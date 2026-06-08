package audit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

type Store interface {
	Append(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error)
	List(ctx context.Context, opts ListOptions) ([]domain.AuditRecord, error)
}

type ListOptions struct {
	Limit int
}

type MemoryStore struct {
	mu      sync.RWMutex
	records []domain.AuditRecord
	counter atomic.Uint64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Append(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return domain.AuditRecord{}, err
	}
	if record.ID == "" {
		record.ID = fmt.Sprintf("audit-%d", s.counter.Add(1))
	}
	if record.RecordedAt.IsZero() {
		record.RecordedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
	return record, nil
}

func (s *MemoryStore) List(ctx context.Context, opts ListOptions) ([]domain.AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	start := len(s.records) - limit
	if start < 0 {
		start = 0
	}
	result := append([]domain.AuditRecord(nil), s.records[start:]...)
	return result, nil
}
