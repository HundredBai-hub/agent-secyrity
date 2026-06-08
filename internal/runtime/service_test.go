package runtime

import (
	"context"
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policy"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
)

func TestServiceEvaluateStoresAuditRecord(t *testing.T) {
	store := audit.NewMemoryStore()
	service := NewService(policy.NewEngine([]domain.Policy{
		{
			ID:       "deny-secret-file-access",
			TenantID: "tenant-a",
			Enabled:  true,
			Priority: 100,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeFileAccess},
				DataLabels: []string{"secret"},
			},
			Decision: domain.DecisionDeny,
			Reason:   "secret file access is blocked",
		},
	}), store)

	result, err := service.Evaluate(context.Background(), domain.RuntimeEvent{
		TenantID:   "tenant-a",
		AgentID:    "agent-code-001",
		UserID:     "user-001",
		TaskID:     "task-001",
		EventType:  domain.EventTypeFileAccess,
		Resource:   "/repo/.env",
		Action:     "read",
		DataLabels: []string{"secret"},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Decision != domain.DecisionDeny {
		t.Fatalf("Decision = %s, want deny", result.Decision)
	}
	if result.AuditID == "" {
		t.Fatal("AuditID is empty")
	}

	records, err := store.List(context.Background(), audit.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
}

func TestServiceEvaluateLoadsEnabledPolicyPacks(t *testing.T) {
	store := audit.NewMemoryStore()
	packStore := policypack.NewMemoryStore()
	if err := packStore.Upsert(context.Background(), domain.PolicyPack{
		ID:       "default-runtime",
		TenantID: "tenant-a",
		Enabled:  true,
		Policies: []domain.Policy{
			{
				ID:       "deny-secret-file-access",
				Enabled:  true,
				Priority: 100,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeFileAccess},
					DataLabels: []string{"secret"},
				},
				Decision: domain.DecisionDeny,
				Reason:   "secret file access is blocked",
			},
		},
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	service := NewServiceWithPolicyPacks(packStore, store)

	result, err := service.Evaluate(context.Background(), domain.RuntimeEvent{
		TenantID:   "tenant-a",
		AgentID:    "agent-code-001",
		UserID:     "user-001",
		TaskID:     "task-001",
		EventType:  domain.EventTypeFileAccess,
		Resource:   "/repo/.env",
		Action:     "read",
		DataLabels: []string{"secret"},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Decision != domain.DecisionDeny {
		t.Fatalf("Decision = %s, want deny", result.Decision)
	}

	if err := packStore.SetEnabled(context.Background(), "tenant-a", "default-runtime", false); err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}
	result, err = service.Evaluate(context.Background(), domain.RuntimeEvent{
		TenantID:   "tenant-a",
		AgentID:    "agent-code-001",
		UserID:     "user-001",
		TaskID:     "task-001",
		EventType:  domain.EventTypeFileAccess,
		Resource:   "/repo/.env",
		Action:     "read",
		DataLabels: []string{"secret"},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Decision != domain.DecisionAllow {
		t.Fatalf("Decision = %s, want allow after pack disabled", result.Decision)
	}
}
