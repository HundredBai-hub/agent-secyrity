# PostgreSQL Storage

## 配置

服务默认使用内存 Store。设置 `DATABASE_URL` 后启用 PostgreSQL Store：

```bash
export DATABASE_URL='postgres://agent_security_user:change-me@localhost:5432/agent_security?sslmode=disable'
go run ./cmd/server
```

启动时会执行幂等 migration，创建所需表和索引。

## Migration

部署系统也可以显式执行：

```text
migrations/postgres/001_init.sql
```

## 表

| 表 | 说明 |
|---|---|
| `audit_records` | 保存完整审计记录 JSONB，并冗余 tenant、event_type、decision、recorded_at |
| `policy_packs` | 保存完整策略包 JSONB，并冗余 tenant、enabled、version、updated_at |
| `approval_requests` | 保存完整审批单 JSONB，并冗余 tenant、status、requested_at、expires_at |

## 集成测试

默认 `go test ./...` 不依赖 PostgreSQL。需要显式设置 DSN：

```bash
AGENT_SECURITY_POSTGRES_TEST_DSN='postgres://agent_security_user:change-me@localhost:5432/agent_security_test?sslmode=disable' \
  go test ./internal/storage/postgres -run Integration
```
