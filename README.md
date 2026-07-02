# gin-vue-admin

> 企业级后台管理系统：**Gin** 后端 + [vue-admin/vue-admin](https://github.com/vue-admin/vue-admin) 前端，遵循业界标准（SOLID / KISS / Clean Architecture）实现。

## 架构

```
modules(web) ──HTTP(/api)──→ app(server)
```

| 模块 | 技术栈 | 路径 | 说明 |
|------|--------|------|------|
| 前端 | Vue3 + Vite + TS + Element Plus + Pinia | `web/` (submodule) | 直接使用 vue-admin/vue-admin |
| 后端 | Go + Gin + GORM + JWT + zap | `server/` | 自研，严格对齐前端 API 契约 |
| 部署 | Docker + docker-compose + Nginx | `deploy/` | 一键起全栈 |
| 文档 | Markdown + Swagger | `docs/` | 架构 / 部署 / API |

## API 契约（核心约定）

- **成功**：`HTTP 200 + { code: 0, data, msg, traceId? }`
- **失败**：`HTTP 4xx/5xx + ProblemDetail`（RFC 7807，含 `type/title/status/detail/code/errors/traceId`）
- 鉴权：请求头 `Authorization: Bearer <accessToken>`，JWT access/refresh 双 token（refresh 轮换）
- 权限：RBAC 权限码模型（`*` 通配 + `user:read` 等具体码）

> 后端任何端点的路径/方法/字段必须与 `web/src/mock/handlers/*.ts` 保持一致。

## 快速开始

### 后端
```bash
make server-dev          # 默认 :8080（PORT 环境变量可覆盖）
curl http://localhost:8080/api/health
```

### 前端
```bash
make web-install         # 首次安装依赖
make web-dev             # http://localhost:5173/vue-admin/
```

### 一键部署（M5）
```bash
make compose-up          # mysql + server + web(nginx)
```

## 测试账号（对齐前端 mock）

| 用户名 | 密码 | 角色 | 权限 |
|--------|------|------|------|
| admin | 123456 | super_admin | 全部（`*`） |
| user | 123456 | user | `user:read` |

## 文档

- [实施计划](docs/plan) — 分里程碑演进路线
- [架构规范](docs/architecture.md) — 前后端分层与依赖方向（M5）
- [部署指南](docs/deployment.md) — Docker / Nginx / 裸机三方案（M5）
- Swagger — 运行后访问 `http://localhost:8080/swagger/index.html`（M5）

## License

MIT
