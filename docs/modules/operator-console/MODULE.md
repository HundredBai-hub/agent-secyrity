# Operator Console 模块说明

## 模块职责

`web/console` 提供 Agent Security Platform 的安全运营控制台。第一版使用 typed mock API 建立页面、状态和测试基线，覆盖运行时总览、策略包、审批队列和审计查询四个核心工作区。

## 关键依赖

| 依赖 | 说明 |
|---|---|
| React | 控制台 UI 组件和状态管理 |
| TypeScript | 固化前端数据模型和 API contract |
| Vite | 开发服务器和生产构建 |
| Vitest | 前端单元测试和组件测试 |
| Testing Library | DOM 交互和可访问查询测试 |

## 文件清单

| 文件 | 作用 |
|---|---|
| `web/console/package.json` | 前端依赖、测试、构建和开发脚本 |
| `web/console/vite.config.ts` | Vite 和 Vitest 配置 |
| `web/console/tsconfig.json` | TypeScript 编译约束 |
| `web/console/index.html` | 前端入口 HTML |
| `web/console/src/main.tsx` | React 挂载入口 |
| `web/console/src/App.tsx` | 控制台主页面、tab、过滤和审批交互 |
| `web/console/src/styles.css` | 控制台视觉、布局和响应式样式 |
| `web/console/src/model.ts` | 控制台数据类型 |
| `web/console/src/mockData.ts` | 本地 mock 数据 |
| `web/console/src/consoleApi.ts` | 控制台 API 接口和 mock 实现 |
| `web/console/src/consoleApi.test.ts` | API 过滤和审批状态测试 |
| `web/console/src/App.test.tsx` | 页面渲染、审计过滤和审批交互测试 |
| `web/console/src/testSetup.ts` | 测试环境扩展 |

## 重要约束

- 第一版不存储真实 API Key，不接入真实登录。
- mock 数据不包含真实客户、密钥或生产环境凭据。
- 后续接真实管理 API 时，应替换 `ConsoleApi` 实现，尽量不改 UI 组件结构。
- 页面以运营工作台为首屏，不做营销落地页。

## 验证方式

```bash
cd web/console
npm test -- --run
npm run build
npm audit --audit-level=moderate
```
