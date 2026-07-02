# M2 认证模块设计

> 状态：已批准（2026-07-02）
> 范围：实现登录/刷新/登出/当前用户四个认证端点，对齐前端 MSW 契约，建立三层架构与测试基础设施。
> 前置：M0 骨架、M1 基础设施（数据库/RBAC 模型/JWT/hash/response/apperr）已完成并跑通。

## 1. 目标与边界

### 目标
- 实现 4 个认证端点，严格对齐前端 `web/src/mock/handlers/auth.ts` 与 `web/src/lib/auth/types.ts` 契约。
- 建立 handler→service→repository 三层架构与构造注入模式，为 M3 立模板。
- 建立 service 层单测基础设施（mock repo + testify），后续 service 复用。
- 启动时幂等种入 admin/user 种子账户。

### 非目标（YAGNI）
- refresh token 不落库，无服务端吊销/黑名单（接受自然过期，TTL 7 天）。
- 不引入依赖注入框架（wire 等），手动组装。
- 不做请求体大小限制中间件（Gin 默认处理足够）。
- 不为纯 CRUD repository 预先抽接口——仅 auth 的 repo 抽接口（为单测）。

## 2. 关键决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 错误响应 | RFC 7807 ProblemDetail | CLAUDE.md 目标契约，已实现 problem.go |
| refresh token | 纯 JWT 不落库 | KISS，自然过期，接受多端旧 token 仍有效的权衡 |
| 种子账户 | 启动时自动幂等 seed | 首启即可登录，FirstOrCreate 不清已有数据 |
| 分层 | handler→service→repository 三层 | Clean Architecture，可测试，对齐 roadmap |
| 依赖注入 | 构造函数注入 + main.go 手动组装 | 显式依赖、可测、KISS，无框架 |
| repository 接口 | 仅 auth repo 抽接口 | 为 service 单测；纯 CRUD repo 按需再抽 |
| 测试 | service 层单测（mock repo + testify） | 隔离 DB，毫秒级，复利基础设施 |
| id 类型 | uint（JSON 数字） | 前端 `string\|number` 兼容，无分布式需求不上 UUID |
| 超管语义 | 判断 `*` in permissions，不判断角色码 | 权限持有语义，解耦角色码，对齐 Spring Security 模式 |
| 登录统计更新 | 异步 + 独立 context + 失败告警 | 不阻塞登录，请求结束后不被取消 |

## 3. 端点契约（对齐前端）

### 端点 1：登录 `POST /api/auth/sessions`
```
请求: {"username":"admin","password":"123456"}
成功(200): {"code":0,"data":{"accessToken":"...","refreshToken":"...","expiresIn":3600},"msg":"ok"}
失败(401): ProblemDetail{status:401,title:"Unauthorized",detail:"用户名或密码错误",code:"UNAUTHORIZED"}
失败(422): ProblemDetail{status:422,...,errors:{field:[msg]}}  // 请求体校验失败
```

数据流：
```
handler.Login(c)
  → 绑定+校验 JSON body（ShouldBindJSON，binding tag）
  → service.Login(ctx, username, password)
      → repo.FindByUsername(username)
      → user == nil → apperr.Unauthorized("用户名或密码错误")  // 与密码错同一文案防枚举
      → hash.Compare(user.Password, password) 失败 → apperr.Unauthorized("用户名或密码错误")
      → user.Status != "active" → apperr.Unauthorized("账户已禁用")
      → jwt.GenerateAccess(user.ID, username)
      → jwt.GenerateRefresh(user.ID, username)
      → 异步: repo.UpdateLoginStats(user.ID) // 独立 context，失败仅告警
      → return AuthResult{AccessToken, RefreshToken, ExpiresIn: jwtMgr.AccessTTLSeconds()}
  → response.Success(c, result)
```

### 端点 2：刷新 `POST /api/auth/tokens/refresh`
```
请求: {"refreshToken":"..."}
成功(200): 同登录 data
失败(401): ProblemDetail,detail:"Invalid refresh token",code:"UNAUTHORIZED"
```

数据流：
```
handler.Refresh(c)
  → 绑定 JSON body
  → service.Refresh(ctx, refreshTokenStr)
      → jwt.Parse(refreshTokenStr) 失败 → apperr.Unauthorized("Invalid refresh token")
      → claims.Type != TypeRefresh → apperr.Unauthorized("Invalid refresh token")
      → repo.FindByID(claims.UserID)
      → user == nil 或 Status != "active" → apperr.Unauthorized
      → 重新签发 access + refresh（旧 refresh 自然过期）
      → return AuthResult{...}
  → response.Success(c, result)
```

权衡说明：纯 JWT 不落库，旧 refresh 在 TTL（7 天）内仍可解析，无法真正删除。接受。

### 端点 3：登出 `DELETE /api/auth/sessions`
```
成功(200): {"code":0,"data":null,"msg":"ok"}
```
纯 JWT 模式下 service.Logout 为空操作（对齐 mock）；前端清本地 storage 实现"登出"。未来需服务端吊销再加黑名单表。

### 端点 4：当前用户 `GET /api/auth/users/me`
```
请求: Authorization: Bearer <accessToken>
成功(200): {"code":0,"data":{id,username,nickname,avatar,roles:[],permissions:[]},"msg":"ok"}
失败(401): ProblemDetail,detail:"未授权"
```

数据流：
```
handler.Me(c)
  → 从 c 取 userID（AuthMiddleware 注入）
  → service.GetProfile(ctx, userID)
      → repo.FindByIDWithRoles(userID)  // 预加载 Roles
      → 收集 role.Code → roles[]
      → 汇总所有 role 的 permission.Code → permissionSet
      → if permissionSet 含 "*" → permissions = ["*"]   // 超管短路
      → return UserProfile{...}
  → response.Success(c, profile)
```

## 4. 认证中间件

`internal/middleware/auth.go`：`AuthRequired(jwtMgr *jwt.Manager)`
- 从 Authorization header 取 Bearer token（缺失/格式错 → apperr.Unauthorized）
- jwt.Parse 失败/过期 → apperr.Unauthorized
- claims.Type != TypeAccess → apperr.Unauthorized（refresh token 不能用于访问）
- c.Set("userID", claims.UserID)、c.Set("username", claims.Username)
- c.Next()

## 5. 数据结构

### AuthResult（对齐 mock AuthResultData / 前端 AuthResult）
```go
type AuthResult struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
    ExpiresIn    int    `json:"expiresIn"`  // 秒
}
```

### UserProfile（对齐 mock SafeUser / 前端 UserProfile）
```go
type UserProfile struct {
    ID          uint     `json:"id"`
    Username    string   `json:"username"`
    Nickname    string   `json:"nickname"`
    Avatar      string   `json:"avatar"`
    Roles       []string `json:"roles"`        // role code 数组
    Permissions []string `json:"permissions"` // permission code 数组，超管为 ["*"]
}
```

### 请求体（binding 校验）
```go
type LoginRequest struct {
    Username string `json:"username" binding:"required,min=3,max=64"`
    Password string `json:"password" binding:"required,min=6,max=72"`
}
type RefreshRequest struct {
    RefreshToken string `json:"refreshToken" binding:"required"`
}
```
bcrypt 72 字节上限由 Password max=72 在 binding 层兜底（M3 注册/改密码同理）。

## 6. 文件布局

新增：
```
internal/handler/auth.go           登录/刷新/登出/me 四端点
internal/service/auth.go           认证业务 + Seed
internal/repository/user.go        UserRepository 接口 + 实现
internal/middleware/auth.go        AuthRequired 中间件
internal/service/auth_test.go      service 单测 + mockUserRepo
```

修改：
```
internal/server/router.go           NewRouter(authHandler, jwtMgr) 注册 /api/auth
cmd/api/main.go                     组装依赖 + 调 Seed
internal/pkg/apperr/apperr.go       补 Write() 统一写出 + INTERNAL_ERROR
internal/pkg/response/problem.go    (可能)补 422 标题（已有）
```

依赖方向（单向）：
```
handler → service → repository(interface) → model
              ↘ jwt.Manager, hash, config
```

## 7. 依赖组装（main.go）

```go
jwtMgr := jwt.NewManager(cfg.JWT)
userRepo := repository.NewUserRepository(gdb)
authSvc := service.NewAuthService(userRepo, jwtMgr, cfg)
if err := authSvc.Seed(context.Background()); err != nil {
    log.L.Fatal("种子数据初始化失败", zap.Error(err))
}
authHandler := handler.NewAuthHandler(authSvc)
r := server.NewRouter(authHandler, jwtMgr)
```

router.go 改 `NewRouter()` 无参 → `NewRouter(authHandler *handler.AuthHandler, jwtMgr *jwt.Manager)`，显式注入：
```go
auth := api.Group("/auth")
{
    h := authHandler
    auth.POST("/sessions", h.Login)
    auth.DELETE("/sessions", h.Logout)
    auth.POST("/tokens/refresh", h.Refresh)
    auth.GET("/users/me", h.Me)  // 加 AuthRequired 中间件
}
```

## 8. 统一错误处理

apperr.go 新增：
```go
func Write(c *gin.Context, err error) {
    if e, ok := As(err); ok {
        response.Problem(c, e.Status, e.Title, e.Detail,
            response.WithCode(e.Code), response.WithErrors(e.Errors))
        return
    }
    log.S.Errorw("未预期错误", "err", err)
    response.Problem(c, 500, "", "服务暂时不可用", response.WithCode("INTERNAL_ERROR"))
}
```

handler 错误路径统一：
```go
if err := h.svc.Login(...); err != nil {
    apperr.Write(c, err)
    return
}
response.Success(c, data)
```

错误映射表：

| 场景 | HTTP | code | detail |
|---|---|---|---|
| 登录密码错/用户不存在 | 401 | UNAUTHORIZED | 用户名或密码错误 |
| 登录账户禁用 | 401 | UNAUTHORIZED | 账户已禁用 |
| 登录请求体校验失败 | 422 | VALIDATION_ERROR | 字段级 errors |
| 刷新 token 无效/过期/类型错 | 401 | UNAUTHORIZED | Invalid refresh token |
| me 未授权/token无效 | 401 | UNAUTHORIZED | 未授权 |
| 服务器异常 | 500 | INTERNAL_ERROR | 服务暂时不可用 |

安全原则：登录失败不区分"用户不存在"与"密码错"，统一文案防用户枚举（OWASP）。

## 9. 种子数据

FirstOrCreate 幂等 seed，顺序按依赖：Permission → Role → RolePermission → User → UserRole。

| 账户 | 密码 | 角色 | 权限 |
|---|---|---|---|
| admin | 123456 | super_admin | `*` |
| user | 123456 | user | `user:read` |

实体：
```
Permission: {code:"*",name:"通配权限",type:"api"}  {code:"user:read",name:"用户查看",type:"api"}
Role:       {code:"super_admin",name:"超级管理员",status:"active"}  {code:"user",name:"普通用户",status:"active"}
RolePermission: super_admin→*  user→user:read
User:       {username:"admin",password:hash("123456"),nickname:"Admin",status:"active"}
            {username:"user",password:hash("123456"),nickname:"User",status:"active"}
UserRole:   admin→super_admin  user→user
```

用 FirstOrCreate 按 unique key 查/建，已有数据不清不删。Seed 放 auth service.Seed（M2 不抽独立 seeder，YAGNI）。

## 10. 测试策略

`internal/service/auth_test.go`，mockUserRepo 实现 UserRepository 接口，内存 map 存储，不依赖真实 MySQL。

覆盖用例：
1. 登录成功 → token 对 + expiresIn=3600
2. 登录密码错 → apperr.Unauthorized
3. 登录用户不存在 → apperr.Unauthorized（同一文案防枚举）
4. 登录被禁用用户 → apperr.Unauthorized
5. 刷新有效 token → 新 token 对
6. 刷新用 access token → 失败（类型不符）
7. 刷新无效 token → 失败
8. GetProfile 普通用户 → roles/permissions 正确
9. GetProfile 超管（持 `*`）→ permissions=["*"]
10. Seed 幂等：调两次数据不重复

## 11. 边界与防御

1. **bcrypt 72 字节**：binding Password max=72 兜底，M2 seed 密码远小于 72 无问题。
2. **并发登录**：旧 token 仍有效（纯 JWT 权衡），接受，用户需主动登出。
3. **refresh 滥用**：TTL 7 天内旧 refresh 可用，接受自然过期缓解。
4. **时序攻击**：hash.Compare 常量时间比较，无风险。
5. **异步更新登录统计**：独立 context.Background()，不继承请求 ctx，失败仅 Warn 日志。

## 12. 验收标准

- [ ] `go build ./...` 通过
- [ ] 启动后日志含"种子数据就绪"
- [ ] `curl POST /api/auth/sessions` 用 admin/123456 返回 token 对
- [ ] 用错密码返回 401 ProblemDetail，文案"用户名或密码错误"
- [ ] refresh 端点用返回的 refreshToken 换新 token 对
- [ ] me 端点带 Bearer token 返回 UserProfile，admin 的 permissions=["*"]
- [ ] me 不带 token 返回 401
- [ ] `go test ./internal/service/...` 10 用例全过
- [ ] 用 user/123456 登录，me 返回 permissions=["user:read"]
