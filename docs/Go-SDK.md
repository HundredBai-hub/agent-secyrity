# Go Runtime SDK

`sdk/go/agentsec` 是 Agent Security Platform 的 Go 接入 SDK。它面向 Go Agent、工具代理、网关插件和企业内部编排服务，负责把运行时事件提交到安全控制面，并把策略决策、审批单和错误响应转换成稳定的 Go 类型。

SDK 只使用 Go 标准库，不依赖服务端 `internal` 包。业务项目可以直接导入 SDK 包，不需要理解服务端内部领域模型。

## 安装与导入

当前仓库模块路径：

```go
import "github.com/HundredBai-hub/agent-secyrity/sdk/go/agentsec"
```

## 创建 Client

```go
client, err := agentsec.NewClient(
	"http://127.0.0.1:8080",
	agentsec.WithUserAgent("my-agent-runtime/1.0"),
	agentsec.WithAPIKey(os.Getenv("AGENT_SECURITY_API_KEY")),
)
if err != nil {
	return err
}
```

`WithAPIKey` 会自动发送 `Authorization: Bearer <key>`。真实密钥应由环境变量、密钥管理系统或部署平台注入，不应写死在业务代码中。

如需统一超时、代理、TLS 或网关 Header，可注入自定义 `http.Client`：

```go
httpClient := &http.Client{Timeout: 3 * time.Second}
client, err := agentsec.NewClient(
	"http://127.0.0.1:8080",
	agentsec.WithHTTPClient(httpClient),
)
```

## 提交运行时事件

Agent 在执行工具、访问文件、访问网络或输出响应前，可以调用 `Evaluate` 获取安全决策。

```go
result, err := client.Evaluate(ctx, agentsec.RuntimeEvent{
	TenantID:  "tenant-a",
	AgentID:   "agent-code-001",
	UserID:    "dev-001",
	TaskID:    "fix-build",
	EventType: agentsec.EventTypeToolCall,
	Subject: agentsec.Subject{
		Type:      agentsec.SubjectTypeUser,
		ID:        "dev-001",
		Roles:     []string{"developer"},
		Groups:    []string{"engineering"},
		RiskLevel: "medium",
	},
	ToolName:   "shell",
	Resource:   "/repo/.env",
	Action:     "read",
	DataLabels: []string{"secret"},
	Intent:     "debug build failure",
})
if err != nil {
	return err
}

switch {
case result.Allowed():
	// 继续执行工具调用。
case result.RequiresApproval():
	// 暂停执行，将 result.ApprovalID 展示给审批流或运营控制台。
case result.Denied():
	// 阻断工具调用，并把 result.Reason 返回给上层编排。
}
```

## 审批复用

当决策为 `require_approval` 时，服务端会返回 `approval_id`。审批通过后，Agent 需要重新提交同一运行时事件，并带上该 `approval_id`。

```go
event := agentsec.RuntimeEvent{
	TenantID:  "tenant-a",
	AgentID:   "agent-code-001",
	UserID:    "dev-001",
	TaskID:    "fix-build",
	EventType: agentsec.EventTypeToolCall,
	ToolName:  "shell",
	Resource:  "/repo/.env",
	Action:    "read",
}

first, err := client.Evaluate(ctx, event)
if err != nil {
	return err
}
if first.RequiresApproval() {
	event.ApprovalID = first.ApprovalID
	second, err := client.Evaluate(ctx, event)
	if err != nil {
		return err
	}
	if !second.Allowed() {
		return fmt.Errorf("approved event is still blocked: %s", second.Reason)
	}
}
```

服务端会校验审批单租户、审批状态和事件绑定字段。不同 Agent、用户、任务、工具、资源或动作不能复用同一个审批单。

## 查询和处理审批单

```go
approvals, err := client.ListApprovals(ctx, "tenant-a")
if err != nil {
	return err
}
for _, approval := range approvals {
	if approval.Status == agentsec.ApprovalStatusPending {
		// 展示给安全运营人员或业务负责人。
	}
}
```

审批通过：

```go
approved, err := client.DecideApproval(ctx, "tenant-a", "approval-1", agentsec.ApprovalDecisionInput{
	Decision:  agentsec.ApprovalStatusApproved,
	DecidedBy: "secops-001",
	Reason:    "approved for incident response",
})
if err != nil {
	return err
}
_ = approved
```

审批拒绝：

```go
rejected, err := client.DecideApproval(ctx, "tenant-a", "approval-1", agentsec.ApprovalDecisionInput{
	Decision:  agentsec.ApprovalStatusRejected,
	DecidedBy: "secops-001",
	Reason:    "secret file access is not allowed for this task",
})
if err != nil {
	return err
}
_ = rejected
```

## 错误处理

服务端返回非 2xx 状态码时，SDK 返回 `*agentsec.APIError`。

```go
result, err := client.Evaluate(ctx, event)
if err != nil {
	var apiErr *agentsec.APIError
	if errors.As(err, &apiErr) {
		log.Printf("agent security API failed: status=%d code=%s message=%s",
			apiErr.StatusCode,
			apiErr.Code,
			apiErr.Message,
		)
		return err
	}
	return err
}
_ = result
```

## 当前限制

- SDK 暂不内置 API Key、OAuth 或 mTLS 鉴权；后续鉴权能力会与服务端身份体系一起设计。
- SDK 暂不内置 retry/backoff，避免重复提交审批和审计事件；调用方可以通过自定义 `http.Client` 和上层编排控制重试策略。
- SDK 当前覆盖运行时评估和审批接口，策略包管理、审计查询和多语言 SDK 属于后续演进范围。
