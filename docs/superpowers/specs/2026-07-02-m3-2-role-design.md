# M3.2 角色模块设计

> 状态：已批准（2026-07-02）
> 范围：role 域 9 端点（CRUD + 列表/导出 + 权限分配子资源），复用 M3.1 通用基础设施。
> 前置：M3.1 已入库（提交 6b22265），提供 pagination/csvutil/middleware.RequirePermission/apperr.Validation/testutil。
> M3 拆分：M3.2 为第二个子项。M3.3 用户管理后续。

## 1. 目标与边界

### 目标
- 实现 role 域 9 端点，严格对齐前端 `web/src/mock/handlers/role.ts` 与 `web/src/modules/system/role/api.ts` 契约。
- 权限分配子资源：GET/PUT `/api/role/:id/permissions`，API 用权限 code、表存 permission id（code↔id 转换）。
- 复用 M3.1 的 pagination（泛型 Paginate）、csvutil（导出）、middleware（RequirePermission + InvalidateAll）、apperr（Validation/Conflict/NotFound）。

### 非目标（YAGNI）
- M3.2 只做 role；user CRUD 是 M3.3。
- 权限分配 PUT 用全量替换（Replace），不做增量 add/remove。
- 缓存失效仍用 InvalidateAll 全量（M3.1 既定），不引入精细失效。

## 2. 关键决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| Role 字段 | 加 Description，保留 Remark/Sort | 对齐 mock RoleInfo.description；Sort 后端排序用，Remark 备注用 |
| 权限分配映射 | API 用 code、表存 id | 对齐前端 mock（string[] 是 code） |
| 未知 code | 严格校验报错（404） | 避免静默丢失分配 |
| 角色删除 | 软删 + 解除 user_roles/role_permissions 关联 | 数据一致，已分配用户失去该角色 |
| 删除事务 | GORM 事务包裹软删 + Association Clear | 保证一致 |
| 权限分配/删除后 | InvalidateAll | 用户权限缓存失效 |
| RoleRepository 接口 | 抽接口 | service 含业务逻辑（code↔id 转换、删除解除关联）值得单测 |
| 导出 | ApiResult 包裹 csv，msg="导出成功" | 对齐 mock，与 M3.1 一致 |
| 时间字段 | createTime/updateTime（base.Model 已改） | M3.1 已对齐 |

## 3. 数据模型变更

Role 加 Description（对齐 mock RoleInfo）：
```go
type Role struct {
	Model
	Code        string         `gorm:"uniqueIndex;size:64;not null" json:"code"`
	Name        string         `gorm:"size:64;not null" json:"name"`
	Description string         `gorm:"size:255" json:"description"`   // 新增
	Status      string         `gorm:"size:16;default:active" json:"status"`
	Sort        int            `gorm:"default:0" json:"sort"`
	Remark      string         `gorm:"size:255" json:"remark"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Permissions []Permission   `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}
```
> RoleInfo 无 sort，但 model 保留 Sort（后端排序）。前端收到 sort 附加字段，与 M3.1 Permission 的 type/parentId 同类，前端兼容。

AutoMigrate 自动加 Description 列。

## 4. 端点契约（9 端点，对齐 mock）

| 方法 | 路径 | 请求 | 响应 data |
|---|---|---|---|
| GET | `/api/role` | keyword/status/page/size | `{records,total,current,size}` |
| GET | `/api/role/export` | — | csv（ApiResult，msg="导出成功"） |
| GET | `/api/role/:id` | — | RoleInfo |
| POST | `/api/role` | `{name,code,description?,status}` | RoleInfo（msg="创建成功"） |
| PUT | `/api/role/:id` | 同 POST（partial） | RoleInfo（msg="更新成功"） |
| DELETE | `/api/role/:id` | — | true（msg="删除成功"） |
| DELETE | `/api/role` | `{ids:[]}` | true（msg="删除成功"） |
| GET | `/api/role/:id/permissions` | — | `string[]`（权限 code 数组） |
| PUT | `/api/role/:id/permissions` | `{permissions:string[]}` | true（msg="设置成功"） |

RoleInfo 字段对齐前端 api.ts：`{id,name,code,description,status,createTime,updateTime}`。

### 权限中间件挂载
```
/api/role              GET    → AuthRequired + RequirePermission("role:list")
/api/role/export       GET    → AuthRequired + RequirePermission("role:list")
/api/role/:id          GET    → AuthRequired + RequirePermission("role:list")
/api/role             POST   → AuthRequired + RequirePermission("role:create")
/api/role/:id         PUT    → AuthRequired + RequirePermission("role:edit")
/api/role/:id         DELETE → AuthRequired + RequirePermission("role:delete")
/api/role             DELETE → AuthRequired + RequirePermission("role:delete")
/api/role/:id/permissions GET  → AuthRequired + RequirePermission("role:permission")
/api/role/:id/permissions PUT  → AuthRequired + RequirePermission("role:permission")
```
路由注册顺序：`/export` 与 `/:id/permissions` 在 `/:id` 前注册（Gin 静态优先）。注意 `/:id/permissions` 是子路径，不与 `/:id` 冲突。

## 5. 权限分配子资源（核心）

### GET `/api/role/:id/permissions`
```
→ 校验角色存在（不存在→404）
→ repo.GetRolePermissionCodes(roleID)  // 查 role→permissions 的 code
→ response.Success(c, codes)
```

### PUT `/api/role/:id/permissions`
```
→ 校验角色存在（不存在→404）
→ 严格校验：所有 code 必须存在
   repo.FindPermissionIDsByCodes(codes) → 批量查，未知 code→404 "权限 xxx 不存在"
→ repo.ReplaceRolePermissions(roleID, permissionIDs)  // 事务内 Association Replace
→ middleware.InvalidateAll()  // 用户权限缓存失效
→ response.Success(c, true, "设置成功")
```

### RoleRepository 新增方法
```go
GetRolePermissionCodes(ctx, roleID uint) ([]string, error)
ReplaceRolePermissions(ctx, roleID uint, permissionIDs []uint) error
FindPermissionIDsByCodes(ctx, codes []string) (map[string]uint, error)  // code→id，供校验+转换
```
FindPermissionIDsByCodes 一次查回所有 code 对应 id，service 据此校验完整性 + 转换。

## 6. 删除约束

角色删除（单/批量）：事务内软删 + 解除关联。
```
db.Transaction(func(tx) error {
    tx.Delete(&role)                              // 软删
    tx.Model(&role).Association("Permissions").Clear()  // 解除 role_permissions
    tx.Model(&role).Association("Users").Clear()        // 解除 user_roles
    return nil
})
→ middleware.InvalidateAll()  // 用户失去角色，缓存失效
```
批量删除循环上述（或事务内批量）。Association Clear 解除关联但不删关联实体记录。

> User.Roles 反向关联：model.User 有 `Roles []Role gorm:"many2many:user_roles"`，`Association("Users")` 在 Role 上需反向。实际用 raw 清 user_roles where role_id in (...) 更直接。Plan 落实时确认。

## 7. 错误处理

| 场景 | HTTP | code | detail |
|---|---|---|---|
| 角色不存在 | 404 | NOT_FOUND | 角色不存在 |
| code 重复（含软删同名） | 409 | CONFLICT | 角色编码已存在 |
| 权限分配含未知 code | 404 | NOT_FOUND | 权限 xxx 不存在 |
| 校验失败 | 422 | VALIDATION_ERROR | 字段级 |
| 无权限 | 403 | FORBIDDEN | 禁止访问 |
| DB 故障 | 500 | INTERNAL_ERROR | 服务暂时不可用（透传） |

isDuplicateKey 复用 M3.1 的双方言判断（gorm.ErrDuplicatedKey + "Duplicate entry" + "UNIQUE constraint failed"）。

## 8. 文件结构

新增：
```
internal/repository/role.go        RoleRepository 接口 + gorm 实现
internal/service/role.go           RoleService（CRUD+list+export+权限分配）
internal/service/role_test.go       单测
internal/handler/role.go           RoleHandler（9端点）
```

修改：
```
internal/model/rbac.go            Role 加 Description
internal/service/auth.go          Seed：super_admin/user 角色补 Description（可选，保持一致）
internal/server/router.go         注册 /api/role 路由组
cmd/api/main.go                   组装 Role 依赖
```

依赖方向（沿用 M2/M3.1）：
```
handler → service → repository(接口) → model
              ↘ pagination, csvutil, apperr, response
              ↘ middleware.InvalidateAll
```

### RoleRepository 接口
```go
type RoleRepository interface {
	List(ctx, q pagination.Query) ([]model.Role, int64, error)
	FindByID(ctx, id uint) (*model.Role, error)
	Create(ctx, *model.Role) error
	Update(ctx, *model.Role) error
	Delete(ctx, id uint) error              // 事务内软删+解除关联
	BatchDelete(ctx, ids []uint) error
	GetRolePermissionCodes(ctx, roleID uint) ([]string, error)
	ReplaceRolePermissions(ctx, roleID uint, permissionIDs []uint) error
	FindPermissionIDsByCodes(ctx, codes []string) (map[string]uint, error)
}
```

## 9. 种子数据

M2 的 Seed 已建 super_admin/user 角色（无 Description）。M3.2 补 Description：
- super_admin: Description="超级管理员，拥有所有权限"
- user: Description="普通用户"

可选（保持一致建议补）。FirstOrCreate 幂等，已存在角色补 Description 用 Update 或 FirstOrCreate 后赋值。

## 10. 测试策略

`service/role_test.go`，SQLite 隔离（testutil.NewTestDB），复用 M3.1 模式。

### RoleService 用例
1. 列表分页 + keyword/status 筛选
2. 详情不存在 → 404
3. 创建成功
4. 创建 code 重复 → 409
5. 更新成功
6. 软删后查不到
7. 删除后解除关联（角色删后，关联的 user 不再有该角色 / role_permissions 清空）
8. 批量删除
9. 权限分配 GET 返回 code 数组
10. 权限分配 PUT（全量替换）
11. 权限分配 PUT 含未知 code → 404
12. 权限分配 PUT 后用户权限码变化（InvalidateAll 生效，需中间件或重新查）

### 删除解除关联验证
建 role→permission、user→role 关联，删除 role 后验证 user_roles 和 role_permissions 记录清除。

## 11. 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./...` 全过（M2+M3.1+M3.2 回归）
- [ ] admin GET /api/role 返回分页列表
- [ ] user（无 role:list）GET /api/role → 403
- [ ] admin POST 创建角色 → 成功；重复 code → 409
- [ ] admin DELETE 角色 → 软删，关联解除
- [ ] admin GET /api/role/:id/permissions 返回 code 数组
- [ ] admin PUT /api/role/:id/permissions 设置权限 → 成功；含未知 code → 404
- [ ] admin 权限分配后，受影响用户权限立即生效（InvalidateAll）
- [ ] admin GET /api/role/export 返回 csv
- [ ] 启动日志含"种子数据就绪"，角色有 Description

## 12. 与 mock 契约对齐核查

- id：后端 uint，前端 string 兼容（M2 已确认）
- 字段名：createTime/updateTime（base.Model 已对齐）
- RoleInfo：name/code/description/status（model 加 Description 对齐）
- 响应结构：{records,total,current,size}（pagination.Result）
- 导出：ApiResult 包裹 csv，msg="导出成功"
- 删除返回 true
- 权限分配：GET 返 string[]（code）、PUT 接 {permissions:string[]}（code）
