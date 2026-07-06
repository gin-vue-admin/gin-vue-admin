# 概览

**gin-vue-admin** 是一套**业界优秀的通用管理框架基座**——不是业务系统，不绑定任何业务实体（crud 等仅作脚手架范例）。

## 定位

对标 RuoYi / go-admin / django-admin，但更现代、更干净：

- **后端**：Go 1.25 + Gin + GORM + JWT + zap + viper，自研三层架构
- **前端**：Vue3 + Vite + TypeScript + Element Plus + Pinia（复用 [vue-admin](https://github.com/vue-admin/vue-admin)）
- **部署**：Docker + docker-compose + Nginx，一键起全栈

## 差异化能力

| 能力 | 说明 |
|------|------|
| 审计字段 | GORM 回调自动注入 CreatedBy/UpdatedBy；软删记录 DeletedBy |
| 数据范围 | `pkg/datascope` 按角色 `DataScope` 控制可见数据，超管短路 |
| 登录防枚举 | 失败响应文案统一，真实原因仅写入登录日志 |
| 字典/部门 | 三级字典 + 树形部门（递归级联删） |
| 系统配置 | `sys_config` + 内存缓存 + 编程 API，运营改配置不发版 |
| 代码生成器 | `cmd/scaffold` 一键生成四层代码 |
| Swagger 文档 | 84 端点全注解，`/swagger/index.html` 在线调试 |

## 功能模块

| 模块 | 状态 | 说明 |
|------|------|------|
| 认证 (M2) | ✅ | 登录/刷新/登出/me，JWT 双 token，防枚举，bcrypt |
| 权限 (M3.1) | ✅ | 权限码 CRUD + 分页/CSV + 权限中间件（超管 `*` 短路） |
| 角色 (M3.2) | ✅ | 角色 CRUD + 权限分配子资源 |
| 用户 (M3.3) | ✅ | 用户 CRUD + 多角色 + 防自删/自禁 + 批量删/导出 |
| 菜单 (M4.1) | ✅ | 动态路由菜单树 |
| 部署 (M5) | ✅ | 多阶段 Dockerfile + docker-compose + nginx |
| 数据层基座 (M6) | ✅ | 泛型 `GenericRepository[T]` + 审计字段 + 开发指南 |
| 组织基座 (M7) | ✅ | 部门 + 字典（三级） |
| 日志与权限 (M8) | ✅ | 操作日志 + 登录日志 + 数据范围 + Swagger |
| 开发体验 (M9) | ✅ | 代码生成器 `cmd/scaffold` |
| 系统配置 (M10) | ✅ | `sys_config` + 内存缓存 + CRUD |

## 工程化

MIT License · golangci-lint v2 · GitHub Actions CI · 11 个测试包 · 完整贡献流程。
