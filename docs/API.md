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
  "agent_id": "agent-code-001",
  "user_id": "dev-001",
  "task_id": "fix-build",
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
