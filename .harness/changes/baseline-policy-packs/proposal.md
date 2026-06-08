# Baseline Policy Packs

## 背景

当前服务启动时在 `cmd/server` 内硬编码了一个 `defaultPolicyPack`。这能演示基础能力，但不利于产品化：策略来源不清晰、场景不可复用、测试覆盖不足，也无法沉淀真实业务场景。

本变更新增内置基线策略包模块，将高频业务场景下的 Agent 运行时风险转成可复用 Policy Pack，作为企业接入时的第一批默认策略模板。

## 目标

- 新增 `internal/baseline` 模块，提供内置策略包集合。
- 覆盖第一批高频业务场景：
  - 代码仓库 Agent：保护 secrets、SSH key、环境变量、危险 shell。
  - 客服 Agent：PII / customer_data 响应脱敏。
  - 财务 Agent：付款、转账、退款等高风险工具调用需要审批。
  - 数据分析 Agent：导出客户数据、访问生产数据库需要审批或阻断。
- 服务启动时使用 baseline 模块 seed 默认策略包。
- 运行时测试覆盖典型场景。
- 文档说明内置策略包的业务场景和策略意图。

## 范围

- In scope:
  - `internal/baseline`。
  - `cmd/server` 默认策略包接入。
  - `internal/runtime` 场景测试。
  - API / 项目文档。
- Out of scope:
  - 不做策略包市场。
  - 不做 UI 管理。
  - 不做行业包动态下载。
  - 不做 AI 自动生成策略。

## 验收标准

- baseline 至少返回 4 个 Policy Pack。
- 每个内置 Policy Pack 都有稳定 ID、租户、版本、启用状态和策略列表。
- 代码仓库场景能阻断 secret 文件访问。
- 客服场景能对 PII 响应返回 redact。
- 财务场景能对转账工具调用返回 require_approval。
- 数据分析场景能对生产数据库导出返回 require_approval 或 deny。
- `cmd/server` 不再直接硬编码默认策略列表。
- `go test ./internal/policypack ./internal/runtime` 通过。
- `go test ./...` 通过。
