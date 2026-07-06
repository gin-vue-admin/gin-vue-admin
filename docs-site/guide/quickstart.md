# 快速开始

## 前置依赖

- Go 1.25+、Node 22+、pnpm 9+、Docker（部署用）
- MySQL 8（本地开发）或直接用 Docker

## 一、Docker 全栈部署（推荐）

```bash
git clone <repo>
cd gin-vue-admin
cp deploy/.env.example deploy/.env   # 按需改密码/端口
make compose-build                    # 构建并启动 mysql + server + web
```

访问 `http://localhost`，受限网络在 `.env` 设 `REGISTRY=docker.m.daocloud.io/library`。

## 二、本地开发

```bash
# 1. 起 MySQL（Docker）
docker run -d --name gva-mysql -p 13306:3306 \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=gva mysql:8

# 2. 后端
cd server && cp configs/config.example.yaml configs/config.yaml   # 首次：从模板创建本地配置（config.yaml 被 gitignore）
cd .. && make server-dev  # :8080（端口冲突见下方"端口冲突处理"）
curl http://localhost:8080/api/health

# 3. 前端
make web-install         # 首次安装依赖
make web-dev             # http://localhost:5173（端口冲突见下方"端口冲突处理"）
```

::: tip 环境变量覆盖
后端配置支持 `GVA_` 前缀环境变量覆盖（双下划线分隔嵌套）：
`GVA_DB__HOST`、`GVA_DB__PORT`、`GVA_JWT__SECRET`、`GVA_SERVER__PORT`。
:::

## 端口冲突处理

框架默认 Gin 标准 8080、vite 标准 5173。本机端口被占用时用环境变量覆盖（无需改代码）：

| 服务 | 环境变量 | 示例 |
|------|----------|------|
| 后端监听端口 | `GVA_SERVER__PORT` | `GVA_SERVER__PORT=8088 make server-dev` |
| 前端 dev 端口 | `VITE_PORT` | `VITE_PORT=5174 make web-dev` |
| 前端连后端地址 | `GVA_API_TARGET` | 后端非 8080 时前端需同步：`GVA_API_TARGET=http://localhost:8088 make web-dev` |

::: info
vite 默认 `strictPort:false`，5173 被占会自动递增到下一个可用端口；需固定端口则命令行加 `--strictPort`。
:::

## 测试账号（启动自动 seed）

| 用户名 | 密码 | 角色 | 权限 |
|--------|------|------|------|
| admin | 123456 | super_admin | 全部（`*`） |
| user | 123456 | user | `user:read` |

::: warning
生产环境务必修改默认密码与 JWT secret。
:::

## Swagger 文档

启动后端后访问 [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)，84 个端点可在线调试。

## 开发命令

```bash
make server-lint        # golangci-lint 静态检查（CI 门禁）
make server-test        # 后端单测
make server-cover       # 单测 + 覆盖率报告
make server-swag        # 重新生成 Swagger docs（改 handler 注解后）
make server-ci          # CI 本地复现：lint → swagger → vet → test → build
make compose-up/down    # 启停全栈
make docs-dev           # 本地预览文档站
```
