# PostgreSQL Storage

## 背景

当前 Audit Record 和 Policy Pack 都使用内存 Store。服务重启后策略包和审计记录会丢失，无法满足生产环境对策略运营、审计追溯和故障恢复的要求。下一步需要引入 PostgreSQL 持久化存储，同时保持现有 Store 接口稳定，避免业务层绑定具体数据库实现。

## 目标

- 增加 PostgreSQL migration SQL。
- 增加 `internal/storage/postgres` 包。
- 实现 PostgreSQL Audit Store。
- 实现 PostgreSQL Policy Pack Store。
- server 支持通过 `DATABASE_URL` 启用 PostgreSQL，否则继续使用内存 Store。
- 保持 `go test ./...` 可在无 PostgreSQL 环境下通过。
- PostgreSQL 集成测试通过环境变量显式开启，避免本地/CI 没有数据库时失败。

## 技术栈约束

- 使用 PostgreSQL。
- Go 侧使用轻量驱动，不引入 ORM。
- 默认单元测试不依赖外部数据库。
- 集成测试通过 `AGENT_SECURITY_POSTGRES_TEST_DSN` 显式开启。

## 范围

- In scope:
  - migration SQL。
  - Postgres Store 实现。
  - server 存储选择逻辑。
  - 文档和测试。
- Out of scope:
  - 不做数据库连接池高级调优。
  - 不做自动迁移执行器。
  - 不做多库分片。
  - 不做复杂审计查询 DSL。

## 验收标准

- 无数据库环境下 `go test ./...` 通过。
- 配置 `DATABASE_URL` 后 server 使用 PostgreSQL Store。
- 配置 `AGENT_SECURITY_POSTGRES_TEST_DSN` 后可运行 PostgreSQL 集成测试。
- Audit Record 和 Policy Pack JSON 能写入、读取、列出、启停。

## Superpowers 工作流要求

| 阶段 | 本变更要求 |
|---|---|
| Spec | 明确 PostgreSQL、接口边界和默认测试策略 |
| TDD | 先写 Store 契约测试和集成测试入口 |
| Debugging | 失败先定位 SQL、JSON 序列化、tenant 过滤 |
| Verification | 完成前重新运行 `go test ./...` 和 harness run |
