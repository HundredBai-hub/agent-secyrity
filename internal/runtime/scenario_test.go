package runtime

import (
	"context"
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policy"
)

func TestProductionScenarios(t *testing.T) {
	service := NewService(policy.NewEngine([]domain.Policy{
		{
			ID:       "deny-secret-file-access",
			Enabled:  true,
			Priority: 100,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeFileAccess},
				Resources:  []string{".env"},
				DataLabels: []string{"secret"},
			},
			Decision: domain.DecisionDeny,
			Reason:   "secret file access is blocked",
		},
		{
			ID:       "require-approval-dangerous-tool",
			Enabled:  true,
			Priority: 90,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeToolCall},
				ToolNames:  []string{"shell"},
				Actions:    []string{"execute"},
			},
			Decision: domain.DecisionRequireApproval,
			Reason:   "dangerous tool execution requires approval",
		},
		{
			ID:       "redact-sensitive-response",
			Enabled:  true,
			Priority: 80,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeResponse},
				DataLabels: []string{"pii"},
			},
			Decision: domain.DecisionRedact,
			Reason:   "sensitive response must be redacted",
		},
	}), audit.NewMemoryStore())

	tests := []struct {
		name  string
		event domain.RuntimeEvent
		want  domain.Decision
	}{
		{
			name: "blocks sensitive file reads",
			event: domain.RuntimeEvent{
				AgentID:    "agent-code-001",
				UserID:     "dev-001",
				TaskID:     "fix-build",
				EventType:  domain.EventTypeFileAccess,
				Resource:   "/repo/.env",
				Action:     "read",
				DataLabels: []string{"secret"},
			},
			want: domain.DecisionDeny,
		},
		{
			name: "requires approval for dangerous tool calls",
			event: domain.RuntimeEvent{
				AgentID:   "agent-code-001",
				UserID:    "dev-001",
				TaskID:    "fix-build",
				EventType: domain.EventTypeToolCall,
				ToolName:  "shell",
				Action:    "execute",
				Intent:    "run migration",
			},
			want: domain.DecisionRequireApproval,
		},
		{
			name: "redacts sensitive customer response",
			event: domain.RuntimeEvent{
				AgentID:    "agent-support-001",
				UserID:     "support-001",
				TaskID:     "ticket-001",
				EventType:  domain.EventTypeResponse,
				Action:     "write",
				DataLabels: []string{"pii"},
			},
			want: domain.DecisionRedact,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Evaluate(context.Background(), tt.event)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if result.Decision != tt.want {
				t.Fatalf("Decision = %s, want %s", result.Decision, tt.want)
			}
			if result.AuditID == "" {
				t.Fatal("AuditID is empty")
			}
		})
	}
}
