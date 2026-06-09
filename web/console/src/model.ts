export type Decision = "allow" | "record" | "redact" | "require_approval" | "deny";

export type EventType = "tool_call" | "file_access" | "response" | "network_access" | "prompt" | "approval";

export type ApprovalStatus = "pending" | "approved" | "rejected" | "expired";

export interface RiskMetric {
  id: string;
  label: string;
  value: number;
  trend: string;
  tone: "neutral" | "good" | "warning" | "critical";
}

export interface DecisionStat {
  decision: Decision;
  count: number;
}

export interface PolicyPackItem {
  id: string;
  name: string;
  scenario: string;
  enabled: boolean;
  policyCount: number;
  coverage: string;
}

export interface ApprovalItem {
  id: string;
  tenantId: string;
  agentId: string;
  userId: string;
  taskId: string;
  toolName: string;
  action: string;
  reason: string;
  status: ApprovalStatus;
  requestedAt: string;
}

export interface AuditRecord {
  id: string;
  tenantId: string;
  agentId: string;
  userId: string;
  taskId: string;
  eventType: EventType;
  decision: Decision;
  resource: string;
  action: string;
  recordedAt: string;
  reason: string;
}

export interface AuditFilters {
  tenantId?: string;
  agentId?: string;
  userId?: string;
  taskId?: string;
  decision?: Decision | "";
}

export interface ConsoleSnapshot {
  tenantName: string;
  environment: string;
  metrics: RiskMetric[];
  decisionStats: DecisionStat[];
  policyPacks: PolicyPackItem[];
  approvals: ApprovalItem[];
  auditRecords: AuditRecord[];
}
