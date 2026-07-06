# API 契约

## 统一响应

### 成功（HTTP 200）

```json
{ "code": 0, "data": {}, "msg": "ok", "traceId": "uuid" }
```

### 失败（HTTP 4xx/5xx）

RFC 7807 ProblemDetail：

```json
{
  "type": "about:blank",
  "title": "Conflict",
  "status": 409,
  "detail": "用户名已存在",
  "code": "DUPLICATE_KEY",
  "errors": { "username": ["已存在"] },
  "traceId": "uuid",
  "instance": "/api/user"
}
```

## 鉴权

请求头：`Authorization: Bearer <accessToken>`

- access token：短期（默认 2h），用于 API 鉴权
- refresh token：长期（默认 7d），用于换取新 access

登录 `POST /api/auth/sessions` 返回双 token；刷新 `POST /api/auth/tokens/refresh`。

## 权限码

RBAC 权限码模型：
- `*` 通配——超管，短路所有 `RequirePermission`
- 具体码：`user:list`、`role:create`、`system:log` 等

权限变更（角色分配、权限增删）后系统自动调 `InvalidateAll()` 失效缓存。

## 端点总览

84 个端点，13 个 Tag 分组：

| Tag | 路径前缀 | 说明 |
|-----|----------|------|
| auth | `/api/auth` | 登录/刷新/登出/me |
| user | `/api/user` | 用户 CRUD（受数据范围约束） |
| role | `/api/role` | 角色 CRUD + 权限分配 |
| permission | `/api/permission` | 权限码 CRUD |
| crud | `/api/crud` | 通用 CRUD 范例 |
| dept | `/api/system/dept` | 部门（树形） |
| dict-category / dict / dict-item | `/api/dict/...` | 字典三级 |
| menu | `/api/system/menu` | 菜单 + 拖拽排序 |
| operation-log | `/api/system/operation-log` | 操作日志查询 |
| login-log | `/api/system/login-log` | 登录日志查询 |
| sys-config | `/api/system/config` | 系统参数配置 |

## 在线文档

启动后端后访问 [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)：

- 按 Tag 浏览，点击展开看参数/响应 schema
- 点 **Authorize** 输入 `Bearer <token>` 后可在页面直接发请求调试
- 原始 spec：`server/docs/swagger.json`

## 常用状态码

| 状态 | 含义 |
|------|------|
| 200 | 成功 |
| 401 | 未授权（token 缺失/失效） |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 冲突（重复、防自删） |
| 422 | 参数校验失败 |
