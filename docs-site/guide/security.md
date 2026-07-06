# 安全

## 报告漏洞

**请勿通过公开 issue 报告安全漏洞。** 见仓库根 [SECURITY.md](https://github.com/gin-vue-admin/gin-vue-admin/blob/main/SECURITY.md) 的报告流程。

## 内置安全约定

| 项 | 实现 |
|----|------|
| 密码 | bcrypt（cost 10），72 字节上限硬校验 |
| 登录防枚举 | 失败响应文案统一，真实原因仅写登录日志 |
| JWT | 双 token，secret 配置注入，生产必须改默认值 |
| 权限缓存 | `RequirePermission` 5 分钟 TTL；变更调 `InvalidateAll()` |
| 数据范围 | service 层按角色 DataScope 过滤，超管（`*`）短路 |
| 防自删/自禁 | 用户禁用/删除自己 → 409 |
| 操作审计 | 所有写操作经 `OperationLog` 中间件异步记录 |

## 默认凭据（仅开发）

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin | 123456 | super_admin |
| user | 123456 | user |

::: danger
生产环境必须改：所有 seed 密码、`JWT_SECRET`、MySQL root 密码。
:::

## 生产加固清单

- [ ] 改 `GVA_JWT__SECRET` 为强随机串（`openssl rand -hex 32`）
- [ ] 改 MySQL root 密码 + 限制来源 IP
- [ ] 反向代理（nginx）启用 TLS、限流、安全响应头
- [ ] 关闭或网关鉴权 `/swagger` 路由（生产不应公开 API 文档）
- [ ] MySQL 每日 `mysqldump` 备份 + 异地存储
- [ ] 日志（`server/logs/`）轮转 + 监控告警
- [ ] 最小权限 DB 账号（勿用 root 连应用）
