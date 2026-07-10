# 前端构建 + 文档站构建 + nginx 托管。
# 构建阶段：pnpm build 生成 web dist + VitePress 生成 docs；运行阶段：nginx 托管静态并反代后端。
# 用法：docker build -f deploy/web.Dockerfile -t gva-web:latest .
# 受限网络：--build-arg REGISTRY=docker.m.daocloud.io/library

ARG REGISTRY=docker.io/library
ARG NPM_REGISTRY=https://registry.npmjs.org

# ---------- 前端构建阶段 ----------
FROM ${REGISTRY}/node:22-alpine AS builder
RUN corepack enable && corepack prepare pnpm@10 --activate
WORKDIR /build
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm config set registry ${NPM_REGISTRY} && pnpm install --frozen-lockfile
COPY web/ ./
# 生产 base 路径按需：根路径部署用 VITE_BASE=/
RUN pnpm build

# ---------- 文档站构建阶段（VitePress）----------
FROM ${REGISTRY}/node:22-alpine AS docs-builder
RUN corepack enable && corepack prepare pnpm@10 --activate
WORKDIR /docsbuild
COPY docs-site/package.json docs-site/pnpm-lock.yaml ./
RUN pnpm config set registry ${NPM_REGISTRY} && pnpm install --no-frozen-lockfile
COPY docs-site/ ./
# DOCS_BASE 决定文档站 base 路径：云主机子路径 /docs/（默认），独立域名用 /
ARG DOCS_BASE=/docs/
ENV DOCS_BASE=${DOCS_BASE}
RUN pnpm build

# ---------- 运行阶段 ----------
FROM ${REGISTRY}/nginx:1.27-alpine
# 前端构建产物
COPY --from=builder /build/dist /usr/share/nginx/html
# 文档站构建产物（挂 /docs 子路径，与 DOCS_BASE 对应）
COPY --from=docs-builder /docsbuild/.vitepress/dist /usr/share/nginx/html/docs
# nginx 配置：SPA history 回退 + 反代 /api 到 server + /docs 静态
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
