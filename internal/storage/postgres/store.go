package postgres

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
