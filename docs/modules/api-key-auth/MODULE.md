# API Key Auth 模块说明

## 模块职责

`internal/auth` 提供服务端最小生产可用的 API Key 认证能力。它负责解析 `AGENT_SECURITY_API_KEYS` 配置、认证 `Authorization: Bearer <key>` 请求头，并判断 API Key 是否允许访问指定租户。

## 关键依赖

| 依赖 | 用途 |
|---|---|
| Go 标准库 `crypto/subtle` | 常量时间比较 API Key secret |
| Go 标准库 `net/http` | 读取请求 Header |
| Go 标准库 `strings` | 解析环境变量和 Bearer Header |

## 文件清单

| 文件 | 作用 |
|---|---|
| `internal/auth/apikey.go` | API Key 配置、解析、认证和租户授权实现 |
| `internal/auth/apikey_test.go` | 覆盖配置解析、错误配置、空配置禁用、Bearer 认证和无效 Header |

## 配置格式

```text
AGENT_SECURITY_API_KEYS=key-id:secret:tenant-a,tenant-b;admin:admin-secret:*
```

| 片段 | 说明 |
|---|---|
| `key-id` | 密钥标识 |
| `secret` | Bearer Token 的实际值 |
| `tenant-a,tenant-b` | 允许访问的租户列表 |
| `*` | 允许访问所有租户 |

## 重要约束

- 真实密钥不得提交到仓库，测试和文档只能使用假值。
- `secret` 字段不导出，不参与错误输出。
- 当前模块不负责密钥持久化、轮换、禁用和细粒度角色权限。
