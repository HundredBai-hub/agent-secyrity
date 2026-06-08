# Production Foundation 设计

## 方案概述

生产基础骨架采用分层单体架构。`domain` 定义稳定领域对象；`policy` 负责策略匹配和决策；`audit` 提供审计存储接口；`runtime` 串联评估流程；`transport/httpapi` 暴露 API；`cmd/server` 作为服务入口。

```text
HTTP API
  -> Runtime Service
  -> Policy Engine
  -> Audit Store
  -> Decision Response
```

## 语言与运行时

| 项 | 选择 |
|---|---|
| 目标项目语言 | Go |
| 验证入口 | `go test ./...` |
| 执行环境 | 本地开发目录，后续使用 harness worktree |
| 依赖策略 | 第一阶段仅 Go 标准库 |

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `cmd/server` | 新增 | HTTP 服务入口，负责启动和优雅关闭 |
| `internal/domain` | 新增 | RuntimeEvent、Policy、Decision、AuditRecord |
| `internal/policy` | 新增 | 策略匹配、动作优先级、默认行为 |
| `internal/audit` | 新增 | 审计存储接口和内存实现 |
| `internal/runtime` | 新增 | 事件评估编排，写入审计 |
| `internal/transport/httpapi` | 新增 | REST API、请求解析、响应编码 |
| `docs` | 更新 | 生产级需求、设计、API 文档 |
| `.harness` | 更新 | 生产级 change、tasks、验证配置 |

## API 设计

### `GET /healthz`

返回服务健康状态。

```json
{"status":"ok"}
```

### `POST /v1/evaluate`

输入 Runtime Event，返回决策。

```json
{
  "agent_id": "agent-code-001",
  "user_id": "u-1001",
  "task_id": "task-fix-build",
  "event_type": "file_access",
  "resource": "/repo/.env",
  "action": "read",
  "data_labels": ["secret"],
  "intent": "debug build failure"
}
```

响应：

```json
{
  "decision": "deny",
  "reason": "matched policy deny-secret-file-access",
  "matched_policy_ids": ["deny-secret-file-access"],
  "audit_id": "..."
}
```

### `GET /v1/audit/events`

返回审计事件列表，第一阶段支持内存分页的基础能力。

## 策略模型

| 字段 | 说明 |
|---|---|
| `id` | 策略 ID |
| `name` | 策略名称 |
| `enabled` | 是否启用 |
| `priority` | 优先级，数字越大越优先 |
| `conditions` | 事件类型、工具、资源、动作、数据标签、用户、Agent 条件 |
| `decision` | allow / deny / redact / require_approval / record |
| `reason` | 决策原因 |

## 决策优先级

当多个策略命中时，按以下优先级返回最终决策：

```text
deny > require_approval > redact > record > allow
```

同一处置动作内按 `priority` 降序选择主要原因，同时保留所有命中策略 ID。

## Osmani Harness 设计检查

| 检查项 | 本方案如何满足 |
|---|---|
| 文件系统和 Git 状态 | 新仓库独立管理，后续每个 change 使用短分支/worktree |
| 工具和命令 | `.harness/config.json` 配置 `go test ./...` |
| 沙箱和权限 | 不接生产系统，不读取真实密钥，测试数据全部合成 |
| Hooks 和策略 | Policy Engine 是运行时安全 hook |
| 上下文管理 | proposal、design、task spec、README、docs 作为上下文 |
| 观测与报告 | Audit Store 记录评估结果，harness 输出执行报告 |
| 恢复路径 | 测试失败进入系统化调试和 harness fix loop |

## Superpowers 工作流检查

| 检查项 | 本方案如何满足 |
|---|---|
| 需求澄清 | proposal 明确生产级目标、范围、非目标 |
| 计划拆解 | tasks.md 拆为 8 个可执行任务 |
| TDD / 测试先行 | 每个核心包先写测试再实现 |
| 隔离执行 | 首次初始化在当前目录，后续迭代启用 worktree |
| 系统化调试 | 失败先读错误、复现、定位包边界 |
| 代码审查 | 完成前审查 API、错误处理、敏感信息和测试覆盖 |
| 完成前验证 | 完成前重新运行 `go test ./...` |

## 风险与取舍

| 风险 | 影响 | 应对 |
|---|---|---|
| 第一阶段仅内存存储 | 重启丢失审计记录 | 通过 Store 接口预留持久化替换 |
| 策略条件表达简单 | 复杂客户策略暂不支持 | 先固化核心事件和动作，后续扩展 DSL |
| 没有多租户模型 | 企业部署隔离不足 | 第二阶段补 tenant_id 和授权模型 |
| 没有 UI | 产品体验不完整 | 后续基于 API 增加控制台 |
