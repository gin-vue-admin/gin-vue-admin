# 安全策略

## 报告漏洞

**请勿通过公开 issue 报告安全漏洞。**

发现安全问题请发送邮件至维护者，或在 GitHub 私下报告（Security Advisories → Report a vulnerability）。我们会在收到报告后 **3 个工作日内**响应，并在修复后致谢。

为便于评估，请在报告中包含：
- 受影响版本与组件
- 复现步骤（PoC）
- 影响范围与攻击场景
- 你期望的披露时间窗

## 支持版本

仅最新 `main` 分支与最近一个 release 接收安全更新（项目当前未发正式 release，以 main 为准）。

## 已知安全约定（自研基座内置）

- **密码**：bcrypt（cost 10），72 字节上限硬校验。
- **登录**：失败响应文案统一防枚举；真实原因仅写入登录日志（`login_logs`），不返回客户端。
- **JWT**：双 token（access + refresh），secret 经配置注入，生产务必改默认值。
- **权限**：`RequirePermission` 中间件 + 5 分钟权限缓存；角色/权限变更后调 `middleware.InvalidateAll()` 失效。
- **数据范围**：`pkg/datascope` 在 service 层按角色 `DataScope` 过滤可见数据，超管（权限含 `*`）短路。
- **防自删/自禁**：用户禁用/删除自己 → 409。
- **默认凭据**：seed 的 `admin/123456`、`user/123456` 仅用于开发，**生产环境必须修改**。

## 加固建议（生产部署）

1. 改 `GVA_JWT__SECRET` 与所有 seed 密码。
2. 反向代理（nginx）启用 TLS、限流、安全响应头（见 `deploy/nginx.conf`）。
3. 数据库账号最小权限，限制来源 IP。
4. 关闭 `/swagger` 路由或加网关鉴权（生产不应暴露 API 文档）。
