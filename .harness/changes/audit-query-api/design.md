# Audit Query API 设计

## 方案概述

审计查询能力在现有 `audit.Store.List(ctx, opts)` 上演进，不新增独立服务层。核心设计是把查询契约集中到 `audit.ListOptions`，由内存存储、PostgreSQL 存储和 HTTP API 复用同一组选项。

## 查询模型

`audit.ListOptions` 扩展字段：

```go
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
```

分页规则：

- `Limit <= 0` 使用默认 100。
- `Limit > 1000` 限制为 1000。
- `Offset < 0` 在 API 边界返回 400；内部 store 对负值按 0 防御处理。
- 排序按 `recorded_at DESC`，内存存储用 append 顺序模拟最新记录优先。

## HTTP 响应

新增响应结构：

```go
type auditEventsResponse struct {
    Events     []domain.AuditRecord `json:"events"`
    Pagination paginationResponse   `json:"pagination"`
}
```

`pagination` 不引入总数，避免 PostgreSQL 每次查询都做 `COUNT(*)`。`has_more` 通过查询 `limit + 1` 条判断，响应只返回前 `limit` 条。

## PostgreSQL 查询

SQL 继续只查询 `record` JSONB，过滤条件使用已有列和 JSONB 字段：

- `tenant_id` 使用列过滤。
- `decision` 使用列过滤。
- `event_type` 使用列过滤。
- `agent_id/user_id/task_id` 使用 `record->'event'->>'agent_id'` 等 JSONB 字段过滤。

所有条件使用 `$n` 参数化传值，不拼接用户输入值。SQL 字符串只拼接固定条件片段。

## 安全边界

| 风险 | 设计处理 |
|---|---|
| 跨租户审计泄露 | API Key 开启时 `tenant_id` 必填并走 `authorizeTenant` |
| SQL 注入 | 查询值全部参数化，不把 query 参数拼进 SQL |
| 大分页拖垮服务 | `limit` 最大 1000 |
| 非法枚举导致语义不明 | HTTP 边界校验 `decision` 和 `event_type` |

## 模块影响

| 模块 | 影响 |
|---|---|
| `internal/audit` | 扩展 ListOptions、内存过滤与分页 |
| `internal/storage/postgres` | 扩展审计查询 SQL |
| `internal/transport/httpapi` | 扩展 query 参数、响应结构和错误处理 |
| `docs/API.md` | 更新审计查询 API 文档 |
| `docs/PROJECT.md` | 补审计模块文档索引 |
| `docs/modules/audit-query-api/MODULE.md` | 新增模块说明 |
| `docs/EXECUTION-BACKLOG.md` | 完成后移动执行指针 |

## 验证方式

- 红灯：先新增 audit/httpapi 查询测试，确认现有实现无法满足过滤和分页。
- 绿灯：实现内存 store、HTTP handler 和 PostgreSQL 查询。
- 局部测试：`go test ./internal/audit ./internal/storage/postgres ./internal/transport/httpapi`。
- 全量测试：`go test ./...`。
- harness validate/run。
- 敏感信息扫描。
