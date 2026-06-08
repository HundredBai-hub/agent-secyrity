package policypack

import (
	"context"
	"testing"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

func TestMemoryStoreUpsertGetListAndSetEnabled(t *testing.T) {
	store := NewMemoryStore()
	pack := domain.PolicyPack{
		ID:       "default-runtime",
		TenantID: "tenant-a",
		Name:     "Default Runtime",
		Version:  "1.0.0",
		Enabled:  true,
		Policies: []domain.Policy{{ID: "record-all", Enabled: true, Decision: domain.DecisionRecord}},
	}

	if err := store.Upsert(context.Background(), pack); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	got, err := store.Get(context.Background(), "tenant-a", "default-runtime")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.TenantID != "tenant-a" || got.ID != "default-runtime" {
		t.Fatalf("Get() = %#v", got)
	}

	packs, err := store.List(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(packs) != 1 {
		t.Fatalf("len(packs) = %d, want 1", len(packs))
	}

	if err := store.SetEnabled(context.Background(), "tenant-a", "default-runtime", false); err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}
	enabled, err := store.ListEnabled(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}
	if len(enabled) != 0 {
		t.Fatalf("len(enabled) = %d, want 0", len(enabled))
	}
}

func TestMemoryStoreIsolatesTenants(t *testing.T) {
	store := NewMemoryStore()
	if err := store.Upsert(context.Background(), domain.PolicyPack{
		ID:       "default-runtime",
		TenantID: "tenant-a",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	if _, err := store.Get(context.Background(), "tenant-b", "default-runtime"); err == nil {
		t.Fatal("Get() error = nil, want not found for different tenant")
	}
}
