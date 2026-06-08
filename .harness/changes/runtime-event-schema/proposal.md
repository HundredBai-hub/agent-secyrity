# Runtime Event Schema

## 背景

`RuntimeEvent` 是 Agent Security Platform 的核心输入契约。当前字段已经覆盖租户、Agent、用户、任务、事件类型、主体、工具、资源、动作、数据标签和意图，但还缺少明确的 schema version、结构化校验错误和 API 层稳定错误语义。

如果不先固化事件契约，后续 SDK、策略模拟、基线策略包、审计查询和业务 benchmark 都会缺少稳定基础。

## 目标

- 给 `RuntimeEvent` 增加 `schema_version` 字段。
- 定义当前默认版本 `runtime_event.v1`。
- 允许缺省版本，服务端自动归一化为当前版本，保持旧调用兼容。
- 对不支持的版本返回结构化校验错误。
- 将运行时事件校验错误从普通字符串升级为可机器读取的字段错误列表。
- HTTP API 对事件校验失败返回 422，并包含 `details.fields`。
- Go SDK 同步 `schema_version` 字段和常量。
- API 文档补充 Runtime Event Schema v1。

## 范围

- In scope:
  - `internal/domain.RuntimeEvent`。
  - 事件校验错误类型。
  - HTTP `/v1/evaluate` 校验错误响应。
  - SDK wire type。
  - 文档和测试。
- Out of scope:
  - 不新增多版本兼容转换器。
  - 不改变策略引擎匹配语义。
  - 不改变数据库 schema。
  - 不实现 OpenAPI 生成。

## 验收标准

- 缺省 `schema_version` 的事件仍可通过校验，并归一化为 `runtime_event.v1`。
- 显式 `runtime_event.v1` 的事件可通过校验。
- 不支持的版本返回字段错误 `schema_version`。
- 缺少必填字段返回字段级错误列表。
- `POST /v1/evaluate` 对校验失败返回 422，而不是 400 或 500。
- SDK `RuntimeEvent` 暴露 `SchemaVersion` 和 `RuntimeEventSchemaV1`。
- `go test ./internal/domain ./internal/transport/httpapi ./sdk/go/agentsec` 通过。
- `go test ./...` 通过。
