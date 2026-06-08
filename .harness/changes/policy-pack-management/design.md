# Policy Pack Management 设计

## 方案概述

新增 `internal/policypack` 包，提供 `Store` 接口和 `MemoryStore`。Runtime service 从原来的固定 `policy.Engine` 变为可以接收 `PolicySource`，每次评估按 `tenant_id` 加载启用策略包并构造策略引擎。HTTP API 新增策略包管理路由，与 evaluate 共用同一个 Store。

```text
Policy Pack API
  -> policypack.Store

Evaluate API
  -> Runtime Service
  -> policypack.Store.ListEnabled(tenant_id)
  -> policy.NewEngineFromPacks(...)
  -> Audit Store
```

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `internal/policypack` | 新增 | Store 接口和内存实现 |
| `internal/runtime` | 扩展 | 支持从策略包 Store 动态加载策略 |
| `internal/transport/httpapi` | 扩展 | 新增 Policy Pack 管理 API |
| `cmd/server` | 更新 | 初始化 Policy Pack Store |
| `docs/API.md` | 更新 | 补充策略包管理 API |

## API 设计

### Upsert Policy Pack

`PUT /v1/tenants/{tenant_id}/policy-packs/{pack_id}`

请求体为 `PolicyPack`，路径中的 tenant 和 pack ID 优先，服务端会覆盖请求体中的对应字段。

### List Policy Packs

`GET /v1/tenants/{tenant_id}/policy-packs`

仅返回该租户策略包。

### Get Policy Pack

`GET /v1/tenants/{tenant_id}/policy-packs/{pack_id}`

找不到返回 404。

### Enable / Disable Policy Pack

`PATCH /v1/tenants/{tenant_id}/policy-packs/{pack_id}/enabled`

请求：

```json
{"enabled": false}
```

## 关键设计决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 是否持久化 | 暂用内存 Store | 先稳定接口，后续替换数据库 |
| Runtime 是否缓存策略 | 暂不缓存 | 保证 API 启停后立即影响评估 |
| 路径 tenant 是否覆盖 body tenant | 覆盖 | 防止客户端提交跨租户 body |
| 找不到策略包 | 返回 404 | 明确资源不存在 |

## 验证方式

```bash
go test ./...
```
