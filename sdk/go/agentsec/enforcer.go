// Package agentsec provides runtime enforcement helpers for protected Agent actions.
package agentsec

import (
	"context"
	"fmt"
)

// ActionFunc is the protected operation executed only after policy enforcement allows it.
type ActionFunc func(ctx context.Context) (any, error)

// ApprovalWaiter waits for an approval request to reach a terminal decision.
type ApprovalWaiter interface {
	WaitApproval(ctx context.Context, client *Client, result EvaluationResult, event RuntimeEvent) (ApprovalRequest, error)
}

// Operation describes one protected runtime action.
type Operation struct {
	Event  RuntimeEvent
	Action ActionFunc
}

// EnforcementResult contains the final evaluation, optional approval and protected action output.
type EnforcementResult struct {
	Evaluation EvaluationResult
	Approval   *ApprovalRequest
	Output     any
}

// EnforcementError reports that policy enforcement did not allow the action to run.
type EnforcementError struct {
	Decision   Decision
	Reason     string
	ApprovalID string
}

// Error returns a readable enforcement error string.
func (e *EnforcementError) Error() string {
	if e == nil {
		return ""
	}
	if e.ApprovalID != "" {
		return fmt.Sprintf("agentsec enforcement blocked: decision=%s approval_id=%s reason=%s", e.Decision, e.ApprovalID, e.Reason)
	}
	return fmt.Sprintf("agentsec enforcement blocked: decision=%s reason=%s", e.Decision, e.Reason)
}

// EnforcerOption customizes Enforcer construction.
type EnforcerOption func(*Enforcer)

// WithApprovalWaiter configures how the SDK waits for require_approval decisions.
func WithApprovalWaiter(waiter ApprovalWaiter) EnforcerOption {
	return func(e *Enforcer) {
		e.approvalWaiter = waiter
	}
}

// Enforcer evaluates runtime events and runs protected actions only when allowed.
type Enforcer struct {
	client         *Client
	approvalWaiter ApprovalWaiter
}

// NewEnforcer creates an execution enforcer for a Client.
func NewEnforcer(client *Client, opts ...EnforcerOption) (*Enforcer, error) {
	if client == nil {
		return nil, fmt.Errorf("agentsec client is required")
	}
	enforcer := &Enforcer{client: client}
	for _, opt := range opts {
		if opt != nil {
			opt(enforcer)
		}
	}
	return enforcer, nil
}

// Execute evaluates the operation, optionally waits for approval, re-evaluates, and then runs the action.
func (e *Enforcer) Execute(ctx context.Context, op Operation) (EnforcementResult, error) {
	if e == nil || e.client == nil {
		return EnforcementResult{}, fmt.Errorf("agentsec enforcer client is required")
	}
	if op.Action == nil {
		return EnforcementResult{}, fmt.Errorf("agentsec operation action is required")
	}
	result, err := e.client.Evaluate(ctx, op.Event)
	if err != nil {
		return EnforcementResult{}, err
	}
	if result.Allowed() {
		return e.executeAllowed(ctx, result, nil, op.Action)
	}
	if !result.RequiresApproval() {
		return EnforcementResult{Evaluation: result}, enforcementError(result)
	}
	if e.approvalWaiter == nil {
		return EnforcementResult{Evaluation: result}, enforcementError(result)
	}
	approval, err := e.approvalWaiter.WaitApproval(ctx, e.client, result, op.Event)
	if err != nil {
		return EnforcementResult{Evaluation: result}, err
	}
	if approval.Status != ApprovalStatusApproved {
		return EnforcementResult{Evaluation: result, Approval: &approval}, &EnforcementError{
			Decision:   result.Decision,
			Reason:     fmt.Sprintf("approval status is %s", approval.Status),
			ApprovalID: approval.ID,
		}
	}
	approvedEvent := op.Event
	approvedEvent.ApprovalID = approval.ID
	approvedResult, err := e.client.Evaluate(ctx, approvedEvent)
	if err != nil {
		return EnforcementResult{Evaluation: result, Approval: &approval}, err
	}
	if !approvedResult.Allowed() {
		return EnforcementResult{Evaluation: approvedResult, Approval: &approval}, enforcementError(approvedResult)
	}
	return e.executeAllowed(ctx, approvedResult, &approval, op.Action)
}

// executeAllowed runs the protected action and wraps its output in an enforcement result.
func (e *Enforcer) executeAllowed(ctx context.Context, result EvaluationResult, approval *ApprovalRequest, action ActionFunc) (EnforcementResult, error) {
	output, err := action(ctx)
	if err != nil {
		return EnforcementResult{Evaluation: result, Approval: approval}, err
	}
	return EnforcementResult{Evaluation: result, Approval: approval, Output: output}, nil
}

// enforcementError converts a policy decision result into an action-blocking error.
func enforcementError(result EvaluationResult) *EnforcementError {
	return &EnforcementError{
		Decision:   result.Decision,
		Reason:     result.Reason,
		ApprovalID: result.ApprovalID,
	}
}
