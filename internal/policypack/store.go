package policypack

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

var ErrNotFound = errors.New("policy pack not found")

type Store interface {
	Upsert(ctx context.Context, pack domain.PolicyPack) error
	Get(ctx context.Context, tenantID string, packID string) (domain.PolicyPack, error)
	List(ctx context.Context, tenantID string) ([]domain.PolicyPack, error)
	ListEnabled(ctx context.Context, tenantID string) ([]domain.PolicyPack, error)
	SetEnabled(ctx context.Context, tenantID string, packID string, enabled bool) error
}

type MemoryStore struct {
	mu    sync.RWMutex
	packs map[string]domain.PolicyPack
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{packs: make(map[string]domain.PolicyPack)}
}

func (s *MemoryStore) Upsert(ctx context.Context, pack domain.PolicyPack) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if pack.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if pack.ID == "" {
		return fmt.Errorf("policy pack id is required")
	}
	for i := range pack.Policies {
		if pack.Policies[i].TenantID == "" {
			pack.Policies[i].TenantID = pack.TenantID
		}
		if pack.Policies[i].PolicyPackID == "" {
			pack.Policies[i].PolicyPackID = pack.ID
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.packs[key(pack.TenantID, pack.ID)] = clonePack(pack)
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, tenantID string, packID string) (domain.PolicyPack, error) {
	if err := ctx.Err(); err != nil {
		return domain.PolicyPack{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	pack, ok := s.packs[key(tenantID, packID)]
	if !ok {
		return domain.PolicyPack{}, ErrNotFound
	}
	return clonePack(pack), nil
}

func (s *MemoryStore) List(ctx context.Context, tenantID string) ([]domain.PolicyPack, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []domain.PolicyPack
	for _, pack := range s.packs {
		if pack.TenantID == tenantID {
			result = append(result, clonePack(pack))
		}
	}
	return result, nil
}

func (s *MemoryStore) ListEnabled(ctx context.Context, tenantID string) ([]domain.PolicyPack, error) {
	packs, err := s.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var result []domain.PolicyPack
	for _, pack := range packs {
		if pack.Enabled {
			result = append(result, pack)
		}
	}
	return result, nil
}

func (s *MemoryStore) SetEnabled(ctx context.Context, tenantID string, packID string, enabled bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mapKey := key(tenantID, packID)
	pack, ok := s.packs[mapKey]
	if !ok {
		return ErrNotFound
	}
	pack.Enabled = enabled
	s.packs[mapKey] = pack
	return nil
}

func key(tenantID string, packID string) string {
	return tenantID + "\x00" + packID
}

func clonePack(pack domain.PolicyPack) domain.PolicyPack {
	pack.Policies = append([]domain.Policy(nil), pack.Policies...)
	return pack
}
