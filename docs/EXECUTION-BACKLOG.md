# Agent Security Platform Execution Backlog

本文档是后续自动开发的任务池。默认执行顺序为 P0 从上到下，然后 P1，再 P2。每个任务必须先生成 harness change 和 task specs，再进入实现循环。

## 执行状态说明

| 状态 | 含义 |
|---|---|
| `todo` | 已进入任务池，尚未创建 change |
| `planned` | 已创建 `.harness/changes/<change-id>` 和 task specs |
| `in_progress` | 正在按 TDD / harness 循环实现 |
| `done` | 已测试、提交并推送 |
| `blocked` | 需要用户决策、外部依赖或方向调整 |

## P0：生产核心闭环

| 顺序 | Change ID | 状态 | 目标 | 验收命令 |
|---:|---|---|---|---|
| 1 | `runtime-event-schema` | done | 固化 Runtime Event Schema、字段版本、校验错误和兼容策略 | `go test ./internal/domain ./internal/transport/httpapi ./sdk/go/agentsec` |
| 2 | `policy-simulator` | done | 支持策略包 dry-run / simulate，用于上线前验证策略命中和误拦截 | `go test ./internal/policy ./internal/transport/httpapi` |
| 3 | `baseline-policy-packs` | done | 提供内置高频场景策略包：代码仓库、客服、财务审批、数据分析 | `go test ./internal/policypack ./internal/runtime` |
| 4 | `audit-query-api` | done | 增强审计查询过滤、分页、按 Agent / User / Task / Decision 检索 | `go test ./internal/audit ./internal/storage/postgres ./internal/transport/httpapi` |
| 5 | `runtime-enforcement-sdk` | todo | Go SDK 增加执行拦截器，封装 evaluate -> approval -> retry -> execute 流程 | `go test ./sdk/go/agentsec` |
| 6 | `business-scenario-benchmarks` | todo | 建立真实业务场景 benchmark 和端到端验收用例 | `go test ./...` |

## P1：企业运营能力

| 顺序 | Change ID | 状态 | 目标 | 验收命令 |
|---:|---|---|---|---|
| 7 | `api-key-lifecycle` | todo | API Key 创建、禁用、轮换、审计，不再只依赖环境变量 | `go test ./internal/auth ./internal/storage/postgres ./internal/transport/httpapi` |
| 8 | `rbac-and-roles` | todo | 管理 API 角色权限模型，区分 runtime writer、policy admin、approver、auditor | `go test ./internal/auth ./internal/transport/httpapi` |
| 9 | `policy-pack-release` | todo | 策略包版本、发布、回滚、启用范围和变更审计 | `go test ./internal/policypack ./internal/storage/postgres ./internal/transport/httpapi` |
| 10 | `management-api` | todo | 管理面 API 聚合：租户、策略包、审批、审计、API Key | `go test ./internal/transport/httpapi` |
| 11 | `operator-console` | todo | 控制台基础页面：策略包、审批队列、审计查询、运行时事件 | `npm test` 或对应前端测试命令 |
| 12 | `observability` | todo | 结构化日志、指标、请求 ID、审计链路 ID、运行状态探针 | `go test ./...` |

## P2：规模化和产品壁垒

| 顺序 | Change ID | 状态 | 目标 | 验收命令 |
|---:|---|---|---|---|
| 13 | `agent-behavior-baseline` | todo | Agent 行为基线、异常行为检测和风险评分 | `go test ./internal/runtime ./internal/policy` |
| 14 | `shadow-mode` | todo | 策略影子运行、只记录不阻断、误报分析 | `go test ./internal/runtime ./internal/transport/httpapi` |
| 15 | `mcp-gateway-adapter` | todo | MCP 工具调用网关适配，拦截工具列表和工具调用 | `go test ./...` |
| 16 | `python-sdk` | todo | Python SDK 支持运行时评估、审批和执行拦截器 | `pytest` |
| 17 | `deployment-packaging` | todo | Docker、compose、生产配置模板、迁移脚本、健康检查 | `go test ./...` 和容器构建命令 |
| 18 | `ci-quality-gates` | todo | GitHub Actions：测试、lint、敏感信息扫描、harness validate | CI 通过 |

## 当前执行指针

下一项：`runtime-enforcement-sdk`

## 单任务执行模板

每个 backlog 任务必须按以下步骤推进：

1. 创建 change：`go run ./cmd/harness change init --workspace <repo> --id <change-id> --title "<title>" --force`
2. 写 proposal/design/tasks。
3. 生成 task specs：`go run ./cmd/harness change task-specs --workspace <repo> --id <change-id> --force`
4. 写红灯测试。
5. 运行局部测试，确认预期失败。
6. 最小实现。
7. 运行局部测试和 `go test ./...`。
8. 更新文档和模块索引。
9. 运行 harness validate/run。
10. 敏感信息扫描。
11. code review。
12. commit + push。
13. 更新本 backlog 状态和当前执行指针。

## 阻塞规则

只有以下情况需要暂停等待用户：

- 涉及产品方向取舍，例如是否先做控制台还是 SDK。
- 涉及外部系统凭据、真实生产环境、数据库连接或部署权限。
- 涉及破坏性迁移、删除数据、强推、回滚等高风险操作。
- 测试连续三轮失败且无法定位根因。

除此之外，默认按任务池顺序持续推进。
