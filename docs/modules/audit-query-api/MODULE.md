# Audit Query API 模块说明

## 模块职责

审计查询能力为企业安全运营提供运行时记录检索入口。它支持按租户、Agent、用户、任务、决策和事件类型过滤审计记录，并通过 `limit` / `offset` 返回分页结果。

## 关键依赖

| 依赖 | 说明 |
|---|---|
| `internal/domain` | 审计记录、运行时事件、事件类型和决策枚举 |
| `internal/audit` | 查询选项、内存存储和存储接口 |
| `internal/storage/postgres` | PostgreSQL 参数化查询实现 |
| `internal/transport/httpapi` | REST 查询参数解析、鉴权和响应输出 |

## 文件清单

| 文件 | 作用 |
|---|---|
| `internal/audit/store.go` | 定义 `Store`、`ListOptions`、分页规整和内存查询实现 |
| `internal/audit/memory_test.go` | 覆盖内存 store 的过滤、排序和分页语义 |
| `internal/storage/postgres/store.go` | PostgreSQL 审计记录写入和参数化查询实现 |
| `internal/storage/postgres/store_integration_test.go` | PostgreSQL store 集成验证，需配置 `AGENT_SECURITY_POSTGRES_TEST_DSN` |
| `internal/transport/httpapi/handler.go` | `GET /v1/audit/events` 查询参数、分页响应和租户鉴权 |
| `internal/transport/httpapi/handler_test.go` | HTTP 查询过滤、分页、非法参数和鉴权测试 |

## 查询语义

- 审计记录按 `recorded_at DESC` 返回。
- `limit` 默认 100，最大 1000。
- `offset` 默认 0。
- `pagination.has_more` 通过查询 `limit + 1` 条判断，不额外执行总数统计。
- API Key 开启时，`tenant_id` 必填并受当前 API Key 授权范围约束。

## 安全约束

- HTTP 边界校验 `decision` 和 `event_type`，避免非法枚举进入存储层。
- PostgreSQL 查询值全部通过参数传入，不拼接用户输入。
- 不提供无租户授权的跨租户审计查询能力。
