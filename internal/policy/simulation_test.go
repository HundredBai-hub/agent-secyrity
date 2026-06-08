package policy

import (
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

func TestSimulateReturnsDecisionFromCandidatePolicyPacks(t *testing.T) {
	t.Parallel()

	result, err := Simulate(SimulationRequest{
		Event: shellEvent("tenant-a"),
		PolicyPacks: []domain.PolicyPack{
			{
				ID:       "candidate-runtime",
				TenantID: "tenant-a",
				Enabled:  true,
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
						Reason:   "shell is blocked",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}
	if result.SchemaVersion != PolicySimulationSchemaV1 {
		t.Fatalf("SchemaVersion = %q, want %q", result.SchemaVersion, PolicySimulationSchemaV1)
	}
	if result.Result.Decision != domain.DecisionDeny {
		t.Fatalf("Decision = %s, want deny", result.Result.Decision)
	}
	if len(result.Result.MatchedPolicyIDs) != 1 || result.Result.MatchedPolicyIDs[0] != "deny-shell" {
		t.Fatalf("MatchedPolicyIDs = %+v, want deny-shell", result.Result.MatchedPolicyIDs)
	}
}

func TestSimulateSkipsDisabledPolicyPacks(t *testing.T) {
	t.Parallel()

	result, err := Simulate(SimulationRequest{
		Event: shellEvent("tenant-a"),
		PolicyPacks: []domain.PolicyPack{
			{
				ID:       "disabled",
				TenantID: "tenant-a",
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
		},
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}
	if result.Result.Decision != domain.DecisionAllow {
		t.Fatalf("Decision = %s, want allow", result.Result.Decision)
	}
}

func TestSimulateValidatesRuntimeEvent(t *testing.T) {
	t.Parallel()

	_, err := Simulate(SimulationRequest{
		Event: domain.RuntimeEvent{SchemaVersion: "runtime_event.v9"},
	})
	if err == nil {
		t.Fatalf("Simulate() error = nil, want validation error")
	}
}

func shellEvent(tenantID string) domain.RuntimeEvent {
	return domain.RuntimeEvent{
		SchemaVersion: domain.RuntimeEventSchemaV1,
		TenantID:      tenantID,
		AgentID:       "agent-code-001",
		UserID:        "user-001",
		TaskID:        "task-001",
		EventType:     domain.EventTypeToolCall,
		ToolName:      "shell",
		Action:        "execute",
	}
}
