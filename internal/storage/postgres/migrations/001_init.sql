CREATE TABLE IF NOT EXISTS audit_records (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  recorded_at TIMESTAMPTZ NOT NULL,
  event_type TEXT NOT NULL,
  decision TEXT NOT NULL,
  record JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_records_tenant_recorded_at
  ON audit_records (tenant_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_records_tenant_decision
  ON audit_records (tenant_id, decision);

CREATE INDEX IF NOT EXISTS idx_audit_records_tenant_event_type
  ON audit_records (tenant_id, event_type);

CREATE TABLE IF NOT EXISTS policy_packs (
  tenant_id TEXT NOT NULL,
  id TEXT NOT NULL,
  version TEXT,
  enabled BOOLEAN NOT NULL,
  pack JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_policy_packs_tenant_enabled
  ON policy_packs (tenant_id, enabled);
