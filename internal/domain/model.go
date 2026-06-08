package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type EventType string

const (
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
	ID         string    `json:"event_id,omitempty"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
	TenantID   string    `json:"tenant_id"`
	AgentID    string    `json:"agent_id"`
	UserID     string    `json:"user_id"`
	TaskID     string    `json:"task_id"`
	EventType  EventType `json:"event_type"`
	Subject    Subject   `json:"subject,omitempty"`
	ToolName   string    `json:"tool_name,omitempty"`
	Resource   string    `json:"resource,omitempty"`
	Action     string    `json:"action"`
	Input      string    `json:"input,omitempty"`
	Output     string    `json:"output,omitempty"`
	DataLabels []string  `json:"data_labels,omitempty"`
	Intent     string    `json:"intent,omitempty"`
}

func (e RuntimeEvent) Validate() error {
	var missing []string
	if strings.TrimSpace(e.TenantID) == "" {
		missing = append(missing, "tenant_id")
	}
	if strings.TrimSpace(e.AgentID) == "" {
		missing = append(missing, "agent_id")
	}
	if strings.TrimSpace(e.UserID) == "" {
		missing = append(missing, "user_id")
	}
	if strings.TrimSpace(e.TaskID) == "" {
		missing = append(missing, "task_id")
	}
	if strings.TrimSpace(string(e.EventType)) == "" {
		missing = append(missing, "event_type")
	}
	if strings.TrimSpace(e.Action) == "" {
		missing = append(missing, "action")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	if !e.EventType.Valid() {
		return fmt.Errorf("unsupported event_type %q", e.EventType)
	}
	return nil
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
}

type AuditRecord struct {
	ID         string           `json:"id"`
	RecordedAt time.Time        `json:"recorded_at"`
	Event      RuntimeEvent     `json:"event"`
	Result     EvaluationResult `json:"result"`
}

var ErrInvalidRuntimeEvent = errors.New("invalid runtime event")
