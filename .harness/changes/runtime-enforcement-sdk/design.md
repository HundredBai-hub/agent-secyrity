# Runtime Enforcement SDK 设计

## 方案概述

SDK 新增一个薄的执行拦截层 `Enforcer`，它组合现有 `Client`，在真实业务动作执行前完成安全评估和审批控制。设计目标是让调用方获得一个明确的同步 API，同时保留审批等待策略的可替换性。

## 核心流程

```text
Operation(Event, Action)
  -> Client.Evaluate(Event)
  -> allow / record: execute Action
  -> deny / redact: return EnforcementError, do not execute Action
  -> require_approval:
       -> ApprovalWaiter.WaitApproval(...)
       -> approved: Event.ApprovalID = approval.ID
       -> Client.Evaluate(Event)
       -> allow / record: execute Action
       -> other decision: return EnforcementError
```

`redact` 作为响应类决策时通常意味着调用方需要进行脱敏处理。执行拦截器默认不擅自替调用方修改输出，因此把 `redact` 视为非立即执行决策，返回结构化错误，由调用方或后续响应拦截器处理。

## 公共类型

| 类型 | 说明 |
|---|---|
| `ActionFunc` | 被保护的真实动作 |
| `Operation` | 一次被保护执行，包含事件和动作 |
| `ApprovalWaiter` | 审批等待接口，由调用方或后续 SDK helper 实现 |
| `Enforcer` | 执行编排器 |
| `EnforcementResult` | 返回评估结果、审批结果和动作输出 |
| `EnforcementError` | 表示策略阻断、审批未通过、重评估仍不允许等安全语义错误 |

## 错误语义

`EnforcementError` 包含：

- `Decision`：导致动作未执行的决策。
- `Reason`：策略或 SDK 原因。
- `ApprovalID`：如存在审批单，便于调用方展示。

调用方可以用 `errors.As(err, *EnforcementError)` 识别安全阻断，与网络错误、执行函数错误区分。

## 审批等待策略

第一阶段只定义接口：

```go
type ApprovalWaiter interface {
    WaitApproval(ctx context.Context, client *Client, result EvaluationResult, event RuntimeEvent) (ApprovalRequest, error)
}
```

SDK 不默认轮询，原因：

- 不同企业审批来源可能是控制台、IM、工单或外部 SOAR。
- 自动轮询需要超时、间隔、退避和取消策略，不能在第一版隐式决定。
- 注入接口更便于测试，也方便后续新增 `PollingApprovalWaiter`。

若未配置 `ApprovalWaiter` 且遇到 `require_approval`，返回 `EnforcementError`，并携带 `ApprovalID`。

## 安全边界

| 风险 | 设计处理 |
|---|---|
| 未评估直接执行 | `Enforcer.Execute` 内部先 Evaluate，只有允许类决策才调用 Action |
| 审批后直接执行 | 审批通过后必须带 `approval_id` 再次 Evaluate |
| 审批拒绝仍执行 | 只接受 `ApprovalStatusApproved` |
| 调用方无法区分安全阻断和系统错误 | 使用 `EnforcementError` |
| SDK 隐式重复提交事件 | 不做网络层自动重试 |

## 验证方式

- `httptest` 模拟 API 服务，验证请求顺序和 `approval_id` 回写。
- 测试 `allow`、`deny`、`require_approval approved`、`require_approval rejected`、无 waiter、nil 输入。
- `go test ./sdk/go/agentsec`。
- `go test ./...`。
- harness validate/run 和敏感信息扫描。
