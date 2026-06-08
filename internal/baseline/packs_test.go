// Package baseline tests the built-in policy pack catalog.
package baseline

import (
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

// TestDefaultPolicyPacksContainBusinessBaselines verifies that all baseline packs are tenant-scoped and enabled.
func TestDefaultPolicyPacksContainBusinessBaselines(t *testing.T) {
	t.Parallel()

	packs := DefaultPolicyPacks("tenant-a")
	if len(packs) != 4 {
		t.Fatalf("len(packs) = %d, want 4", len(packs))
	}

	wantPackIDs := []string{
		"baseline-code-repository",
		"baseline-customer-support",
		"baseline-finance-operations",
		"baseline-data-analysis",
	}
	for _, packID := range wantPackIDs {
		pack := findPack(packs, packID)
		if pack == nil {
			t.Fatalf("missing pack %s", packID)
		}
		if pack.TenantID != "tenant-a" {
			t.Fatalf("pack %s tenant_id = %s, want tenant-a", packID, pack.TenantID)
		}
		if !pack.Enabled {
			t.Fatalf("pack %s enabled = false, want true", packID)
		}
		if pack.Version == "" {
			t.Fatalf("pack %s version is empty", packID)
		}
	}
}

// TestDefaultPolicyPacksContainKeyRiskPolicies verifies that high-risk baseline policies are present.
func TestDefaultPolicyPacksContainKeyRiskPolicies(t *testing.T) {
	t.Parallel()

	packs := DefaultPolicyPacks("tenant-a")
	assertPolicy(t, packs, "baseline-code-repository.deny.secret-file-access", domain.DecisionDeny)
	assertPolicy(t, packs, "baseline-code-repository.require-approval.dangerous-shell", domain.DecisionRequireApproval)
	assertPolicy(t, packs, "baseline-customer-support.redact.customer-data-response", domain.DecisionRedact)
	assertPolicy(t, packs, "baseline-finance-operations.require-approval.money-transfer", domain.DecisionRequireApproval)
	assertPolicy(t, packs, "baseline-data-analysis.require-approval.production-database-export", domain.DecisionRequireApproval)
}

// findPack returns the policy pack with the requested ID.
func findPack(packs []domain.PolicyPack, id string) *domain.PolicyPack {
	for i := range packs {
		if packs[i].ID == id {
			return &packs[i]
		}
	}
	return nil
}

// assertPolicy verifies that a policy exists and has the expected decision.
func assertPolicy(t *testing.T, packs []domain.PolicyPack, id string, decision domain.Decision) {
	t.Helper()
	for _, pack := range packs {
		for _, policy := range pack.Policies {
			if policy.ID == id {
				if policy.Decision != decision {
					t.Fatalf("policy %s decision = %s, want %s", id, policy.Decision, decision)
				}
				if !policy.Enabled {
					t.Fatalf("policy %s enabled = false, want true", id)
				}
				return
			}
		}
	}
	t.Fatalf("missing policy %s", id)
}
