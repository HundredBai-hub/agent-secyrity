# Baseline Policy Packs 设计

## 方案概述

新增 `internal/baseline` 包，负责提供内置 Policy Pack：

```go
func DefaultPolicyPacks(tenantID string) []domain.PolicyPack
```

服务启动时调用该函数，将返回的策略包写入当前策略包 store。这样默认策略从 `cmd/server` 中解耦，后续可以扩展为策略模板、行业包和版本治理。

## 第一批策略包

| Pack ID | 场景 | 目标 |
|---|---|---|
| `baseline-code-repository` | 代码仓库 Agent | 阻断 secrets / SSH key / 环境变量访问，危险 shell 需要审批 |
| `baseline-customer-support` | 客服 Agent | 客户数据和 PII 输出脱敏，高风险账号操作需要审批 |
| `baseline-finance-operations` | 财务 Agent | 付款、转账、退款等动作需要审批 |
| `baseline-data-analysis` | 数据分析 Agent | 生产数据库访问、客户数据导出需要审批或阻断 |

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `internal/baseline` | 新增 | 内置策略包定义和测试 |
| `cmd/server` | 修改 | 使用 baseline seed 默认策略包 |
| `internal/runtime` | 修改 | 增加典型业务场景测试 |
| `docs/PROJECT.md` | 修改 | 增加 baseline 模块索引 |
| `docs/API.md` | 修改 | 增加内置策略包说明 |
| `docs/EXECUTION-BACKLOG.md` | 修改 | 完成后指针移动到 `audit-query-api` |

## 策略命名约定

```text
<pack-id>.<decision>.<business-risk>
```

示例：

```text
baseline-code-repository.deny.secret-file-access
baseline-finance-operations.require-approval.money-transfer
```

## 验证方式

- `go test ./internal/baseline`
- `go test ./internal/policypack ./internal/runtime`
- `go test ./...`
- harness validate/run。

## 风险与取舍

| 风险 | 影响 | 应对 |
|---|---|---|
| 默认策略过强 | 本地演示或客户试用可能被误拦截 | 先作为 baseline 模板，可按租户启停策略包 |
| 场景不够完整 | 不能覆盖所有行业 | 第一批覆盖高频共性场景，后续通过 benchmark 和行业模板扩展 |
| 策略写死在代码 | 修改需要发版 | 现阶段作为内置模板，后续策略包发布能力会支持外部导入 |
