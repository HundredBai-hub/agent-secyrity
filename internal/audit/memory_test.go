package audit

import (
	"context"
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

func TestMemoryStoreAppendAndList(t *testing.T) {
	store := NewMemoryStore()
	record := domain.AuditRecord{
		Event: domain.RuntimeEvent{
			TenantID:  "tenant-a",
			AgentID:   "agent-code-001",
			UserID:    "user-001",
			TaskID:    "task-001",
			EventType: domain.EventTypeToolCall,
			ToolName:  "shell",
			Action:    "execute",
		},
		Result: domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
	}

	saved, err := store.Append(context.Background(), record)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if saved.ID == "" {
		t.Fatal("Append() returned empty ID")
	}

	records, err := store.List(context.Background(), ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if records[0].Result.Decision != domain.DecisionRequireApproval {
		t.Fatalf("Decision = %s", records[0].Result.Decision)
	}
}
