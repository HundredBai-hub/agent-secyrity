// Package runtime tests runtime decisions against built-in baseline policy packs.
package runtime

import (
	"context"
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/baseline"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
)

// TestBaselinePolicyPacksCoverCommonBusinessScenarios verifies baseline packs through the runtime evaluation path.
func TestBaselinePolicyPacksCoverCommonBusinessScenarios(t *testing.T) {
	t.Parallel()

	packStore := policypack.NewMemoryStore()
	for _, pack := range baseline.DefaultPolicyPacks("tenant-a") {
		if err := packStore.Upsert(context.Background(), pack); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
	}
	service := NewServiceWithPolicyPacks(packStore, audit.NewMemoryStore())

	tests := []struct {
		name     string
		event    domain.RuntimeEvent
		decision domain.Decision
	}{
		{
			name: "code repository agent secret file access is denied",
			event: domain.RuntimeEvent{
				TenantID:   "tenant-a",
				AgentID:    "agent-code-001",
				UserID:     "dev-001",
				TaskID:     "fix-build",
				EventType:  domain.EventTypeFileAccess,
				Resource:   "/repo/.env",
				Action:     "read",
				DataLabels: []string{"secret"},
			},
			decision: domain.DecisionDeny,
		},
		{
			name: "customer support response with pii is redacted",
			event: domain.RuntimeEvent{
				TenantID:   "tenant-a",
				AgentID:    "agent-support-001",
				UserID:     "support-001",
				TaskID:     "ticket-001",
				EventType:  domain.EventTypeResponse,
				Action:     "write",
				DataLabels: []string{"pii"},
			},
			decision: domain.DecisionRedact,
		},
		{
			name: "finance money transfer requires approval",
			event: domain.RuntimeEvent{
				TenantID:  "tenant-a",
				AgentID:   "agent-finance-001",
				UserID:    "finance-001",
				TaskID:    "pay-invoice-001",
				EventType: domain.EventTypeToolCall,
				ToolName:  "wire_transfer",
				Action:    "transfer",
			},
			decision: domain.DecisionRequireApproval,
		},
		{
			name: "data analysis production customer export requires approval",
			event: domain.RuntimeEvent{
				TenantID:   "tenant-a",
				AgentID:    "agent-data-001",
				UserID:     "analyst-001",
				TaskID:     "customer-churn-analysis",
				EventType:  domain.EventTypeToolCall,
				ToolName:   "sql_query",
				Resource:   "production.customer_orders",
				Action:     "export",
				DataLabels: []string{"customer_data"},
			},
			decision: domain.DecisionRequireApproval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Evaluate(context.Background(), tt.event)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if result.Decision != tt.decision {
				t.Fatalf("Decision = %s, want %s", result.Decision, tt.decision)
			}
		})
	}
}
