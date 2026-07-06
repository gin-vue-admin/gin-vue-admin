# gva web

> 本目录是 [gin-vue-admin](../) 项目的 Vue 3 前端，基于 [vue-admin/vue-admin](https://github.com/vue-admin/vue-admin) 演进，已作为普通目录并入本仓库统一维护。

## 开发

```bash
nvm use 22.23.1   # Node 版本要求（package.json engines）
npm install       # 安装依赖（包管理器见根目录说明）
npm run dev                  # 启动开发服务器（默认 5173；本机端口冲突用 VITE_PORT=5174）
```

开发模式下前端通过 `vite.config.ts` 的 proxy 转发 `/api` 与 `/swagger` 到后端（默认 `http://localhost:8080`，可用 `GVA_API_TARGET` 覆盖）。

## 走真实后端 vs Mock

默认走真实后端（`VITE_ENABLE_MOCK` 默认关闭）。如需启用 Mock Service Worker，设 `VITE_ENABLE_MOCK=true`。

## 详细文档

- 项目总览、架构、部署：见仓库根目录 [README.md](../README.md)
- 工程规范（架构 / HTTP / 状态 / 命名）：见 [docs/standards/](./docs/standards/)
- 工程上下文索引：见 [CLAUDE.md](./CLAUDE.md)

## License

MIT © gin-vue-admin contributors
