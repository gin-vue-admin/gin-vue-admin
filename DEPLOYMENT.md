# 部署指南

gin-vue-admin 的部署方案、TLS、备份、文档站部署。

## 方案对比

| 方案 | 适用 | 复杂度 |
|------|------|--------|
| **Docker Compose 单机 + TLS**（本文主方案） | 个人/中小团队/POC → 中小生产 | 低 |
| 二进制 + systemd + 托管 MySQL | 单实例、极简 | 中 |
| Kubernetes + Helm | 多实例/滚动发布 | 高 |

---

## 一、Docker Compose 部署（主方案）

### 拓扑

```
云主机（推荐 4C8G+）
└── docker compose up -d
    ├── mysql        (卷 mysql-data 持久化)
    ├── server       (Go API :8080，仅容器内可见，healthcheck)
    ├── web          (nginx：前端 SPA + VitePress 文档 + 反代 /api + /swagger)
    └── db-backup    (可选 profile，每日 mysqldump)
```

### 步骤

```bash
# 1. 拉代码
git clone <repo>
cd gin-vue-admin

# 2. 配环境
cp deploy/.env.example deploy/.env
vim deploy/.env   # 改 MYSQL_ROOT_PASSWORD / JWT_SECRET（必须）

# 3. 构建并启动
make compose-build       # = docker compose -f deploy/docker-compose.yml build
make compose-up          # = docker compose up -d

# 4. 验证
curl http://localhost/api/health        # {"code":0,"data":{"status":"up"}}
curl -I http://localhost/docs/          # 200，文档站
open http://localhost                    # 前端
```

首次启动自动 AutoMigrate + seed（admin/123456，**生产必须改**）。

### 服务清单

| 服务 | 端口 | 说明 |
|------|------|------|
| mysql | 13306:3306 | 持久化卷 + healthcheck |
| server | 仅内部 :8080 | healthcheck = `/api/health`，非 root 运行 |
| web (nginx) | 80:80 | 前端 + 文档 + 反代 |
| db-backup | — | 可选，`--profile backup` 启用 |

---

## 二、TLS / HTTPS（certbot + nginx）

默认 compose 跑 HTTP（nginx:80）。生产 TLS 由 `docker-compose.tls.yml` override 叠加，证书用 Let's Encrypt 自动签发+续期。

### 前置

- 真实域名 `DOMAIN` 的 A 记录已指向云主机公网 IP
- 宿主 80/443 端口对公网开放、80 空闲（首次签发用 standalone 模式临时占用 80）
- `.env` 设 `DOMAIN` 与 `EMAIL`

```ini
# deploy/.env
DOMAIN=example.com
EMAIL=you@example.com
```

### 首次签发

```bash
bash deploy/scripts/init-certs.sh
```

脚本做两件事：
1. `docker run certbot --standalone` 签发证书到宿主 `/etc/letsencrypt/live/$DOMAIN/`
2. `sed` 替换 `nginx.tls.conf` 的 `DOMAIN_PLACEHOLDER` → 生成 `deploy/data/nginx.runtime.conf`

### 启动 TLS 全栈

```bash
cd deploy
docker compose -f docker-compose.yml -f docker-compose.tls.yml --profile tls up -d
```

`--profile tls` 启用 certbot 续期服务（每 12h 检查，临近过期自动更新 + HUP nginx reload）。

### 验证

```bash
curl -I https://example.com/api/health    # HTTP/2 200
open https://example.com/docs/             # 文档站
```

HTTP 全部 301 重定向到 HTTPS（HSTS 已开启）。

### 文件说明

| 文件 | 作用 |
|------|------|
| `deploy/docker-compose.tls.yml` | TLS override（web 暴露 443 + 挂证书/conf；certbot 续期服务） |
| `deploy/nginx.tls.conf` | 443 server 模板（含 DOMAIN_PLACEHOLDER） |
| `deploy/scripts/init-certs.sh` | 首次签发 + 生成 runtime conf |
| `deploy/data/nginx.runtime.conf` | sed 生成的实际 nginx 配置（gitignore） |
| `/etc/letsencrypt` | 证书（certbot 默认，宿主全局） |

### 备选 TLS 方案

若不想要 certbot 的复杂度：

- **边界网关**：云 SLB / Cloudflare 在 compose 前做 TLS 终止，nginx 仍跑 80（改动最小，证书托管）
- **Caddy 替换 nginx**：改 `web.Dockerfile` 用 `caddy:2-alpine` + Caddyfile，自动 TLS：

  ```caddyfile
  example.com {
      reverse_proxy /api/* server:8080
      reverse_proxy /swagger/* server:8080
      root * /srv
      try_files {path} /index.html
      file_server
  }
  ```

---

## 三、数据持久化与备份

### 卷

- `mysql-data`：MySQL 数据，docker volume，重建容器不丢

### 自动备份（db-backup 服务）

```bash
# 启用备份（compose profile）
docker compose --profile backup -f deploy/docker-compose.yml up -d

# 产物：deploy/data/backups/gva-YYYYMMDD-HHMMSS.sql.gz，保留 7 天（KEEP_DAYS 可配）
```

### 手动备份/恢复

```bash
# 备份
docker exec gva-mysql mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction gva \
  | gzip > deploy/data/backups/manual-$(date +%F).sql.gz

# 恢复
gunzip < deploy/data/backups/gva-xxx.sql.gz \
  | docker exec -i gva-mysql mysql -uroot -p"$MYSQL_ROOT_PASSWORD" gva
```

---

## 四、配置覆盖

所有配置支持环境变量覆盖（`GVA_` 前缀，双下划线分隔嵌套）。compose 的 `server.environment` 是生产配置主入口：

```yaml
GVA_DB__HOST: mysql
GVA_DB__PASSWORD: ${MYSQL_ROOT_PASSWORD}
GVA_JWT__SECRET: ${JWT_SECRET}
GVA_APP__MODE: release
```

也可挂 `configs/config.local.yaml` 覆盖默认值（已被 gitignore）。

---

## 五、文档站部署

VitePress 文档站已集成进 `web.Dockerfile`：

- 构建期：`pnpm docs:build` 生成静态 `.html`（`cleanUrls=false`，nginx 直出无需 SPA fallback）
- 运行期：产物挂到 nginx `/usr/share/nginx/html/docs`，nginx `location /docs/` 服务
- base 路径由 `DOCS_BASE` 决定（默认 `/docs/`，独立域名部署改 `/`）

访问：`http://localhost/docs/`

本地预览（不改部署）：

```bash
cd docs-site && pnpm install   # 首次（docs-site/package.json）
cd docs-site && pnpm dev       # http://localhost:5173/（端口冲突：VITE_PORT=5175 pnpm dev）
```

---

## 六、更新版本

```bash
git pull
make compose-build     # 重建镜像
make compose-up        # 重启（数据卷保留）
```

平滑更新（零停机）需多实例 + 负载均衡，见下方 Kubernetes 方案。

---

## 七、其他部署形态

### 二进制 + systemd（单机极简）

```bash
# 编译
cd server && CGO_ENABLED=0 go build -o /opt/gva/api ./cmd/api

# systemd unit: /etc/systemd/system/gva.service
[Unit]
Description=gva server
After=network.target mysql.service

[Service]
ExecStart=/opt/gva/api
Environment=GVA_DB__HOST=127.0.0.1 GVA_JWT__SECRET=xxx GVA_APP__MODE=release
Restart=always
User=gva

[Install]
WantedBy=multi-user.target
```

MySQL 用云 RDS 或本机 systemd 管理。前端 `pnpm build` 后 dist 丢 nginx。

### Kubernetes

写 Helm chart（Deployment + Service + Ingress + StatefulSet MySQL 或用云 RDS）。适合多实例/滚动发布/弹性伸缩。本仓库暂不提供 chart，可按 server/web 镜像自行编排。

---

## 八、生产加固清单

- [ ] 改 `MYSQL_ROOT_PASSWORD`、所有 seed 密码、`JWT_SECRET`（`openssl rand -hex 32`）
- [ ] TLS 启用（边界网关或 certbot/Caddy）
- [ ] 启用 db-backup（`--profile backup`）+ 异地存储备份
- [ ] `/swagger` 路由生产关闭或加网关鉴权（不应公开 API 文档）
- [ ] 最小权限 DB 账号（应用不用 root）
- [ ] 日志轮转（`server/logs/` + lumberjack 已内置）+ 监控告警
- [ ] 资源限制（compose 加 `mem_limit`/`cpus`）
- [ ] 防火墙：仅暴露 80/443，13306 不对公网
