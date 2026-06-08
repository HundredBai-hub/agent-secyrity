# Production Roadmap And Backlog 设计

## 方案概述

用三份项目级文档承载后续自动执行体系：

| 文档 | 职责 |
|---|---|
| `docs/ROADMAP.md` | 定义北极星目标、当前状态、阶段路线和差异化建设方向 |
| `docs/EXECUTION-BACKLOG.md` | 定义 P0/P1/P2 任务池、状态、验收命令和当前执行指针 |
| `docs/HARNESS-EXECUTION-LOOP.md` | 定义每个任务如何进入 harness change、TDD、验证、报告、提交和下一项 |

该 change 不改变运行时代码，只改变项目执行方式。后续每个 backlog 任务都必须通过 `.harness/changes/<change-id>` 和 `.harness/tasks/<change-id>` 管理。

## 模块影响

| 模块 | 影响 | 说明 |
|---|---|---|
| `docs/ROADMAP.md` | 新增 | 产品和工程阶段路线 |
| `docs/EXECUTION-BACKLOG.md` | 新增 | 可执行任务池 |
| `docs/HARNESS-EXECUTION-LOOP.md` | 新增 | 自动执行循环 |
| `README.md` | 修改 | 增加路线图和 backlog 入口 |
| `docs/PROJECT.md` | 修改 | 增加文档索引 |
| `.harness/changes/production-roadmap` | 新增 | 将本执行体系纳入 harness |

## 执行方式

后续默认循环：

```text
读取 backlog 当前指针
  -> 创建 change
  -> 写 proposal/design/tasks
  -> 生成 task specs
  -> 写红灯测试
  -> 实现
  -> 局部测试
  -> 全量测试
  -> harness validate/run
  -> 敏感扫描
  -> code review
  -> commit
  -> push
  -> 更新 backlog
  -> 下一项
```

## 验证方式

- `go test ./...`
- `go run ./cmd/harness validate --task <production-roadmap-task>`
- `go run ./cmd/harness run --task <production-roadmap-task>`
- 文档索引检查。

## 风险与应对

| 风险 | 影响 | 应对 |
|---|---|---|
| backlog 太大导致执行发散 | 任务选择不稳定 | 使用当前执行指针和 P0/P1/P2 顺序约束 |
| 文档和实际实现脱节 | 后续 agent 走错方向 | 每个完成 change 必须更新 backlog 状态 |
| 任务粒度过大 | 单轮实现不可控 | 每个 change 再拆 task specs，并坚持 TDD |
| 遇到产品方向取舍 | 可能做错优先级 | 仅方向性决策暂停等待用户 |
