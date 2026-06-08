# API Key Authentication

## 背景

当前服务端已经提供运行时评估、策略包管理、审批流、审计查询和 Go SDK，但 HTTP API 默认未鉴权。生产部署时，如果只依赖网络边界，任何能访问服务的人都可以提交事件、查询审计、修改策略包或处理审批单。

本变更新增最小生产可用的 API Key 认证与租户访问边界。它不是完整 IAM / RBAC，而是服务端接入真实企业环境前必须具备的基础控制面防线。

## 目标

- 服务端支持通过环境变量配置静态 API Key。
- 客户端通过 `Authorization: Bearer <key>` 调用 API。
- 未配置 API Key 时保持本地开发兼容，默认不启用鉴权。
- 启用鉴权后，除 `GET /healthz` 外所有 API 都需要有效 API Key。
- 支持按 API Key 绑定允许访问的租户集合。
- 路径租户、请求体租户和 API Key 允许租户不匹配时返回 403。
- 审计查询支持 `tenant_id` 过滤，避免跨租户审计数据泄露。
- Go SDK 支持 `WithAPIKey` 自动注入 Authorization Header。

## 技术栈约束

- 语言：Go。
- 依赖：只使用 Go 标准库，不引入第三方鉴权库。
- 测试命令：`go test ./internal/transport/httpapi ./sdk/go/agentsec`、`go test ./...`。
- 运行方式：本地环境变量配置，不接入外部密钥管理系统。

## 范围

- In scope:
  - HTTP middleware。
  - 静态 API Key 配置解析。
  - 租户访问校验。
  - 审计查询租户过滤。
  - SDK Header 注入。
  - 测试和文档。
- Out of scope:
  - 不做用户登录。
  - 不做 OAuth / OIDC。
  - 不做密钥创建、轮换、禁用 API。
  - 不做细粒度角色权限。
  - 不做数据库持久化 API Key。

## 验收标准

- [ ] 未配置 API Key 时，现有测试与本地开发行为保持兼容。
- [ ] 启用 API Key 后，无 Authorization Header 返回 401。
- [ ] 无效 API Key 返回 401。
- [ ] API Key 与租户不匹配返回 403。
- [ ] `GET /healthz` 不需要鉴权。
- [ ] `GET /v1/audit/events?tenant_id=tenant-a` 只返回对应租户审计记录。
- [ ] Go SDK `WithAPIKey` 会发送 Bearer Token。
- [ ] `go test ./...` 通过。

## Osmani Harness 约束映射

| 约束 | 本变更要求 |
|---|---|
| Behavior-first | 先以 HTTP 状态码、Header、租户匹配和审计过滤定义可观察行为 |
| Context | 读取 HTTP handler、audit store、cmd/server、SDK 和现有 API 文档 |
| Boundaries | 不引入完整 IAM，不接触真实密钥，不改变现有领域模型主流程 |
| Verification | 使用 handler/SDK 单元测试、全量 `go test ./...`、harness validate/run |
| Feedback loop | 认证或租户测试失败时，根据状态码、错误码和请求路径定位 |
| Auditability | change/task 文档、测试输出、diff、提交记录可追踪 |

## Superpowers 工作流借鉴

| 阶段 | 本变更要求 |
|---|---|
| Spec | 明确本次是静态 API Key 和租户边界，不是账号体系 |
| Writing Plans | 拆成 auth 配置、middleware、租户校验、审计过滤、SDK、文档 |
| TDD | 先写 handler 和 SDK 红灯测试，再实现 |
| Systematic Debugging | 鉴权失败先区分 401 认证失败与 403 授权失败 |
| Code Review | 重点检查 secret 泄露、跨租户访问、错误语义和默认兼容性 |
| Verification Before Completion | 提交前重新运行完整测试和敏感信息扫描 |
