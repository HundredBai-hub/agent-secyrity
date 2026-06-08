# Audit Query API

## 背景

运行时审计已经能记录 Agent 事件和策略决策，但查询能力仍停留在演示阶段：只支持 `tenant_id` 和 `limit`，无法按 Agent、用户、任务、决策或事件类型定位记录，也没有分页元信息。

企业落地时，审计查询是安全运营闭环的入口。安全团队需要回答这些问题：

- 某个 Agent 最近触发了哪些阻断或审批。
- 某个用户发起的任务是否访问了敏感资源。
- 某个业务任务的完整执行过程是否可追溯。
- 某类高风险决策在一段时间内是否集中出现。

## 目标

- 扩展 `audit.ListOptions`，支持按 `TenantID`、`AgentID`、`UserID`、`TaskID`、`Decision`、`EventType` 查询。
- 支持 `limit` 和 `offset` 分页，并返回 `pagination` 元信息。
- 内存存储和 PostgreSQL 存储保持一致查询语义。
- `GET /v1/audit/events` 保持原路径，新增查询参数，不破坏老客户端读取 `events` 的能力。
- 启用 API Key 认证时继续强制租户授权，避免跨租户审计泄露。

## 范围

- In scope:
  - `internal/audit` 查询选项、过滤和分页。
  - `internal/storage/postgres` 审计查询 SQL。
  - `internal/transport/httpapi` 查询参数解析、响应结构和错误处理。
  - API 文档、项目文档、模块文档和 backlog。
- Out of scope:
  - 不做全文搜索。
  - 不做时间范围查询。
  - 不做审计导出。
  - 不新增前端控制台。

## API 契约

`GET /v1/audit/events`

查询参数：

| 参数 | 说明 |
|---|---|
| `tenant_id` | 租户过滤；启用 API Key 后必填并受授权校验 |
| `agent_id` | Agent ID 过滤 |
| `user_id` | 用户 ID 过滤 |
| `task_id` | 任务 ID 过滤 |
| `decision` | 决策过滤，例如 `allow`、`deny`、`require_approval`、`redact`、`record` |
| `event_type` | 事件类型过滤，例如 `tool_call`、`file_access`、`response` |
| `limit` | 返回数量，默认 100，最大 1000 |
| `offset` | 跳过数量，默认 0 |

响应继续包含 `events`，并新增 `pagination`：

```json
{
  "events": [],
  "pagination": {
    "limit": 100,
    "offset": 0,
    "count": 0,
    "has_more": false
  }
}
```

## 验收标准

- 内存存储可以按 Agent / User / Task / Decision / EventType 组合过滤。
- 内存存储按最新记录优先返回，并支持 `offset`。
- PostgreSQL 存储使用参数化查询实现同等过滤。
- HTTP API 能解析新增查询参数并返回 `pagination`。
- 非法 `limit`、`offset`、`decision`、`event_type` 返回 400。
- API Key 开启时，无 `tenant_id` 或访问未授权租户仍被拒绝。
- `go test ./internal/audit ./internal/storage/postgres ./internal/transport/httpapi` 通过。
- `go test ./...` 通过。
