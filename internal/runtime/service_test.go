package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/approval"
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
	if records[0].Event.SchemaVersion != domain.RuntimeEventSchemaV1 {
		t.Fatalf("audit schema_version = %q, want %q", records[0].Event.SchemaVersion, domain.RuntimeEventSchemaV1)
	}
}

func TestServiceEvaluateCreatesApprovalRequest(t *testing.T) {
	approvalStore := approval.NewMemoryStore()
	service := NewServiceWithOptions(Options{
		Engine: policy.NewEngine([]domain.Policy{
			{
				ID:       "require-approval-dangerous-tool",
				TenantID: "tenant-a",
				Enabled:  true,
				Priority: 100,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeToolCall},
					ToolNames:  []string{"shell"},
					Actions:    []string{"execute"},
				},
				Decision: domain.DecisionRequireApproval,
				Reason:   "dangerous tool execution requires approval",
			},
		}),
		AuditStore:    audit.NewMemoryStore(),
		ApprovalStore: approvalStore,
		ApprovalTTL:   time.Hour,
	})

	result, err := service.Evaluate(context.Background(), domain.RuntimeEvent{
		TenantID:  "tenant-a",
		AgentID:   "agent-code-001",
		UserID:    "user-001",
		TaskID:    "task-001",
		EventType: domain.EventTypeToolCall,
		ToolName:  "shell",
		Action:    "execute",
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Decision != domain.DecisionRequireApproval {
		t.Fatalf("Decision = %s, want require_approval", result.Decision)
	}
	if result.ApprovalID == "" {
		t.Fatal("ApprovalID is empty")
	}
	request, err := approvalStore.Get(context.Background(), "tenant-a", result.ApprovalID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if request.Status != domain.ApprovalStatusPending {
		t.Fatalf("Status = %s, want pending", request.Status)
	}
}

func TestServiceEvaluateAllowsApprovedMatchingApproval(t *testing.T) {
	approvalStore := approval.NewMemoryStore()
	event := approvalEvent("tenant-a")
	request, err := approvalStore.Create(context.Background(), domain.ApprovalRequest{
		TenantID:  "tenant-a",
		Event:     event,
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := approvalStore.Decide(context.Background(), "tenant-a", request.ID, approval.DecisionInput{
		Status:    domain.ApprovalStatusApproved,
		DecidedBy: "secops-001",
		Now:       time.Now(),
	}); err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	event.ApprovalID = request.ID
	service := NewServiceWithOptions(Options{
		Engine:        policy.NewEngine(requireApprovalPolicies()),
		AuditStore:    audit.NewMemoryStore(),
		ApprovalStore: approvalStore,
	})

	result, err := service.Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Decision != domain.DecisionAllow {
		t.Fatalf("Decision = %s, want allow", result.Decision)
	}
}

func TestServiceEvaluateDeniesMismatchedApproval(t *testing.T) {
	approvalStore := approval.NewMemoryStore()
	request, err := approvalStore.Create(context.Background(), domain.ApprovalRequest{
		TenantID:  "tenant-a",
		Event:     approvalEvent("tenant-a"),
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := approvalStore.Decide(context.Background(), "tenant-a", request.ID, approval.DecisionInput{
		Status:    domain.ApprovalStatusApproved,
		DecidedBy: "secops-001",
		Now:       time.Now(),
	}); err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	event := approvalEvent("tenant-a")
	event.ApprovalID = request.ID
	event.ToolName = "exec"
	service := NewServiceWithOptions(Options{
		Engine:        policy.NewEngine(requireApprovalPolicies()),
		AuditStore:    audit.NewMemoryStore(),
		ApprovalStore: approvalStore,
	})

	result, err := service.Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Decision != domain.DecisionDeny {
		t.Fatalf("Decision = %s, want deny", result.Decision)
	}
}

func TestServiceEvaluateDeniesPendingApproval(t *testing.T) {
	approvalStore := approval.NewMemoryStore()
	event := approvalEvent("tenant-a")
	request, err := approvalStore.Create(context.Background(), domain.ApprovalRequest{
		TenantID:  "tenant-a",
		Event:     event,
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	event.ApprovalID = request.ID
	service := NewServiceWithOptions(Options{
		Engine:        policy.NewEngine(requireApprovalPolicies()),
		AuditStore:    audit.NewMemoryStore(),
		ApprovalStore: approvalStore,
	})

	result, err := service.Evaluate(context.Background(), event)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Decision != domain.DecisionDeny {
		t.Fatalf("Decision = %s, want deny", result.Decision)
	}
}

func approvalEvent(tenantID string) domain.RuntimeEvent {
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

func requireApprovalPolicies() []domain.Policy {
	return []domain.Policy{
		{
			ID:       "require-approval-dangerous-tool",
			TenantID: "tenant-a",
			Enabled:  true,
			Priority: 100,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeToolCall},
				ToolNames:  []string{"shell", "exec"},
				Actions:    []string{"execute"},
			},
			Decision: domain.DecisionRequireApproval,
			Reason:   "dangerous tool execution requires approval",
		},
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
