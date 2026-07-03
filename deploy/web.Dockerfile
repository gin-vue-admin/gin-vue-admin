# 前端构建 + nginx 托管。
# 构建阶段：pnpm build 生成 dist；运行阶段：nginx 托管静态并反代后端。
# 用法：docker build -f deploy/web.Dockerfile -t gva-web:latest .
# 受限网络：--build-arg REGISTRY=docker.m.daocloud.io/library

ARG REGISTRY=docker.io/library

# ---------- 构建阶段 ----------
FROM ${REGISTRY}/node:22-alpine AS builder
RUN corepack enable && corepack prepare pnpm@10 --activate
WORKDIR /build
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
# 生产 base 路径按需：根路径部署用 VITE_BASE=/
RUN pnpm build

# ---------- 运行阶段 ----------
FROM ${REGISTRY}/nginx:1.27-alpine
# 拷构建产物到 nginx html 目录
COPY --from=builder /build/dist /usr/share/nginx/html
# nginx 配置：SPA history 回退 + 反代 /api 到 server
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
