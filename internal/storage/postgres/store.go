package postgres

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/approval"
	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/001_init.sql
var migrationSQL string

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, migrationSQL)
	return err
}

type AuditStore struct {
	db *sql.DB
}

func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

func (s *AuditStore) Append(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error) {
	if record.ID == "" {
		record.ID = fmt.Sprintf("audit-%d", time.Now().UTC().UnixNano())
	}
	if record.RecordedAt.IsZero() {
		record.RecordedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return domain.AuditRecord{}, fmt.Errorf("marshal audit record: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO audit_records (id, tenant_id, recorded_at, event_type, decision, record)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  recorded_at = EXCLUDED.recorded_at,
  event_type = EXCLUDED.event_type,
  decision = EXCLUDED.decision,
  record = EXCLUDED.record
`, record.ID, record.Event.TenantID, record.RecordedAt, record.Event.EventType, record.Result.Decision, payload)
	if err != nil {
		return domain.AuditRecord{}, err
	}
	return record, nil
}

func (s *AuditStore) List(ctx context.Context, opts audit.ListOptions) ([]domain.AuditRecord, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if opts.TenantID != "" {
		rows, err := s.db.QueryContext(ctx, `
SELECT record
FROM audit_records
WHERE tenant_id = $1
ORDER BY recorded_at DESC
LIMIT $2
`, opts.TenantID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanAuditRecords(rows)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT record
FROM audit_records
ORDER BY recorded_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditRecords(rows)
}

func scanAuditRecords(rows *sql.Rows) ([]domain.AuditRecord, error) {
	var records []domain.AuditRecord
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var record domain.AuditRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, fmt.Errorf("unmarshal audit record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

type PolicyPackStore struct {
	db *sql.DB
}

func NewPolicyPackStore(db *sql.DB) *PolicyPackStore {
	return &PolicyPackStore{db: db}
}

func (s *PolicyPackStore) Upsert(ctx context.Context, pack domain.PolicyPack) error {
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
	payload, err := json.Marshal(pack)
	if err != nil {
		return fmt.Errorf("marshal policy pack: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO policy_packs (tenant_id, id, version, enabled, pack, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, id) DO UPDATE SET
  version = EXCLUDED.version,
  enabled = EXCLUDED.enabled,
  pack = EXCLUDED.pack,
  updated_at = EXCLUDED.updated_at
`, pack.TenantID, pack.ID, pack.Version, pack.Enabled, payload, time.Now().UTC())
	return err
}

func (s *PolicyPackStore) Get(ctx context.Context, tenantID string, packID string) (domain.PolicyPack, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `
SELECT pack
FROM policy_packs
WHERE tenant_id = $1 AND id = $2
`, tenantID, packID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PolicyPack{}, policypack.ErrNotFound
	}
	if err != nil {
		return domain.PolicyPack{}, err
	}
	return unmarshalPack(raw)
}

func (s *PolicyPackStore) List(ctx context.Context, tenantID string) ([]domain.PolicyPack, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT pack
FROM policy_packs
WHERE tenant_id = $1
ORDER BY id
`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPacks(rows)
}

func (s *PolicyPackStore) ListEnabled(ctx context.Context, tenantID string) ([]domain.PolicyPack, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT pack
FROM policy_packs
WHERE tenant_id = $1 AND enabled = true
ORDER BY id
`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPacks(rows)
}

func (s *PolicyPackStore) SetEnabled(ctx context.Context, tenantID string, packID string, enabled bool) error {
	pack, err := s.Get(ctx, tenantID, packID)
	if err != nil {
		return err
	}
	pack.Enabled = enabled
	return s.Upsert(ctx, pack)
}

func scanPacks(rows *sql.Rows) ([]domain.PolicyPack, error) {
	var packs []domain.PolicyPack
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		pack, err := unmarshalPack(raw)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return packs, nil
}

func unmarshalPack(raw []byte) (domain.PolicyPack, error) {
	var pack domain.PolicyPack
	if err := json.Unmarshal(raw, &pack); err != nil {
		return domain.PolicyPack{}, fmt.Errorf("unmarshal policy pack: %w", err)
	}
	return pack, nil
}

type ApprovalStore struct {
	db *sql.DB
}

func NewApprovalStore(db *sql.DB) *ApprovalStore {
	return &ApprovalStore{db: db}
}

func (s *ApprovalStore) Create(ctx context.Context, request domain.ApprovalRequest) (domain.ApprovalRequest, error) {
	if request.TenantID == "" {
		return domain.ApprovalRequest{}, fmt.Errorf("tenant_id is required")
	}
	if request.ID == "" {
		request.ID = fmt.Sprintf("approval-%d", time.Now().UTC().UnixNano())
	}
	if request.Status == "" {
		request.Status = domain.ApprovalStatusPending
	}
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return domain.ApprovalRequest{}, fmt.Errorf("marshal approval request: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO approval_requests (tenant_id, id, status, requested_at, expires_at, request)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, id) DO UPDATE SET
  status = EXCLUDED.status,
  requested_at = EXCLUDED.requested_at,
  expires_at = EXCLUDED.expires_at,
  request = EXCLUDED.request
`, request.TenantID, request.ID, request.Status, request.RequestedAt, request.ExpiresAt, payload)
	if err != nil {
		return domain.ApprovalRequest{}, err
	}
	return request, nil
}

func (s *ApprovalStore) Get(ctx context.Context, tenantID string, approvalID string) (domain.ApprovalRequest, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `
SELECT request
FROM approval_requests
WHERE tenant_id = $1 AND id = $2
`, tenantID, approvalID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ApprovalRequest{}, approval.ErrNotFound
	}
	if err != nil {
		return domain.ApprovalRequest{}, err
	}
	return unmarshalApproval(raw)
}

func (s *ApprovalStore) List(ctx context.Context, tenantID string, opts approval.ListOptions) ([]domain.ApprovalRequest, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT request
FROM approval_requests
WHERE tenant_id = $1
ORDER BY requested_at DESC
LIMIT $2
`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var requests []domain.ApprovalRequest
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		request, err := unmarshalApproval(raw)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

func (s *ApprovalStore) Decide(ctx context.Context, tenantID string, approvalID string, input approval.DecisionInput) (domain.ApprovalRequest, error) {
	request, err := s.Get(ctx, tenantID, approvalID)
	if err != nil {
		return domain.ApprovalRequest{}, err
	}
	if input.Status != domain.ApprovalStatusApproved && input.Status != domain.ApprovalStatusRejected {
		return domain.ApprovalRequest{}, approval.ErrInvalidDecision
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if request.Status != domain.ApprovalStatusPending {
		return domain.ApprovalRequest{}, approval.ErrApprovalAlreadyDecided
	}
	if !request.ExpiresAt.IsZero() && now.After(request.ExpiresAt) {
		request.Status = domain.ApprovalStatusExpired
		if _, err := s.Create(ctx, request); err != nil {
			return domain.ApprovalRequest{}, err
		}
		return domain.ApprovalRequest{}, approval.ErrApprovalExpired
	}
	request.Status = input.Status
	request.DecidedAt = now
	request.DecidedBy = input.DecidedBy
	request.DecisionReason = input.Reason
	if _, err := s.Create(ctx, request); err != nil {
		return domain.ApprovalRequest{}, err
	}
	return request, nil
}

func unmarshalApproval(raw []byte) (domain.ApprovalRequest, error) {
	var request domain.ApprovalRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return domain.ApprovalRequest{}, fmt.Errorf("unmarshal approval request: %w", err)
	}
	return request, nil
}
