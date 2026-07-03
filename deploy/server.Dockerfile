# 后端服务多阶段构建。
# 构建阶段：编译静态二进制；运行阶段：最小 alpine 镜像。
# 用法：docker build -f deploy/server.Dockerfile -t gva-server:latest .
# 受限网络可用镜像源前缀：--build-arg REGISTRY=docker.m.daocloud.io/library

ARG REGISTRY=docker.io/library

# ---------- 构建阶段 ----------
FROM ${REGISTRY}/golang:1.25-alpine AS builder

# CGO 禁用 + 静态链接，适配 alpine（无 glibc）
# GOPROXY 可配：受限网络用 https://goproxy.cn（构建参数 GOPROXY 覆盖）
# GOARCH 不硬编码：用构建机架构，避免交叉编译 OOM；多架构部署用 buildx --platform
ARG GOPROXY=https://goproxy.cn
ENV CGO_ENABLED=0 GOOS=linux GOPROXY=${GOPROXY}

WORKDIR /build

# 先拷依赖清单，利用层缓存
COPY server/go.mod server/go.sum ./
RUN go mod download

# 拷源码并编译。-p 1 限制并行包编译，降低内存峰值（避免受限内存环境 OOM）
COPY server/ ./
RUN go build -p 1 -ldflags="-s -w" -o /out/api ./cmd/api

# ---------- 运行阶段 ----------
FROM ${REGISTRY}/alpine:3.20

# ca-certificates（HTTPS）、tzdata（时区）
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

ENV TZ=Asia/Shanghai

WORKDIR /app
# 拷二进制与默认配置
COPY --from=builder /out/api ./api
COPY server/configs ./configs

USER app
EXPOSE 8080

# 通过环境变量覆盖配置（连 compose 内的 mysql、端口等）
# GVA_DB__HOST / GVA_DB__PORT / GVA_DB__PASSWORD / GVA_SERVER__PORT / GVA_APP__MODE ...
ENTRYPOINT ["./api"]
