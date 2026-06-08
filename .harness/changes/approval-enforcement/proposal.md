# Approval Enforcement

## 背景

系统现在可以在 `require_approval` 时生成审批单，也可以 approve / reject。但生产环境还缺少执行侧校验：Agent 在审批通过后需要带着 `approval_id` 再次提交同一运行时事件，Runtime 必须确认该审批属于同一租户、同一事件、状态已 approved，且未被串用到其他资源或工具上。

本变更把审批从“可处理对象”推进为“可执行凭证”，补上执行控制闭环。

## 目标

- Runtime Event 增加 `approval_id`。
- Runtime service 在事件带 `approval_id` 时执行审批校验。
- 审批通过且事件与原审批事件一致时，允许执行。
- 审批拒绝、过期、未决、跨租户、事件不匹配时拒绝。
- 评估结果明确返回原因。
- 审计记录保留 approval_id。

## 技术栈约束

- 沿用 Go 标准库和现有 Store。
- 默认 `go test ./...` 必须通过。
- 不引入新依赖。

## 范围

- In scope:
  - `approval_id` 输入字段。
  - 审批状态校验。
  - 事件绑定校验。
  - Runtime 测试和 API 文档。
- Out of scope:
  - 不做一次性 token 消耗。
  - 不做审批人权限校验。
  - 不做多级审批。

## 验收标准

- 已 approved 审批可以放行同一事件。
- pending / rejected / expired 审批不能放行。
- 跨租户 approval_id 不能使用。
- approval_id 不能用于不同工具、资源或动作。
- `go test ./...` 通过。
