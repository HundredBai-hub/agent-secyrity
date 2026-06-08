# API Key Authentication Tasks

- [ ] API-KEY-AUTH-001: 新增 `internal/auth` 配置解析和 API Key 认证单元测试
- [ ] API-KEY-AUTH-002: 实现 `internal/auth` API Key 配置、Bearer 解析、常量时间比较和租户授权
- [ ] API-KEY-AUTH-003: 新增 HTTP 鉴权红灯测试，覆盖 401、403、healthz bypass、evaluate 租户校验
- [ ] API-KEY-AUTH-004: 实现 HTTP middleware、router 注入、路径租户和请求体租户授权
- [ ] API-KEY-AUTH-005: 新增审计查询租户过滤测试，覆盖内存和 PostgreSQL store 行为
- [ ] API-KEY-AUTH-006: 实现 `audit.ListOptions.TenantID` 和 PostgreSQL 审计租户过滤
- [ ] API-KEY-AUTH-007: 新增 SDK `WithAPIKey` 红灯测试并实现 Authorization Header 注入
- [ ] API-KEY-AUTH-008: 更新 README、API 文档、Go SDK 文档和模块索引
- [ ] API-KEY-AUTH-009: 运行 harness validate/run、`go test ./...`、敏感信息扫描，完成审查、提交并推送
