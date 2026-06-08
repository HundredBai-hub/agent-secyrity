# Go Runtime SDK 模块说明

## 模块职责

`sdk/go/agentsec` 提供 Agent Security Platform 的 Go 公共 SDK。它面向 Go Agent、工具代理、网关插件和内部编排服务，封装运行时评估和审批 API，让调用方不用手工拼接 HTTP 请求、解析 JSON 错误或依赖服务端 `internal` 包。

## 关键依赖

| 依赖 | 用途 |
|---|---|
| Go 标准库 `net/http` | 发送 API 请求 |
| Go 标准库 `encoding/json` | 编解码 REST / JSON 请求响应 |
| Go 标准库 `context` | 传递取消和超时控制 |
| Go 标准库 `net/url` | 校验 base URL、转义路径参数 |

## 文件清单

| 文件 | 作用 |
|---|---|
| `sdk/go/agentsec/client.go` | SDK 公共类型、Client 配置、Evaluate、Approval API、错误处理和决策 helper |
| `sdk/go/agentsec/enforcer.go` | 执行拦截器，封装评估、审批等待、重评估和真实动作执行 |
| `sdk/go/agentsec/client_test.go` | 基于 `httptest` 的 SDK 行为测试，覆盖请求路径、JSON 字段、错误解析、审批接口和 helper |
| `sdk/go/agentsec/enforcer_test.go` | 覆盖 allow/deny/require_approval、审批拒绝、无 waiter、重评估阻断和 nil 输入 |

## 公共能力

| 能力 | 说明 |
|---|---|
| Runtime Evaluate | 提交 `RuntimeEvent`，返回 `EvaluationResult` |
| Decision Helper | 通过 `Allowed()`、`Denied()`、`RequiresApproval()` 简化调用方分支 |
| Approval Query | 查询租户审批单列表和单个审批单 |
| Approval Decide | 提交审批通过或拒绝动作 |
| Runtime Enforcer | 封装 `evaluate -> approval -> retry -> execute` 运行时安全闭环 |
| APIError | 非 2xx 响应统一返回 `StatusCode`、`Code`、`Message` |
| Client Option | 支持自定义 `http.Client`、`User-Agent` 和 API Key |

## 重要约束

- SDK 不导入 `internal/domain`，避免外部项目无法导入 internal 包。
- SDK 只负责注入 API Key Header，不负责密钥存储、轮换或刷新。
- SDK 暂不内置重试，避免在安全决策链路中隐式重复提交事件。
- `Enforcer` 遇到审批通过后必须重新评估，不能把审批结果直接等同于放行。
- SDK wire type 与 HTTP API JSON 字段保持一致，新增字段应优先做到向后兼容。
