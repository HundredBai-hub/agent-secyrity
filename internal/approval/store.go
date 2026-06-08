package approval

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

var (
	ErrNotFound               = errors.New("approval request not found")
	ErrApprovalExpired        = errors.New("approval request expired")
	ErrApprovalAlreadyDecided = errors.New("approval request already decided")
	ErrInvalidDecision        = errors.New("invalid approval decision")
)

type Store interface {
	Create(ctx context.Context, request domain.ApprovalRequest) (domain.ApprovalRequest, error)
	Get(ctx context.Context, tenantID string, approvalID string) (domain.ApprovalRequest, error)
	List(ctx context.Context, tenantID string, opts ListOptions) ([]domain.ApprovalRequest, error)
	Decide(ctx context.Context, tenantID string, approvalID string, input DecisionInput) (domain.ApprovalRequest, error)
}

type ListOptions struct {
	Limit int
}

type DecisionInput struct {
	Status    domain.ApprovalStatus
	DecidedBy string
	Reason    string
	Now       time.Time
}

type MemoryStore struct {
	mu       sync.RWMutex
	requests map[string]domain.ApprovalRequest
	counter  atomic.Uint64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{requests: make(map[string]domain.ApprovalRequest)}
}

func (s *MemoryStore) Create(ctx context.Context, request domain.ApprovalRequest) (domain.ApprovalRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.ApprovalRequest{}, err
	}
	if request.TenantID == "" {
		return domain.ApprovalRequest{}, fmt.Errorf("tenant_id is required")
	}
	if request.ID == "" {
		request.ID = fmt.Sprintf("approval-%d", s.counter.Add(1))
	}
	if request.Status == "" {
		request.Status = domain.ApprovalStatusPending
	}
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests[key(request.TenantID, request.ID)] = cloneRequest(request)
	return request, nil
}

func (s *MemoryStore) Get(ctx context.Context, tenantID string, approvalID string) (domain.ApprovalRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.ApprovalRequest{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	request, ok := s.requests[key(tenantID, approvalID)]
	if !ok {
		return domain.ApprovalRequest{}, ErrNotFound
	}
	return cloneRequest(request), nil
}

func (s *MemoryStore) List(ctx context.Context, tenantID string, opts ListOptions) ([]domain.ApprovalRequest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []domain.ApprovalRequest
	for _, request := range s.requests {
		if request.TenantID == tenantID {
			result = append(result, cloneRequest(request))
		}
	}
	if len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
}

func (s *MemoryStore) Decide(ctx context.Context, tenantID string, approvalID string, input DecisionInput) (domain.ApprovalRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.ApprovalRequest{}, err
	}
	if input.Status != domain.ApprovalStatusApproved && input.Status != domain.ApprovalStatusRejected {
		return domain.ApprovalRequest{}, ErrInvalidDecision
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mapKey := key(tenantID, approvalID)
	request, ok := s.requests[mapKey]
	if !ok {
		return domain.ApprovalRequest{}, ErrNotFound
	}
	if request.Status != domain.ApprovalStatusPending {
		return domain.ApprovalRequest{}, ErrApprovalAlreadyDecided
	}
	if !request.ExpiresAt.IsZero() && now.After(request.ExpiresAt) {
		request.Status = domain.ApprovalStatusExpired
		s.requests[mapKey] = request
		return domain.ApprovalRequest{}, ErrApprovalExpired
	}
	request.Status = input.Status
	request.DecidedAt = now
	request.DecidedBy = input.DecidedBy
	request.DecisionReason = input.Reason
	s.requests[mapKey] = request
	return cloneRequest(request), nil
}

func key(tenantID string, approvalID string) string {
	return tenantID + "\x00" + approvalID
}

func cloneRequest(request domain.ApprovalRequest) domain.ApprovalRequest {
	return request
}
