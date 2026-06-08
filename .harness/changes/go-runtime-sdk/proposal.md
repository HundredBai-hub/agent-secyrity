# Go Runtime SDK

## 背景

服务端已经具备运行时评估、策略包、审批、审批执行校验和 PostgreSQL 存储。但 Agent / 工具调用方如果直接拼 HTTP 请求，容易出现字段不一致、错误处理不统一、审批复用不规范等问题。生产落地需要一个稳定 SDK，作为 Agent 接入运行时安全控制面的标准方式。

本变更新增 Go Runtime SDK，为 Go Agent、工具代理、网关插件提供最小稳定接入层。

## 目标

- 新增 `sdk/go/agentsec` 包。
- 提供 `Client.Evaluate` 调用 `/v1/evaluate`。
- 提供 `Client.DecideApproval` 调用审批决定接口。
- 提供 `Client.ListApprovals` 和 `Client.GetApproval`。
- 提供 `Decision` helper：`Allowed()`、`Denied()`、`RequiresApproval()`。
- 支持自定义 `http.Client`、base URL、User-Agent。
- 支持 context、超时和非 2xx 错误返回。
- 提供 httptest 单元测试和 README 示例。

## 技术栈约束

- SDK 只使用 Go 标准库。
- 不引入新依赖。
- 默认 `go test ./...` 通过。

## 范围

- In scope:
  - SDK Client。
  - evaluate / approval APIs。
  - 错误处理。
  - 测试和文档。
- Out of scope:
  - 不做重试/backoff。
  - 不做 API Key 鉴权。
  - 不做 Node/Python SDK。
  - 不做自动工具拦截器。

## 验收标准

- SDK 可以提交 Runtime Event 并解析 EvaluationResult。
- SDK 可以处理非 2xx 错误。
- SDK 可以 approve/reject 审批。
- SDK helper 能准确识别 allow / deny / require_approval。
- `go test ./...` 通过。
