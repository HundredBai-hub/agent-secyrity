# Go Runtime SDK Tasks

- [ ] GO-RUNTIME-SDK-001: 新增 `sdk/go/agentsec` 公共类型、Client 配置、Option 和 `APIError`
- [ ] GO-RUNTIME-SDK-002: 使用 `httptest` 编写 `Evaluate` 红灯测试，覆盖请求路径、JSON 字段、响应解析和决策 helper
- [ ] GO-RUNTIME-SDK-003: 实现 `Client.Evaluate`、请求构造、JSON 编解码和非 2xx 错误解析
- [ ] GO-RUNTIME-SDK-004: 使用 `httptest` 编写 Approval 红灯测试，覆盖审批列表、详情和审批决定
- [ ] GO-RUNTIME-SDK-005: 实现 `ListApprovals`、`GetApproval`、`DecideApproval`
- [ ] GO-RUNTIME-SDK-006: 补充 `docs/Go-SDK.md`、README SDK 说明和项目模块索引
- [ ] GO-RUNTIME-SDK-007: 运行 harness validate/run、`go test ./...`、敏感信息扫描，完成审查、提交并推送
