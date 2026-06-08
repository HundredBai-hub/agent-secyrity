package domain

import (
	"errors"
	"testing"
)

func TestRuntimeEventValidateRequiresCoreFields(t *testing.T) {
	event := RuntimeEvent{
		TenantID:  "tenant-a",
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

func TestRuntimeEventValidateRequiresTenantID(t *testing.T) {
	event := RuntimeEvent{
		AgentID:   "agent-code-001",
		UserID:    "user-001",
		TaskID:    "task-001",
		EventType: EventTypeToolCall,
		ToolName:  "shell",
		Action:    "execute",
	}

	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing tenant_id error")
	}
}

func TestRuntimeEventNormalizeDefaultsSchemaVersion(t *testing.T) {
	event := RuntimeEvent{
		TenantID:  "tenant-a",
		AgentID:   "agent-code-001",
		UserID:    "user-001",
		TaskID:    "task-001",
		EventType: EventTypeToolCall,
		ToolName:  "shell",
		Action:    "execute",
	}

	normalized, err := event.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if normalized.SchemaVersion != RuntimeEventSchemaV1 {
		t.Fatalf("SchemaVersion = %q, want %q", normalized.SchemaVersion, RuntimeEventSchemaV1)
	}
}

func TestRuntimeEventValidateReportsFieldErrors(t *testing.T) {
	event := RuntimeEvent{
		SchemaVersion: "runtime_event.v9",
		EventType:     EventType("unsupported"),
	}

	err := event.Validate()
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validate() error = %T, want *ValidationError", err)
	}

	wantFields := map[string]string{
		"schema_version": "unsupported",
		"tenant_id":      "required",
		"agent_id":       "required",
		"user_id":        "required",
		"task_id":        "required",
		"action":         "required",
		"event_type":     "unsupported",
	}
	for field, code := range wantFields {
		if !validationErr.HasField(field, code) {
			t.Fatalf("Validate() missing field error %s/%s in %+v", field, code, validationErr.Fields)
		}
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
