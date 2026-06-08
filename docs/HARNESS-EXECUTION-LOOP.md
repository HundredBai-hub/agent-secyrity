# Harness Execution Loop

本文档定义本仓库后续开发的默认自动执行循环。除非用户明确改变优先级，开发 agent 应从 `docs/EXECUTION-BACKLOG.md` 的当前执行指针开始，按顺序持续推进。

## 循环入口

当前仓库：

```text
/Users/hb/Code/murphysec/agent-sec/agent-security-dev
```

Harness 仓库：

```text
/Users/hb/Code/murphysec/agent-sec/ai-dev-harness
```

默认验证命令：

```bash
go test ./...
```

## 单个 Change 状态机

```text
todo
  -> planned
  -> red_test
  -> implemented
  -> verified
  -> documented
  -> reviewed
  -> committed
  -> pushed
  -> next
```

## 标准循环

### 1. 选择任务

- 读取 `docs/EXECUTION-BACKLOG.md`。
- 选择 `当前执行指针` 对应任务。
- 如果该任务不存在或已完成，选择下一个 `todo` 任务。

### 2. 创建 change

```bash
go run ./cmd/harness change init --workspace /Users/hb/Code/murphysec/agent-sec/agent-security-dev --id <change-id> --title "<title>" --force
```

必须补齐：

- `.harness/changes/<change-id>/proposal.md`
- `.harness/changes/<change-id>/design.md`
- `.harness/changes/<change-id>/tasks.md`

### 3. 生成 task specs

```bash
go run ./cmd/harness change task-specs --workspace /Users/hb/Code/murphysec/agent-sec/agent-security-dev --id <change-id> --force
```

### 4. 测试先行

- 新增或修改测试。
- 先运行局部测试，确认因为目标行为未实现而失败。
- 失败原因必须是预期缺失行为，不是测试语法错误或环境错误。

### 5. 最小实现

- 只实现当前 task 所需行为。
- 避免顺手重构无关模块。
- 公共 API 改动必须同步文档。

### 6. 验证

至少运行：

```bash
go test ./...
```

并按 change 类型运行局部测试，例如：

```bash
go test ./internal/domain ./internal/transport/httpapi ./sdk/go/agentsec
```

### 7. Harness validate/run

至少验证当前 change 的所有 task spec：

```bash
go run ./cmd/harness validate --task <task-json>
```

至少运行一个代表性 task：

```bash
go run ./cmd/harness run --task <task-json>
```

### 8. 安全扫描

提交前运行敏感信息扫描：

```bash
rg -n "password|api[_-]?key|token|BEGIN (RSA |OPENSSH |EC )?PRIVATE KEY|sk-[A-Za-z0-9]" README.md docs cmd internal sdk .harness/changes .harness/tasks
```

命中 placeholder、测试假值或字段名时，在总结里说明。

### 9. 文档更新

涉及新增模块、公共 API、配置、部署、SDK、数据库、策略语义时，必须同步：

- `README.md`
- `docs/PROJECT.md`
- 对应 `docs/modules/<module>/MODULE.md`
- 对应专题文档，例如 `docs/API.md`、`docs/Go-SDK.md`

### 10. 提交与推送

提交前检查：

```bash
git status --short --branch
git diff --cached --stat
go test ./...
```

提交并推送：

```bash
git commit -m "<type>: <summary>"
git push
```

### 11. 更新 backlog

任务完成后：

- 将对应 change 状态改为 `done`。
- 将 `当前执行指针` 移动到下一项。
- 如产生新任务，追加到对应优先级区块。

## 默认暂停条件

遇到以下情况暂停并向用户说明：

- 需要真实密钥、生产数据库、外部服务账号。
- 需要做破坏性操作。
- 需求方向和 backlog 冲突。
- 连续三轮修复仍无法通过同一个验证。

## 默认不中断条件

以下情况不需要等待用户：

- 继续 backlog 下一项。
- 为当前 change 补测试、补文档、补 harness task。
- 修复当前 change 引入的测试失败。
- 提交并推送已经验证通过的原子变更。
