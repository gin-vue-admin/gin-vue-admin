# 新增业务模块

基座已提供 **泛型 Repository 基类 + 审计字段 + 软删除 + 统一响应 + Swagger 注解**，新增模块只写"差异部分"。

最佳参考：`internal/{model,repository,service,handler}/crud.go`（最简范例）。复杂模板参考 `user`/`role`（含关联、权限缓存、唯一约束）。

## 基座能力（直接复用）

| 能力 | 位置 | 说明 |
|------|------|------|
| 泛型 Repository | `repository/generic.go` | `GenericRepository[T]` 提供 FindByID/List/Create/Update/Delete/BatchDelete + 分页 |
| 审计字段 | `model/base.go` 的 `Model` | CreatedBy/UpdatedBy/DeletedBy，GORM 回调自动注入 |
| 软删除 | `Model.DeletedAt` | gorm.DeletedAt，查询自动过滤 |
| 分页 | `pkg/pagination` | `Query{Keyword,Status,Page,Size}` + `Result[T]` + 泛型 `Paginate[T]` |
| 统一响应 | `pkg/response` | 成功 `Success(c,data,msg)`；失败 `apperr.Write(c,err)` |
| 错误模型 | `pkg/apperr` | NotFound/Conflict/Validation/... → RFC 7807 ProblemDetail |

## Checklist（以新增 `foo` 模块为例）

### 1. Model — `internal/model/foo.go`

```go
type Foo struct {
    model.Model                       // ID/时间戳/审计/软删
    Name   string `gorm:"size:64;not null" json:"name"`
    Status string `gorm:"size:16;default:active" json:"status"`
}
func (Foo) TableName() string { return "foos" }
```

### 2. 迁移 — `internal/model/rbac.go::AutoMigrate`

```go
return db.AutoMigrate(&User{}, &Role{}, ..., &Foo{})
```

### 3. Repository — `internal/repository/foo.go`

嵌入 `GenericRepository[Foo]` 复用全部 CRUD，只写过滤函数 + 接口：

```go
type FooRepository interface {
    List(ctx context.Context, q pagination.Query) (pagination.Result[model.Foo], error)
    // FindByID/Create/Update/Delete/BatchDelete 由 GenericRepository 嵌入提供
}

type fooRepository struct {
    *GenericRepository[model.Foo]
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

### 4. Service — `internal/service/foo.go`

DTO + 业务方法：`q.Normalize()` → 调 repo → NotFound 翻译。参考 `service/crud.go`。

### 5. Handler — `internal/handler/foo.go`

绑定 → 调 service → `response.Success` / `apperr.Write`。参考 `handler/crud.go`。

### 6-8. 路由 / 装配 / 种子

- `server/router.go`：`NewRouter` 签名加 `fooHandler`，注册路由组 + `AuthRequired` + 每条 `RequirePermission`
- `cmd/api/main.go`：构造 repo→service→handler，传入 `NewRouter`
- `service/auth.go`：`seedPermissionCodes` 加 `foo:list/create/edit/delete`；`seedMenus` 加菜单项

::: warning 路由顺序
`/export`、批量 DELETE（无 `:id`）等具体路径必须在 `/:id` 前注册（Gin 树形路由静态优先）。
:::

### 9. Swagger 注解 — `internal/handler/foo.go`

```go
// List GET /api/foo
// @Summary      Foo 分页列表
// @Tags         foo
// @Produce      json
// @Security     BearerAuth
// @Param        page    query int    false "页码" default(1)
// @Success      200  {object} response.ApiResult
// @Router       /foo [get]
func (h *FooHandler) List(c *gin.Context) { ... }
```

改完 handler 注解必须重新生成 docs 包：

```bash
make server-swag   # swag init -g cmd/api/main.go -o docs --parseInternal --parseDependency
```

## 关键约定（避坑）

1. **审计自动注入**：`middleware.AuthRequired` 写入 request context，GORM 回调自动填。service 用 `c.Request.Context()` 生效。
2. **泛型基类 T 为值类型**：`GenericRepository[Foo]` 非 `*Foo`。
3. **权限变更**：role/permission 改完调 `middleware.InvalidateAll()` 失效缓存（5min TTL）。
4. **`NewRouter` 位置参数**：加模块要改签名（已知技术债，未来可改 struct 参数）。
