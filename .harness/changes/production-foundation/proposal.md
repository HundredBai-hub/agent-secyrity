# Production Foundation

## 背景

企业 AI Agent 正从问答走向执行：客服 Agent 调用工单系统，研发 Agent 读取代码和执行命令，数据 Agent 查询内部数据，运营 Agent 调用 SaaS API。生产环境需要的不只是提示词过滤，而是一套运行时安全控制面，能在 Agent 每一次工具调用、资源访问、响应输出和审批动作发生时，完成可解释的策略判断、处置和审计。

本变更建立生产级基础骨架。目标不是演示 MVP，而是形成可以持续演进为企业产品的核心后端平台。

## 目标

- 建立稳定的领域模型：Runtime Event、Policy、Decision、Audit Record。
- 实现可测试的策略评估引擎，支持条件匹配和处置动作。
- 实现 Runtime Evaluation Service，串联事件接收、策略评估、审计记录和决策返回。
- 提供最小生产可用 HTTP API：`POST /v1/evaluate`、`GET /v1/audit/events`、`GET /healthz`。
- 提供内存审计存储接口，为后续 PostgreSQL / ClickHouse / OpenSearch 替换预留边界。
- 建立测试体系和 harness 验证闭环。
- 初始化独立 Git 仓库并推送 GitHub。

## 技术栈约束

- 当前生产基础骨架采用 Go，实现安全网关、策略评估、审计服务和 HTTP API。
- 后续 SDK、控制台、采集插件不强制使用 Go。
- 第一阶段只使用 Go 标准库，降低供应链风险。
- 所有任务必须通过 `go test ./...` 验证。

## 范围

- In scope:
  - Go module 初始化。
  - 领域模型与输入校验。
  - 策略引擎。
  - 审计存储接口和内存实现。
  - Runtime service 编排。
  - HTTP API 和健康检查。
  - 单元测试和 API 测试。
  - harness change、task、report。
  - GitHub 初始化推送。
- Out of scope:
  - 不接真实生产 Agent 平台。
  - 不接真实密钥、生产数据库和真实用户敏感数据。
  - 不做 Web 控制台。
  - 不做复杂机器学习研判。
  - 不做分布式部署和多租户计费。

## 验收标准

- `go test ./...` 通过。
- `POST /v1/evaluate` 能对示例事件返回 allow、deny、redact、require_approval 或 record。
- 每次评估都会写入 audit record。
- `GET /v1/audit/events` 能返回已记录事件。
- 高风险示例事件能被策略阻断。
- 敏感输出示例能被策略要求脱敏。
- 代码不包含真实密钥、生产凭据或硬编码敏感配置。

## Osmani Harness 约束映射

| 约束 | 本变更要求 |
|---|---|
| Behavior-first | 以 Agent 运行时事件评估和决策返回作为用户可观察行为 |
| Context | proposal、design、task spec、测试用例共同描述上下文 |
| Boundaries | 禁止生产系统、真实密钥、真实用户数据和自动合并 |
| Verification | `go test ./...` 作为完成前验证命令 |
| Feedback loop | harness 报告记录失败命令、失败类型和修复尝试 |
| Auditability | Runtime service 每次评估都产生 audit record |

## Superpowers 工作流借鉴

| 阶段 | 本变更要求 |
|---|---|
| Brainstorm / Spec | 先明确生产级范围、非目标和验收标准 |
| Writing Plans | 任务拆成领域模型、策略、审计、服务、API、验证和仓库管理 |
| Test-Driven Development | 领域模型、策略引擎、Runtime service、HTTP API 测试先行 |
| Using Git Worktrees | 后续真实迭代默认启用 worktree，首个仓库初始化在当前目录完成 |
| Systematic Debugging | 验证失败先定位根因，再做单一最小修复 |
| Code Review | 完成前检查安全边界、错误处理、测试覆盖和 API 稳定性 |
| Verification Before Completion | 未重新运行 `go test ./...` 前不声明完成 |
