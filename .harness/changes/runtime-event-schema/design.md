# Runtime Event Schema 设计

## 方案概述

为运行时事件定义 v1 版本契约：

```json
{
  "schema_version": "runtime_event.v1",
  "tenant_id": "tenant-a",
  "agent_id": "agent-code-001",
  "user_id": "dev-001",
  "task_id": "fix-build",
  "event_type": "tool_call",
  "subject": {"type": "user", "id": "dev-001"},
  "tool_name": "shell",
  "resource": "/repo/.env",
  "action": "execute",
  "data_labels": ["secret"],
  "intent": "debug build failure"
}
```

`schema_version` 缺省时视为 `runtime_event.v1`，服务端在校验时归一化。这样旧 SDK 和已有测试不需要立即改造。

## 数据模型

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `schema_version` | string | 否 | 缺省按 `runtime_event.v1` 处理 |
| `tenant_id` | string | 是 | 租户隔离边界 |
| `agent_id` | string | 是 | Agent 身份 |
| `user_id` | string | 是 | 发起用户 |
| `task_id` | string | 是 | 业务任务 |
| `event_type` | enum | 是 | prompt / tool_call / file_access / network_access / response / approval |
| `action` | string | 是 | read / write / execute / call 等动作 |

## 校验错误

新增领域错误类型：

| 类型 | 说明 |
|---|---|
| `ValidationError` | 表示运行时事件或后续领域对象校验失败 |
| `FieldError` | 包含 `field`、`code`、`message` |

示例：

```json
{
  "error": "invalid_runtime_event",
  "message": "invalid runtime event",
  "details": {
    "fields": [
      {"field": "tenant_id", "code": "required", "message": "tenant_id is required"}
    ]
  }
}
```

## HTTP 行为

| 场景 | 状态码 | 错误码 |
|---|---|---|
| JSON 格式错误 | 400 | `invalid_json` |
| RuntimeEvent 校验失败 | 422 | `invalid_runtime_event` |
| 鉴权失败 | 保持 401 / 403 | 不变 |
| 策略评估内部失败 | 500 | `evaluation_failed` |

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `internal/domain` | 修改 | 新增版本字段、版本常量、结构化校验错误 |
| `internal/runtime` | 轻微影响 | 接收归一化后的事件并写审计 |
| `internal/transport/httpapi` | 修改 | 422 和 details.fields 错误响应 |
| `sdk/go/agentsec` | 修改 | wire type 同步 schema version |
| `docs/API.md` | 修改 | 增加 Runtime Event Schema v1 说明 |
| `docs/EXECUTION-BACKLOG.md` | 修改 | 完成后更新状态和执行指针 |

## 验证方式

- `go test ./internal/domain`
- `go test ./internal/transport/httpapi`
- `go test ./sdk/go/agentsec`
- `go test ./...`
- harness validate/run。

## 风险与取舍

| 风险 | 影响 | 应对 |
|---|---|---|
| 增加版本字段可能影响旧调用方 | 旧请求没有该字段 | 缺省归一化为 v1，保持兼容 |
| 422 改变错误状态码 | 客户端若依赖 400 需要适配 | 语义上校验失败应为 422，SDK 按非 2xx 统一处理 |
| 结构化错误类型扩展到未来对象 | API 形态需要稳定 | 先只用于 RuntimeEvent，类型设计保持通用 |
