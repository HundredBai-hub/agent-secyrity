import type { ConsoleSnapshot } from "./model";

export const initialSnapshot: ConsoleSnapshot = {
  tenantName: "MurphySec 内部试点租户",
  environment: "Production Guarded Runtime",
  metrics: [
    { id: "events", label: "今日运行时事件", value: 1284, trend: "+18%", tone: "neutral" },
    { id: "blocked", label: "已阻断高风险动作", value: 37, trend: "-6%", tone: "critical" },
    { id: "approval", label: "待审批动作", value: 8, trend: "+3", tone: "warning" },
    { id: "coverage", label: "策略覆盖场景", value: 4, trend: "baseline", tone: "good" }
  ],
  decisionStats: [
    { decision: "allow", count: 842 },
    { decision: "record", count: 286 },
    { decision: "require_approval", count: 91 },
    { decision: "redact", count: 28 },
    { decision: "deny", count: 37 }
  ],
  policyPacks: [
    {
      id: "baseline-code-repository",
      name: "代码仓库 Agent",
      scenario: "CI 修复、代码检索、依赖升级",
      enabled: true,
      policyCount: 2,
      coverage: "secret 文件访问、危险 shell"
    },
    {
      id: "baseline-customer-support",
      name: "客服 Agent",
      scenario: "工单总结、账号操作、退款辅助",
      enabled: true,
      policyCount: 2,
      coverage: "PII 脱敏、账号变更审批"
    },
    {
      id: "baseline-finance-operations",
      name: "财务 Agent",
      scenario: "付款、转账、退款执行",
      enabled: true,
      policyCount: 1,
      coverage: "资金动作审批"
    },
    {
      id: "baseline-data-analysis",
      name: "数据分析 Agent",
      scenario: "客户分析、生产库查询、报表导出",
      enabled: true,
      policyCount: 1,
      coverage: "生产客户数据导出审批"
    }
  ],
  approvals: [
    {
      id: "approval-1001",
      tenantId: "tenant-a",
      agentId: "agent-finance-001",
      userId: "finance-chen",
      taskId: "pay-invoice-7781",
      toolName: "wire_transfer",
      action: "transfer",
      reason: "money movement requires approval",
      status: "pending",
      requestedAt: "2026-06-08T09:42:00Z"
    },
    {
      id: "approval-1002",
      tenantId: "tenant-a",
      agentId: "agent-code-014",
      userId: "dev-lin",
      taskId: "incident-hotfix",
      toolName: "terminal",
      action: "execute",
      reason: "dangerous shell execution requires approval",
      status: "pending",
      requestedAt: "2026-06-08T10:05:00Z"
    }
  ],
  auditRecords: [
    {
      id: "audit-9004",
      tenantId: "tenant-a",
      agentId: "agent-code-014",
      userId: "dev-lin",
      taskId: "incident-hotfix",
      eventType: "tool_call",
      decision: "require_approval",
      resource: "production-shell",
      action: "execute",
      recordedAt: "2026-06-08T10:05:00Z",
      reason: "dangerous shell execution requires approval"
    },
    {
      id: "audit-9003",
      tenantId: "tenant-a",
      agentId: "agent-support-002",
      userId: "support-wang",
      taskId: "ticket-5420",
      eventType: "response",
      decision: "redact",
      resource: "customer-profile",
      action: "write",
      recordedAt: "2026-06-08T09:58:00Z",
      reason: "customer data responses must be redacted"
    },
    {
      id: "audit-9002",
      tenantId: "tenant-a",
      agentId: "agent-finance-001",
      userId: "finance-chen",
      taskId: "pay-invoice-7781",
      eventType: "tool_call",
      decision: "require_approval",
      resource: "vendor-bank-account",
      action: "transfer",
      recordedAt: "2026-06-08T09:42:00Z",
      reason: "money movement requires approval"
    },
    {
      id: "audit-9001",
      tenantId: "tenant-a",
      agentId: "agent-code-001",
      userId: "dev-zhou",
      taskId: "dependency-upgrade",
      eventType: "file_access",
      decision: "deny",
      resource: "/repo/.env",
      action: "read",
      recordedAt: "2026-06-08T09:21:00Z",
      reason: "secret file access is blocked"
    }
  ]
};
