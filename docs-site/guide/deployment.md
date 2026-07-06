# 部署

主方案：**Docker Compose 单机 + Nginx 反代 + TLS**。适用于个人/中小团队/POC → 中小生产。

## 拓扑

```
云主机（4C8G+）
└── docker compose up -d
    ├── mysql        (卷持久化 + 每日 mysqldump)
    ├── server       (Go API :8080，仅容器内可见)
    ├── web          (Vue 静态 + VitePress 文档，nginx 托管)
    └── nginx        (80→443 重定向 + TLS + 反代)
                      location /api/   → server
                      location /docs/  → 文档静态
                      location /       → 前端静态
```

## 快速部署

```bash
git clone <repo>
cd gin-vue-admin
cp deploy/.env.example deploy/.env
# 编辑 .env：改 MYSQL_ROOT_PASSWORD、JWT_SECRET、DOMAIN
make compose-build
make compose-up
```

首次启动自动 AutoMigrate + seed（admin/123456，生产务必改）。

## TLS / HTTPS

两种方式：

### 方式一：certbot 容器（推荐）

`.env` 设 `DOMAIN=your.domain.com` + `ENABLE_TLS=true`，compose 的 nginx 暴露 80+443，certbot 容器自动签发 + 续期 Let's Encrypt 证书。

### 方式二：Caddy 替换 nginx

Caddy 自动 TLS，配置更短。改 `web.Dockerfile` 用 caddy 镜像 + Caddyfile。

::: tip
HTTP 全部重定向到 HTTPS（nginx 80 → 443 return 301）。
:::

## 数据持久化与备份

- MySQL 数据卷 `mysql-data` 持久化
- `db-backup` 服务每日 `mysqldump` 到 `deploy/data/backups/`，保留 7 天
- 恢复：`gunzip < backup.sql.gz | docker exec -i gva-mysql mysql -uroot -p gva`

## 配置覆盖

所有配置项支持环境变量覆盖（`GVA_` 前缀，双下划线分隔嵌套）。compose 的 `server.environment` 段是生产配置的主入口：

```yaml
GVA_DB__HOST: mysql
GVA_DB__PASSWORD: ${MYSQL_ROOT_PASSWORD}
GVA_JWT__SECRET: ${JWT_SECRET}
GVA_APP__MODE: release
```

## 更新版本

```bash
git pull
make compose-build     # 重建镜像
make compose-up        # 滚动重启（数据卷保留）
```

## 其他部署形态

| 方案 | 适用 | 备注 |
|------|------|------|
| 二进制 + systemd + 托管 MySQL | 单实例、极简启动 | `go build` + systemd unit + 云 RDS |
| Kubernetes + Helm | 多实例/滚动发布 | 需自写 Helm chart |
| 阿里云 ACK / AWS ECS | 云原生托管 | 用容器服务 + RDS |

详见仓库根 `DEPLOYMENT.md`。

## 文档站部署

VitePress 构建产物集成在 nginx 镜像内，访问 `/docs/`。本地预览：

```bash
pnpm install           # 首次（根 package.json）
make docs-dev          # 或 pnpm docs:dev，http://localhost:5173/docs/（端口冲突：VITE_PORT=5175 pnpm docs:dev）
```

修改 `docs-site/` 后在 CI 自动重新构建。
