# Tenant Policy Foundation 设计

## 方案概述

本轮在现有领域模型上增加企业级控制面基础：`tenant_id` 标识租户隔离边界，`Subject` 表达用户、Agent、服务账号等执行主体，`PolicyPack` 承载策略集合、版本和启用状态。策略引擎从单一策略列表扩展为策略包输入，但保留兼容的 `NewEngine([]Policy)` 构造方式。

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `internal/domain` | 扩展 | RuntimeEvent、Policy、PolicyConditions、AuditRecord 增加租户/主体/策略包字段 |
| `internal/policy` | 扩展 | 新增 `NewEngineFromPacks`，策略评估按 tenant 隔离 |
| `internal/runtime` | 扩展 | 保持服务编排，审计记录自动保留新增字段 |
| `internal/transport/httpapi` | 扩展 | JSON 字段自然兼容，补 API 测试 |
| `docs/API.md` | 更新 | 说明 tenant_id、subject、policy_pack_id |

## 数据模型

### Subject

| 字段 | 说明 |
|---|---|
| `type` | user / agent / service_account / workflow |
| `id` | 主体唯一 ID |
| `roles` | 角色，如 developer、support、admin |
| `groups` | 组织或团队 |
| `risk_level` | low / medium / high |

### Policy Pack

| 字段 | 说明 |
|---|---|
| `id` | 策略包 ID |
| `tenant_id` | 所属租户 |
| `name` | 名称 |
| `version` | 版本 |
| `enabled` | 是否启用 |
| `policies` | 策略集合 |

## 策略评估规则

1. 事件必须包含 `tenant_id`。
2. 只评估同租户策略。
3. 禁用策略包不参与评估。
4. 策略自身禁用不参与评估。
5. 主体条件可按 subject type、role、group、risk level 匹配。
6. 决策优先级保持：`deny > require_approval > redact > record > allow`。

## Superpowers 工作流检查

| 检查项 | 本方案如何满足 |
|---|---|
| 需求澄清 | proposal 明确生产级拼图和非目标 |
| 计划拆解 | tasks.md 拆分模型、引擎、API、文档和验证 |
| TDD / 测试先行 | 先写跨租户隔离、主体匹配、策略包禁用测试 |
| 系统化调试 | 失败优先定位模型校验和条件匹配 |
| 完成前验证 | 重新运行 `go test ./...` 和 harness run |

## 风险与取舍

| 风险 | 影响 | 应对 |
|---|---|---|
| 事件强制 tenant_id 可能打破旧调用 | API 使用者需要补字段 | 当前仍是早期生产骨架，优先建立正确边界 |
| 策略包只在内存构造 | 无法动态管理 | 后续增加策略包 Store 和管理 API |
| 主体模型过复杂 | 策略难维护 | 第一阶段仅支持 type、role、group、risk level |
