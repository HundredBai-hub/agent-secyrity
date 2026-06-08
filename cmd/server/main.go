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

	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
	runtimeSvc "github.com/HundredBai-hub/agent-secyrity/internal/runtime"
	"github.com/HundredBai-hub/agent-secyrity/internal/transport/httpapi"
)

func main() {
	addr := getenv("ADDR", ":8080")
	store := audit.NewMemoryStore()
	packStore := policypack.NewMemoryStore()
	if err := packStore.Upsert(context.Background(), defaultPolicyPack()); err != nil {
		slog.Error("seed default policy pack failed", "error", err)
		os.Exit(1)
	}
	service := runtimeSvc.NewServiceWithPolicyPacks(packStore, store)
	server := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewRouterWithPolicyPacks(service, store, packStore),
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
