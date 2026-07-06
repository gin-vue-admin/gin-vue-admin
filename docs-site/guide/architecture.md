# 架构

## 总览

```
┌─────────────┐   HTTP /api    ┌──────────────────────────────┐
│  web (Vue3) │ ──────────────▶│  server (Go/Gin)             │
│  Element Plus│  ◀─ ApiResult  │  handler→service→repository   │
│  Pinia      │   ◀─ ProblemDet │  →model (GORM/MySQL)          │
└─────────────┘                └──────────────────────────────┘
```

| 模块 | 技术栈 | 路径 |
|------|--------|------|
| 前端 | Vue3 + Vite + TS + Element Plus + Pinia | `web/`（基于 vue-admin，已并入） |
| 后端 | Go + Gin + GORM + JWT + zap + viper | `server/` |
| 部署 | Docker + docker-compose + Nginx | `deploy/` |

## 后端分层（Clean Architecture）

```
handler (HTTP) → service (业务) → repository (接口) → model (GORM)
                    ↘ pkg: jwt / hash / pagination / csvutil / async / apperr / datascope
                    ↘ middleware: AuthRequired / RequirePermission (缓存 TTL)
```

**依赖方向单向**，repository 抽接口便于单测注入 mock。handler 只翻译 HTTP↔业务，service 编排业务，repository 隔离持久化。

## 目录结构

```
gin-vue-admin/
├── server/                    # Go 后端
│   ├── cmd/
│   │   ├── api/main.go        # 入口（依赖组装）
│   │   └── scaffold/          # 代码生成器 CLI
│   ├── internal/
│   │   ├── handler/           # HTTP 端点 + Swagger 注解
│   │   ├── service/           # 业务逻辑（含 *_test.go）
│   │   ├── repository/        # 数据访问（接口 + gorm 实现 + 泛型基类）
│   │   ├── model/             # GORM 实体
│   │   ├── middleware/        # AuthRequired / RequirePermission / OperationLog
│   │   ├── server/            # 路由装配
│   │   └── pkg/               # apperr/response/jwt/hash/pagination/csvutil/async/log/audit/datascope
│   ├── docs/                  # Swagger 产物 + 开发指南
│   └── configs/config.yaml    # 配置（环境变量可覆盖）
├── web/                       # Vue3 前端（基于 vue-admin 演进，普通目录）
├── deploy/                    # Dockerfile / docker-compose / nginx
├── docs-site/                 # 本文档站（VitePress）
└── .github/                   # CI / Issue 模板
```

## API 契约

- **成功**：`HTTP 200 + { code: 0, data, msg, traceId? }`
- **失败**：`HTTP 4xx/5xx + ProblemDetail`（RFC 7807：`type/title/status/detail/code/errors/traceId`）
- **鉴权**：请求头 `Authorization: Bearer <accessToken>`，JWT access/refresh 双 token
- **权限**：RBAC 权限码（`*` 通配超管 + `user:list` 等具体码）
- **校验错误**：HTTP 422 + `VALIDATION_ERROR`

## 关键设计决策

1. **泛型基类 T 为值类型**：`GenericRepository[Foo]`（非 `*Foo`），内部 `new(T)` 取指针，与 `pagination.Paginate[T]` 对齐。
2. **审计字段下沉**：CreatedBy/UpdatedBy 经 GORM 回调注入；DeletedBy 因 GORM 软删 UPDATE 不经回调，下沉到 repository 显式双写。
3. **数据范围无 `custom`**：YAGNI，避免引入 role_dept 关联表 + 分配 UI；4 种 scope 覆盖 95% 场景。
4. **登录防枚举**：响应文案统一，真实失败原因仅写登录日志，解耦安全与可观测。
5. **现有模块不重构**：user/role/permission 维持现状，风险无收益；新模块用泛型基类。
