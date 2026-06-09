# Operator Console 设计

## 信息架构

控制台默认展示一个运营工作台，顶部是租户和环境状态，左侧是功能导航，主区域按 tab 切换：

| 视图 | 目标用户问题 |
|---|---|
| Overview | 今天有哪些 Agent 风险、阻断、审批和高风险趋势 |
| Policy Packs | 当前 baseline 策略覆盖了哪些业务场景，哪些启用 |
| Approvals | 哪些高风险动作等待审批，业务上下文是什么 |
| Audit | 某个 Agent / 用户 / 任务 / 决策对应的审计记录是什么 |

## 前端目录

```text
web/console
  package.json
  vite.config.ts
  tsconfig.json
  index.html
  src/
    main.tsx
    App.tsx
    App.test.tsx
    model.ts
    mockData.ts
    consoleApi.ts
    consoleApi.test.ts
    styles.css
```

## 数据边界

第一版使用 `ConsoleApi` 接口和 in-memory mock 实现：

```ts
interface ConsoleApi {
  getSnapshot(): Promise<ConsoleSnapshot>
  decideApproval(id: string, decision: 'approved' | 'rejected'): Promise<ApprovalItem>
  queryAudit(filters: AuditFilters): Promise<AuditRecord[]>
}
```

后续接真实后端时，只替换 `consoleApi.ts` 的实现，不改页面结构。

## UI 风格

采用克制的企业安全运营台风格：

- 高密度但清晰，适合反复扫描。
- 主色使用深墨绿色和冷灰，不做紫色渐变。
- 风险用红/琥珀/绿色语义色，配合文字标签，不只依赖颜色。
- 页面主体不是卡片堆叠，重复数据项才使用小半径卡片或表格行。
- 按钮使用明确动作文案，审批动作区分 approve/reject。

## 测试策略

| 层级 | 测试 |
|---|---|
| 数据层 | `consoleApi.test.ts` 覆盖审计过滤和审批状态更新 |
| UI 层 | `App.test.tsx` 覆盖关键视图渲染、过滤交互、审批交互 |
| 构建 | `npm run build` |
| 仓库整体 | `go test ./...` |

## 安全边界

- 不在前端代码写真实 API Key。
- mock 数据不包含真实客户、密钥或生产地址。
- 后续真实 API 接入时，API Key 应由服务端 session 或安全配置注入，不存 localStorage。
- UI 不使用 `dangerouslySetInnerHTML`。

## 验证方式

- `npm install` 或 `npm ci`。
- `npm test -- --run`。
- `npm run build`。
- 启动 dev server 后用浏览器检查桌面和移动宽度。
- harness validate/run。
- 敏感信息扫描。
