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

func NewEngineFromPacks(packs []domain.PolicyPack) *Engine {
	var policies []domain.Policy
	for _, pack := range packs {
		if !pack.Enabled {
			continue
		}
		for _, policy := range pack.Policies {
			policy.TenantID = firstNonEmpty(policy.TenantID, pack.TenantID)
			policy.PolicyPackID = firstNonEmpty(policy.PolicyPackID, pack.ID)
			policies = append(policies, policy)
		}
	}
	return NewEngine(policies)
}

func (e *Engine) Evaluate(event domain.RuntimeEvent) domain.EvaluationResult {
	best := domain.EvaluationResult{
		Decision: domain.DecisionAllow,
		Reason:   "no policy matched",
	}
	bestPolicyPriority := -1
	for _, policy := range e.policies {
		if !policy.Enabled || !matchesTenant(policy, event) || !matches(policy.Conditions, event) {
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

func matchesTenant(policy domain.Policy, event domain.RuntimeEvent) bool {
	if strings.TrimSpace(policy.TenantID) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(policy.TenantID), strings.TrimSpace(event.TenantID))
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
		matchString(conditions.UserIDs, event.UserID) &&
		matchSubjectType(conditions.SubjectTypes, event.Subject.Type) &&
		matchString(conditions.SubjectIDs, event.Subject.ID) &&
		matchAnyString(conditions.SubjectRoles, event.Subject.Roles) &&
		matchAnyString(conditions.SubjectGroups, event.Subject.Groups) &&
		matchString(conditions.SubjectRiskLevels, event.Subject.RiskLevel)
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

func matchSubjectType(allowed []domain.SubjectType, actual domain.SubjectType) bool {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
