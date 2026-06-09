# Operator Console

## 背景

当前平台已经具备运行时评估、策略包、审批、审计查询和 Go SDK 拦截能力，但还没有页面。安全运营人员、研发负责人和售前演示都缺少一个可视化入口，无法直观看到 Agent 运行时风险、策略命中、待审批动作和审计记录。

前端控制台第一版用于建立产品体验基座，不等待完整管理 API 全部完成。页面先通过 typed mock API 工作，保留真实 API 客户端边界，后续可以逐步替换为后端接口。

## 目标

- 新增 `web/console` 前端工程。
- 提供生产级控制台首屏，而不是营销落地页。
- 覆盖四个核心工作区：
  - 运行时总览：风险指标、决策分布、近期高风险事件。
  - 策略包：baseline 策略包状态和业务场景。
  - 审批队列：待审批高风险动作。
  - 审计查询：按租户、Agent、用户、任务、决策过滤记录。
- 建立前端测试、构建和文档基线。

## 范围

- In scope:
  - React + TypeScript + Vite 前端工程。
  - Vitest 单元测试。
  - 控制台布局、导航、指标卡、表格、过滤器和审批动作 UI。
  - `docs/PROJECT.md` 和模块文档。
  - harness change/task specs。
- Out of scope:
  - 不接入真实登录。
  - 不实现完整 RBAC。
  - 不把前端打包进 Go server。
  - 不新增后端管理 API。
  - 不做部署流水线。

## 技术选型

| 选型 | 理由 |
|---|---|
| React | 适合管理台组件化和后续复杂状态演进 |
| TypeScript | 固化 API contract 和 UI 状态类型 |
| Vite | 轻量、快速、与 Vitest 配套 |
| Vitest | 测试入口简单，适合当前 harness 验收 |
| CSS Modules / 普通 CSS | 第一版避免引入重型 UI 框架，保持可控 |

## 验收标准

- `web/console` 可独立安装、测试、构建。
- 控制台首屏是实际运营界面，不是 landing page。
- 页面包含运行时总览、策略包、审批队列、审计查询四个视图。
- 审计查询过滤器可改变列表结果。
- 审批队列支持 approve/reject UI 状态变化。
- 表格和关键按钮有可访问标签或可见文本。
- 适配桌面和移动宽度，文本不重叠。
- `npm test` / `npm run build` 在 `web/console` 下通过。
- `go test ./...` 仍通过。
