# Production Roadmap And Backlog

## 背景

前期开发已经完成多项基础能力，但推进方式是“完成一个 change 后等待继续指令”。这不符合项目级 AI 工程 harness 的目标。后续需要把产品目标、工程阶段、任务池、执行循环和验收命令统一沉淀下来，让 agent 可以按 backlog 顺序持续推进。

## 目标

- 新增项目级路线图 `docs/ROADMAP.md`。
- 新增执行任务池 `docs/EXECUTION-BACKLOG.md`。
- 新增 harness 执行循环规则 `docs/HARNESS-EXECUTION-LOOP.md`。
- 将路线图和任务池挂入 README 和 `docs/PROJECT.md`。
- 明确后续默认执行方式：从 backlog 当前指针开始，按 harness 循环持续执行。

## 范围

- In scope:
  - 文档体系。
  - backlog 状态字段。
  - change/task 生成规则。
  - 验证、扫描、提交、推送规则。
- Out of scope:
  - 不实现新的运行时代码。
  - 不改变 API 行为。
  - 不调整 harness CLI 源码。

## 验收标准

- `docs/ROADMAP.md` 清晰描述阶段路线和差异化方向。
- `docs/EXECUTION-BACKLOG.md` 至少列出 P0 / P1 / P2 任务和当前执行指针。
- `docs/HARNESS-EXECUTION-LOOP.md` 描述完整的 harness 自动循环。
- README 和 `docs/PROJECT.md` 能索引到三份文档。
- 生成 `.harness/tasks/production-roadmap`。
- `go test ./...` 通过。
