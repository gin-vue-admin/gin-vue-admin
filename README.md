# gin-vue-admin

> 企业级后台管理系统：**Gin** 后端 + [vue-admin/vue-admin](https://github.com/vue-admin/vue-admin) 前端，遵循业界标准（SOLID / KISS / Clean Architecture）实现。

后端自研，**严格对齐前端 MSW mock 契约**（端点路径、方法、字段一一对应）；前端作为 git submodule 直接复用 vue-admin。

## 架构

```
┌─────────────┐   HTTP /api    ┌──────────────────────────────┐
│  web (Vue3) │ ──────────────▶│  server (Go/Gin)             │
│  Element Plus│  ◀─ ApiResult  │  handler→service→repository   │
│  Pinia      │   ◀─ ProblemDet │  →model (GORM/MySQL)          │
└─────────────┘                └──────────────────────────────┘
```

| 模块 | 技术栈 | 路径 | 说明 |
|------|--------|------|------|
| 前端 | Vue3 + Vite + TS + Element Plus + Pinia | `web/` (submodule) | 直接使用 vue-admin/vue-admin |
| 后端 | Go + Gin + GORM + JWT + zap + viper | `server/` | 自研，三层架构 + 构造注入 |
| 部署 | Docker + docker-compose + Nginx | `deploy/` | 一键起全栈 |

### 后端分层（Clean Architecture）

```
handler (HTTP) → service (业务) → repository (接口) → model (GORM)
                    ↘ pkg: jwt / hash / pagination / csvutil / async / apperr
                    ↘ middleware: AuthRequired / RequirePermission (缓存 TTL)
```

依赖方向单向，repository 抽接口便于单测注入 mock。

## 功能模块

| 模块 | 状态 | 说明 |
|------|------|------|
| 认证 (M2) | ✅ | 登录/刷新/登出/me，JWT 双 token，防枚举，bcrypt |
| 权限 (M3.1) | ✅ | 权限码 CRUD + 通用分页/CSV + 权限中间件（超管 `*` 短路） |
| 角色 (M3.2) | ✅ | 角色 CRUD + 权限分配子资源（code↔id 严格校验） |
| 用户 (M3.3) | ✅ | 用户 CRUD + 多角色 + 防自删/自禁 + 批量删/导出 |
| 菜单 (M4.1) | ✅ | 动态路由菜单树（MenuDTO + meta.permissions） |
| 部署 (M5) | ✅ | 多阶段 Dockerfile + docker-compose + nginx 反代 |

## API 契约（核心约定）

- **成功**：`HTTP 200 + { code: 0, data, msg, traceId? }`
- **失败**：`HTTP 4xx/5xx + ProblemDetail`（RFC 7807，含 `type/title/status/detail/code/errors/traceId`）
- **鉴权**：请求头 `Authorization: Bearer <accessToken>`，JWT access/refresh 双 token（refresh 轮换）
- **权限**：RBAC 权限码模型（`*` 通配超管 + `user:list` 等具体码）
- **校验错误**：HTTP 422 + `VALIDATION_ERROR`

> 后端任何端点的路径/方法/字段必须与 `web/src/mock/handlers/*.ts` 保持一致。

## 快速开始

### 前置依赖

- Go 1.25+、Node 22+、pnpm 10+、Docker（部署用）
- MySQL 8（本地开发）或直接用 Docker（见部署）

### 一、Docker 全栈部署（推荐）

```bash
git clone --recurse-submodules <repo>
cd gin-vue-admin
cp deploy/.env.example deploy/.env   # 按需改密码/端口
make compose-build                    # 构建并启动 mysql + server + web
```

访问 `http://localhost`，受限网络在 `.env` 设 `REGISTRY=docker.m.daocloud.io/library`。

### 二、本地开发

```bash
# 1. 起 MySQL（Docker）
docker run -d --name gva-mysql -p 13306:3306 \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=gva mysql:8

# 2. 后端
make server-dev          # :8088（config.yaml 可改，GVA_SERVER__PORT 覆盖）
curl http://localhost:8088/api/health

# 3. 前端
make web-install         # 首次安装依赖
make web-dev             # http://localhost:5173
```

> 后端配置支持环境变量覆盖（`GVA_` 前缀，双下划线分隔嵌套）：
> `GVA_DB__HOST`、`GVA_DB__PORT`、`GVA_JWT__SECRET`、`GVA_SERVER__PORT` 等。

## 测试账号（启动自动 seed）

| 用户名 | 密码 | 角色 | 权限 |
|--------|------|------|------|
| admin | 123456 | super_admin | 全部（`*`） |
| user | 123456 | user | `user:read` |

> 生产环境务必修改默认密码与 JWT secret。

## 开发命令

```bash
make server-test        # 后端单测
make server-build       # 编译二进制到 server/bin
make server-tidy        # 整理依赖
make compose-up/down    # 启停全栈
```

## 项目结构

```
gin-vue-admin/
├── server/                    # Go 后端
│   ├── cmd/api/main.go        # 入口（依赖组装）
│   ├── internal/
│   │   ├── handler/           # HTTP 端点
│   │   ├── service/           # 业务逻辑（含 *_test.go）
│   │   ├── repository/        # 数据访问（接口 + gorm 实现）
│   │   ├── model/             # GORM 实体
│   │   ├── middleware/        # AuthRequired / RequirePermission
│   │   ├── pkg/               # apperr/response/jwt/hash/pagination/csvutil/async/log
│   │   └── server/            # 路由装配
│   └── configs/config.yaml    # 配置（环境变量可覆盖）
├── web/                       # Vue3 前端（submodule）
├── deploy/                    # Dockerfile / docker-compose / nginx
└── docs/superpowers/          # 设计 spec 与实现计划
```

## 测试

后端 service/repository/middleware/pagination/csvutil 单测全覆盖（SQLite 隔离）：

```bash
cd server && go test ./...
```

## License

MIT
