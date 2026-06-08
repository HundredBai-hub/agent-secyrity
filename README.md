# Agent Security Platform

Agent Security Platform 是面向企业 AI Agent 运行时安全的生产级工程。它用于在 Agent 执行业务任务时，对工具调用、文件访问、网络访问、模型响应和审批动作进行统一采集、策略评估、处置和审计。

本项目从生产级架构起步，不以演示 MVP 为目标。第一阶段交付的是可长期演进的运行时安全平台基础骨架：领域模型、策略引擎、审计存储、HTTP API、测试体系、harness 研发闭环和 GitHub 版本管理。

## 产品目标

- 看见 Agent 运行时行为：prompt、tool call、file access、network access、response、approval。
- 控制 Agent 执行边界：基于身份、任务、工具、资源、数据标签、风险等级做策略判断。
- 处置高风险行为：allow、deny、redact、require approval、record。
- 审计和追责：按 Agent、用户、任务、资源、策略命中和决策结果回放事件。
- 企业级隔离：按 `tenant_id` 隔离事件、策略和审计记录。
- 主体身份治理：用结构化 `subject` 表达用户、Agent、服务账号和工作流身份。
- 策略包治理：用 Policy Pack 承载策略集合、版本和启用状态。
- 支撑生产落地：稳定 API、明确模块边界、可测试、可审计、可扩展。

## 工程约束

- 后端实现语言不强制长期绑定 Go；当前生产基础骨架采用 Go，优先保证安全网关、策略评估和审计服务的稳定性。
- 所有开发任务必须配置可执行验证命令。
- 真实代码改动优先使用 worktree、容器或 CI 沙箱。
- 遵循 Osmani Agent Harness Engineering：上下文、边界、工具、验证、反馈循环、沙箱、审计。
- 借鉴 Superpowers 工作流：需求澄清、计划拆解、TDD、系统化调试、代码审查、完成前验证。
- 不接入真实生产系统、真实密钥、真实用户敏感数据作为测试输入。

## 初始技术栈

| 层级 | 选择 | 说明 |
|---|---|---|
| Runtime Service | Go | 单二进制、标准库 HTTP、适合安全网关和策略评估 |
| Storage | In-memory first | 第一阶段用接口隔离，后续替换 PostgreSQL / ClickHouse / OpenSearch |
| API | REST / JSON | 先提供稳定可测接口，再扩展 SDK 和控制台 |
| Test | Go test | 单元测试和 API 测试先行 |
| Harness | ai-dev-harness | 管理 change、task、验证和报告 |

## 目录结构

```text
agent-security-dev/
├── cmd/server/                 # HTTP 服务入口
├── internal/
│   ├── audit/                  # 审计记录与存储接口
│   ├── domain/                 # 运行时事件、策略、决策领域模型
│   ├── policy/                 # 策略评估引擎
│   ├── runtime/                # 运行时评估服务
│   └── transport/httpapi/      # HTTP API
├── docs/                       # 产品与工程文档
├── .harness/                   # AI Dev Harness 配置、changes、tasks、reports
└── README.md
```

## 本地验证

```bash
go test ./...
```

## Harness

本项目使用 `/Users/hb/Code/murphysec/agent-sec/ai-dev-harness` 管理自动研发闭环。当前生产级 change：

```text
.harness/changes/production-foundation/
```
