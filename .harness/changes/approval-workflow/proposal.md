# Approval Workflow

## 背景

当前策略引擎可以返回 `require_approval`，但系统还没有真正创建审批单，也没有审批通过、拒绝、过期和审计闭环。生产环境中，高风险 Agent 行为不能只返回一个状态码，需要把控制流程落到可追踪、可运营的审批对象上。

本变更将 `require_approval` 升级为审批工作流：运行时评估触发审批单，审批人做出通过或拒绝，系统记录审批过程，并确保跨租户隔离。

## 目标

- 新增审批领域模型：ApprovalRequest、ApprovalStatus、ApprovalDecision。
- 新增 Approval Store 接口和内存实现。
- Runtime service 在 `require_approval` 决策时自动创建审批单。
- Evaluation response 返回 `approval_id`。
- 新增审批 API：
  - `GET /v1/tenants/{tenant_id}/approvals`
  - `GET /v1/tenants/{tenant_id}/approvals/{approval_id}`
  - `POST /v1/tenants/{tenant_id}/approvals/{approval_id}/decide`
- PostgreSQL Store 支持审批持久化。
- 审批创建和审批决定均可审计。

## 技术栈约束

- 沿用 Go 标准库和当前 PostgreSQL 驱动。
- 默认 `go test ./...` 不依赖外部数据库。
- PostgreSQL 集成测试通过 `AGENT_SECURITY_POSTGRES_TEST_DSN` 显式开启。

## 范围

- In scope:
  - 审批模型。
  - 内存 Approval Store。
  - PostgreSQL Approval Store。
  - Runtime 自动创建审批单。
  - 审批 API。
  - 审批场景测试和文档。
- Out of scope:
  - 不做真实通知。
  - 不做审批人权限模型。
  - 不做 Web 审批页面。
  - 不做多级审批和会签。

## 验收标准

- 高风险工具调用命中 `require_approval` 时返回 `approval_id`。
- 可以查询审批列表和单个审批。
- 可以通过 API approve / reject。
- 已过期或已处理审批不可再次处理。
- tenant-a 不可查询 tenant-b 审批。
- `go test ./...` 通过。

## Superpowers 工作流要求

| 阶段 | 本变更要求 |
|---|---|
| Spec | 明确审批对象、状态流转、非目标 |
| TDD | 先写 Store、Runtime、HTTP、过期和跨租户测试 |
| Debugging | 失败优先定位状态机、租户隔离和审计记录 |
| Verification | 完成前重新运行 `go test ./...` 和 harness run |
