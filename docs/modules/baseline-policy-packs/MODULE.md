# Baseline Policy Packs 模块说明

## 模块职责

`internal/baseline` 提供内置高频业务场景策略包模板。它把代码仓库、客服、财务、数据分析等常见 Agent 落地风险转成可复用 Policy Pack，作为企业接入和本地默认启动的基础策略集合。

## 文件清单

| 文件 | 作用 |
|---|---|
| `internal/baseline/packs.go` | 定义 `DefaultPolicyPacks` 和四类内置策略包 |
| `internal/baseline/packs_test.go` | 校验策略包数量、ID、租户、版本和关键风险策略 |

## 内置策略包

| Pack ID | 场景 | 核心策略 |
|---|---|---|
| `baseline-code-repository` | 代码仓库 Agent | 阻断 secret 文件访问，危险 shell 需要审批 |
| `baseline-customer-support` | 客服 Agent | PII / customer_data 响应脱敏，高风险账号操作需要审批 |
| `baseline-finance-operations` | 财务 Agent | 付款、转账、退款需要审批 |
| `baseline-data-analysis` | 数据分析 Agent | 生产客户数据导出需要审批 |

## 重要约束

- 内置策略包是默认模板，不是不可修改的系统策略。
- 每个策略包按租户生成，避免跨租户策略污染。
- 后续策略包发布和回滚能力应复用这些模板，而不是继续在 `cmd/server` 中硬编码策略。
