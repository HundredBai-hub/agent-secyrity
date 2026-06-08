package agentsec

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnforcerExecutesAllowedOperation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/evaluate" {
			t.Fatalf("path = %s, want /v1/evaluate", r.URL.Path)
		}
		writeTestJSON(t, w, http.StatusOK, EvaluationResult{Decision: DecisionAllow})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	enforcer, err := NewEnforcer(client)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	executed := false
	result, err := enforcer.Execute(context.Background(), Operation{
		Event: testRuntimeEvent(),
		Action: func(ctx context.Context) (any, error) {
			executed = true
			return "tool-output", nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !executed {
		t.Fatalf("action was not executed")
	}
	if result.Output != "tool-output" {
		t.Fatalf("output = %v, want tool-output", result.Output)
	}
	if result.Evaluation.Decision != DecisionAllow {
		t.Fatalf("decision = %s, want allow", result.Evaluation.Decision)
	}
}

func TestEnforcerBlocksDeniedOperation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, http.StatusOK, EvaluationResult{
			Decision: DecisionDeny,
			Reason:   "shell is blocked",
		})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	enforcer, err := NewEnforcer(client)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	executed := false
	_, err = enforcer.Execute(context.Background(), Operation{
		Event: testRuntimeEvent(),
		Action: func(ctx context.Context) (any, error) {
			executed = true
			return nil, nil
		},
	})
	if executed {
		t.Fatalf("action executed for denied operation")
	}
	var enforcementErr *EnforcementError
	if !errors.As(err, &enforcementErr) {
		t.Fatalf("error = %T, want *EnforcementError", err)
	}
	if enforcementErr.Decision != DecisionDeny || enforcementErr.Reason != "shell is blocked" {
		t.Fatalf("unexpected enforcement error: %+v", enforcementErr)
	}
}

func TestEnforcerWaitsForApprovalAndReevaluatesBeforeExecution(t *testing.T) {
	t.Parallel()

	var evaluateCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		evaluateCalls++
		var event RuntimeEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		switch evaluateCalls {
		case 1:
			if event.ApprovalID != "" {
				t.Fatalf("first approval_id = %q, want empty", event.ApprovalID)
			}
			writeTestJSON(t, w, http.StatusOK, EvaluationResult{
				Decision:   DecisionRequireApproval,
				Reason:     "money movement requires approval",
				ApprovalID: "approval-1",
			})
		case 2:
			if event.ApprovalID != "approval-1" {
				t.Fatalf("second approval_id = %q, want approval-1", event.ApprovalID)
			}
			writeTestJSON(t, w, http.StatusOK, EvaluationResult{Decision: DecisionAllow})
		default:
			t.Fatalf("unexpected evaluate call %d", evaluateCalls)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	waiter := staticApprovalWaiter{approval: ApprovalRequest{ID: "approval-1", Status: ApprovalStatusApproved}}
	enforcer, err := NewEnforcer(client, WithApprovalWaiter(waiter))
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	executed := false
	result, err := enforcer.Execute(context.Background(), Operation{
		Event: testRuntimeEvent(),
		Action: func(ctx context.Context) (any, error) {
			executed = true
			return "paid", nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !executed {
		t.Fatalf("action was not executed after approval")
	}
	if result.Approval == nil || result.Approval.ID != "approval-1" {
		t.Fatalf("approval = %+v, want approval-1", result.Approval)
	}
	if result.Output != "paid" {
		t.Fatalf("output = %v, want paid", result.Output)
	}
}

func TestEnforcerDoesNotExecuteRejectedApproval(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, http.StatusOK, EvaluationResult{
			Decision:   DecisionRequireApproval,
			Reason:     "needs approval",
			ApprovalID: "approval-1",
		})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	waiter := staticApprovalWaiter{approval: ApprovalRequest{ID: "approval-1", Status: ApprovalStatusRejected}}
	enforcer, err := NewEnforcer(client, WithApprovalWaiter(waiter))
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	executed := false
	_, err = enforcer.Execute(context.Background(), Operation{
		Event: testRuntimeEvent(),
		Action: func(ctx context.Context) (any, error) {
			executed = true
			return nil, nil
		},
	})
	if executed {
		t.Fatalf("action executed after rejected approval")
	}
	var enforcementErr *EnforcementError
	if !errors.As(err, &enforcementErr) {
		t.Fatalf("error = %T, want *EnforcementError", err)
	}
	if enforcementErr.ApprovalID != "approval-1" {
		t.Fatalf("approval id = %q, want approval-1", enforcementErr.ApprovalID)
	}
}

func TestEnforcerReturnsApprovalRequiredWhenNoWaiterConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, http.StatusOK, EvaluationResult{
			Decision:   DecisionRequireApproval,
			Reason:     "needs approval",
			ApprovalID: "approval-1",
		})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	enforcer, err := NewEnforcer(client)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	executed := false
	_, err = enforcer.Execute(context.Background(), Operation{
		Event: testRuntimeEvent(),
		Action: func(ctx context.Context) (any, error) {
			executed = true
			return nil, nil
		},
	})
	if executed {
		t.Fatalf("action executed without approval waiter")
	}
	var enforcementErr *EnforcementError
	if !errors.As(err, &enforcementErr) {
		t.Fatalf("error = %T, want *EnforcementError", err)
	}
	if enforcementErr.ApprovalID != "approval-1" {
		t.Fatalf("approval id = %q, want approval-1", enforcementErr.ApprovalID)
	}
}

func TestEnforcerDoesNotExecuteExpiredApproval(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, http.StatusOK, EvaluationResult{
			Decision:   DecisionRequireApproval,
			Reason:     "needs approval",
			ApprovalID: "approval-1",
		})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	enforcer, err := NewEnforcer(client, WithApprovalWaiter(staticApprovalWaiter{
		approval: ApprovalRequest{ID: "approval-1", Status: ApprovalStatusExpired},
	}))
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	executed := false
	_, err = enforcer.Execute(context.Background(), Operation{
		Event: testRuntimeEvent(),
		Action: func(ctx context.Context) (any, error) {
			executed = true
			return nil, nil
		},
	})
	if executed {
		t.Fatalf("action executed after expired approval")
	}
	var enforcementErr *EnforcementError
	if !errors.As(err, &enforcementErr) {
		t.Fatalf("error = %T, want *EnforcementError", err)
	}
}

func TestEnforcerDoesNotExecuteWhenApprovedReevaluationStillBlocks(t *testing.T) {
	t.Parallel()

	var evaluateCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		evaluateCalls++
		switch evaluateCalls {
		case 1:
			writeTestJSON(t, w, http.StatusOK, EvaluationResult{
				Decision:   DecisionRequireApproval,
				Reason:     "needs approval",
				ApprovalID: "approval-1",
			})
		case 2:
			writeTestJSON(t, w, http.StatusOK, EvaluationResult{
				Decision: DecisionDeny,
				Reason:   "approved event still violates policy",
			})
		default:
			t.Fatalf("unexpected evaluate call %d", evaluateCalls)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	enforcer, err := NewEnforcer(client, WithApprovalWaiter(staticApprovalWaiter{
		approval: ApprovalRequest{ID: "approval-1", Status: ApprovalStatusApproved},
	}))
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	executed := false
	_, err = enforcer.Execute(context.Background(), Operation{
		Event: testRuntimeEvent(),
		Action: func(ctx context.Context) (any, error) {
			executed = true
			return nil, nil
		},
	})
	if executed {
		t.Fatalf("action executed after denied reevaluation")
	}
	var enforcementErr *EnforcementError
	if !errors.As(err, &enforcementErr) {
		t.Fatalf("error = %T, want *EnforcementError", err)
	}
	if enforcementErr.Decision != DecisionDeny {
		t.Fatalf("decision = %s, want deny", enforcementErr.Decision)
	}
}

func TestNewEnforcerRejectsNilClient(t *testing.T) {
	t.Parallel()

	if _, err := NewEnforcer(nil); err == nil {
		t.Fatalf("NewEnforcer(nil) error = nil, want error")
	}
}

func TestEnforcerRejectsNilAction(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, http.StatusOK, EvaluationResult{Decision: DecisionAllow})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	enforcer, err := NewEnforcer(client)
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	if _, err := enforcer.Execute(context.Background(), Operation{Event: testRuntimeEvent()}); err == nil {
		t.Fatalf("Execute(nil action) error = nil, want error")
	}
}

type staticApprovalWaiter struct {
	approval ApprovalRequest
	err      error
}

func (w staticApprovalWaiter) WaitApproval(ctx context.Context, client *Client, result EvaluationResult, event RuntimeEvent) (ApprovalRequest, error) {
	if w.err != nil {
		return ApprovalRequest{}, w.err
	}
	return w.approval, nil
}

func testRuntimeEvent() RuntimeEvent {
	return RuntimeEvent{
		SchemaVersion: RuntimeEventSchemaV1,
		TenantID:      "tenant-a",
		AgentID:       "agent-1",
		UserID:        "user-1",
		TaskID:        "task-1",
		EventType:     EventTypeToolCall,
		ToolName:      "shell",
		Action:        "execute",
	}
}
