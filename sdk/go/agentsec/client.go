// Package agentsec provides a Go client for Agent Security Platform runtime APIs.
package agentsec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultUserAgent = "agentsec-go-sdk/0.1"

// EventType identifies the kind of Agent runtime event being evaluated.
type EventType string

const (
	// EventTypePrompt records a prompt sent to an LLM.
	EventTypePrompt EventType = "prompt"
	// EventTypeToolCall records an Agent tool invocation.
	EventTypeToolCall EventType = "tool_call"
	// EventTypeFileAccess records file system access.
	EventTypeFileAccess EventType = "file_access"
	// EventTypeNetworkAccess records outbound network access.
	EventTypeNetworkAccess EventType = "network_access"
	// EventTypeResponse records an LLM or Agent response.
	EventTypeResponse EventType = "response"
	// EventTypeApproval records an approval action.
	EventTypeApproval EventType = "approval"
)

// Decision is the policy decision returned by runtime evaluation.
type Decision string

const (
	// DecisionAllow lets the runtime action continue.
	DecisionAllow Decision = "allow"
	// DecisionRecord records the runtime action without blocking it.
	DecisionRecord Decision = "record"
	// DecisionRedact requires sensitive content redaction.
	DecisionRedact Decision = "redact"
	// DecisionRequireApproval blocks until an approval request is approved.
	DecisionRequireApproval Decision = "require_approval"
	// DecisionDeny blocks the runtime action.
	DecisionDeny Decision = "deny"
)

// SubjectType identifies the non-human or human subject behind a runtime action.
type SubjectType string

const (
	// SubjectTypeUser represents an end user.
	SubjectTypeUser SubjectType = "user"
	// SubjectTypeAgent represents an AI Agent identity.
	SubjectTypeAgent SubjectType = "agent"
	// SubjectTypeServiceAccount represents a service account.
	SubjectTypeServiceAccount SubjectType = "service_account"
	// SubjectTypeWorkflow represents an automated workflow.
	SubjectTypeWorkflow SubjectType = "workflow"
)

// ApprovalStatus is the lifecycle state of an approval request.
type ApprovalStatus string

const (
	// ApprovalStatusPending means the request is waiting for a human or system decision.
	ApprovalStatusPending ApprovalStatus = "pending"
	// ApprovalStatusApproved means the request was approved.
	ApprovalStatusApproved ApprovalStatus = "approved"
	// ApprovalStatusRejected means the request was rejected.
	ApprovalStatusRejected ApprovalStatus = "rejected"
	// ApprovalStatusExpired means the request expired before a decision was made.
	ApprovalStatusExpired ApprovalStatus = "expired"
)

// Subject describes the actor identity used by policy conditions.
type Subject struct {
	Type      SubjectType `json:"type,omitempty"`
	ID        string      `json:"id,omitempty"`
	Roles     []string    `json:"roles,omitempty"`
	Groups    []string    `json:"groups,omitempty"`
	RiskLevel string      `json:"risk_level,omitempty"`
}

// RuntimeEvent is the event submitted by an Agent runtime before or after an action.
type RuntimeEvent struct {
	ID         string    `json:"event_id,omitempty"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
	TenantID   string    `json:"tenant_id"`
	AgentID    string    `json:"agent_id"`
	UserID     string    `json:"user_id"`
	TaskID     string    `json:"task_id"`
	EventType  EventType `json:"event_type"`
	Subject    Subject   `json:"subject,omitempty"`
	ApprovalID string    `json:"approval_id,omitempty"`
	ToolName   string    `json:"tool_name,omitempty"`
	Resource   string    `json:"resource,omitempty"`
	Action     string    `json:"action"`
	Input      string    `json:"input,omitempty"`
	Output     string    `json:"output,omitempty"`
	DataLabels []string  `json:"data_labels,omitempty"`
	Intent     string    `json:"intent,omitempty"`
}

// EvaluationResult is the policy decision returned for a runtime event.
type EvaluationResult struct {
	Decision         Decision `json:"decision"`
	Reason           string   `json:"reason"`
	MatchedPolicyIDs []string `json:"matched_policy_ids"`
	AuditID          string   `json:"audit_id,omitempty"`
	ApprovalID       string   `json:"approval_id,omitempty"`
}

// Allowed reports whether the runtime event can continue immediately.
func (r EvaluationResult) Allowed() bool {
	return r.Decision == DecisionAllow || r.Decision == DecisionRecord
}

// Denied reports whether the runtime event must be blocked.
func (r EvaluationResult) Denied() bool {
	return r.Decision == DecisionDeny
}

// RequiresApproval reports whether the runtime event needs approval before retrying.
func (r EvaluationResult) RequiresApproval() bool {
	return r.Decision == DecisionRequireApproval
}

// ApprovalRequest is the wire representation of a runtime approval request.
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

// ApprovalDecisionInput is the request body used to approve or reject an approval request.
type ApprovalDecisionInput struct {
	Decision  ApprovalStatus `json:"decision"`
	DecidedBy string         `json:"decided_by"`
	Reason    string         `json:"reason"`
}

// APIError represents a non-2xx response from Agent Security Platform.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

// Error returns a readable API error string.
func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return fmt.Sprintf("agentsec API error: status=%d message=%s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("agentsec API error: status=%d code=%s message=%s", e.StatusCode, e.Code, e.Message)
}

// Client calls Agent Security Platform runtime APIs.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
	apiKey     string
}

// Option customizes Client construction.
type Option func(*Client)

// WithHTTPClient configures the HTTP client used for outbound requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithUserAgent configures the User-Agent header sent by the SDK.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		if strings.TrimSpace(userAgent) != "" {
			c.userAgent = userAgent
		}
	}
}

// WithAPIKey configures the Bearer token used for authenticated API calls.
func WithAPIKey(apiKey string) Option {
	return func(c *Client) {
		if strings.TrimSpace(apiKey) != "" {
			c.apiKey = apiKey
		}
	}
}

// NewClient creates a Client for a running Agent Security Platform API server.
func NewClient(baseURL string, opts ...Option) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("base URL must include scheme and host")
	}

	client := &Client{
		baseURL:    strings.TrimRight(parsed.String(), "/"),
		httpClient: http.DefaultClient,
		userAgent:  defaultUserAgent,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client, nil
}

// Evaluate submits a runtime event to the policy evaluation API.
func (c *Client) Evaluate(ctx context.Context, event RuntimeEvent) (EvaluationResult, error) {
	var result EvaluationResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/evaluate", event, &result); err != nil {
		return EvaluationResult{}, err
	}
	return result, nil
}

// ListApprovals returns recent approval requests for a tenant.
func (c *Client) ListApprovals(ctx context.Context, tenantID string) ([]ApprovalRequest, error) {
	var result struct {
		Approvals []ApprovalRequest `json:"approvals"`
	}
	path := "/v1/tenants/" + url.PathEscape(tenantID) + "/approvals"
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result.Approvals, nil
}

// GetApproval returns a single approval request.
func (c *Client) GetApproval(ctx context.Context, tenantID string, approvalID string) (ApprovalRequest, error) {
	var result ApprovalRequest
	path := "/v1/tenants/" + url.PathEscape(tenantID) + "/approvals/" + url.PathEscape(approvalID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &result); err != nil {
		return ApprovalRequest{}, err
	}
	return result, nil
}

// DecideApproval approves or rejects an approval request.
func (c *Client) DecideApproval(ctx context.Context, tenantID string, approvalID string, input ApprovalDecisionInput) (ApprovalRequest, error) {
	var result ApprovalRequest
	path := "/v1/tenants/" + url.PathEscape(tenantID) + "/approvals/" + url.PathEscape(approvalID) + "/decide"
	if err := c.doJSON(ctx, http.MethodPost, path, input, &result); err != nil {
		return ApprovalRequest{}, err
	}
	return result, nil
}

func (c *Client) doJSON(ctx context.Context, method string, path string, input any, output any) error {
	req, err := c.newRequest(ctx, method, path, input)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(resp)
	}
	if output == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return nil
}

func (c *Client) newRequest(ctx context.Context, method string, path string, input any) (*http.Request, error) {
	var body io.Reader
	if input != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(input); err != nil {
			return nil, fmt.Errorf("encode %s %s request: %w", method, path, err)
		}
		body = &buf
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build %s %s request: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func decodeAPIError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("read error response: %w", err)
	}

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    strings.TrimSpace(string(body)),
	}
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		apiErr.Code = payload.Error
		apiErr.Message = payload.Message
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	return apiErr
}
