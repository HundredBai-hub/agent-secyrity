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
- 策略运营 API：支持按租户创建、查询、列出、启停策略包。
- 审批流：`require_approval` 自动生成审批单，支持审批通过、拒绝、过期和审计。
- Go Runtime SDK：为 Go Agent、工具代理和网关插件提供标准接入方式。
- Operator Console：提供运行时总览、策略包、审批队列和审计查询的安全运营页面。
- API Key 认证：支持按密钥绑定租户访问范围，保护运行时控制面 API。
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
| Storage | PostgreSQL / In-memory | 默认内存，设置 `DATABASE_URL` 后使用 PostgreSQL |
| API | REST / JSON | 先提供稳定可测接口，再扩展 SDK 和控制台 |
| SDK | Go 标准库 | `sdk/go/agentsec` 封装运行时评估和审批 API |
| Console | React / TypeScript / Vite | `web/console` 提供安全运营控制台 |
| Auth | Static API Key | 通过环境变量配置 API Key 和租户访问范围 |
| Test | Go test / Vitest | 后端、SDK、前端测试先行 |
| Harness | ai-dev-harness | 管理 change、task、验证和报告 |

## 目录结构

```text
agent-security-dev/
├── cmd/server/                 # HTTP 服务入口
├── internal/
│   ├── audit/                  # 审计记录与存储接口
│   ├── approval/               # 审批请求、状态流转与存储接口
│   ├── domain/                 # 运行时事件、策略、决策领域模型
│   ├── policy/                 # 策略评估引擎
│   ├── policypack/             # 策略包存储与管理
│   ├── runtime/                # 运行时评估服务
│   └── transport/httpapi/      # HTTP API
├── sdk/go/agentsec/            # Go Runtime SDK
├── web/console/                # Operator Console 前端控制台
├── docs/                       # 产品与工程文档
├── .harness/                   # AI Dev Harness 配置、changes、tasks、reports
└── README.md
```

## 本地验证

```bash
go test ./...
```

SDK 局部验证：

```bash
go test ./sdk/go/agentsec
```

前端控制台验证：

```bash
cd web/console
npm test -- --run
npm run build
```

## Go Runtime SDK

Go Agent、工具代理和网关插件可以通过 SDK 接入运行时安全控制面：

```go
client, err := agentsec.NewClient("http://127.0.0.1:8080")
if err != nil {
	return err
}

result, err := client.Evaluate(ctx, agentsec.RuntimeEvent{
	TenantID:  "tenant-a",
	AgentID:   "agent-code-001",
	UserID:    "dev-001",
	TaskID:    "fix-build",
	EventType: agentsec.EventTypeToolCall,
	ToolName:  "shell",
	Action:    "execute",
})
```

完整接入说明见 [docs/Go-SDK.md](docs/Go-SDK.md)。

## Operator Console

前端控制台位于 `web/console`，用于安全运营人员查看 Agent 运行时风险、baseline 策略包、待审批动作和审计记录。

```bash
cd web/console
npm install
npm run dev -- --port 4174
```

本地访问：`http://127.0.0.1:4174/`。

## 本地运行

内存模式：

```bash
go run ./cmd/server
```

PostgreSQL 模式：

```bash
export DATABASE_URL='postgres://agent_security_user:change-me@localhost:5432/agent_security?sslmode=disable'
go run ./cmd/server
```

启用 API Key 认证：

```bash
export AGENT_SECURITY_API_KEYS='runtime:replace-with-runtime-key:tenant-a,tenant-b'
go run ./cmd/server
```

配置格式为 `key-id:secret:tenant-list`，多个 Key 用 `;` 分隔，多个租户用 `,` 分隔。`tenant-list` 可使用 `*` 表示允许访问所有租户。真实密钥应由部署环境注入，不应提交到仓库。

## Harness

本项目使用 `/Users/hb/Code/murphysec/agent-sec/ai-dev-harness` 管理自动研发闭环。当前生产级 change：

```text
.harness/changes/production-foundation/
```

后续开发默认按项目级任务池持续推进：

- [docs/ROADMAP.md](docs/ROADMAP.md)：产品和工程阶段路线。
- [docs/EXECUTION-BACKLOG.md](docs/EXECUTION-BACKLOG.md)：可执行任务池、优先级、状态和当前执行指针。
- [docs/HARNESS-EXECUTION-LOOP.md](docs/HARNESS-EXECUTION-LOOP.md)：harness 自动执行循环规则。
