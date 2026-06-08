# Agent Security Platform API

## Authentication

服务端默认不启用鉴权，便于本地开发和测试。生产环境可以通过 `AGENT_SECURITY_API_KEYS` 启用 API Key 认证：

```bash
export AGENT_SECURITY_API_KEYS='runtime:replace-with-runtime-key:tenant-a,tenant-b;admin:replace-with-admin-key:*'
```

配置格式：

| 片段 | 说明 |
|---|---|
| `runtime` | Key ID，用于配置识别 |
| `replace-with-runtime-key` | Bearer Token，真实环境应由密钥管理或部署系统注入 |
| `tenant-a,tenant-b` | 允许访问的租户列表 |
| `*` | 允许访问所有租户 |

调用 API 时使用：

```http
Authorization: Bearer replace-with-runtime-key
```

启用认证后的错误语义：

| 场景 | 状态码 | 响应 |
|---|---|---|
| 缺少或错误 API Key | 401 | `{"error":"unauthorized","message":"missing or invalid API key"}` |
| API Key 无权访问租户 | 403 | `{"error":"forbidden","message":"API key is not allowed to access tenant"}` |

`GET /healthz` 不需要 API Key。

## `GET /healthz`

健康检查。

响应：

```json
{"status":"ok"}
```

## `POST /v1/evaluate`

提交 Agent 运行时事件，返回安全决策。

请求：

```json
{
  "schema_version": "runtime_event.v1",
  "tenant_id": "tenant-a",
  "agent_id": "agent-code-001",
  "user_id": "dev-001",
  "task_id": "fix-build",
  "subject": {
    "type": "user",
    "id": "dev-001",
    "roles": ["developer"],
    "groups": ["engineering"],
    "risk_level": "medium"
  },
  "event_type": "file_access",
  "resource": "/repo/.env",
  "action": "read",
  "data_labels": ["secret"],
  "intent": "debug build failure"
}
```

响应：

```json
{
  "decision": "deny",
  "reason": "secret file access is blocked",
  "matched_policy_ids": ["deny-secret-file-access"],
  "audit_id": "audit-1"
}
```

校验失败响应：

```json
{
  "error": "invalid_runtime_event",
  "message": "invalid runtime event",
  "details": {
    "fields": [
      {"field": "tenant_id", "code": "required", "message": "tenant_id is required"}
    ]
  }
}
```

## `GET /v1/audit/events`

查询审计事件。响应按 `recorded_at` 倒序返回，用于安全运营按租户、Agent、用户、任务和决策快速定位运行时风险记录。

可选查询参数：

| 参数 | 说明 |
|---|---|
| `limit` | 返回数量，默认 100，最大 1000 |
| `offset` | 分页偏移量，默认 0 |
| `tenant_id` | 租户过滤条件；启用 API Key 认证后必填，并且必须属于当前 API Key 允许范围 |
| `agent_id` | Agent ID 过滤条件 |
| `user_id` | 用户 ID 过滤条件 |
| `task_id` | 业务任务 ID 过滤条件 |
| `decision` | 决策过滤条件，可选 `allow`、`record`、`redact`、`require_approval`、`deny` |
| `event_type` | 事件类型过滤条件，可选 `prompt`、`tool_call`、`file_access`、`network_access`、`response`、`approval` |

响应：

```json
{
  "events": [
    {
      "id": "audit-1",
      "recorded_at": "2026-06-08T12:00:00Z",
      "event": {
        "tenant_id": "tenant-a",
        "agent_id": "agent-code-001",
        "user_id": "dev-001",
        "task_id": "fix-build",
        "event_type": "file_access",
        "resource": "/repo/.env",
        "action": "read",
        "data_labels": ["secret"]
      },
      "result": {
        "decision": "deny",
        "reason": "secret file access is blocked",
        "matched_policy_ids": ["deny-secret-file-access"]
      }
    }
  ],
  "pagination": {
    "limit": 100,
    "offset": 0,
    "count": 1,
    "has_more": false
  }
}
```

## Runtime Event 字段

| 字段 | 说明 |
|---|---|
| `schema_version` | Runtime Event Schema 版本；当前版本为 `runtime_event.v1`，缺省时服务端按 v1 处理 |
| `tenant_id` | 租户 ID，必填，用于策略隔离和审计隔离 |
| `agent_id` | Agent 身份 |
| `user_id` | 发起用户 |
| `task_id` | 业务任务 |
| `subject` | 结构化主体信息，可表达 user、agent、service_account、workflow |
| `event_type` | prompt / tool_call / file_access / network_access / response / approval |
| `tool_name` | 工具名称 |
| `resource` | 文件、URL、API、数据库表等资源 |
| `action` | read / write / execute 等动作 |
| `data_labels` | pii、secret、customer_data、source_code 等数据标签 |

## Policy Pack 管理

服务启动时会为 `default` 租户写入内置 baseline 策略包。内置策略包面向常见 Agent 业务场景：

| Pack ID | 场景 | 策略意图 |
|---|---|---|
| `baseline-code-repository` | 代码仓库 Agent | 保护 secrets、SSH key、环境变量，危险 shell 需要审批 |
| `baseline-customer-support` | 客服 Agent | 客户数据和 PII 输出脱敏，高风险账号操作需要审批 |
| `baseline-finance-operations` | 财务 Agent | 付款、转账、退款等资金动作需要审批 |
| `baseline-data-analysis` | 数据分析 Agent | 生产数据库访问和客户数据导出需要审批 |

### `PUT /v1/tenants/{tenant_id}/policy-packs/{pack_id}`

创建或更新策略包。路径中的 `tenant_id` 和 `pack_id` 优先，服务端会覆盖请求体中的同名字段，避免跨租户写入。

```json
{
  "name": "Default Runtime",
  "version": "1.0.0",
  "enabled": true,
  "policies": [
    {
      "id": "deny-shell",
      "enabled": true,
      "priority": 100,
      "conditions": {
        "event_types": ["tool_call"],
        "tool_names": ["shell"]
      },
      "decision": "deny",
      "reason": "shell is blocked"
    }
  ]
}
```

### `GET /v1/tenants/{tenant_id}/policy-packs`

列出租户策略包。

### `GET /v1/tenants/{tenant_id}/policy-packs/{pack_id}`

查询单个策略包。不存在返回 404。

### `PATCH /v1/tenants/{tenant_id}/policy-packs/{pack_id}/enabled`

启用或禁用策略包。禁用后，运行时评估不会加载该策略包。

```json
{"enabled": false}
```

## Policy Simulation

### `POST /v1/tenants/{tenant_id}/policy-simulations`

提交一个运行时事件和候选策略包，执行上线前 dry-run。该接口只返回模拟决策，不写审计记录，不创建审批单，不修改策略包状态。

请求：

```json
{
  "event": {
    "schema_version": "runtime_event.v1",
    "tenant_id": "tenant-a",
    "agent_id": "agent-code-001",
    "user_id": "dev-001",
    "task_id": "fix-build",
    "event_type": "tool_call",
    "tool_name": "shell",
    "action": "execute"
  },
  "policy_packs": [
    {
      "id": "candidate-runtime",
      "tenant_id": "tenant-a",
      "enabled": true,
      "policies": [
        {
          "id": "deny-shell",
          "enabled": true,
          "priority": 100,
          "conditions": {
            "event_types": ["tool_call"],
            "tool_names": ["shell"]
          },
          "decision": "deny",
          "reason": "shell is blocked"
        }
      ]
    }
  ]
}
```

响应：

```json
{
  "schema_version": "policy_simulation.v1",
  "result": {
    "decision": "deny",
    "reason": "shell is blocked",
    "matched_policy_ids": ["deny-shell"]
  }
}
```

约束：

| 条件 | 要求 |
|---|---|
| 租户 | `event.tenant_id` 必须与路径 `tenant_id` 一致 |
| 鉴权 | 启用 API Key 时，Key 必须允许访问该租户 |
| 校验 | Runtime Event 校验失败返回 422 和 `details.fields` |
| 副作用 | 不写审计、不创建审批、不保存策略包 |

## Approval Workflow

当评估结果为 `require_approval` 时，响应会包含 `approval_id`：

```json
{
  "decision": "require_approval",
  "reason": "dangerous tool execution requires approval",
  "matched_policy_ids": ["require-approval-dangerous-tool"],
  "audit_id": "audit-1",
  "approval_id": "approval-1"
}
```

### `GET /v1/tenants/{tenant_id}/approvals`

列出租户审批单。

### `GET /v1/tenants/{tenant_id}/approvals/{approval_id}`

查询单个审批单。跨租户查询返回 404。

### `POST /v1/tenants/{tenant_id}/approvals/{approval_id}/decide`

审批通过或拒绝。

```json
{
  "decision": "approved",
  "decided_by": "secops-001",
  "reason": "approved for incident response"
}
```

`decision` 支持：

| 值 | 说明 |
|---|---|
| `approved` | 审批通过 |
| `rejected` | 审批拒绝 |

## Approval Enforcement

审批通过后，Agent 需要在重新提交同一运行时事件时带上 `approval_id`：

```json
{
  "tenant_id": "tenant-a",
  "agent_id": "agent-code-001",
  "user_id": "dev-001",
  "task_id": "fix-build",
  "event_type": "tool_call",
  "tool_name": "shell",
  "action": "execute",
  "approval_id": "approval-1"
}
```

Runtime 会校验：

| 条件 | 要求 |
|---|---|
| 租户 | `approval_id` 必须属于同一 `tenant_id` |
| 状态 | 审批状态必须为 `approved` |
| 事件绑定 | agent、user、task、event_type、tool_name、resource、action、data_labels 必须与原审批事件一致 |

校验通过返回 `allow`；校验失败返回 `deny` 并写入审计。
