# Policy Pack Management

## 背景

当前系统已经具备 `PolicyPack` 模型和按租户隔离的策略评估能力，但策略包仍然只能在代码中静态构造。生产环境需要让租户策略可以被创建、查询、启停和用于运行时评估，否则无法支撑企业客户的策略运营、策略回滚和差异化配置。

本变更将策略包从“代码内对象”推进为“可管理资源”，建立 Policy Pack Store 和管理 API。

## 目标

- 新增 Policy Pack Store 接口和并发安全内存实现。
- Runtime service 评估时从 Store 动态加载租户启用策略包。
- 新增 Policy Pack 管理 API：
  - `PUT /v1/tenants/{tenant_id}/policy-packs/{pack_id}`
  - `GET /v1/tenants/{tenant_id}/policy-packs`
  - `GET /v1/tenants/{tenant_id}/policy-packs/{pack_id}`
  - `PATCH /v1/tenants/{tenant_id}/policy-packs/{pack_id}/enabled`
- 保证跨租户策略包隔离。
- 更新文档、测试和 harness task。

## 技术栈约束

- 沿用 Go 标准库。
- 所有行为通过 `go test ./...` 验证。
- 仍使用内存 Store，但通过接口隔离，后续替换持久化数据库。

## 范围

- In scope:
  - Policy Pack Store。
  - 管理 API。
  - Runtime service 动态加载启用策略包。
  - HTTP 测试和文档。
- Out of scope:
  - 不做数据库持久化。
  - 不做鉴权登录。
  - 不做策略包发布审批流。
  - 不做 Web 控制台。

## 验收标准

- 可以通过 API 创建 / 更新策略包。
- 可以按租户列出策略包。
- 可以启停策略包。
- 禁用策略包后，Runtime evaluate 不再命中其中策略。
- tenant-a 无法查询 tenant-b 的策略包。
- `go test ./...` 通过。

## Superpowers 工作流要求

| 阶段 | 本变更要求 |
|---|---|
| Spec | 明确 Store、API、动态评估和非目标 |
| Plan | 拆成 Store、Runtime、HTTP、文档和验证任务 |
| TDD | 先写 Store、Runtime 动态加载、HTTP API 测试 |
| Debugging | 失败优先定位租户路径解析、Store 隔离和策略加载 |
| Verification | 完成前重新运行 `go test ./...` 和 harness run |
