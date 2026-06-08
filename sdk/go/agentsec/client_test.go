package agentsec

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientEvaluate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/evaluate" {
			t.Fatalf("path = %s, want /v1/evaluate", r.URL.Path)
		}
		if got := r.Header.Get("User-Agent"); got != "agentsec-test/1.0" {
			t.Fatalf("user agent = %q, want agentsec-test/1.0", got)
		}

		var event RuntimeEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if event.TenantID != "tenant-a" || event.AgentID != "agent-1" || event.EventType != EventTypeToolCall {
			t.Fatalf("unexpected event: %+v", event)
		}
		if event.Subject.Type != SubjectTypeUser || event.Subject.RiskLevel != "medium" {
			t.Fatalf("unexpected subject: %+v", event.Subject)
		}

		writeTestJSON(t, w, http.StatusOK, EvaluationResult{
			Decision:         DecisionRequireApproval,
			Reason:           "payment transfer requires approval",
			MatchedPolicyIDs: []string{"policy-payment-approval"},
			AuditID:          "audit-1",
			ApprovalID:       "approval-1",
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, WithUserAgent("agentsec-test/1.0"))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.Evaluate(context.Background(), RuntimeEvent{
		TenantID:  "tenant-a",
		AgentID:   "agent-1",
		UserID:    "user-1",
		TaskID:    "task-1",
		EventType: EventTypeToolCall,
		Subject: Subject{
			Type:      SubjectTypeUser,
			ID:        "user-1",
			Roles:     []string{"finance"},
			RiskLevel: "medium",
		},
		ToolName:   "wire_transfer",
		Resource:   "bank-account:vendor-1",
		Action:     "transfer",
		DataLabels: []string{"payment", "financial"},
		Intent:     "pay approved invoice",
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	if !result.RequiresApproval() {
		t.Fatalf("requires approval = false, want true")
	}
	if result.Allowed() {
		t.Fatalf("allowed = true, want false")
	}
	if result.ApprovalID != "approval-1" {
		t.Fatalf("approval id = %q, want approval-1", result.ApprovalID)
	}
}

func TestClientReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, http.StatusForbidden, map[string]string{
			"error":   "policy_denied",
			"message": "tool call is blocked",
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Evaluate(context.Background(), RuntimeEvent{TenantID: "tenant-a"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", apiErr.StatusCode)
	}
	if apiErr.Code != "policy_denied" || apiErr.Message != "tool call is blocked" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestClientApprovals(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tenants/tenant-a/approvals":
			writeTestJSON(t, w, http.StatusOK, map[string][]ApprovalRequest{
				"approvals": {testApproval(now)},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tenants/tenant-a/approvals/approval-1":
			writeTestJSON(t, w, http.StatusOK, testApproval(now))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tenants/tenant-a/approvals/approval-1/decide":
			var input ApprovalDecisionInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatalf("decode decision: %v", err)
			}
			if input.Decision != ApprovalStatusApproved || input.DecidedBy != "manager-1" {
				t.Fatalf("unexpected decision input: %+v", input)
			}
			approval := testApproval(now)
			approval.Status = ApprovalStatusApproved
			approval.DecidedBy = "manager-1"
			approval.DecisionReason = "invoice was verified"
			writeTestJSON(t, w, http.StatusOK, approval)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	approvals, err := client.ListApprovals(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(approvals) != 1 || approvals[0].ID != "approval-1" {
		t.Fatalf("unexpected approvals: %+v", approvals)
	}

	approval, err := client.GetApproval(context.Background(), "tenant-a", "approval-1")
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if approval.Status != ApprovalStatusPending {
		t.Fatalf("approval status = %s, want pending", approval.Status)
	}

	approved, err := client.DecideApproval(context.Background(), "tenant-a", "approval-1", ApprovalDecisionInput{
		Decision:  ApprovalStatusApproved,
		DecidedBy: "manager-1",
		Reason:    "invoice was verified",
	})
	if err != nil {
		t.Fatalf("decide approval: %v", err)
	}
	if approved.Status != ApprovalStatusApproved || approved.DecidedBy != "manager-1" {
		t.Fatalf("unexpected approved request: %+v", approved)
	}
}

func TestDecisionHelpers(t *testing.T) {
	t.Parallel()

	if !(EvaluationResult{Decision: DecisionAllow}).Allowed() {
		t.Fatalf("allow decision should be allowed")
	}
	if !(EvaluationResult{Decision: DecisionDeny}).Denied() {
		t.Fatalf("deny decision should be denied")
	}
	if !(EvaluationResult{Decision: DecisionRequireApproval}).RequiresApproval() {
		t.Fatalf("require_approval decision should require approval")
	}
}

func testApproval(now time.Time) ApprovalRequest {
	return ApprovalRequest{
		ID:       "approval-1",
		TenantID: "tenant-a",
		Status:   ApprovalStatusPending,
		Event: RuntimeEvent{
			TenantID:  "tenant-a",
			AgentID:   "agent-1",
			UserID:    "user-1",
			TaskID:    "task-1",
			EventType: EventTypeToolCall,
			ToolName:  "wire_transfer",
			Resource:  "bank-account:vendor-1",
			Action:    "transfer",
		},
		Result: EvaluationResult{
			Decision:         DecisionRequireApproval,
			Reason:           "payment transfer requires approval",
			MatchedPolicyIDs: []string{"policy-payment-approval"},
			AuditID:          "audit-1",
			ApprovalID:       "approval-1",
		},
		Reason:      "payment transfer requires approval",
		RequestedAt: now,
		ExpiresAt:   now.Add(30 * time.Minute),
	}
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
