# Policy Simulator 设计

## 方案概述

新增策略模拟能力，复用现有 policy engine，但绕开 runtime service 的审计和审批副作用：

```text
HTTP request
  -> auth / tenant check
  -> decode SimulationRequest
  -> event.Normalize()
  -> policy.Simulate(request)
  -> response
```

模拟只基于请求体中的候选 policy packs，不读取当前启用策略包。这样它可以用于策略包上线前验证。

## 数据结构

| 类型 | 模块 | 说明 |
|---|---|---|
| `SimulationRequest` | `internal/policy` | 包含 RuntimeEvent 和候选 PolicyPack 列表 |
| `SimulationResult` | `internal/policy` | 包含 schema version 和 EvaluationResult |
| `PolicySimulationSchemaV1` | `internal/policy` | 当前响应 schema |

## API 行为

| 场景 | 状态码 | 说明 |
|---|---|---|
| 模拟成功 | 200 | 返回 `schema_version` 和 `result` |
| JSON 错误 | 400 | `invalid_json` |
| event 校验失败 | 422 | `invalid_runtime_event` + `details.fields` |
| 路径租户与 event 租户不一致 | 403 | `forbidden` |
| API Key 无权访问租户 | 403 | 复用已有授权逻辑 |

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `internal/policy` | 修改 | 新增 `Simulate`、请求和响应模型 |
| `internal/transport/httpapi` | 修改 | 新增 `POST /v1/tenants/{tenant_id}/policy-simulations` |
| `docs/API.md` | 修改 | 增加模拟 API 文档 |
| `docs/EXECUTION-BACKLOG.md` | 修改 | 完成后将指针移动到 `baseline-policy-packs` |

## 测试策略

- `internal/policy` 单元测试：
  - enabled pack 参与模拟。
  - disabled pack 不参与模拟。
  - 返回 policy_simulation.v1。
- `internal/transport/httpapi` 测试：
  - 成功模拟 deny。
  - event tenant 与路径 tenant 不一致返回 403。
  - event 校验失败返回 422。
  - 审计 store 无新增记录。

## 风险与取舍

| 风险 | 影响 | 应对 |
|---|---|---|
| 模拟只使用请求体策略包 | 不能直接模拟当前线上策略全集 | 后续 `policy-pack-release` 增加读取数据库策略包能力 |
| 不写审计 | 无法追踪每次模拟 | 当前模拟用于上线前验证，后续可增加模拟操作审计 |
| 与 evaluate API 接近 | 调用方可能混淆 | 路径命名为 `policy-simulations`，文档强调无副作用 |
