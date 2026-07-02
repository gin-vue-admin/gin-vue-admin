# Gin + Vue-Admin 后台管理系统 — 顶层编排
# 用法：make <target>   （Windows 建议在 WSL/Git Bash 下使用）

SERVER_DIR := server
WEB_DIR    := web
SERVER_BIN := $(SERVER_DIR)/bin/api

.DEFAULT_GOAL := help

.PHONY: help
help: ## 显示所有命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ---------------- 后端 ----------------
.PHONY: server-dev server-build server-run server-test server-tidy server-swag
server-dev: ## 后端开发模式（热重载需自行接 air）
	cd $(SERVER_DIR) && PORT=8080 go run ./cmd/api

server-build: ## 编译后端二进制到 server/bin
	cd $(SERVER_DIR) && go build -o bin/api ./cmd/api

server-run: $(SERVER_BIN) ## 运行编译后的后端
	$(SERVER_BIN)

server-test: ## 运行后端单测
	cd $(SERVER_DIR) && go test ./...

server-tidy: ## 整理后端依赖
	cd $(SERVER_DIR) && go mod tidy

server-swag: ## 生成 Swagger 文档（M5 启用）
	cd $(SERVER_DIR) && swag init -g cmd/api/main.go -o docs

# ---------------- 前端 ----------------
.PHONY: web-install web-dev web-build
web-install: ## 安装前端依赖
	cd $(WEB_DIR) && pnpm install

web-dev: ## 启动前端开发服务器
	cd $(WEB_DIR) && pnpm dev

web-build: ## 构建前端生产包
	cd $(WEB_DIR) && pnpm build

# ---------------- 部署 ----------------
.PHONY: compose-up compose-down compose-build
compose-up: ## 启动全栈（mysql+server+web）
	cd deploy && docker compose up -d

compose-down: ## 停止全栈
	cd deploy && docker compose down

compose-build: ## 构建并启动全栈
	cd deploy && docker compose up -d --build
