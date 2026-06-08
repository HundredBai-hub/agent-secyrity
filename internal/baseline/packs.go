// Package baseline defines built-in policy packs for common Agent risk scenarios.
package baseline

import "github.com/HundredBai-hub/agent-secyrity/internal/domain"

const baselineVersion = "1.0.0"

// DefaultPolicyPacks returns the built-in policy packs generated for one tenant.
func DefaultPolicyPacks(tenantID string) []domain.PolicyPack {
	return []domain.PolicyPack{
		codeRepositoryPack(tenantID),
		customerSupportPack(tenantID),
		financeOperationsPack(tenantID),
		dataAnalysisPack(tenantID),
	}
}

// codeRepositoryPack protects source code Agents from secret access and dangerous shell execution.
func codeRepositoryPack(tenantID string) domain.PolicyPack {
	return domain.PolicyPack{
		ID:       "baseline-code-repository",
		TenantID: tenantID,
		Name:     "Baseline Code Repository",
		Version:  baselineVersion,
		Enabled:  true,
		Policies: []domain.Policy{
			{
				ID:       "baseline-code-repository.deny.secret-file-access",
				TenantID: tenantID,
				Enabled:  true,
				Priority: 100,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeFileAccess},
					Resources:  []string{".env", "id_rsa", "id_ed25519", ".ssh/", "credentials"},
					DataLabels: []string{"secret"},
				},
				Decision: domain.DecisionDeny,
				Reason:   "secret file access is blocked",
			},
			{
				ID:       "baseline-code-repository.require-approval.dangerous-shell",
				TenantID: tenantID,
				Enabled:  true,
				Priority: 90,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeToolCall},
					ToolNames:  []string{"shell", "exec", "terminal"},
					Actions:    []string{"execute"},
				},
				Decision: domain.DecisionRequireApproval,
				Reason:   "dangerous shell execution requires approval",
			},
		},
	}
}

// customerSupportPack protects customer support Agents from leaking customer data or changing accounts without review.
func customerSupportPack(tenantID string) domain.PolicyPack {
	return domain.PolicyPack{
		ID:       "baseline-customer-support",
		TenantID: tenantID,
		Name:     "Baseline Customer Support",
		Version:  baselineVersion,
		Enabled:  true,
		Policies: []domain.Policy{
			{
				ID:       "baseline-customer-support.redact.customer-data-response",
				TenantID: tenantID,
				Enabled:  true,
				Priority: 100,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeResponse},
					DataLabels: []string{"pii", "customer_data"},
				},
				Decision: domain.DecisionRedact,
				Reason:   "customer data responses must be redacted",
			},
			{
				ID:       "baseline-customer-support.require-approval.account-change",
				TenantID: tenantID,
				Enabled:  true,
				Priority: 90,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeToolCall},
					ToolNames:  []string{"account_admin", "crm"},
					Actions:    []string{"change_account_status", "reset_password", "refund"},
				},
				Decision: domain.DecisionRequireApproval,
				Reason:   "high-risk customer account operation requires approval",
			},
		},
	}
}

// financeOperationsPack requires approval before Agents perform money movement.
func financeOperationsPack(tenantID string) domain.PolicyPack {
	return domain.PolicyPack{
		ID:       "baseline-finance-operations",
		TenantID: tenantID,
		Name:     "Baseline Finance Operations",
		Version:  baselineVersion,
		Enabled:  true,
		Policies: []domain.Policy{
			{
				ID:       "baseline-finance-operations.require-approval.money-transfer",
				TenantID: tenantID,
				Enabled:  true,
				Priority: 100,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeToolCall},
					ToolNames:  []string{"wire_transfer", "payment", "refund"},
					Actions:    []string{"transfer", "pay", "refund"},
				},
				Decision: domain.DecisionRequireApproval,
				Reason:   "money movement requires approval",
			},
		},
	}
}

// dataAnalysisPack requires approval before Agents export production customer data.
func dataAnalysisPack(tenantID string) domain.PolicyPack {
	return domain.PolicyPack{
		ID:       "baseline-data-analysis",
		TenantID: tenantID,
		Name:     "Baseline Data Analysis",
		Version:  baselineVersion,
		Enabled:  true,
		Policies: []domain.Policy{
			{
				ID:       "baseline-data-analysis.require-approval.production-database-export",
				TenantID: tenantID,
				Enabled:  true,
				Priority: 100,
				Conditions: domain.PolicyConditions{
					EventTypes: []domain.EventType{domain.EventTypeToolCall},
					ToolNames:  []string{"sql_query", "warehouse", "database"},
					Resources:  []string{"prod", "production", "customer"},
					Actions:    []string{"export", "query", "dump"},
					DataLabels: []string{"customer_data", "pii"},
				},
				Decision: domain.DecisionRequireApproval,
				Reason:   "production customer data export requires approval",
			},
		},
	}
}
