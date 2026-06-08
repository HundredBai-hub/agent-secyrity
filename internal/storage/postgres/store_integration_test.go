package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/approval"
	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
)

func TestIntegrationStoresAuditAndPolicyPacks(t *testing.T) {
	dsn := os.Getenv("AGENT_SECURITY_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("AGENT_SECURITY_POSTGRES_TEST_DSN is not set")
	}
	ctx := context.Background()
	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	auditStore := NewAuditStore(db)
	record, err := auditStore.Append(ctx, domain.AuditRecord{
		Event: domain.RuntimeEvent{
			TenantID:  "tenant-pg",
			AgentID:   "agent-code-001",
			UserID:    "user-001",
			TaskID:    "task-001",
			EventType: domain.EventTypeToolCall,
			ToolName:  "shell",
			Action:    "execute",
		},
		Result: domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if record.ID == "" {
		t.Fatal("Append() returned empty ID")
	}
	if _, err := auditStore.Append(ctx, domain.AuditRecord{
		Event: domain.RuntimeEvent{
			TenantID:  "tenant-other",
			AgentID:   "agent-code-001",
			UserID:    "user-001",
			TaskID:    "task-001",
			EventType: domain.EventTypeToolCall,
			ToolName:  "shell",
			Action:    "execute",
		},
		Result: domain.EvaluationResult{Decision: domain.DecisionDeny},
	}); err != nil {
		t.Fatalf("Append(other tenant) error = %v", err)
	}
	records, err := auditStore.List(ctx, audit.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) == 0 {
		t.Fatal("List() returned no audit records")
	}
	tenantRecords, err := auditStore.List(ctx, audit.ListOptions{Limit: 10, TenantID: "tenant-pg"})
	if err != nil {
		t.Fatalf("List(tenant) error = %v", err)
	}
	if len(tenantRecords) == 0 {
		t.Fatal("List(tenant) returned no audit records")
	}
	for _, tenantRecord := range tenantRecords {
		if tenantRecord.Event.TenantID != "tenant-pg" {
			t.Fatalf("tenant record = %s, want tenant-pg", tenantRecord.Event.TenantID)
		}
	}

	packStore := NewPolicyPackStore(db)
	pack := domain.PolicyPack{
		ID:       "runtime-pg",
		TenantID: "tenant-pg",
		Version:  "1.0.0",
		Enabled:  true,
		Policies: []domain.Policy{{ID: "record-all", Enabled: true, Decision: domain.DecisionRecord}},
	}
	if err := packStore.Upsert(ctx, pack); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	got, err := packStore.Get(ctx, "tenant-pg", "runtime-pg")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != pack.ID || got.TenantID != pack.TenantID {
		t.Fatalf("Get() = %#v", got)
	}
	if err := packStore.SetEnabled(ctx, "tenant-pg", "runtime-pg", false); err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}
	enabled, err := packStore.ListEnabled(ctx, "tenant-pg")
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}
	if len(enabled) != 0 {
		t.Fatalf("len(enabled) = %d, want 0", len(enabled))
	}
	if _, err := packStore.Get(ctx, "tenant-other", "runtime-pg"); err == nil {
		t.Fatal("Get() error = nil, want tenant isolation not found")
	} else if err != policypack.ErrNotFound {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}

	approvalStore := NewApprovalStore(db)
	approvalRequest, err := approvalStore.Create(ctx, domain.ApprovalRequest{
		TenantID:  "tenant-pg",
		Event:     minimalEvent("tenant-pg"),
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		Reason:    "dangerous tool execution requires approval",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	decided, err := approvalStore.Decide(ctx, "tenant-pg", approvalRequest.ID, approval.DecisionInput{
		Status:    domain.ApprovalStatusApproved,
		DecidedBy: "secops-001",
		Reason:    "approved",
		Now:       time.Now(),
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decided.Status != domain.ApprovalStatusApproved {
		t.Fatalf("Status = %s, want approved", decided.Status)
	}
	if _, err := approvalStore.Get(ctx, "tenant-other", approvalRequest.ID); err == nil {
		t.Fatal("Get() error = nil, want tenant isolation not found")
	} else if err != approval.ErrNotFound {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func minimalEvent(tenantID string) domain.RuntimeEvent {
	return domain.RuntimeEvent{
		TenantID:  tenantID,
		AgentID:   "agent-code-001",
		UserID:    "user-001",
		TaskID:    "task-001",
		EventType: domain.EventTypeToolCall,
		ToolName:  "shell",
		Action:    "execute",
	}
}
