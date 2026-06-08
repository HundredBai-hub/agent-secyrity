package policy

import (
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

func TestEngineEvaluateReturnsHighestPriorityDecision(t *testing.T) {
	engine := NewEngine([]domain.Policy{
		{
			ID:       "record-all-file-access",
			TenantID: "tenant-a",
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
			TenantID: "tenant-a",
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
		TenantID:   "tenant-a",
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
		TenantID:  "tenant-a",
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

func TestEngineEvaluateIsolatesPoliciesByTenant(t *testing.T) {
	engine := NewEngine([]domain.Policy{
		{
			ID:       "tenant-a-deny-shell",
			TenantID: "tenant-a",
			Enabled:  true,
			Priority: 100,
			Conditions: domain.PolicyConditions{
				EventTypes: []domain.EventType{domain.EventTypeToolCall},
				ToolNames:  []string{"shell"},
			},
			Decision: domain.DecisionDeny,
			Reason:   "tenant-a blocks shell",
		},
	})

	result := engine.Evaluate(domain.RuntimeEvent{
		TenantID:  "tenant-b",
		AgentID:   "agent-code-001",
		UserID:    "user-001",
		TaskID:    "task-001",
		EventType: domain.EventTypeToolCall,
		ToolName:  "shell",
		Action:    "execute",
	})

	if result.Decision != domain.DecisionAllow {
		t.Fatalf("Decision = %s, want allow for different tenant", result.Decision)
	}
}

func TestEngineEvaluateMatchesSubjectRoles(t *testing.T) {
	engine := NewEngine([]domain.Policy{
		{
			ID:       "support-response-redaction",
			TenantID: "tenant-a",
			Enabled:  true,
			Priority: 100,
			Conditions: domain.PolicyConditions{
				EventTypes:   []domain.EventType{domain.EventTypeResponse},
				SubjectRoles: []string{"support"},
				DataLabels:   []string{"pii"},
			},
			Decision: domain.DecisionRedact,
			Reason:   "support responses with pii must be redacted",
		},
	})

	result := engine.Evaluate(domain.RuntimeEvent{
		TenantID:   "tenant-a",
		AgentID:    "agent-support-001",
		UserID:     "user-001",
		TaskID:     "ticket-001",
		EventType:  domain.EventTypeResponse,
		Action:     "write",
		DataLabels: []string{"pii"},
		Subject: domain.Subject{
			Type:  domain.SubjectTypeUser,
			ID:    "user-001",
			Roles: []string{"support"},
		},
	})

	if result.Decision != domain.DecisionRedact {
		t.Fatalf("Decision = %s, want redact", result.Decision)
	}
}

func TestEngineFromPolicyPacksSkipsDisabledPacks(t *testing.T) {
	engine := NewEngineFromPacks([]domain.PolicyPack{
		{
			ID:       "pack-disabled",
			TenantID: "tenant-a",
			Version:  "1.0.0",
			Enabled:  false,
			Policies: []domain.Policy{
				{
					ID:       "deny-shell",
					Enabled:  true,
					Priority: 100,
					Conditions: domain.PolicyConditions{
						EventTypes: []domain.EventType{domain.EventTypeToolCall},
						ToolNames:  []string{"shell"},
					},
					Decision: domain.DecisionDeny,
				},
			},
		},
	})

	result := engine.Evaluate(domain.RuntimeEvent{
		TenantID:  "tenant-a",
		AgentID:   "agent-code-001",
		UserID:    "user-001",
		TaskID:    "task-001",
		EventType: domain.EventTypeToolCall,
		ToolName:  "shell",
		Action:    "execute",
	})

	if result.Decision != domain.DecisionAllow {
		t.Fatalf("Decision = %s, want allow because pack is disabled", result.Decision)
	}
}
