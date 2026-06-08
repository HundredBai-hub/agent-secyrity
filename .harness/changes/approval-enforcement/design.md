# Approval Enforcement 设计

## 方案概述

Runtime Event 新增 `approval_id`。当请求携带 `approval_id` 时，Runtime service 先从 Approval Store 读取审批单，校验：

1. 审批单属于同一 `tenant_id`。
2. 审批状态为 `approved`。
3. 当前事件与审批单中的原始事件匹配。

校验通过后，直接返回 `allow` 并写入审计；校验失败返回 `deny` 并写入审计。

## 事件绑定字段

第一阶段比较以下字段：

| 字段 | 说明 |
|---|---|
| `tenant_id` | 租户 |
| `agent_id` | Agent |
| `user_id` | 用户 |
| `task_id` | 任务 |
| `event_type` | 事件类型 |
| `tool_name` | 工具名称 |
| `resource` | 资源 |
| `action` | 动作 |
| `data_labels` | 数据标签集合 |

## 决策规则

| 场景 | 决策 |
|---|---|
| approval_id 不存在 | deny |
| 审批 pending | deny |
| 审批 rejected | deny |
| 审批 expired | deny |
| 审批 approved 且事件匹配 | allow |
| 审批 approved 但事件不匹配 | deny |

## 验证方式

```bash
go test ./...
```
