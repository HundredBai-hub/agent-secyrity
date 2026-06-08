package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/approval"
	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/auth"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
	runtimeSvc "github.com/HundredBai-hub/agent-secyrity/internal/runtime"
	postgresStore "github.com/HundredBai-hub/agent-secyrity/internal/storage/postgres"
	"github.com/HundredBai-hub/agent-secyrity/internal/transport/httpapi"
)

func main() {
	addr := getenv("ADDR", ":8080")
	stores := initStores(context.Background())
	defer stores.Close()
	if err := stores.PolicyPacks.Upsert(context.Background(), defaultPolicyPack()); err != nil {
		slog.Error("seed default policy pack failed", "error", err)
		os.Exit(1)
	}
	service := runtimeSvc.NewServiceWithOptions(runtimeSvc.Options{
		PolicyPacks:   stores.PolicyPacks,
		AuditStore:    stores.Audit,
		ApprovalStore: stores.Approvals,
		ApprovalTTL:   15 * time.Minute,
	})
	apiKeys, err := auth.ParseAPIKeys(os.Getenv("AGENT_SECURITY_API_KEYS"))
	if err != nil {
		slog.Error("parse API key configuration failed", "error", err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewRouterWithOptions(httpapi.Options{Service: service, Audit: stores.Audit, PolicyPacks: stores.PolicyPacks, Approvals: stores.Approvals, APIKeys: apiKeys}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("agent security server listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
}

type stores struct {
	Audit       audit.Store
	PolicyPacks policypack.Store
	Approvals   approval.Store
	Close       func()
}

func initStores(ctx context.Context) stores {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return stores{
			Audit:       audit.NewMemoryStore(),
			PolicyPacks: policypack.NewMemoryStore(),
			Approvals:   approval.NewMemoryStore(),
			Close:       func() {},
		}
	}
	db, err := postgresStore.Open(ctx, dsn)
	if err != nil {
		slog.Error("open postgres failed", "error", err)
		os.Exit(1)
	}
	if err := postgresStore.Migrate(ctx, db); err != nil {
		_ = db.Close()
		slog.Error("migrate postgres failed", "error", err)
		os.Exit(1)
	}
	return stores{
		Audit:       postgresStore.NewAuditStore(db),
		PolicyPacks: postgresStore.NewPolicyPackStore(db),
		Approvals:   postgresStore.NewApprovalStore(db),
		Close: func() {
			if err := db.Close(); err != nil {
				slog.Error("close postgres failed", "error", err)
			}
		},
	}
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func defaultPolicyPack() domain.PolicyPack {
	return domain.PolicyPack{
		ID:       "default-runtime",
		TenantID: "default",
		Name:     "Default Runtime",
		Version:  "1.0.0",
		Enabled:  true,
		Policies: []domain.Policy{
			{
				ID:       "deny-secret-file-access",
				TenantID: "default",
				Name:     "Deny secret file access",
				Enabled:  true,
				Priority: 100,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeFileAccess},
					Resources:  []string{".env", "id_rsa", "id_ed25519", ".ssh/"},
					DataLabels: []string{"secret"},
				},
				Decision: domain.DecisionDeny,
				Reason:   "secret file access is blocked",
			},
			{
				ID:       "require-approval-dangerous-tool",
				TenantID: "default",
				Name:     "Require approval for dangerous tools",
				Enabled:  true,
				Priority: 90,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeToolCall},
					ToolNames:  []string{"shell", "exec", "terminal"},
					Actions:    []string{"execute"},
				},
				Decision: domain.DecisionRequireApproval,
				Reason:   "dangerous tool execution requires approval",
			},
			{
				ID:       "redact-sensitive-response",
				TenantID: "default",
				Name:     "Redact sensitive responses",
				Enabled:  true,
				Priority: 80,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeResponse},
					DataLabels: []string{"pii", "secret", "customer_data"},
				},
				Decision: domain.DecisionRedact,
				Reason:   "sensitive response must be redacted",
			},
		},
	}
}
