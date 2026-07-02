# M3.1 权限模块 + 通用基础设施设计

> 状态：已批准（2026-07-02）
> 范围：permission 域 7 端点 + 权限中间件 + 通用分页/CSV 导出基础设施。为 M3.2（角色）/M3.3（用户）复用立模板。
> 前置：M0 骨架、M1 基础设施、M2 认证已完成并入库（提交 b744198）。
> M3 拆分：M3 拆 3 子项，本 spec 为第一个（M3.1）。M3.2 角色、M3.3 用户各自独立 spec。

## 1. 目标与边界

### 目标
- 实现 permission 域 7 端点，严格对齐前端 `web/src/mock/handlers/permission.ts` 与 `web/src/modules/system/permission/api.ts` 契约。
- 建权限中间件（`*` 通配短路 + hasAny/hasAll 语义），内存缓存 + TTL，供所有受保护路由复用。
- 建通用分页（泛型 Paginate）、CSV 导出工具，M3.2/M3.3 复用。
- 扩展 Permission model 对齐前端契约（加 module/description/status/软删除）。

### 非目标（YAGNI）
- M3.1 只做 permission CRUD + 中间件 + 基础设施；role/user CRUD 是 M3.2/M3.3。
- 权限缓存不做分布式（仅进程内 map+RWMutex+TTL）。
- 软删除唯一索引冲突不做自动改 code，接受 conflict 提示。
- 不做精细缓存失效，permission CRUD 后 InvalidateAll 清全量。

## 2. 关键决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| M3 拆分 | 3 子项，先 M3.1 | 23 端点过大，独立 spec/plan 风险分散 |
| model 对齐 | 扩展 Permission 加 module/description/status/DeletedAt | 对齐 mock 契约 |
| 删除 | 软删除（DeletedAt） | 数据可恢复，后台惯例 |
| 软删唯一索引 | 接受 conflict（409） | 简单，符合"code 已存在"语义 |
| 权限中间件 | 取权限码 + 内存缓存 TTL 5min | 平衡性能与实时性 |
| 超管语义 | permSet 含 `*` 短路放行 | 与 M2 一致，判断权限码非角色名 |
| 分页 | 通用 Query/Result + 泛型 Paginate | 三域复用，DRY |
| CSV 导出 | ApiResult 包裹 csv（msg="导出成功"） | 对齐 mock ok(csv)，前端拦截器解包 |
| 时间字段 JSON | createTime/updateTime | 对齐前端 api.ts，改 base.Model tag |

## 3. 数据模型变更

### base.Model 时间 JSON tag（影响所有实体）
```go
type Model struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createTime"`
	UpdatedAt time.Time `json:"updateTime"`
}
```
影响面：所有继承 Model 的实体（User/Role/Permission/Menu）时间序列化字段名变化。M2 的 UserProfile 不含时间字段，无回归。M2 me 端点需回归验证（不报错即可）。

### Permission 扩展
```go
type Permission struct {
	Model
	Code        string         `gorm:"uniqueIndex;size:128;not null" json:"code"`
	Name        string         `gorm:"size:64;not null" json:"name"`
	Type        string         `gorm:"size:16" json:"type"`             // menu|button|api，后端用
	Module      string         `gorm:"size:32;index" json:"module"`    // 新增：system/user/role/permission/dict/config
	Description string         `gorm:"size:255" json:"description"`     // 新增
	Status      string         `gorm:"size:16;default:active" json:"status"` // 新增：active|inactive
	ParentID    uint           `gorm:"index;default:0" json:"parentId"`
	Sort        int            `gorm:"default:0" json:"sort"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-""`                 // 新增：软删除
}
```
AutoMigrate 自动加列。GORM 查询自动过滤已删除。

## 4. 通用基础设施

### 分页 `internal/pkg/pagination/pagination.go`
```go
package pagination

// Query 通用列表查询参数（对齐前端 PermissionSearchRequest 公共字段）。
type Query struct {
	Keyword string `form:"keyword"`
	Status  string `form:"status"`
	Page    int    `form:"page,default=1"`
	Size    int    `form:"size,default=10"`
}

// Normalize 补默认值：page<1→1，size<1→10，size>100→100。
func (q *Query) Normalize()

// Result 分页响应（对齐前端 {records,total,current,size}）。
type Result[T any] struct {
	Records []T   `json:"records"`
	Total   int64 `json:"total"`
	Current int   `json:"current"`
	Size    int   `json:"size"`
}

// Paginate 在 gorm 查询上叠加分页 + count，返回 Result。build 回调叠加域特有 Where（如 module）。
func Paginate[T any](db *gorm.DB, q Query, build func(*gorm.DB) *gorm.DB) (Result[T], error)
```
Go 1.25 泛型支持，三域复用。各域 keyword/module 等筛选在 build 回调叠加。

### CSV `internal/pkg/csvutil/csvutil.go`
```go
package csvutil

// Build 将对象数组转为 CSV 文本（含表头），字段含逗号/引号/换行时双引号包裹转义。
// 对齐前端 mock toCsv 语义。
func Build(rows []map[string]any, headers []string) string
```
handler：`csv := csvutil.Build(rows, headers)` → `response.Success(c, csv, "导出成功")`。

## 5. 权限中间件 `internal/middleware/permission.go`

### PermissionReader 最小接口
```go
// PermissionReader 中间件仅需"按 userID 查权限码集合"，最小接口避免依赖完整 repo。
type PermissionReader interface {
	GetUserPermissionCodes(ctx context.Context, userID uint) ([]string, error)
}
```

### 缓存与中间件
```go
// 内存缓存：userID → (permSet, expireAt)，RWMutex 保护，TTL 5 分钟。
// RequirePermission(codes...) 需任一权限码（hasAny）。
// RequireAllPermissions(codes...) 需全部（hasAll）。
// 超管语义：permSet 含 "*" 短路放行。

func RequirePermission(repo PermissionReader, codes ...string) gin.HandlerFunc
func RequireAllPermissions(repo PermissionReader, codes ...string) gin.HandlerFunc

// InvalidateAll 清全量缓存。permission CRUD 后调用。
func InvalidateAll()
```

### 数据流
```
RequirePermission(["permission:list"])
  → 从 c 取 userID（AuthRequired 已注入）
  → 查缓存(命中) / repo.GetUserPermissionCodes（未命中，写缓存 TTL 5min）
  → permSet 含 "*" 或 codes 任一命中 → c.Next()
  → 否则 apperr.Forbidden
```
缓存失效：permission CRUD 后 InvalidateAll 清全量（后台操作低频，可接受）。

## 6. 端点契约（permission 域）

| 方法 | 路径 | 请求 | 成功响应 data |
|---|---|---|---|
| GET | `/api/permission` | query: keyword/module/status/page/size/all | all=true→`PermissionInfo[]`；否则`{records,total,current,size}` |
| GET | `/api/permission/export` | — | csv 字符串（ApiResult 包裹，msg="导出成功"） |
| GET | `/api/permission/:id` | — | PermissionInfo |
| POST | `/api/permission` | `{name,code,description?,module,status}` | PermissionInfo（msg="创建成功"） |
| PUT | `/api/permission/:id` | 同 POST（partial） | PermissionInfo（msg="更新成功"） |
| DELETE | `/api/permission/:id` | — | `true`（msg="删除成功"） |
| DELETE | `/api/permission` | `{ids:[]}` | `true`（msg="删除成功"） |

PermissionInfo 字段对齐前端 api.ts：`{id,name,code,description,module,status,createTime,updateTime}`。

### 权限中间件挂载
```
/api/permission      GET    → AuthRequired + RequirePermission("permission:list")
/api/permission/export GET   → AuthRequired + RequirePermission("permission:list")
/api/permission/:id   GET    → AuthRequired + RequirePermission("permission:list")
/api/permission       POST   → AuthRequired + RequirePermission("permission:create")
/api/permission/:id   PUT    → AuthRequired + RequirePermission("permission:edit")
/api/permission/:id   DELETE → AuthRequired + RequirePermission("permission:delete")
/api/permission       DELETE → AuthRequired + RequirePermission("permission:delete")
```
注意路由注册顺序：`/export` 与 `/:id` 都在 GET，Gin 需 `export` 先注册或用明确路径，避免 `:id` 捕获 "export"。Gin 树形路由会精确匹配 `export` 优先，但需确认。

## 7. 种子数据

启动 seed 补 20 个权限码定义（对齐 mock permissionCodes），分模块：
- system: system:setting/system:log/system:operation
- user: user:list/user:create/user:edit/user:delete
- role: role:list/role:create/role:edit/role:delete/role:permission
- permission: permission:list/permission:create/permission:edit/permission:delete
- dict: dict:list/dict:create/dict:edit/dict:delete
- config: config:system/config:parameter/config:email

M2 的 Seed 已配 super_admin→`*`。M3.1 补：种入这 20 个 Permission 定义（不预分配给 user 角色，M3.2 处理）。

## 8. 错误处理

| 场景 | HTTP | code | detail |
|---|---|---|---|
| 未登录 | 401 | UNAUTHORIZED | 未授权（AuthRequired） |
| 无权限 | 403 | FORBIDDEN | 禁止访问（RequirePermission） |
| 权限不存在 | 404 | NOT_FOUND | 权限不存在 |
| code 唯一冲突（含软删同名） | 409 | CONFLICT | 权限编码已存在 |
| 请求体校验失败 | 422 | VALIDATION_ERROR | 字段级 errors |
| DB 故障 | 500 | INTERNAL_ERROR | 服务暂时不可用（透传 apperr.Write） |

软删 conflict：service.Create/Update 捕获 gorm 唯一约束错误 → apperr.Conflict("权限编码已存在")。

## 9. 文件结构

新增：
```
internal/pkg/pagination/pagination.go    通用 Query/Result/Paginate
internal/pkg/csvutil/csvutil.go           Build
internal/middleware/permission.go        RequirePermission/RequireAll/InvalidateAll + 缓存
internal/repository/permission.go        PermissionRepository 接口 + gorm 实现
internal/service/permission.go           PermissionService（CRUD+list+export）
internal/service/permission_test.go       service 单测
internal/handler/permission.go           PermissionHandler（7端点）
internal/testutil/db.go                  复用 newTestDB（从 M2 service/auth_test.go 提取）
```

修改：
```
internal/model/base.go                    Model 时间 JSON tag → createTime/updateTime
internal/model/rbac.go                    Permission 加字段 + 软删除
internal/service/auth.go                  Seed 补 20 权限定义
internal/server/router.go                注册 /api/permission 路由组
cmd/api/main.go                          组装 Permission 依赖 + 中间件
```

依赖方向（沿用 M2）：
```
handler → service → repository(接口) → model
              ↘ pagination, csvutil, apperr, response
middleware/permission → PermissionReader(接口) + 缓存
```

### PermissionRepository 接口
```go
type PermissionRepository interface {
	List(ctx, q pagination.Query, module string) ([]model.Permission, int64, error)
	ListAll(ctx, q pagination.Query, module string) ([]model.Permission, error)
	FindByID(ctx, id uint) (*model.Permission, error)
	Create(ctx, *model.Permission) error
	Update(ctx, *model.Permission) error
	Delete(ctx, id uint) error              // 软删
	BatchDelete(ctx, ids []uint) error       // 批量软删
	GetUserPermissionCodes(ctx, userID uint) ([]string, error)  // 中间件用
}
```
注：`GetUserPermissionCodes` 查 user→roles→permissions 跨表，M3.1 实现，M3.2 复用。

## 10. 测试策略

抽 `internal/testutil/db.go`：从 M2 auth_test.go 提取 newTestDB（SQLite 临时文件库 + AutoMigrate），M3 复用。

### PermissionService 用例（service/permission_test.go）
1. 列表分页正确（total/current/size）
2. keyword 筛选（name/code/description 模糊）
3. module 筛选
4. status 筛选
5. all=true 返回全量不分页
6. 详情不存在 → 404
7. 创建成功
8. 创建 code 重复 → 409
9. 更新成功
10. 软删除后查询查不到
11. 批量删除
12. export 生成 CSV（表头 + 转义）

### 中间件用例（middleware/permission_test.go）
1. 有权限放行
2. 无权限 → 403
3. 超管 `*` 短路放行
4. 缓存命中（不重复查 DB）
5. InvalidateAll 后重查

## 11. 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./...` 全过（含 M2 回归 + M3.1 新测）
- [ ] M2 me 端点回归正常（base.Model tag 改动无破坏）
- [ ] admin 登录后 GET /api/permission 返回分页列表
- [ ] user（无 permission:list）GET /api/permission → 403
- [ ] admin POST 创建权限 → 成功；重复 code → 409
- [ ] admin DELETE 软删后 GET 列表查不到
- [ ] admin GET /api/permission?all=true 返回全量
- [ ] admin GET /api/permission/export 返回 csv（ApiResult 包裹）
- [ ] admin 批量删除
- [ ] 启动日志含"种子数据就绪"且库内有 20 权限

## 12. 与 mock 契约对齐核查

- id：后端 uint，前端 string/number 兼容（M2 已确认）
- 字段名：createTime/updateTime（base.Model tag 已对齐）
- 响应结构：{records,total,current,size}（pagination.Result 对齐）
- 导出：ApiResult 包裹 csv，msg="导出成功"（对齐 ok(csv)）
- 删除返回 true（对齐 mock ok(true)）
- all=true 跳过分页（对齐 mock）
