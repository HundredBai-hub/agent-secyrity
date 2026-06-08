# Tenant Policy Foundation

## 背景

当前生产基础骨架已经具备运行时事件、策略评估、审计记录和 HTTP API。企业级落地还缺少三个关键拼图：租户隔离、主体身份和策略包。没有租户维度，审计和策略无法服务多企业部署；没有主体身份，策略只能按 user_id / agent_id 粗粒度匹配；没有策略包，策略无法按企业、团队、场景进行版本化治理。

本变更建立企业控制面的基础数据模型和评估边界，让后续接入审批、策略分发、多租户存储和控制台时有稳定核心。

## 目标

- Runtime Event 增加 `tenant_id` 和结构化 `subject`。
- Policy 增加 `tenant_id`、`policy_pack_id` 和主体条件。
- 新增 Policy Pack 模型，支持启用状态、版本和策略集合。
- Policy Engine 按租户隔离评估，禁止跨租户策略命中。
- Runtime service 审计记录保留租户和主体信息。
- HTTP API 兼容新字段。
- 增加生产级场景测试：跨租户策略隔离、主体角色匹配、策略包禁用。

## 技术栈约束

- 沿用 Go 和标准库。
- 所有行为通过 `go test ./...` 验证。
- 仅使用合成测试数据，不引入真实租户、用户、密钥和生产资源。

## 范围

- In scope:
  - 领域模型扩展。
  - 策略包模型。
  - 策略引擎租户隔离。
  - 主体条件匹配。
  - HTTP JSON 兼容。
  - 测试与文档更新。
- Out of scope:
  - 不做租户鉴权登录。
  - 不做持久化数据库迁移。
  - 不做策略包发布流程和控制台。
  - 不做多租户计费和配额。

## 验收标准

- 缺少 `tenant_id` 的 Runtime Event 校验失败。
- tenant-a 的策略不会命中 tenant-b 的事件。
- subject role / type 条件可以命中策略。
- 禁用的 Policy Pack 不参与评估。
- `go test ./...` 通过。

## Osmani Harness 约束映射

| 约束 | 本变更要求 |
|---|---|
| Behavior-first | 以跨租户隔离、主体匹配和策略包启停作为可观察行为 |
| Context | 变更上下文写入 proposal、design、task spec 和测试 |
| Boundaries | 不接真实租户数据、不做登录鉴权、不改生产系统 |
| Verification | `go test ./...` 覆盖领域、策略、运行时和 API |
| Feedback loop | harness 报告记录验证失败和修复循环 |
| Auditability | 审计记录保留 tenant 和 subject 字段 |

## Superpowers 工作流借鉴

| 阶段 | 本变更要求 |
|---|---|
| Brainstorm / Spec | 明确租户、主体、策略包的生产价值和非目标 |
| Writing Plans | 拆成模型、策略引擎、服务、API、场景测试和文档任务 |
| Test-Driven Development | 先写跨租户、主体匹配、策略包禁用测试 |
| Systematic Debugging | 失败先定位匹配逻辑和模型校验边界 |
| Verification Before Completion | 完成前重新运行 `go test ./...` |
