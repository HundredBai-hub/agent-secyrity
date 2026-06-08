# Agent Security Platform API

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

## `GET /v1/audit/events`

查询审计事件。

可选查询参数：

| 参数 | 说明 |
|---|---|
| `limit` | 返回数量，默认 100，最大 1000 |

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
  ]
}
```

## Runtime Event 字段

| 字段 | 说明 |
|---|---|
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
