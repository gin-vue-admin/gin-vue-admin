# 变更日志

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### Added
- 后端基座里程碑 M6–M10 全部交付：
  - **M6 数据层基座**：泛型 `GenericRepository[T]` + 审计字段（CreatedBy/UpdatedBy/DeletedBy）+ GORM 回调自动注入 + 新模块开发指南。
  - **M7 组织基座**：部门（树形+递归级联删）+ 字典（三级 categories/dicts/items）。
  - **M8 日志与权限**：操作日志中间件、登录日志、数据范围（`pkg/datascope`，按角色 DataScope 控制可见数据）、Swagger API 文档（84 操作 / 41 路径）。
  - **M9 开发体验**：代码生成器 `cmd/scaffold`（CLI 生成 model/repo/service/handler）。
  - **M10 系统参数配置**：`sys_config` + 内存缓存 + CRUD + 编程 API（`GetValue/GetBool/GetInt`）。
- 开源工程基础设施：MIT LICENSE、golangci-lint 规则集、GitHub Actions CI、CONTRIBUTING/SECURITY 策略、Issue/PR 模板、.editorconfig。

### Changed
- README 补充 CI/coverage/license 徽章与开发规范指引。

### Security
- 登录失败响应统一防枚举；真实失败原因仅记入登录日志。
- 用户禁用/删除自己被拒绝（409）。
