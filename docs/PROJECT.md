# Agent Security Platform 项目索引

Agent Security Platform 是面向企业 AI Agent 运行时安全的控制面与执行前评估平台。当前工程以 Go 服务端为基础，提供运行时事件采集、策略评估、审计、策略包治理、审批流和 Go Runtime SDK。

## 模块列表

| 模块 | 路径 | 职责 | 模块文档 |
|---|---|---|---|
| HTTP Server | `cmd/server` | 启动 HTTP 服务，选择内存或 PostgreSQL 存储 | 随服务启动与部署变更补齐 |
| Auth | `internal/auth` | API Key 配置解析、Bearer 认证和租户授权 | [API Key Auth 模块说明](modules/api-key-auth/MODULE.md) |
| Baseline Policy Packs | `internal/baseline` | 内置高频业务场景策略包模板 | [Baseline Policy Packs 模块说明](modules/baseline-policy-packs/MODULE.md) |
| Domain | `internal/domain` | 运行时事件、策略、决策、审批、审计领域模型 | 随领域模型稳定化变更补齐 |
| Runtime Service | `internal/runtime` | 串联策略评估、审计记录和审批执行校验 | 随运行时编排变更补齐 |
| Policy Engine | `internal/policy` | 根据租户策略、主体、资源、动作和数据标签计算决策 | 随策略语言变更补齐 |
| Audit Store | `internal/audit` | 审计记录存储接口、过滤分页查询和内存实现 | [Audit Query API 模块说明](modules/audit-query-api/MODULE.md) |
| Policy Pack Store | `internal/policypack` | 策略包治理接口与内存实现 | 随策略包治理变更补齐 |
| Approval Store | `internal/approval` | 审批单状态流转、过期和决策接口 | 随审批流扩展变更补齐 |
| PostgreSQL Store | `internal/storage/postgres` | 审计、策略包、审批单的 PostgreSQL 持久化实现 | 随数据库迁移变更补齐 |
| HTTP API | `internal/transport/httpapi` | REST / JSON API 路由、请求解析和错误响应 | 见 [API.md](API.md) |
| Go Runtime SDK | `sdk/go/agentsec` | Go Agent / 工具代理 / 网关插件接入运行时安全平台的公共 SDK | [Go SDK 模块说明](modules/go-runtime-sdk/MODULE.md) |
| Operator Console | `web/console` | 安全运营控制台，覆盖运行时总览、策略包、审批队列和审计查询 | [Operator Console 模块说明](modules/operator-console/MODULE.md) |

## 文档索引

| 文档 | 说明 |
|---|---|
| [API.md](API.md) | HTTP API、运行时事件、策略包和审批流接口 |
| [Go-SDK.md](Go-SDK.md) | Go Runtime SDK 接入说明和示例 |
| [ROADMAP.md](ROADMAP.md) | 产品和工程阶段路线 |
| [EXECUTION-BACKLOG.md](EXECUTION-BACKLOG.md) | 后续开发任务池、优先级、状态和当前执行指针 |
| [HARNESS-EXECUTION-LOOP.md](HARNESS-EXECUTION-LOOP.md) | harness 自动执行循环规则 |
| [PostgreSQL.md](PostgreSQL.md) | PostgreSQL 本地配置、迁移和验证方式 |
| [项目启动.md](项目启动.md) | 项目启动背景和阶段目标 |

## 约束

- 服务端内部模块保持 `internal` 边界，外部业务项目通过 HTTP API 或 SDK 接入。
- SDK 公共类型不直接依赖服务端 `internal/domain`，以保证外部可导入和接口稳定。
- 新增生产能力需要同步维护对应测试、文档和 harness change / task。
