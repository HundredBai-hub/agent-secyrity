# Policy Simulator

## 背景

策略包上线前需要验证命中效果和误拦截风险。当前 `/v1/evaluate` 会真实写审计，并在 `require_approval` 时创建审批单，不适合用于策略上线前反复试算。

本变更新增 Policy Simulator：给定 Runtime Event 和一组候选策略包，返回模拟决策、命中策略和原因，但不写审计、不创建审批、不改变策略包启用状态。

## 目标

- 提供纯函数级策略模拟能力。
- HTTP API 支持按租户提交 event + policy packs 做 dry-run。
- 响应返回 decision、reason、matched_policy_ids 和 schema version。
- 复用 Runtime Event Schema v1 校验错误。
- 不产生审计记录。
- 不产生审批单。
- 为后续策略包发布、影子模式、业务 benchmark 打基础。

## 范围

- In scope:
  - `internal/policy` 模拟输入和输出。
  - HTTP simulate endpoint。
  - Runtime Event 校验和租户授权复用。
  - 测试和文档。
- Out of scope:
  - 不读取数据库中的策略包做批量模拟。
  - 不做策略差异分析。
  - 不做 UI。
  - 不做影子模式。

## API 草案

`POST /v1/tenants/{tenant_id}/policy-simulations`

请求：

```json
{
  "event": {
    "schema_version": "runtime_event.v1",
    "tenant_id": "tenant-a",
    "agent_id": "agent-code-001",
    "user_id": "dev-001",
    "task_id": "fix-build",
    "event_type": "tool_call",
    "tool_name": "shell",
    "action": "execute"
  },
  "policy_packs": [
    {
      "id": "candidate-runtime",
      "tenant_id": "tenant-a",
      "enabled": true,
      "policies": []
    }
  ]
}
```

响应：

```json
{
  "schema_version": "policy_simulation.v1",
  "result": {
    "decision": "deny",
    "reason": "shell is blocked",
    "matched_policy_ids": ["deny-shell"]
  }
}
```

## 验收标准

- 模拟命中策略时返回与真实 engine 一致的决策。
- disabled policy pack 不参与模拟。
- event tenant 与路径 tenant 不一致返回 403。
- event 校验失败返回 422 和字段详情。
- 模拟请求不会写审计记录。
- `go test ./internal/policy ./internal/transport/httpapi` 通过。
- `go test ./...` 通过。
