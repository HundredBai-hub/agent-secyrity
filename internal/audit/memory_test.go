// Package audit tests audit store query behavior.
package audit

import (
	"context"
	"testing"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

// TestMemoryStoreAppendAndList verifies basic append and list behavior.
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

// TestMemoryStoreListFiltersByTenant verifies tenant-scoped audit listing.
func TestMemoryStoreListFiltersByTenant(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if _, err := store.Append(ctx, domain.AuditRecord{
		Event:  domain.RuntimeEvent{TenantID: "tenant-a", EventType: domain.EventTypeToolCall},
		Result: domain.EvaluationResult{Decision: domain.DecisionAllow},
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if _, err := store.Append(ctx, domain.AuditRecord{
		Event:  domain.RuntimeEvent{TenantID: "tenant-b", EventType: domain.EventTypeToolCall},
		Result: domain.EvaluationResult{Decision: domain.DecisionDeny},
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	records, err := store.List(ctx, ListOptions{Limit: 10, TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if records[0].Event.TenantID != "tenant-a" {
		t.Fatalf("tenant_id = %s, want tenant-a", records[0].Event.TenantID)
	}
}

// TestMemoryStoreListFiltersByActorTaskDecisionAndEvent verifies combined business filters.
func TestMemoryStoreListFiltersByActorTaskDecisionAndEvent(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	records := []domain.AuditRecord{
		{
			ID:         "audit-1",
			RecordedAt: time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC),
			Event: domain.RuntimeEvent{
				TenantID:  "tenant-a",
				AgentID:   "agent-code-001",
				UserID:    "dev-001",
				TaskID:    "task-build",
				EventType: domain.EventTypeToolCall,
			},
			Result: domain.EvaluationResult{Decision: domain.DecisionRequireApproval},
		},
		{
			ID:         "audit-2",
			RecordedAt: time.Date(2026, 6, 8, 10, 1, 0, 0, time.UTC),
			Event: domain.RuntimeEvent{
				TenantID:  "tenant-a",
				AgentID:   "agent-code-001",
				UserID:    "dev-001",
				TaskID:    "task-build",
				EventType: domain.EventTypeFileAccess,
			},
			Result: domain.EvaluationResult{Decision: domain.DecisionDeny},
		},
		{
			ID:         "audit-3",
			RecordedAt: time.Date(2026, 6, 8, 10, 2, 0, 0, time.UTC),
			Event: domain.RuntimeEvent{
				TenantID:  "tenant-a",
				AgentID:   "agent-support-001",
				UserID:    "support-001",
				TaskID:    "ticket-001",
				EventType: domain.EventTypeResponse,
			},
			Result: domain.EvaluationResult{Decision: domain.DecisionRedact},
		},
	}
	for _, record := range records {
		if _, err := store.Append(ctx, record); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got, err := store.List(ctx, ListOptions{
		Limit:     10,
		TenantID:  "tenant-a",
		AgentID:   "agent-code-001",
		UserID:    "dev-001",
		TaskID:    "task-build",
		Decision:  domain.DecisionDeny,
		EventType: domain.EventTypeFileAccess,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(got))
	}
	if got[0].ID != "audit-2" {
		t.Fatalf("record id = %s, want audit-2", got[0].ID)
	}
}

// TestMemoryStoreListUsesNewestFirstOffsetAndLimit verifies stable pagination semantics.
func TestMemoryStoreListUsesNewestFirstOffsetAndLimit(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	for i, id := range []string{"audit-1", "audit-2", "audit-3"} {
		if _, err := store.Append(ctx, domain.AuditRecord{
			ID:         id,
			RecordedAt: time.Date(2026, 6, 8, 10, i, 0, 0, time.UTC),
			Event:      domain.RuntimeEvent{TenantID: "tenant-a", EventType: domain.EventTypeToolCall},
			Result:     domain.EvaluationResult{Decision: domain.DecisionAllow},
		}); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got, err := store.List(ctx, ListOptions{TenantID: "tenant-a", Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(got))
	}
	if got[0].ID != "audit-2" {
		t.Fatalf("record id = %s, want second newest audit-2", got[0].ID)
	}
}
