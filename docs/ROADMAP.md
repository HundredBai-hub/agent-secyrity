# Agent Security Platform Roadmap

本文档定义 Agent Security Platform 从当前基础骨架走向生产级产品的阶段路线。后续开发默认不再按单点口头任务推进，而是围绕本路线图拆分 change、task、验收标准和 harness 报告。

## 北极星目标

把 Agent Security Platform 建成企业 AI Agent 上线与运行时执行安全控制面：

- 上线前可验收：Agent 能力、工具、权限、数据访问和执行边界可声明、可检查、可审批。
- 运行中可拦截：Agent 每次工具调用、文件访问、网络访问、响应输出都能被策略评估和处置。
- 事后可追责：每个决策、审批、执行动作都有审计记录，可按租户、Agent、用户、任务和策略回放。
- 企业可运营：安全团队能用策略包、审批流、审计查询、报表和 SDK 接入企业研发与业务流程。

## 当前状态

| 能力 | 状态 | 说明 |
|---|---|---|
| 领域模型 | 已有基础 | RuntimeEvent、Policy、Decision、AuditRecord、ApprovalRequest |
| 策略评估 | 已有基础 | 支持租户、主体、事件类型、工具、资源、动作、数据标签条件 |
| 策略包 | 已有基础 | 支持创建、查询、列出、启停 |
| 审批流 | 已有基础 | 支持 require_approval、审批通过/拒绝、审批执行校验 |
| 存储 | 已有基础 | 内存和 PostgreSQL |
| HTTP API | 已有基础 | evaluate、audit、policy pack、approval |
| Go SDK | 已有基础 | evaluate、approval、API Key |
| API 认证 | 已有基础 | 静态 API Key + 租户范围 |
| 运营控制台 | 未开始 | 需要后续建设 |
| 策略语言治理 | 未完成 | 需要版本、测试、模拟、导入导出 |
| 真实业务 benchmark | 未完成 | 需要沉淀高频业务场景和验收用例 |
| 部署与观测 | 未完成 | 需要配置、日志、指标、健康检查、CI/CD |

## 阶段路线

### Phase 0：基础控制面

目标：保证运行时安全控制面可以被服务端和 Go Agent 安全接入。

已完成：

- 生产基础骨架
- 租户策略基础
- 策略包管理
- PostgreSQL 存储
- 审批流
- 审批执行校验
- Go Runtime SDK
- API Key 认证

### Phase 1：生产可用核心闭环

目标：让企业可以围绕真实 Agent 业务流程完成策略配置、运行时拦截、审批和审计。

优先任务：

1. Runtime Event Schema 稳定化与版本化。
2. 策略语言与 Policy Pack 测试能力。
3. 内置基线策略包。
4. 审计查询 API 增强。
5. Runtime SDK 执行拦截器。
6. 业务场景 benchmark。

### Phase 2：企业运营能力

目标：让安全团队、AI 平台团队和业务系统 owner 可以持续运营 Agent 风险。

优先任务：

1. 管理面 API 与控制台。
2. API Key 生命周期管理。
3. 角色权限模型。
4. 策略发布、回滚和灰度。
5. 审计报表与风险看板。
6. 多语言 SDK。

### Phase 3：产品壁垒与规模化

目标：形成差异化能力，而不是只做基础网关。

优先任务：

1. 真实业务场景库和行业模板。
2. Agent 行为基线和异常检测。
3. 策略模拟、影子运行和误拦截分析。
4. 与企业身份、数据安全、DevSecOps、MCP / Agent 平台集成。
5. 可复用的评测集、红队用例和行业 benchmark。

## 差异化建设方向

| 方向 | 产品价值 | 工程落点 |
|---|---|---|
| 真实业务场景库 | 让客户看到具体痛点和落地方式 | `benchmarks/`、内置策略包、测试数据 |
| 执行前验收 + 运行中拦截 | 覆盖 Agent 生命周期，而不是只看 prompt | Schema、Policy Pack、SDK、审批校验 |
| 租户与主体身份治理 | 支持企业多团队、多 Agent、多工作流落地 | Subject、API Key、后续 RBAC |
| 策略可测试可发布 | 降低误拦截和上线风险 | Policy simulator、dry run、版本治理 |
| 审计可追责 | 支撑安全运营、合规和事故复盘 | Audit API、报表、证据链 |

## 执行原则

- 每个功能先进入 `docs/EXECUTION-BACKLOG.md`。
- 每个 P0/P1 功能必须有 `.harness/changes/<change-id>/`。
- 每个 change 必须生成 `.harness/tasks/<change-id>/`。
- 每个实现任务必须走 TDD：红灯测试、实现、绿灯验证。
- 每个提交前必须运行 `go test ./...` 和相关 harness validate/run。
- 每个完成的 change 必须提交并推送到 GitHub。
