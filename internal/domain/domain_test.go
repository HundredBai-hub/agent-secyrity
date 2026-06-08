package domain

import "testing"

func TestRuntimeEventValidateRequiresCoreFields(t *testing.T) {
	event := RuntimeEvent{
		AgentID:   "agent-code-001",
		UserID:    "user-001",
		TaskID:    "task-001",
		EventType: EventTypeFileAccess,
		Resource:  "/repo/.env",
		Action:    "read",
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	event.AgentID = ""
	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing agent_id error")
	}
}

func TestDecisionPriority(t *testing.T) {
	if DecisionDeny.Priority() <= DecisionRequireApproval.Priority() {
		t.Fatal("deny must have higher priority than require_approval")
	}
	if DecisionAllow.Priority() >= DecisionRecord.Priority() {
		t.Fatal("allow must have lower priority than record")
	}
}
