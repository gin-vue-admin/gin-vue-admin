# M4.2 菜单管理 CRUD 设计

> 状态：已批准（2026-07-03）
> 范围：/api/system/menu/* 5 端点（树查/增/改/删/拖拽排序），对齐前端 menu-manage 契约。
> 前置：M4.1 已入库（model.Menu 含 Title）。
> 设计原则：沿用 M3 三域 CRUD 模式（repository 接口 + service + handler），复用 pagination/apperr/middleware。

## 1. 目标与边界

### 目标
- 5 端点：GET /api/system/menu（树）、POST（创建）、PUT /:id（更新）、DELETE /:id（级联删）、PATCH /sort（拖拽排序）。
- 拖拽排序：draggingId 相对 targetId 的 before/after/inner 定位，重排同级 sort。
- 级联删除：删父菜单递归删子菜单。
- 复用 M4.1 的 MenuDTO 树构建。

### 非目标（YAGNI）
- 不做菜单缓存失效（菜单低频改，读时直查 DB）。
- 不做多权限码数组（PermissionCode 单值，M4.1 既定）。
- 不做菜单导入导出（前端无此端点）。

## 2. 关键决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| parentId 类型 | API 用 string（"0" 或 null 表根），后端 uint（0=根） | 对齐前端 MenuInfo.parentId:string\|null；handler 转换 |
| 树查询 | 复用 M4.1 buildMenuTree，返 MenuInfo 树（管理用，含 status/id/parentId） | 与 MenuDTO 不同——管理需 id/parentId/sort/status |
| 拖拽排序 | 重排同级 sort（整数化），before/after 调 target.sort±0.5 后重排 | 对齐 mock 算法（sort-0.5/sort+0.5 再整数化） |
| 级联删除 | 递归删子菜单（事务） | mock removeRecordAndChildren 递归 |
| 删除约束 | 不校验引用（菜单无外键被引用） | 简单，菜单是叶子资源 |
| 权限码 | 复用 menu:list/create/edit/delete（M3.1 已种） | 但 mock 未种 menu:* 权限——seed 需补 |
| 软删除 | 菜单硬删除（无 DeletedAt） | model.Menu 无 DeletedAt，管理删除是真删 |

> ⚠️ 权限码：M3.1 seed 未种 menu:list/create/edit/delete。M4.2 Seed 补种这些权限码，并给 super_admin（已持 *，无需分配）。

## 3. 端点契约

| 方法 | 路径 | 权限码 | 请求 | 响应 |
|---|---|---|---|---|
| GET | /api/system/menu | menu:list | ?view=tree（mock 用） | MenuInfo[] 树 |
| POST | /api/system/menu | menu:create | MenuCreateReq | MenuInfo（msg=创建成功） |
| PUT | /api/system/menu/:id | menu:edit | MenuCreateReq（partial） | MenuInfo（msg=更新成功） |
| DELETE | /api/system/menu/:id | menu:delete | — | true（msg=删除成功） |
| PATCH | /api/system/menu/sort | menu:edit | MenuSortReq | true（msg=排序成功） |

### MenuInfo（管理用，对齐前端）
```go
type MenuInfo struct {
	ID       uint        `json:"id"`       // 字符串化？前端 id:string，后端 uint→JSON 数字，前端兼容（M2 已确认）
	ParentID *uint       `json:"parentId"` // nil→null（根），非 nil→数字
	Name     string      `json:"name"`
	Title    string      `json:"title"`    // 显示名（M4.1 已加）
	Path     string      `json:"path"`
	Component string     `json:"component,omitempty"`
	Icon     string      `json:"icon,omitempty"`
	Sort     int         `json:"sort"`
	Status   string      `json:"status"`   // active|inactive（M4.1 model 无 status——需加？见下）
	Children []MenuInfo  `json:"children,omitempty"`
}
```
> ⚠️ model.Menu 当前**无 Status 字段**。前端 MenuInfo 有 status。需加 Status（active|inactive）。

### MenuCreateReq
```go
type MenuCreateReq struct {
	ParentID  *uint  `json:"parentId"`           // nil=根
	Name      string `json:"name" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Path      string `json:"path" binding:"required"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	Status    string `json:"status" binding:"required,oneof=active inactive"`
}
```

### MenuSortReq
```go
type MenuSortReq struct {
	DraggingID uint   `json:"draggingId"`  // 前端 string，handler 转 uint
	TargetID   uint   `json:"targetId"`
	Position   string `json:"position" binding:"required,oneof=before after inner"`
}
```

## 4. 数据模型变更

model.Menu 加 Status（active|inactive）：
```go
Status string `gorm:"size:16;default:active" json:"status"` // 新增
```
AutoMigrate 自动加列。Seed 补种 menu:* 权限码。

## 5. 拖拽排序算法

```
PATCH /sort {draggingId, targetId, position}
→ 取 dragging、target 菜单
→ position=inner: dragging.parentId = target.id
→ position=before/after: dragging.parentId = target.parentId; dragging.sort = target.sort ± 0.5
→ 重排同级（同 parentId 的菜单按 sort 升序，重新赋整数 0,1,2...）
→ 批量 Update sort
```
后端用整数 sort，±0.5 后重排整数化（对齐 mock）。事务保证一致。

## 6. 级联删除

```
DELETE /:id
→ 事务内递归：删自身 + 所有子孙
→ 实现：先查所有菜单，BFS/DFS 找 id 的所有后代 id 集合，批量 Delete
```

## 7. MenuRepository 扩展

M4.1 的 MenuRepository 只有 GetAllMenus。M4.2 扩展：
```go
type MenuRepository interface {
	GetAllMenus(ctx) ([]model.Menu, error)         // M4.1
	FindByID(ctx, id) (*model.Menu, error)         // 新增
	Create(ctx, *model.Menu) error                 // 新增
	Update(ctx, *model.Menu) error                 // 新增
	DeleteByIDs(ctx, ids []uint) error             // 新增（批量硬删，级联用）
	UpdateSort(ctx, id, sort uint, parentID *uint) error  // 新增（排序用）
	BatchUpdateSort(ctx, menus []model.Menu) error // 新增（重排同级）
}
```

## 8. 文件结构

新增：repository/menu.go（扩展）、service/menu.go（扩展 CRUD+排序）、service/menu_test.go（扩展）、handler/menu.go（扩展 5 端点）
修改：model/menu.go（加 Status）、service/auth.go（Seed 补 menu:* 权限码）、router.go、main.go（menuHandler 已组装，路由加 /api/system/menu 组）

## 9. 测试

- 树查询（含 status）
- 创建（parentId nil=根 / 非 nil=子）
- 更新
- 级联删除（删父，子孙全删）
- 拖拽 inner（变 parent）
- 拖拽 before/after（同级重排，sort 整数化）
- 权限码未知（M4.2 无角色关联，N/A）

## 10. 验收

- admin GET /api/system/menu 返回树（含 status）
- admin POST 创建菜单 → 成功
- admin DELETE 父菜单 → 子孙全删
- admin PATCH /sort 拖拽 → sort 重排
- user（无 menu:list）→ 403
- go test ./... 全过
