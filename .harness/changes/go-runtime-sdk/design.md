# Go Runtime SDK 设计

## 方案概述

新增公开 SDK 包 `sdk/go/agentsec`，作为 Go Agent、工具代理、网关插件接入 Agent Security Platform 的标准客户端。SDK 只依赖 Go 标准库，通过 REST / JSON 调用当前服务端 API，屏蔽 URL 拼接、JSON 编解码、非 2xx 错误解析和审批接口差异。

SDK 不依赖 `internal/domain`，避免外部业务项目无法导入 internal 包的问题。SDK 内定义稳定的 wire type，与 HTTP API JSON 字段保持一致，后续服务端内部模型可以演进，SDK 公共接口保持兼容。

## 语言与运行时

| 项 | 选择 |
|---|---|
| 目标项目语言 | Go |
| 验证入口 | `go test ./sdk/go/agentsec`、`go test ./...` |
| 执行环境 | 当前仓库本地工作区，受 harness 任务约束 |

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `sdk/go/agentsec` | 新增 | Go Runtime SDK 公共包，提供 Client、类型、错误处理和 helper |
| `docs/Go-SDK.md` | 新增 | SDK 使用说明、接入示例、审批复用方式 |
| `README.md` | 修改 | 增加 SDK 目录和验证说明 |
| `.harness/changes/go-runtime-sdk` | 修改 | 固化需求、设计、任务与验收标准 |

## 公共接口设计

| 接口 | 说明 |
|---|---|
| `NewClient(baseURL string, opts ...Option) (*Client, error)` | 创建 SDK Client，校验 base URL，支持自定义选项 |
| `WithHTTPClient(*http.Client)` | 注入自定义 HTTP 客户端，用于超时、代理、测试替换 |
| `WithUserAgent(string)` | 设置 User-Agent，便于服务端审计与网关识别 |
| `Client.Evaluate(ctx, RuntimeEvent)` | 调用 `/v1/evaluate`，返回 `EvaluationResult` |
| `Client.ListApprovals(ctx, tenantID)` | 查询租户审批单列表 |
| `Client.GetApproval(ctx, tenantID, approvalID)` | 查询单个审批单 |
| `Client.DecideApproval(ctx, tenantID, approvalID, ApprovalDecisionInput)` | 审批通过或拒绝 |
| `EvaluationResult.Allowed/Denied/RequiresApproval` | 调用侧根据决策分支处理 |
| `APIError` | 承载 HTTP 状态码、服务端错误码和错误消息 |

## 数据模型边界

| 类型 | 说明 |
|---|---|
| `RuntimeEvent` | Agent 运行时事件，包含租户、Agent、用户、任务、事件类型、工具、资源、动作、输入输出、数据标签、意图和审批 ID |
| `Subject` | 结构化主体身份，表达用户、Agent、服务账号和工作流身份 |
| `EvaluationResult` | 策略评估结果，包含决策、原因、命中策略、审计 ID、审批 ID |
| `ApprovalRequest` | 审批单视图，包含原事件、评估结果、状态、过期时间和审批人信息 |
| `ApprovalDecisionInput` | 审批动作输入，包含 `approved` 或 `rejected`、审批人和原因 |

## 错误处理

- SDK 对非 2xx 响应返回 `*APIError`。
- `APIError` 包含 `StatusCode`、`Code`、`Message`。
- 服务端返回标准错误 JSON 时解析 `error` 和 `message` 字段。
- 服务端返回非标准响应时使用 HTTP 状态文本和响应体摘要构造错误。
- 所有请求支持 `context.Context`，调用方可以控制取消和超时。

## Osmani Harness 设计检查

| 检查项 | 本方案如何满足 |
|---|---|
| 文件系统和 Git 状态 | 只修改 SDK、文档和当前 change 文件；变更前后检查 `git status --short --branch` |
| 工具和命令 | 使用 `go test ./sdk/go/agentsec` 做局部验证，使用 `go test ./...` 做全量验证 |
| 沙箱和权限 | 不需要网络下载，不引入新依赖；只在当前仓库可写目录内操作 |
| Hooks 和策略 | 使用 `gofmt`、Go 单元测试、harness validate/run、敏感信息扫描作为完成前检查 |
| 上下文管理 | 读取 HTTP API、domain model、README、现有 docs；大输出不写入代码上下文 |
| 观测与报告 | harness 任务输出保存在 `.harness/reports`，Git diff 和测试结果作为人工审查依据 |
| 恢复路径 | 单个 SDK change 可通过 Git diff 审查和单 commit 回滚；测试失败先定位最小原因再修复 |

## Superpowers 工作流检查

| 检查项 | 本方案如何满足 |
|---|---|
| 需求澄清 | 目标用户是 Go Agent / 工具代理 / 网关插件；本次只做 Evaluate 和 Approval SDK |
| 计划拆解 | 任务拆成公共类型、Evaluate、Approval、错误处理、文档、验证 |
| TDD / 测试先行 | 先写 `httptest` 单元测试并确认失败，再实现 SDK |
| 隔离执行 | 当前主线工作区干净；变更范围小，不额外创建 worktree |
| 系统化调试 | 测试失败时先看请求路径、JSON 字段、错误类型，再做最小修复 |
| 代码审查 | 重点检查公共接口稳定性、internal 依赖边界、错误可诊断性和文档完整性 |
| 完成前验证 | 提交前重新运行 `go test ./...`、harness validate/run 和敏感信息扫描 |

## 风险与取舍

| 风险 | 影响 | 应对 |
|---|---|---|
| SDK wire type 与服务端模型重复 | 后续字段演进需要同步维护 | 公共 SDK 避免依赖 internal，这是对外可用性的必要取舍；用测试覆盖关键 JSON 字段 |
| 暂不做鉴权 | 不能直接用于生产公网暴露服务 | 本阶段平台服务还未完成鉴权体系，SDK 预留自定义 HTTP Client，鉴权在后续 change 中补充 |
| 暂不做重试 | 短暂网络抖动由调用方处理 | 安全决策链路优先保持确定性，避免 SDK 隐式重复提交审批或审计事件 |
| 错误模型依赖服务端 JSON | 非标准错误可读性下降 | 对非 JSON 响应回退到状态码、状态文本和响应体摘要 |
