# PostgreSQL Storage 设计

## 方案概述

新增 PostgreSQL 存储层，复用现有 `audit.Store` 和 `policypack.Store` 接口。数据库表采用“关键字段列 + JSONB 原文”的方式：tenant、id、enabled、recorded_at 等字段用于索引和过滤；完整 `AuditRecord` / `PolicyPack` 使用 JSONB 保存，减少早期 schema 频繁变更成本。

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `migrations/postgres` | 新增 | 建表 SQL |
| `internal/storage/postgres` | 新增 | DB 打开、Store 实现、集成测试 |
| `cmd/server` | 更新 | 根据 `DATABASE_URL` 选择 Postgres 或 Memory |
| `docs` | 更新 | PostgreSQL 配置和迁移说明 |

## 表设计

### `audit_records`

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | text primary key | Audit ID |
| `tenant_id` | text not null | 租户 |
| `recorded_at` | timestamptz not null | 记录时间 |
| `event_type` | text not null | 事件类型 |
| `decision` | text not null | 决策 |
| `record` | jsonb not null | 完整 AuditRecord |

索引：

- `(tenant_id, recorded_at desc)`
- `(tenant_id, decision)`
- `(tenant_id, event_type)`

### `policy_packs`

| 字段 | 类型 | 说明 |
|---|---|---|
| `tenant_id` | text not null | 租户 |
| `id` | text not null | 策略包 ID |
| `version` | text | 版本 |
| `enabled` | boolean not null | 是否启用 |
| `pack` | jsonb not null | 完整 PolicyPack |
| `updated_at` | timestamptz not null | 更新时间 |

主键：

- `(tenant_id, id)`

索引：

- `(tenant_id, enabled)`

## 运行时配置

| 环境变量 | 说明 |
|---|---|
| `DATABASE_URL` | 设置后 server 使用 PostgreSQL Store |
| `AGENT_SECURITY_POSTGRES_TEST_DSN` | 设置后启用 PostgreSQL 集成测试 |

## 关键设计决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 是否引入 ORM | 不引入 | 当前 SQL 简单，标准接口更清晰 |
| 是否默认跑集成测试 | 不默认 | 避免无 PostgreSQL 环境时阻塞开发 |
| 是否自动执行 migration | 不做 | 生产环境迁移应由部署系统控制 |
| JSONB 是否冗余字段 | 是 | 早期领域模型变化快，保留完整事件和策略包 |

## 验证方式

```bash
go test ./...
AGENT_SECURITY_POSTGRES_TEST_DSN='postgres://...' go test ./internal/storage/postgres -run Integration
```
