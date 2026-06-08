package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type EventType string

const (
	RuntimeEventSchemaV1             = "runtime_event.v1"
	EventTypePrompt        EventType = "prompt"
	EventTypeToolCall      EventType = "tool_call"
	EventTypeFileAccess    EventType = "file_access"
	EventTypeNetworkAccess EventType = "network_access"
	EventTypeResponse      EventType = "response"
	EventTypeApproval      EventType = "approval"
)

type Decision string

const (
	DecisionAllow           Decision = "allow"
	DecisionRecord          Decision = "record"
	DecisionRedact          Decision = "redact"
	DecisionRequireApproval Decision = "require_approval"
	DecisionDeny            Decision = "deny"
)

func (d Decision) Priority() int {
	switch d {
	case DecisionDeny:
		return 500
	case DecisionRequireApproval:
		return 400
	case DecisionRedact:
		return 300
	case DecisionRecord:
		return 200
	case DecisionAllow:
		return 100
	default:
		return 0
	}
}

type RuntimeEvent struct {
	SchemaVersion string    `json:"schema_version,omitempty"`
	ID            string    `json:"event_id,omitempty"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
	TenantID      string    `json:"tenant_id"`
	AgentID       string    `json:"agent_id"`
	UserID        string    `json:"user_id"`
	TaskID        string    `json:"task_id"`
	EventType     EventType `json:"event_type"`
	Subject       Subject   `json:"subject,omitempty"`
	ApprovalID    string    `json:"approval_id,omitempty"`
	ToolName      string    `json:"tool_name,omitempty"`
	Resource      string    `json:"resource,omitempty"`
	Action        string    `json:"action"`
	Input         string    `json:"input,omitempty"`
	Output        string    `json:"output,omitempty"`
	DataLabels    []string  `json:"data_labels,omitempty"`
	Intent        string    `json:"intent,omitempty"`
}

func (e RuntimeEvent) Normalize() (RuntimeEvent, error) {
	if strings.TrimSpace(e.SchemaVersion) == "" {
		e.SchemaVersion = RuntimeEventSchemaV1
	}
	if err := e.Validate(); err != nil {
		return RuntimeEvent{}, err
	}
	return e, nil
}

func (e RuntimeEvent) Validate() error {
	var fieldErrors []FieldError
	if strings.TrimSpace(e.SchemaVersion) != "" && e.SchemaVersion != RuntimeEventSchemaV1 {
		fieldErrors = append(fieldErrors, FieldError{Field: "schema_version", Code: "unsupported", Message: "schema_version is unsupported"})
	}
	if strings.TrimSpace(e.TenantID) == "" {
		fieldErrors = append(fieldErrors, FieldError{Field: "tenant_id", Code: "required", Message: "tenant_id is required"})
	}
	if strings.TrimSpace(e.AgentID) == "" {
		fieldErrors = append(fieldErrors, FieldError{Field: "agent_id", Code: "required", Message: "agent_id is required"})
	}
	if strings.TrimSpace(e.UserID) == "" {
		fieldErrors = append(fieldErrors, FieldError{Field: "user_id", Code: "required", Message: "user_id is required"})
	}
	if strings.TrimSpace(e.TaskID) == "" {
		fieldErrors = append(fieldErrors, FieldError{Field: "task_id", Code: "required", Message: "task_id is required"})
	}
	if strings.TrimSpace(string(e.EventType)) == "" {
		fieldErrors = append(fieldErrors, FieldError{Field: "event_type", Code: "required", Message: "event_type is required"})
	}
	if strings.TrimSpace(e.Action) == "" {
		fieldErrors = append(fieldErrors, FieldError{Field: "action", Code: "required", Message: "action is required"})
	}
	if strings.TrimSpace(string(e.EventType)) != "" && !e.EventType.Valid() {
		fieldErrors = append(fieldErrors, FieldError{Field: "event_type", Code: "unsupported", Message: fmt.Sprintf("event_type %q is unsupported", e.EventType)})
	}
	if len(fieldErrors) > 0 {
		return &ValidationError{Message: "invalid runtime event", Fields: fieldErrors}
	}
	return nil
}

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationError struct {
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields"`
}

func (e *ValidationError) Error() string {
	if e == nil || e.Message == "" {
		return "validation failed"
	}
	return e.Message
}

func (e *ValidationError) HasField(field string, code string) bool {
	if e == nil {
		return false
	}
	for _, fieldError := range e.Fields {
		if fieldError.Field == field && fieldError.Code == code {
			return true
		}
	}
	return false
}

type SubjectType string

const (
	SubjectTypeUser           SubjectType = "user"
	SubjectTypeAgent          SubjectType = "agent"
	SubjectTypeServiceAccount SubjectType = "service_account"
	SubjectTypeWorkflow       SubjectType = "workflow"
)

type Subject struct {
	Type      SubjectType `json:"type,omitempty"`
	ID        string      `json:"id,omitempty"`
	Roles     []string    `json:"roles,omitempty"`
	Groups    []string    `json:"groups,omitempty"`
	RiskLevel string      `json:"risk_level,omitempty"`
}

func (t EventType) Valid() bool {
	switch t {
	case EventTypePrompt, EventTypeToolCall, EventTypeFileAccess, EventTypeNetworkAccess, EventTypeResponse, EventTypeApproval:
		return true
	default:
		return false
	}
}

type Policy struct {
	ID           string           `json:"id"`
	TenantID     string           `json:"tenant_id,omitempty"`
	PolicyPackID string           `json:"policy_pack_id,omitempty"`
	Name         string           `json:"name,omitempty"`
	Enabled      bool             `json:"enabled"`
	Priority     int              `json:"priority"`
	Conditions   PolicyConditions `json:"conditions"`
	Decision     Decision         `json:"decision"`
	Reason       string           `json:"reason,omitempty"`
}

type PolicyConditions struct {
	EventTypes        []EventType   `json:"event_types,omitempty"`
	ToolNames         []string      `json:"tool_names,omitempty"`
	Resources         []string      `json:"resources,omitempty"`
	Actions           []string      `json:"actions,omitempty"`
	DataLabels        []string      `json:"data_labels,omitempty"`
	AgentIDs          []string      `json:"agent_ids,omitempty"`
	UserIDs           []string      `json:"user_ids,omitempty"`
	SubjectTypes      []SubjectType `json:"subject_types,omitempty"`
	SubjectIDs        []string      `json:"subject_ids,omitempty"`
	SubjectRoles      []string      `json:"subject_roles,omitempty"`
	SubjectGroups     []string      `json:"subject_groups,omitempty"`
	SubjectRiskLevels []string      `json:"subject_risk_levels,omitempty"`
}

type PolicyPack struct {
	ID       string   `json:"id"`
	TenantID string   `json:"tenant_id"`
	Name     string   `json:"name,omitempty"`
	Version  string   `json:"version,omitempty"`
	Enabled  bool     `json:"enabled"`
	Policies []Policy `json:"policies"`
}

type EvaluationResult struct {
	Decision         Decision `json:"decision"`
	Reason           string   `json:"reason"`
	MatchedPolicyIDs []string `json:"matched_policy_ids"`
	AuditID          string   `json:"audit_id,omitempty"`
	ApprovalID       string   `json:"approval_id,omitempty"`
}

type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

type ApprovalRequest struct {
	ID             string           `json:"id"`
	TenantID       string           `json:"tenant_id"`
	Status         ApprovalStatus   `json:"status"`
	Event          RuntimeEvent     `json:"event"`
	Result         EvaluationResult `json:"result"`
	Reason         string           `json:"reason,omitempty"`
	RequestedAt    time.Time        `json:"requested_at"`
	ExpiresAt      time.Time        `json:"expires_at"`
	DecidedAt      time.Time        `json:"decided_at,omitempty"`
	DecidedBy      string           `json:"decided_by,omitempty"`
	DecisionReason string           `json:"decision_reason,omitempty"`
}

type AuditRecord struct {
	ID         string           `json:"id"`
	RecordedAt time.Time        `json:"recorded_at"`
	Event      RuntimeEvent     `json:"event"`
	Result     EvaluationResult `json:"result"`
}

var ErrInvalidRuntimeEvent = errors.New("invalid runtime event")

type RuntimeEventError struct {
	Err error
}

func (e *RuntimeEventError) Error() string {
	if e == nil || e.Err == nil {
		return ErrInvalidRuntimeEvent.Error()
	}
	return fmt.Sprintf("%s: %v", ErrInvalidRuntimeEvent, e.Err)
}

func (e *RuntimeEventError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *RuntimeEventError) Is(target error) bool {
	return target == ErrInvalidRuntimeEvent
}
