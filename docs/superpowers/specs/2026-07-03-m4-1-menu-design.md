# M4.1 动态路由菜单设计

> 状态：已批准（2026-07-03）
> 范围：GET /api/system/menus —— 登录用户按角色过滤的菜单树（MenuDTO），供前端动态路由装载。
> 前置：M3 已整体交付（提交 4ed2872）。
> M4 拆分：M4.1 动态路由菜单（本 spec）→ M4.2 菜单管理 CRUD（后续）。

## 1. 目标与边界

### 目标
- 实现 GET /api/system/menus，对齐前端 MenuDTO 契约（lib/router/types-menu.ts）。
- 按登录用户角色过滤菜单（超管 `*` 全见，普通用户按 PermissionCode 过滤）。
- 构建菜单树（ParentID 递归）+ 树修剪（父无可见子则不返回）。
- model.Menu 加 Title（显示名），明确 Name 为路由名。

### 非目标（YAGNI）
- M4.1 只做动态路由菜单读端点；菜单管理 CRUD（增删改/拖拽排序）是 M4.2。
- PermissionCode 用单值（不做 meta.permissions.any 数组复杂语义——M4.1 单码足够，超管 `*` 短路）。
- 不做菜单缓存（菜单低频读，每次查 DB 可接受；若需缓存后续加）。

## 2. 关键决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| Name vs Title | 加 Title（显示名），Name 为路由名 | 前端 MenuDTO 需两个独立值：name（路由标识，标识符规范）+ meta.title（显示文字，中文）。职责不同不可合并 |
| 端点权限 | 仅 AuthRequired（登录即可） | 菜单是登录后基础数据，不按权限码限制访问，按角色过滤内容 |
| 角色过滤 | 后端下发完整树 + meta.permissions，**前端过滤**（canSeeMenu + 路由守卫） | 前端已有完整过滤逻辑（permission store canSeeMenu + guards 权限检查），后端不需重复过滤，避免双重逻辑维护 |
| 树构建 | ParentID 递归，sort 排序 | model.Menu 已有 ParentID/Sort |
| PermissionCode | 单值转 meta.permissions.any 数组 | 前端强依赖 meta.permissions.{any,all}（guards.ts:100, permission.ts:37），后端必须下发 |
| 权限码查询 | 不需要（前端过滤） | 简化后端，不查用户权限 |

## 3. 数据模型

model.Menu 加 Title：
```go
type Menu struct {
	Model
	ParentID       uint   `gorm:"index;default:0" json:"parentId"`
	Name           string `gorm:"size:64" json:"name"`         // 路由名（如 systemUser）
	Title          string `gorm:"size:64" json:"title"`        // 新增：显示名（如 用户管理）
	Path           string `gorm:"size:255" json:"path"`
	Component      string `gorm:"size:255" json:"component"`
	Icon           string `gorm:"size:64" json:"icon"`         // PascalCase 全局唯一
	Sort           int    `gorm:"default:0" json:"sort"`
	ShowMenu       bool   `gorm:"default:true" json:"showMenu"`
	PermissionCode string `gorm:"size:128" json:"permissionCode"` // 空则公共可见
}
```
AutoMigrate 自动加 Title 列。

## 4. MenuDTO 响应（对齐前端 types-menu.ts）

```go
// MenuDTO 对齐前端 lib/router/types-menu.ts。
type MenuDTO struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Component string    `json:"component,omitempty"`
	Meta      MenuMeta  `json:"meta"`
	Children  []MenuDTO `json:"children,omitempty"`
}
type MenuMeta struct {
	Title       string           `json:"title"`
	Icon        string           `json:"icon,omitempty"`
	ShowMenu    bool             `json:"showMenu"`
	Permissions *MenuPermissions `json:"permissions,omitempty"` // 前端强依赖（guards.ts:100, permission.ts:37）
}
type MenuPermissions struct {
	Any []string `json:"any,omitempty"` // 任一权限码命中即可见
	All []string `json:"all,omitempty"` // 全部命中才可见
}
```

转换：Menu→MenuDTO
- Path/Name/Component 直接映射
- Meta: { Title←Title, Icon←Icon, ShowMenu←ShowMenu, Permissions←PermissionCode 转 {any:[code]} }
- PermissionCode 非空 → Permissions = &MenuPermissions{Any: []string{code}}；空 → nil（公共可见，前端 canSeeMenu 返回 true）
- Children: 递归构建

## 5. 端点与角色过滤

`GET /api/system/menus`（AuthRequired，无 RequirePermission）：
```
→ 取 userID（AuthRequired 注入，确认登录）
→ menus = repo.GetAllMenus()  // 按 sort, parent_id 排序的全部菜单
→ tree = buildTree(menus)  // ParentID 递归构建完整树
→ dtos = toMenuDTOs(tree)  // 转 MenuDTO，含 meta.permissions
→ response.Success(c, dtos)
```

**后端下发完整树，不过滤**。前端两套机制负责过滤：
- `permission store canSeeMenu`（permission.ts:37）：侧边栏显示前过滤
- `guards.ts:100` 路由守卫：访问路由时检查 meta.permissions

后端只需把 PermissionCode 转成 meta.permissions.any 下发，过滤逻辑全部由前端处理（前端已有成熟实现）。避免后端/前端双重过滤逻辑维护。

## 6. 种子菜单

Seed 补种菜单（对齐 ALL_MENUS 核心结构）。M4.1 种入主菜单：
```
home:     Name=home, Title=首页, Path=/, Component=dashboard/views/Home, Icon=HomeFilled, Sort=0
crud:     Name=crud, Title=增删改查, Path=/crud, Component=crud/views/List, Icon=Document, Sort=10
system:   Name=system, Title=系统管理, Path=/system, Icon=Setting, Sort=20（父，无 component）
  systemUser:       Title=用户管理, Path=/system/user, Component=system/user/views/List, Icon=User, PermissionCode=user:list
  systemRole:       Title=角色管理, Path=/system/role, Component=system/role/views/List, Icon=Avatar, PermissionCode=role:list
  systemPermission: Title=权限管理, Path=/system/permission, Component=system/permission/views/List, Icon=Key, PermissionCode=permission:list
```
> home/crud/system 父节点无 PermissionCode（公共可见）。system 子菜单有 PermissionCode（按权限过滤）。system/access 中间层（M4.1 简化掉，直接 system→user/role/permission）。

## 7. MenuRepository

```go
type MenuRepository interface {
	GetAllMenus(ctx) ([]model.Menu, error)  // 按 sort, parent_id 排序
}
```
> 仅需 GetAllMenus（M4.1 只读）。M4.2 扩展 CRUD 方法。

## 8. 错误处理

| 场景 | HTTP | code | detail |
|---|---|---|---|
| 未登录 | 401 | UNAUTHORIZED | 未授权（AuthRequired） |
| DB 故障 | 500 | INTERNAL_ERROR | 服务暂时不可用（透传） |

M4.1 是读端点，无 404/409/422（菜单不存在不报错，返空数组）。

## 9. 文件结构

新增：
```
internal/repository/menu.go        MenuRepository（GetAllMenus）
internal/service/menu.go           MenuService（过滤+树构建+DTO）+ MenuDTO/MenuMeta 类型
internal/service/menu_test.go       单测
internal/handler/menu.go           MenuHandler（1端点）
```

修改：
```
internal/model/menu.go            Menu 加 Title
internal/service/auth.go          Seed 补菜单数据
internal/server/router.go         注册 /api/system/menus
cmd/api/main.go                   组装 Menu 依赖
```

## 10. 测试策略

`service/menu_test.go`，SQLite 隔离（testutil.NewTestDB）。

### MenuService 用例
1. 返回全部菜单（不过滤，完整树）
2. 树构建正确（ParentID 递归，sort 排序）
3. MenuDTO 转换：Meta.Title/Icon/ShowMenu 正确
4. PermissionCode 非空 → meta.permissions.any = [code]
5. PermissionCode 空 → meta.permissions 为 nil（公共可见）
6. Component 空的父节点（如 system）→ component 字段 omitempty 不下发
7. children 递归嵌套正确

## 11. 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./...` 全过（M2+M3+M4.1 回归）
- [ ] admin GET /api/system/menus → 返回完整菜单树（MenuDTO）
- [ ] user GET /api/system/menus → 仅公共菜单 + user:list 相关（systemUser）
- [ ] 未登录 → 401
- [ ] 菜单树结构正确（children 嵌套，sort 顺序）
- [ ] 启动日志含"种子数据就绪"，菜单已种入

## 12. 与契约对齐核查

- MenuDTO：path/name/component/meta{title,icon,showMenu}/children——对齐前端 types-menu.ts
- name 是路由名（systemUser），title 是显示名（用户管理）——两个字段
- 按角色过滤——对齐 mock（admin 全见，user 部分）
- 公共菜单（无 PermissionCode）——home/crud 等所有用户可见
- 不下发 meta.permissions——后端已过滤，前端不需再过滤（isSuperAdmin 短路 + 后端过滤）

## 13. 风险

1. ~~前端 meta.permissions~~：已确认前端强依赖（guards.ts:100, permission.ts:37），MenuDTO 已含 meta.permissions.any。后端下发完整树 + permissions，前端过滤。
2. 菜单树构建性能：菜单数量少（<50），全量查 + 内存构建树，无性能问题。
3. system/access 中间层：ALL_MENUS 有三层（system→access→user），M4.1 种子简化为两层（system→user/role/permission）。前端 dynamic.ts 只注册叶子 component，中间层不影响路由装载。但侧边栏层级显示可能不同——Task 落实时按前端 Sidebar 验证，若需三层再补种子。
4. 后端不过滤 vs mock 行为：mock 按角色返不同树（admin 全见，user 部分）。本 spec 后端下发完整树 + permissions，前端 canSeeMenu 过滤——最终用户看到的菜单与 mock 过滤后等价，但响应体不同（后端返全量）。前端已适配（visibleMenus 计算属性过滤）。
