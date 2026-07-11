# 变更日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [Unreleased]

## [0.2.0] - 2026-07-11

### Added
- 前端 CI 门禁（`.github/workflows/web-ci.yml`）：eslint + vue-tsc 类型检查 + vitest 单测（pretest 钩子含 check:api 前后端契约一致性检测）+ v8 覆盖率 + playwright smoke e2e（dev server + MSW，25 用例），与后端 ci.yml 平级。
- handler 集成测试全覆盖：新增 9 模块 63 个集成测试（permission/role/menu/dept/dict/sys_config/notice/log/dashboard+health+crud）+ `newAppServer` 全量测试装配，handler 包覆盖率 5.6% → 55.1%。
- server `-version` flag（开源项目常备版本号输出，可由 ldflags 注入）。
- README 动态 CI 徽章（ci.yml + web-ci.yml）。

### Changed
- docs-site 文档站独立成 `docs-site/` 子项目（与 web/ 对齐）：依赖管理从仓库根下沉到 `docs-site/package.json` + `pnpm-lock.yaml` + `pnpm-workspace.yaml`（仅 `allowBuilds`，pnpm v11 配置载体）；删除根 `package.json`/`pnpm-lock.yaml`/`pnpm-workspace.yaml`；`deploy-docs.yml` / `Makefile` / `web.Dockerfile` / `dependabot.yml` 改走 docs-site/；`web-ci.yml` 去掉三处 `--ignore-workspace`（根 workspace 已删，web/ 天然独立，根因消除）。

### Removed
- 删除 `web/docs-site/`（vue-admin 独立项目遗留文档站，根 `docs-site/` 已作为统一文档站，消除双源维护）与 `web/.github/`（vue-admin 遗留 GitHub workflow/templates，子目录 `.github` 不被 GitHub 执行属死代码）；同步移除 `web/package.json` 的 docs 脚本。
- `web/public/md.html`：遗留 md-editor CDN demo 页（引用 zeus 框架，与 gva 无关）。
- 根 `package.json`/`pnpm-lock.yaml`/`pnpm-workspace.yaml`（docs-site 独立化下沉，见 Changed）。

### Fixed
- `check:api` 一致性脚本正则支持模板字符串反引号，检测覆盖 50 → 91 个端点（此前 `` api.get(`/api/.../${id}`) `` 这类反引号端点漏检）。
- 补齐 `system/config` 模块 mock handler（6 端点：列表/按 id/按 key/增/改/删），修复 check:api 检测到的前后端契约不一致；此前该模块仅有前端 API 调用无 mock 落地。
- Makefile `docs-build` 的 `DOCS_BASE` 变量转义（单 `$` → `$$`，原表达式被 make 吞掉一直失效）。
- `docs-site/pnpm-workspace.yaml` 补 `packages: ['.']`（pnpm 9 要求非空，否则 "packages field missing"）。

## [0.1.0] - 2026-07-07

### Added
- 后端基座里程碑 M6–M10 全部交付：
  - **M6 数据层基座**：泛型 `GenericRepository[T]` + 审计字段（CreatedBy/UpdatedBy/DeletedBy）+ GORM 回调自动注入 + 新模块开发指南。
  - **M7 组织基座**：部门（树形+递归级联删）+ 字典（三级 categories/dicts/items）。
  - **M8 日志与权限**：操作日志中间件、登录日志、数据范围（`pkg/datascope`，按角色 DataScope 控制可见数据）、Swagger API 文档（84 操作 / 41 路径）。
  - **M9 开发体验**：代码生成器 `cmd/scaffold`（CLI 生成 model/repo/service/handler）。
  - **M10 系统参数配置**：`sys_config` + 内存缓存 + CRUD + 编程 API（`GetValue/GetBool/GetInt`）。
- 开源工程基础设施：MIT LICENSE、golangci-lint 规则集、GitHub Actions CI、CONTRIBUTING/SECURITY 策略、Issue/PR 模板、.editorconfig。
- 文档站（VitePress）+ 全栈 Docker 部署（docker-compose + nginx + TLS + 自动备份）。

### Security
- 登录失败响应统一防枚举；真实失败原因仅记入登录日志。
- 用户禁用/删除自己被拒绝（409）。
- Swagger 生产关闭、操作日志脱敏、`config.yaml` gitignore 防密钥误提交。
