# Runtime Enforcement SDK

## 背景

当前 Go SDK 已能提交运行时事件、读取评估结果和处理审批接口，但业务调用方仍需要手写以下控制流：

1. 在工具执行前调用 `Evaluate`。
2. 根据 `allow / record / redact / deny / require_approval` 分支处理。
3. 需要审批时等待审批完成。
4. 审批通过后带 `approval_id` 重新评估。
5. 评估允许后再执行真实业务动作。

这段逻辑是运行时安全的核心闭环。如果每个 Agent 项目都自行实现，容易出现漏拦截、审批后不重评估、拒绝后继续执行、错误处理不一致等问题。

## 目标

- 在 `sdk/go/agentsec` 中新增生产可用的执行拦截器 API。
- 封装 `evaluate -> approval -> retry -> execute` 流程。
- 调用方只需要传入运行时事件和真实执行函数。
- SDK 对 `deny`、`require_approval`、`allow/record` 给出稳定语义。
- 审批等待能力通过接口注入，SDK 不强制绑定某种轮询或 UI 实现。

## 范围

- In scope:
  - 新增 `Enforcer`、`Operation`、`ActionFunc`、`ApprovalWaiter` 等 SDK 类型。
  - 支持 allowed/record 时执行动作。
  - 支持 denied 时不执行动作并返回结构化错误。
  - 支持 require_approval 时等待审批、带 `approval_id` 重评估、允许后执行。
  - 补充 SDK 测试和文档。
- Out of scope:
  - 不做后台轮询 worker。
  - 不做审批 UI。
  - 不做自动重试网络错误。
  - 不做异步任务队列。
  - 不做非 Go SDK。

## API 草案

```go
type ActionFunc func(ctx context.Context) (any, error)

type ApprovalWaiter interface {
    WaitApproval(ctx context.Context, client *Client, result EvaluationResult, event RuntimeEvent) (ApprovalRequest, error)
}

type Operation struct {
    Event RuntimeEvent
    Action ActionFunc
}

type EnforcementResult struct {
    Evaluation EvaluationResult
    Approval   *ApprovalRequest
    Output     any
}

type Enforcer struct {
    // constructed by NewEnforcer(client, opts...)
}

func NewEnforcer(client *Client, opts ...EnforcerOption) (*Enforcer, error)
func (e *Enforcer) Execute(ctx context.Context, op Operation) (EnforcementResult, error)
```

## 验收标准

- `allow` / `record` 决策会执行 `Action` 并返回输出。
- `deny` 决策不会执行 `Action`，返回可识别的 enforcement error。
- `require_approval` 决策会调用注入的 `ApprovalWaiter`。
- 审批通过后 SDK 会把 `approval_id` 写回事件并再次 `Evaluate`。
- 重评估允许后才执行 `Action`。
- 审批拒绝、过期或重评估仍不允许时不会执行 `Action`。
- `nil Client`、`nil Action` 等输入错误有明确返回。
- `go test ./sdk/go/agentsec` 通过。
- `go test ./...` 通过。
