package policy

import (
	"fmt"
	"strings"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

type Engine struct {
	policies []domain.Policy
}

func NewEngine(policies []domain.Policy) *Engine {
	copied := append([]domain.Policy(nil), policies...)
	return &Engine{policies: copied}
}

func (e *Engine) Evaluate(event domain.RuntimeEvent) domain.EvaluationResult {
	best := domain.EvaluationResult{
		Decision: domain.DecisionAllow,
		Reason:   "no policy matched",
	}
	bestPolicyPriority := -1
	for _, policy := range e.policies {
		if !policy.Enabled || !matches(policy.Conditions, event) {
			continue
		}
		best.MatchedPolicyIDs = append(best.MatchedPolicyIDs, policy.ID)
		if shouldReplace(policy, best.Decision, bestPolicyPriority) {
			best.Decision = policy.Decision
			best.Reason = policy.Reason
			if best.Reason == "" {
				best.Reason = fmt.Sprintf("matched policy %s", policy.ID)
			}
			bestPolicyPriority = policy.Priority
		}
	}
	return best
}

func shouldReplace(policy domain.Policy, current domain.Decision, currentPolicyPriority int) bool {
	if policy.Decision.Priority() != current.Priority() {
		return policy.Decision.Priority() > current.Priority()
	}
	return policy.Priority > currentPolicyPriority
}

func matches(conditions domain.PolicyConditions, event domain.RuntimeEvent) bool {
	return matchEventType(conditions.EventTypes, event.EventType) &&
		matchString(conditions.ToolNames, event.ToolName) &&
		matchResource(conditions.Resources, event.Resource) &&
		matchString(conditions.Actions, event.Action) &&
		matchAnyString(conditions.DataLabels, event.DataLabels) &&
		matchString(conditions.AgentIDs, event.AgentID) &&
		matchString(conditions.UserIDs, event.UserID)
}

func matchEventType(allowed []domain.EventType, actual domain.EventType) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if item == actual {
			return true
		}
	}
	return false
}

func matchString(allowed []string, actual string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(actual)) {
			return true
		}
	}
	return false
}

func matchResource(patterns []string, actual string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.Contains(actual, pattern) || strings.EqualFold(pattern, actual) {
			return true
		}
	}
	return false
}

func matchAnyString(required []string, actual []string) bool {
	if len(required) == 0 {
		return true
	}
	for _, want := range required {
		for _, got := range actual {
			if strings.EqualFold(strings.TrimSpace(want), strings.TrimSpace(got)) {
				return true
			}
		}
	}
	return false
}
