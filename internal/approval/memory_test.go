package approval

import (
	"context"
	"testing"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

func TestMemoryStoreCreateListGetAndDecide(t *testing.T) {
	store := NewMemoryStore()
	request, err := store.Create(context.Background(), domain.ApprovalRequest{
		TenantID: "tenant-a",
		Event: domain.RuntimeEvent{
			TenantID:  "tenant-a",
			AgentID:   "agent-code-001",
			UserID:    "user-001",
			TaskID:    "task-001",
			EventType: domain.EventTypeToolCall,
			ToolName:  "shell",
			Action:    "execute",
		},
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		Reason:    "dangerous tool execution requires approval",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if request.ID == "" {
		t.Fatal("Create() returned empty ID")
	}
	if request.Status != domain.ApprovalStatusPending {
		t.Fatalf("Status = %s, want pending", request.Status)
	}

	list, err := store.List(context.Background(), "tenant-a", ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	decided, err := store.Decide(context.Background(), "tenant-a", request.ID, DecisionInput{
		Status:    domain.ApprovalStatusApproved,
		DecidedBy: "secops-001",
		Reason:    "approved for incident response",
		Now:       time.Now(),
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decided.Status != domain.ApprovalStatusApproved {
		t.Fatalf("Status = %s, want approved", decided.Status)
	}
	if decided.DecidedBy != "secops-001" {
		t.Fatalf("DecidedBy = %q", decided.DecidedBy)
	}
}

func TestMemoryStoreRejectsExpiredOrAlreadyDecidedRequests(t *testing.T) {
	store := NewMemoryStore()
	expired, err := store.Create(context.Background(), domain.ApprovalRequest{
		TenantID:  "tenant-a",
		Event:     minimalEvent("tenant-a"),
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		ExpiresAt: time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := store.Decide(context.Background(), "tenant-a", expired.ID, DecisionInput{
		Status:    domain.ApprovalStatusApproved,
		DecidedBy: "secops-001",
		Now:       time.Now(),
	}); err != ErrApprovalExpired {
		t.Fatalf("Decide() error = %v, want ErrApprovalExpired", err)
	}

	pending, err := store.Create(context.Background(), domain.ApprovalRequest{
		TenantID:  "tenant-a",
		Event:     minimalEvent("tenant-a"),
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := store.Decide(context.Background(), "tenant-a", pending.ID, DecisionInput{
		Status:    domain.ApprovalStatusRejected,
		DecidedBy: "secops-001",
		Now:       time.Now(),
	}); err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if _, err := store.Decide(context.Background(), "tenant-a", pending.ID, DecisionInput{
		Status:    domain.ApprovalStatusApproved,
		DecidedBy: "secops-002",
		Now:       time.Now(),
	}); err != ErrApprovalAlreadyDecided {
		t.Fatalf("Decide() error = %v, want ErrApprovalAlreadyDecided", err)
	}
}

func TestMemoryStoreIsolatesTenants(t *testing.T) {
	store := NewMemoryStore()
	request, err := store.Create(context.Background(), domain.ApprovalRequest{
		TenantID:  "tenant-a",
		Event:     minimalEvent("tenant-a"),
		Result:    domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := store.Get(context.Background(), "tenant-b", request.ID); err != ErrNotFound {
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
