# API Key Authentication 设计

## 方案概述

新增 `internal/auth` 包承载 API Key 配置与认证逻辑；HTTP 层通过 middleware 对除 `GET /healthz` 以外的请求执行认证。认证通过后，将调用方允许访问的租户集合写入 request context，handler 在处理路径租户、请求体租户和审计查询租户时做授权校验。

配置采用环境变量，满足本地和生产最小接入：

```text
AGENT_SECURITY_API_KEYS=key-id-1:secret-1:tenant-a,tenant-b;key-id-2:secret-2:tenant-c
```

格式说明：

| 片段 | 说明 |
|---|---|
| `key-id` | 密钥标识，仅用于日志和排查，不返回给客户端 |
| `secret` | Bearer Token 的实际值 |
| `tenant-a,tenant-b` | 允许访问的租户 ID 列表 |
| `*` | 允许访问所有租户，用于内部管理服务 |

未配置 `AGENT_SECURITY_API_KEYS` 时，认证 middleware 不启用，保持当前本地开发和测试兼容。

## 语言与运行时

| 项 | 选择 |
|---|---|
| 目标项目语言 | Go |
| 验证入口 | `go test ./internal/transport/httpapi ./sdk/go/agentsec`、`go test ./...` |
| 执行环境 | 当前仓库本地工作区，受 harness 任务约束 |

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `internal/auth` | 新增 | API Key 配置解析、Bearer Header 解析、常量时间比较、租户授权 |
| `internal/transport/httpapi` | 修改 | 增加认证 middleware、租户校验、审计 tenant_id 过滤 |
| `internal/audit` | 修改 | `ListOptions` 增加 `TenantID` |
| `internal/storage/postgres` | 修改 | 审计查询按租户过滤 |
| `cmd/server` | 修改 | 从环境变量加载 API Key 配置并注入 router |
| `sdk/go/agentsec` | 修改 | `WithAPIKey` 设置 Authorization Header |
| `docs/API.md` / `README.md` / `docs/Go-SDK.md` | 修改 | 补充认证配置、错误语义和 SDK 使用方式 |

## HTTP 行为

| 场景 | 状态码 | 响应错误码 |
|---|---|---|
| 未启用 API Key | 保持当前行为 | 无 |
| `GET /healthz` | 200 | 无 |
| 缺少 Authorization Header | 401 | `unauthorized` |
| Header 不是 Bearer | 401 | `unauthorized` |
| Bearer Token 不存在 | 401 | `unauthorized` |
| API Key 无权访问路径租户 | 403 | `forbidden` |
| API Key 无权访问请求体租户 | 403 | `forbidden` |
| API Key 无权访问审计查询租户 | 403 | `forbidden` |

错误响应仍使用当前项目的标准格式：

```json
{"error":"unauthorized","message":"missing or invalid API key"}
```

## 租户边界

- `POST /v1/evaluate`：校验请求体 `tenant_id`。
- `/v1/tenants/{tenant_id}/...`：校验路径 `tenant_id`。
- `GET /v1/audit/events?tenant_id=...`：启用鉴权后必须带 `tenant_id`，并校验该租户。
- API Key 租户为 `*` 时跳过具体租户限制。

## Osmani Harness 设计检查

| 检查项 | 本方案如何满足 |
|---|---|
| 文件系统和 Git 状态 | 只修改 auth、httpapi、audit、postgres、server、SDK、文档和本 change |
| 工具和命令 | 使用 handler/SDK 局部测试、全量 Go 测试、harness validate/run |
| 沙箱和权限 | 不需要网络下载，不引入依赖，不读取真实密钥 |
| Hooks 和策略 | 使用 `gofmt`、`go test ./...`、敏感信息扫描和 staged diff 审查 |
| 上下文管理 | 只读取相关 handler、store、SDK、server 和文档 |
| 观测与报告 | harness report 记录任务运行结果，Git 提交记录实现审计 |
| 恢复路径 | 单一 commit 可回滚；测试失败先按 401/403/200 分类定位 |

## Superpowers 工作流检查

| 检查项 | 本方案如何满足 |
|---|---|
| 需求澄清 | 明确当前只做静态 API Key 与租户边界 |
| 计划拆解 | 拆成配置解析、middleware、租户校验、审计过滤、SDK、文档 |
| TDD / 测试先行 | 先写 HTTP 和 SDK 红灯测试 |
| 隔离执行 | 当前 main 干净，变更范围集中，不额外创建 worktree |
| 系统化调试 | 认证失败优先检查 Header、token、tenant source 和 context |
| 代码审查 | 检查默认兼容性、租户越权、错误暴露、secret 日志风险 |
| 完成前验证 | 重新运行全量测试、harness 和敏感信息扫描 |

## 风险与取舍

| 风险 | 影响 | 应对 |
|---|---|---|
| 静态 API Key 不支持轮换 API | 运维侧需要重启或重载配置 | 当前作为最小生产防线，后续用数据库和控制台管理密钥 |
| 环境变量中包含密钥 | 配置管理需要保护环境变量 | 文档强调不提交真实密钥，测试只用假值 |
| `*` 租户权限过大 | 管理密钥泄露影响全部租户 | 默认示例使用具体租户，`*` 仅用于内部管理服务 |
| 审计查询历史接口新增 tenant_id 要求 | 启用鉴权后调用方需要调整 | 未启用鉴权保持兼容；启用鉴权时安全优先 |
