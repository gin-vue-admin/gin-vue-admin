# 新增业务模块指南

本文档说明在 gva 后端基座上新增一个"标准 CRUD 业务模块"的标准化流程。基座已提供 **泛型 Repository 基类 + 审计字段 + 软删除 + 统一响应**，新增模块只需写"差异部分"。

最佳参考：`internal/{model,repository,service,handler}/crud.go`（M6 范例，最简最全）。复杂模板参考 `user`/`role` 模块（含关联、权限缓存失效、唯一约束）。

---

## 基座能力（直接复用，无需重写）

| 能力 | 位置 | 说明 |
|------|------|------|
| 泛型 Repository | `internal/repository/generic.go` | `GenericRepository[T]` 提供 FindByID/List/ListAll/Create/Update/Delete/BatchDelete + 分页 |
| 审计字段 | `internal/model/base.go` 的 `Model` | `CreatedBy/UpdatedBy/DeletedBy`，由 `internal/pkg/audit` GORM 回调从请求上下文自动注入 |
| 软删除 | `Model.DeletedAt` | gorm.DeletedAt，查询自动过滤 |
| 统一分页 | `internal/pkg/pagination` | `Query{Keyword,Status,Page,Size}` + `Normalize()` + `Result[T]{Records,Total,Current,Size}` + 泛型 `Paginate[T]` |
| 统一响应 | `internal/pkg/response` | 成功 `response.Success(c,data,msg)`；失败 `apperr.Write(c,err)` |
| 错误模型 | `internal/pkg/apperr` | `NotFound/Conflict/Validation/Unauthorized/Forbidden` → RFC 7807 ProblemDetail |
| CSV 导出 | `internal/pkg/csvutil` | `Build(rows []map[string]any, headers []string) string` |

---

## Checklist（以新增 `foo` 模块为例）

### 1. Model — `internal/model/foo.go`

嵌入 `Model`（自动获审计 + 软删），定义业务字段，写 `TableName`。

```go
type Foo struct {
    model.Model                       // ID/时间戳/CreatedBy/UpdatedBy/DeletedAt/DeletedBy
    Name   string `gorm:"size:64;not null" json:"name"`
    Status string `gorm:"size:16;default:active" json:"status"`
}
func (Foo) TableName() string { return "foos" }
```

### 2. 迁移 — `internal/model/rbac.go::AutoMigrate`

```go
return db.AutoMigrate(&User{}, &Role{}, &Permission{}, &Menu{}, &CrudItem{}, &Foo{})
```

### 3. Repository — `internal/repository/foo.go`

嵌入 `GenericRepository[Foo]` 复用全部 CRUD，只写过滤函数 + 接口。

```go
type FooRepository interface {
    List(ctx context.Context, q pagination.Query) (pagination.Result[model.Foo], error)
    FindByID(ctx context.Context, id uint) (*model.Foo, error)
    Create(ctx context.Context, e *model.Foo) error
    Update(ctx context.Context, e *model.Foo) error
    Delete(ctx context.Context, id uint) error
    BatchDelete(ctx context.Context, ids []uint) error
}

type fooRepository struct {
    *GenericRepository[model.Foo]   // 嵌入：FindByID/Create/Update/Delete/BatchDelete 全自动
}

func NewFooRepository(db *gorm.DB) FooRepository {
    return &fooRepository{GenericRepository: NewGenericRepository[model.Foo](db)}
}

func (r *fooRepository) List(ctx context.Context, q pagination.Query) (pagination.Result[model.Foo], error) {
    return r.GenericRepository.List(ctx, q, func(db *gorm.DB) *gorm.DB {
        if q.Keyword != "" {
            return db.Where("name LIKE ?", "%"+q.Keyword+"%")
        }
        return db
    })
}
```

> 有关联表清理的模块（如 user/role 删关联），在 Delete/BatchDelete 里包 `r.DB(ctx).Transaction(...)` 自行处理。

### 4. Service — `internal/service/foo.go`

DTO + 业务方法。`q.Normalize()` → 调 repo → NotFound 翻译。参考 `service/crud.go`。

### 5. Handler — `internal/handler/foo.go`

绑定 → 调 service → `response.Success` / `apperr.Write`。分页参数解析：标准模块用 `pagination.Query` 绑 `page/size`；前端契约不同时（如 crud 用 `current/name`）自定义 query struct 映射。参考 `handler/crud.go`。

### 6. 路由 — `internal/server/router.go`

`NewRouter` 签名加 `fooHandler` 参数；注册路由组 + `AuthRequired` + 每条路由 `RequirePermission`。

> **路由顺序**：`/export`、批量 DELETE（无 `:id`）等具体路径必须在 `/:id` 前注册（Gin 树形路由静态优先）。

```go
foo := api.Group("/foo")
foo.Use(middleware.AuthRequired(jwtMgr))
{
    foo.GET("",      middleware.RequirePermission(permRepo, "foo:list"),   fooHandler.List)
    foo.GET("/:id",  middleware.RequirePermission(permRepo, "foo:list"),   fooHandler.Get)
    foo.POST("",     middleware.RequirePermission(permRepo, "foo:create"), fooHandler.Create)
    foo.PUT("/:id",  middleware.RequirePermission(permRepo, "foo:edit"),   fooHandler.Update)
    foo.DELETE("/:id", middleware.RequirePermission(permRepo, "foo:delete"), fooHandler.Delete)
    foo.DELETE("",     middleware.RequirePermission(permRepo, "foo:delete"), fooHandler.BatchDelete)
}
```

### 7. 装配 — `cmd/api/main.go`

```go
fooRepo := repository.NewFooRepository(gdb)
fooSvc := service.NewFooService(fooRepo)
fooHandler := handler.NewFooHandler(fooSvc)
r := server.NewRouter(..., fooHandler, permRepo, jwtMgr)
```

### 8. 种子 — `internal/service/auth.go`

- `seedPermissionCodes` 追加 `foo:list/create/edit/delete`
- `seedMenus` 追加菜单项（带 `PermissionCode: "foo:list"`）

> 种子幂等（FirstOrCreate）。新增权限码/菜单会自动创建；**已存在记录不更新**（如改了已有菜单的 PermissionCode，需手动 SQL 或重置库）。

### 9. Swagger 注解 — `internal/handler/foo.go`

基座已接入 [swaggo](https://github.com/swaggo/swag)，新 handler 必须加注解以纳入 API 文档。每个端点在 doc comment 中追加注解：

```go
// List GET /api/foo
// @Summary      Foo 分页列表
// @Tags         foo
// @Produce      json
// @Security     BearerAuth
// @Param        page    query int    false "页码" default(1)
// @Param        size    query int    false "每页" default(10)
// @Param        keyword query string false "关键词"
// @Success      200  {object} response.ApiResult
// @Failure      401  {object} response.ProblemDetail
// @Router       /foo [get]
func (h *FooHandler) List(c *gin.Context) { ... }
```

注解要点：
- 契约类型统一用 `response.ApiResult`（成功）/ `response.ProblemDetail`（失败），勿为每个端点自定义 wrapper
- 鉴权端点加 `@Security BearerAuth`；公开端点（登录/健康检查）省略
- `@Tags` 与模块名一致，决定 Swagger UI 的分组
- body 引用本包 DTO struct 或 `service.XxxReq`；swaggo 会递归解析字段

生成与查看：
```bash
swag init -g cmd/api/main.go -o docs --parseInternal --parseDependency   # 重新生成 docs 包
go run ./cmd/api                                                          # 启动后访问 http://localhost:8080/swagger/index.html
```

> **首改需同步 main.go**：全局元注解（`@host`/`@BasePath`/`@securityDefinitions`）在 `cmd/api/main.go` package 声明上方；docs 包由 `_ "gva/docs"` 匿名导入触发注册。路由 `/swagger/*any` 已在 `router.go` 挂载，无需每个模块处理。

---

## 关键约定（避坑）

1. **审计自动注入**：`middleware.AuthRequired` 把 userID 写入 request context，GORM 回调自动填 `CreatedBy/UpdatedBy`。service 用 `c.Request.Context()` 即可生效。`DeletedBy` 由 M8 操作日志统一处理。
2. **泛型基类 T 为值类型**：`GenericRepository[Foo]`（非 `*Foo`），内部 `new(T)` 取指针，与 `pagination.Paginate[T]` 的 `new(T)` 语义对齐。
3. **`isDuplicateKey` 复用**：唯一约束冲突判断在 `service/permission.go` 定义一次，新 service 直接用，勿重复定义。
4. **`NewRouter` 位置参数**：加模块就要改签名（已知技术债，未来可改 struct 参数）。
5. **权限变更后**：role/permission 类变更需调 `middleware.InvalidateAll()` 失效权限缓存（5min TTL）。

---

## 测试

- `repository` 用 `testutil.NewTestDB(t)`（SQLite 临时库 + AutoMigrate）
- 参考 `repository/generic_test.go`、`repository/crud_test.go`
- service 测试参考 `service/*_test.go`（mock repo 或 SQLite）
