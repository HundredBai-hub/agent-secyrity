# Approval Workflow 设计

## 方案概述

新增 `internal/approval` 包，承载审批 Store 和状态流转。Runtime service 增加可选 Approval Store，当策略结果为 `require_approval` 时创建 `ApprovalRequest`，并把 `approval_id` 写回 `EvaluationResult`。HTTP API 新增审批查询和审批决定接口。PostgreSQL 存储层新增 `approval_requests` 表。

```text
Evaluate
  -> Policy Engine
  -> require_approval
  -> Approval Store.Create
  -> Audit Store.Append
  -> Response with approval_id
```

## 状态机

```text
pending -> approved
pending -> rejected
pending -> expired
```

约束：

- 只有 pending 可以被处理。
- `expires_at` 之前可以 approve / reject。
- 过期审批不可 approve / reject。
- 审批查询按 tenant 隔离。

## 数据模型

| 字段 | 说明 |
|---|---|
| `id` | 审批 ID |
| `tenant_id` | 租户 |
| `status` | pending / approved / rejected / expired |
| `event` | 触发审批的 Runtime Event |
| `result` | 原始 EvaluationResult |
| `reason` | 审批原因 |
| `requested_at` | 创建时间 |
| `expires_at` | 过期时间 |
| `decided_at` | 审批时间 |
| `decided_by` | 审批人 |
| `decision_reason` | 审批说明 |

## API

### List Approvals

`GET /v1/tenants/{tenant_id}/approvals`

### Get Approval

`GET /v1/tenants/{tenant_id}/approvals/{approval_id}`

### Decide Approval

`POST /v1/tenants/{tenant_id}/approvals/{approval_id}/decide`

```json
{
  "decision": "approved",
  "decided_by": "secops-001",
  "reason": "business approved"
}
```

## PostgreSQL

新增 `approval_requests` 表，使用关键字段列 + JSONB 原文模式，与 audit/policy pack 保持一致。

## 验证方式

```bash
go test ./...
```
