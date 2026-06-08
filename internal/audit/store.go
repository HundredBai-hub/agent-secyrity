// Package audit defines audit record storage contracts and in-memory storage.
package audit

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
)

// Store persists and queries runtime audit records.
type Store interface {
	Append(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error)
	List(ctx context.Context, opts ListOptions) ([]domain.AuditRecord, error)
}

// ListOptions contains audit query filters and pagination options.
type ListOptions struct {
	Limit     int
	Offset    int
	TenantID  string
	AgentID   string
	UserID    string
	TaskID    string
	Decision  domain.Decision
	EventType domain.EventType
}

// Normalize returns safe pagination options while preserving filters.
func (opts ListOptions) Normalize() ListOptions {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if opts.Limit > 1000 {
		opts.Limit = 1000
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	return opts
}

// MemoryStore stores audit records in process memory for tests and local runs.
type MemoryStore struct {
	mu      sync.RWMutex
	records []domain.AuditRecord
	counter atomic.Uint64
}

// NewMemoryStore returns an empty in-memory audit store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// Append stores one audit record and fills missing ID and timestamp values.
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

// List returns audit records matching filters, ordered newest first.
func (s *MemoryStore) List(ctx context.Context, opts ListOptions) ([]domain.AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	opts = opts.Normalize()
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]domain.AuditRecord, 0, len(s.records))
	for _, record := range s.records {
		if recordMatches(record, opts) {
			records = append(records, record)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].RecordedAt.After(records[j].RecordedAt)
	})
	if opts.Offset >= len(records) {
		return []domain.AuditRecord{}, nil
	}
	end := opts.Offset + opts.Limit
	if end > len(records) {
		end = len(records)
	}
	result := append([]domain.AuditRecord(nil), records[opts.Offset:end]...)
	return result, nil
}

// recordMatches reports whether a record satisfies all query filters.
func recordMatches(record domain.AuditRecord, opts ListOptions) bool {
	if opts.TenantID != "" && record.Event.TenantID != opts.TenantID {
		return false
	}
	if opts.AgentID != "" && record.Event.AgentID != opts.AgentID {
		return false
	}
	if opts.UserID != "" && record.Event.UserID != opts.UserID {
		return false
	}
	if opts.TaskID != "" && record.Event.TaskID != opts.TaskID {
		return false
	}
	if opts.Decision != "" && record.Result.Decision != opts.Decision {
		return false
	}
	if opts.EventType != "" && record.Event.EventType != opts.EventType {
		return false
	}
	return true
}
