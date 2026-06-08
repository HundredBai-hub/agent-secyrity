package policy

import (
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

func TestEngineEvaluateReturnsHighestPriorityDecision(t *testing.T) {
	engine := NewEngine([]domain.Policy{
		{
			ID:       "record-all-file-access",
			Name:     "Record file access",
			Enabled:  true,
			Priority: 10,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeFileAccess},
			},
			Decision: domain.DecisionRecord,
			Reason:   "record file access",
		},
		{
			ID:       "deny-secret-file-access",
			Name:     "Deny secret file access",
			Enabled:  true,
			Priority: 100,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeFileAccess},
				DataLabels: []string{"secret"},
			},
			Decision: domain.DecisionDeny,
			Reason:   "secret file access is blocked",
		},
	})

	result := engine.Evaluate(domain.RuntimeEvent{
		AgentID:    "agent-code-001",
		UserID:     "user-001",
		TaskID:     "task-001",
		EventType:  domain.EventTypeFileAccess,
		Resource:   "/repo/.env",
		Action:     "read",
		DataLabels: []string{"secret"},
	})

	if result.Decision != domain.DecisionDeny {
		t.Fatalf("Decision = %s, want %s", result.Decision, domain.DecisionDeny)
	}
	if len(result.MatchedPolicyIDs) != 2 {
		t.Fatalf("MatchedPolicyIDs = %v, want 2 matches", result.MatchedPolicyIDs)
	}
	if result.Reason != "secret file access is blocked" {
		t.Fatalf("Reason = %q", result.Reason)
	}
}

func TestEngineEvaluateDefaultsToAllow(t *testing.T) {
	result := NewEngine(nil).Evaluate(domain.RuntimeEvent{
		AgentID:   "agent-code-001",
		UserID:    "user-001",
		TaskID:    "task-001",
		EventType: domain.EventTypeToolCall,
		ToolName:  "read_file",
		Action:    "execute",
	})

	if result.Decision != domain.DecisionAllow {
		t.Fatalf("Decision = %s, want allow", result.Decision)
	}
}
