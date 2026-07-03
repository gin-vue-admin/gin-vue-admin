# M3.3 用户管理模块设计

> 状态：已批准（2026-07-03）
> 范围：user 域 7 端点（CRUD + 列表/导出/批量删），复用 M3.1/M3.2 基础设施。
> 前置：M3.2 已入库（提交 3d66168）。
> M3 拆分：M3.3 为最后一个子项，完成后 M3（RBAC+用户管理）整体交付。
> ⚠️ 契约说明：前端 UserInfo 已改为多角色（用户确认），本地 web 子模块因网络受限未同步至最新 main，本 spec 按多角色 `roles: string[]` 设计。前端契约确认后若有差异仅微调 DTO 字段名。

## 1. 目标与边界

### 目标
- 实现 user 域 7 端点，对齐前端 UserInfo 多角色契约。
- 用户 CRUD：创建（密码必填 bcrypt）、更新（密码可选）、删除（软删+解除关联）、列表（keyword/status/role 筛选）、导出、批量删。
- 复用 M3.1/M3.2 的 pagination、csvutil、middleware（AuthRequired + RequirePermission + InvalidateAll）、apperr、testutil、isDuplicateKey、hash。

### 非目标（YAGNI）
- M3.3 只做 user 域；菜单（M4）、部门/字典等后续。
- 不做密码重置独立端点（创建必填、更新可选覆盖）。
- 不做用户头像上传（avatar 字段存 URL，上传是文件服务）。
- 不做邮箱/手机验证。

## 2. 关键决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| role 映射 | 多角色 `roles: string[]`，API 与表都多对多 | 前端已改多角色；与后端 RBAC 多对多天然契合 |
| User→UserInfo | DTO 转换（handler/service 层） | model.User 不直接序列化（Password 不泄漏、Roles→code、LastLoginAt→字符串） |
| 密码 | 创建必填/更新可选，bcrypt | 与 M2 seed 一致；更新留空不改密码 |
| 用户删除 | 软删 + 解除 user_roles 关联（事务） | 与 M3.2 角色删除对称，数据一致 |
| 防自删/自禁 | 禁止（目标==自己→409） | 避免管理员锁死自己 |
| UserRepository | 扩展现有接口（M2 建） | 单一用户仓储；M2 AuthService 仅用原4方法不受影响 |
| role 校验 | 严格校验未知 code→404 | 复用 M3.2 FindRoleIDsByCodes 模式，避免静默丢失 |
| 时间字段 | createTime/lastLoginTime（ISO8601） | 对齐前端 UserInfo |
| 导出 | ApiResult 包裹 csv，msg="导出成功" | 对齐 mock，与 M3.1/M3.2 一致 |

## 3. 数据模型

model.User **保持不变**（已有全部字段：Username/Password/Nickname/RealName/Email/Phone/Avatar/Status/LastLoginAt/LoginCount/DeletedAt/Roles）。无需改 model。

### UserInfo 响应 DTO（对齐前端契约）
```go
type UserInfo struct {
	ID            uint     `json:"id"`
	Username      string   `json:"username"`
	RealName      string   `json:"realName"`
	Email         string   `json:"email"`
	Phone         string   `json:"phone"`
	Roles         []string `json:"roles"`         // 角色 code 数组（多角色）
	Status        string   `json:"status"`
	Avatar        string   `json:"avatar"`
	CreateTime    string   `json:"createTime"`    // CreatedAt→ISO8601
	LastLoginTime string   `json:"lastLoginTime"` // LastLoginAt→ISO8601，空则""
	LoginCount    int      `json:"loginCount"`
}
```
转换规则：
- Roles：预加载 user.Roles，取 []Role.Code
- CreateTime：CreatedAt.Format(time.RFC3339)
- LastLoginTime：LastLoginAt.Valid ? Format(RFC3339) : ""

### 请求 DTO
```go
type UserCreateReq struct {
	Username string   `json:"username" binding:"required,min=3,max=64"`
	RealName string   `json:"realName" binding:"max=64"`
	Email    string   `json:"email" binding:"omitempty,email,max=128"`
	Phone    string   `json:"phone" binding:"max=32"`
	Roles    []string `json:"roles" binding:"required,min=1"`     // 角色 code，至少1个
	Status   string   `json:"status" binding:"required,oneof=active inactive"`
	Password string   `json:"password" binding:"required,min=6,max=72"` // 创建必填
}
type UserUpdateReq struct {
	RealName *string  `json:"realName"`           // 指针区分未传与空值
	Email    *string  `json:"email"`
	Phone    *string  `json:"phone"`
	Roles    []string `json:"roles"`              // 传了才改
	Status   *string  `json:"status"`
	Password *string  `json:"password"`           // 传了才改密码
}
```
> UpdateReq 用指针区分"未传字段（不改）"与"传空值"。Roles 是切片，nil=不改，[]={} 不合法（binding 时排除空）。

## 4. 端点契约（7 端点）

| 方法 | 路径 | 权限码 | 请求 | 响应 data |
|---|---|---|---|---|
| GET | `/api/user` | user:list | keyword/role/status/page/size | `{records,total,current,size}` |
| GET | `/api/user/export` | user:list | 同上 query | csv（ApiResult，msg="导出成功"） |
| GET | `/api/user/:id` | user:list | — | UserInfo |
| POST | `/api/user` | user:create | UserCreateReq | UserInfo（msg="创建成功"） |
| PUT | `/api/user/:id` | user:edit | UserUpdateReq | UserInfo（msg="更新成功"） |
| DELETE | `/api/user/:id` | user:delete | — | true（msg="删除成功"） |
| DELETE | `/api/user` | user:delete | `{ids:[]}` | true（msg="删除成功"） |

权限中间件挂载：AuthRequired + RequirePermission(user:list/create/edit/delete)。超管 `*` 短路。路由顺序：/export 在 /:id 前注册。

## 5. 关键业务逻辑

### 创建（POST）
```
→ username 重复 → 409 "用户名已存在"
→ Roles code 严格校验：repo.FindRoleIDsByCodes(codes)，未知 code → 404 "角色 xxx 不存在"
→ Password bcrypt 哈希（hash.Hash）
→ repo.Create(user) + repo.ReplaceRoles(userID, roleIDs)
→ middleware.InvalidateAll()
→ return UserInfo
```

### 更新（PUT /:id）
```
→ 取当前用户（token userID）与目标对比
   - status 改为 inactive 且目标==自己 → 409 "不能禁用自己"
→ 更新字段（指针非 nil 才改）
→ Password 传了（非空）→ bcrypt 哈希更新
→ Roles 传了（非 nil）→ 严格校验 + ReplaceRoles + InvalidateAll
→ repo.Update(user)
→ return UserInfo
```

### 删除（DELETE /:id）
```
→ 目标==自己 → 409 "不能删除自己"
→ repo.Delete（事务：软删 user + 清 user_roles）
→ middleware.InvalidateAll()
→ return true
```
批量删除：循环校验自删（任一 id==自己→409），再事务批量。

### 列表（GET）
```
→ repo.List(q, roleCode)：分页 + keyword/status/role 筛选
   - keyword: username/realName/email/phone 模糊
   - status 精确
   - role: JOIN user_roles + roles WHERE role.code = ?
→ 每条 user 预加载 Roles → 转 UserInfo（roles code 数组）
→ return Result[UserInfo]
```
> 列表预加载 Roles 用 Preload，避免 N+1。

## 6. UserRepository 扩展

扩展现有接口（M2 建，加方法）：
```go
type UserRepository interface {
	// M2 既有
	FindByUsername(ctx, username string) (*model.User, error)
	FindByID(ctx, id uint) (*model.User, error)
	FindByIDWithRoles(ctx, id uint) (*model.User, error)  // 预加载 Roles
	UpdateLoginStats(ctx, id uint) error
	// M3.3 新增
	List(ctx, q pagination.Query, roleCode string) ([]model.User, int64, error)
	Create(ctx, *model.User) error
	Update(ctx, *model.User) error
	Delete(ctx, id uint) error              // 事务软删+清 user_roles
	BatchDelete(ctx, ids []uint) error
	ReplaceRoles(ctx, userID uint, roleIDs []uint) error
	FindRoleIDsByCodes(ctx, codes []string) (map[string]uint, error)
}
```
> M2 的 AuthService 依赖 UserRepository 接口，扩展后 AuthService 只用原4方法，编译不受影响（接口扩展不需改实现的使用方，只需 userRepository struct 实现新方法）。

List 的 role 筛选用 JOIN：
```go
if roleCode != "" {
	db = db.Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("r.code = ?", roleCode)
}
```

## 7. 错误处理

| 场景 | HTTP | code | detail |
|---|---|---|---|
| 用户不存在 | 404 | NOT_FOUND | 用户不存在 |
| username 重复（含软删同名） | 409 | CONFLICT | 用户名已存在 |
| 角色 code 未知 | 404 | NOT_FOUND | 角色 xxx 不存在 |
| 删除/禁用自己 | 409 | CONFLICT | 不能删除自己 / 不能禁用自己 |
| 校验失败 | 422 | VALIDATION_ERROR | 字段级 |
| 无权限 | 403 | FORBIDDEN | 禁止访问 |
| DB 故障 | 500 | INTERNAL_ERROR | 服务暂时不可用（透传） |

isDuplicateKey 复用 service 包（M3.1）。

## 8. 文件结构

新增：
```
internal/service/user.go           UserService（CRUD+list+export+角色关联）+ UserInfo/UserCreateReq/UserUpdateReq
internal/service/user_test.go       单测
internal/handler/user.go           UserHandler（7端点）
```

修改：
```
internal/repository/user.go        UserRepository 接口扩展 + 新方法实现
internal/server/router.go          注册 /api/user 路由组
cmd/api/main.go                    组装 User 依赖
```

依赖方向（沿用）：
```
handler → service → repository(接口) → model
              ↘ pagination, csvutil, hash, apperr, response
              ↘ middleware.InvalidateAll
```

## 9. 测试策略

`service/user_test.go`，SQLite 隔离（testutil.NewTestDB），复用模式。

### UserService 用例
1. 创建成功（含角色关联、密码哈希）
2. 创建 username 重复 → 409
3. 创建未知角色 code → 404
4. 详情不存在 → 404
5. 列表分页 + keyword/status/role 筛选
6. 更新（改密码、改角色）
7. 更新禁用自己 → 409（需传入 operatorID）
8. 软删后查不到 + user_roles 清除
9. 删除自己 → 409
10. 批量删除（含自删拦截）
11. 导出 CSV（含表头 + 转义）
12. UserInfo 转换（roles code 数组、lastLoginTime 空/非空）

> 防自删/自禁需 service 方法接收 operatorID 参数（handler 从 token 注入）。

## 10. 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./...` 全过（M2+M3.1+M3.2+M3.3 回归）
- [ ] admin GET /api/user 返回分页列表，每条含 roles 数组
- [ ] user（无 user:list）GET /api/user → 403
- [ ] admin POST 创建用户（含密码+角色）→ 成功；username 重复 → 409
- [ ] admin 创建未知角色 → 404
- [ ] admin PUT 改密码 → 登录用新密码成功
- [ ] admin DELETE 自己 → 409；删除他人 → 软删
- [ ] admin 禁用自己 → 409
- [ ] admin GET /api/user/export 返回 csv
- [ ] admin GET /api/user?role=user 按 role 筛选
- [ ] 启动日志含"种子数据就绪"

## 11. 与契约对齐核查

- id：后端 uint，前端兼容（M2 确认）
- roles：多角色 code 数组（前端已改多角色，本 spec 据此设计）
- createTime/lastLoginTime：ISO8601 字符串（LastLoginAt 空→""）
- password：永不序列化（model json:"-"），创建必填/更新可选
- 响应 {records,total,current,size}、导出 ApiResult 包裹 csv、删除返 true——与 M3.1/M3.2 一致

## 12. 风险与待确认

1. **前端契约**：本地 web 未同步最新 main，多角色字段名假设为 `roles: string[]`。前端确认后若字段名不同（如 `roleCodes`）仅改 DTO tag，逻辑不变。
2. **UpdateReq 指针语义**：用指针区分未传/空值，前端需配合（PUT 只传要改的字段）。若前端总是全量传，指针语义仍兼容（全传=全改）。
3. **列表 role 筛选 JOIN**：SQLite/MySQL 标准 JOIN 兼容，Task 落实时测试验证。
