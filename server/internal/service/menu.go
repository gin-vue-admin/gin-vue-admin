package service

import (
	"context"
	"errors"
	"sort"

	"gorm.io/gorm"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/repository"
)

// MenuDTO 对齐前端 lib/router/types-menu.ts。
type MenuDTO struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Component string    `json:"component,omitempty"`
	Meta      MenuMeta  `json:"meta"`
	Children  []MenuDTO `json:"children,omitempty"`
}

// MenuMeta 菜单元信息。Permissions 非空时前端按权限过滤。
type MenuMeta struct {
	Title       string           `json:"title"`
	Icon        string           `json:"icon,omitempty"`
	ShowMenu    bool             `json:"showMenu"`
	Permissions *MenuPermissions `json:"permissions,omitempty"`
}

// MenuPermissions 权限约束。Any=任一命中即可见，All=全部命中才可见。
type MenuPermissions struct {
	Any []string `json:"any,omitempty"`
	All []string `json:"all,omitempty"`
}

// MenuService 菜单业务：构建树 + 转 DTO。后端下发完整树，前端过滤。
type MenuService struct {
	repo repository.MenuRepository
}

func NewMenuService(repo repository.MenuRepository) *MenuService {
	return &MenuService{repo: repo}
}

// GetMenus 查全部菜单，构建树，转 MenuDTO。后端下发完整树，前端过滤。
func (s *MenuService) GetMenus(ctx context.Context) ([]MenuDTO, error) {
	menus, err := s.repo.GetAllMenus(ctx)
	if err != nil {
		return nil, err
	}
	return buildMenuTree(menus), nil
}

// buildMenuTree 扁平菜单构建为树（按 ParentID 递归）。
// 三遍：转 DTO 入 map → 挂载 children（操作指针）→ 收集 roots。
// 必须先完成全部 children 挂载再收集 roots，避免值拷贝丢失已挂载的 children。
func buildMenuTree(menus []model.Menu) []MenuDTO {
	byID := make(map[uint]*MenuDTO, len(menus))
	// 第一遍：全部转 DTO 入 map
	for i := range menus {
		m := &menus[i]
		dto := toMenuDTO(m)
		byID[m.ID] = &dto
	}
	// 第二遍：挂载 children（操作 map 中的指针，按 DB 顺序保证 sort 稳定）
	for i := range menus {
		m := &menus[i]
		if m.ParentID == 0 || byID[m.ParentID] == nil {
			continue // 根菜单，不挂载
		}
		child := byID[m.ID]
		byID[m.ParentID].Children = append(byID[m.ParentID].Children, *child)
	}
	// 第三遍：收集 roots（此时 children 已全部挂载，值拷贝不丢）
	roots := make([]MenuDTO, 0)
	for i := range menus {
		m := &menus[i]
		if m.ParentID == 0 || byID[m.ParentID] == nil {
			roots = append(roots, *byID[m.ID])
		}
	}
	return roots
}

// toMenuDTO 单个 Menu→DTO。PermissionCode 非空转 meta.permissions.any。
func toMenuDTO(m *model.Menu) MenuDTO {
	dto := MenuDTO{
		Path:      m.Path,
		Name:      m.Name,
		Component: m.Component,
		Meta: MenuMeta{
			Title:    m.Title,
			Icon:     m.Icon,
			ShowMenu: m.ShowMenu,
		},
	}
	if m.PermissionCode != "" {
		dto.Meta.Permissions = &MenuPermissions{Any: []string{m.PermissionCode}}
	}
	return dto
}

// MenuInfo 管理用菜单（含 id/parentId/sort/status），对齐前端 MenuInfo。
type MenuInfo struct {
	ID        uint       `json:"id"`
	ParentID  *uint      `json:"parentId"` // nil=根→JSON null
	Name      string     `json:"name"`
	Title     string     `json:"title"`
	Path      string     `json:"path"`
	Component string     `json:"component,omitempty"`
	Icon      string     `json:"icon,omitempty"`
	Sort      int        `json:"sort"`
	Status    string     `json:"status"`
	Children  []MenuInfo `json:"children,omitempty"`
}

// MenuCreateReq 创建/更新菜单请求。
type MenuCreateReq struct {
	ParentID  *uint  `json:"parentId"`
	Name      string `json:"name" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Path      string `json:"path" binding:"required"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	Status    string `json:"status" binding:"required,oneof=active inactive"`
}

// MenuSortReq 拖拽排序请求。
type MenuSortReq struct {
	DraggingID uint   `json:"draggingId"`
	TargetID   uint   `json:"targetId"`
	Position   string `json:"position"`
}

// GetTree 管理用菜单树（MenuInfo，含 status）。
func (s *MenuService) GetTree(ctx context.Context) ([]MenuInfo, error) {
	menus, err := s.repo.GetAllMenus(ctx)
	if err != nil {
		return nil, err
	}
	return buildMenuInfoTree(menus), nil
}

// Get 详情。不存在→404。
func (s *MenuService) Get(ctx context.Context, id uint) (*model.Menu, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("菜单不存在")
		}
		return nil, err
	}
	return m, nil
}

// Create 创建。
func (s *MenuService) Create(ctx context.Context, req *MenuCreateReq) (*model.Menu, error) {
	m := &model.Menu{
		ParentID: parentIDVal(req.ParentID), Name: req.Name, Title: req.Title,
		Path: req.Path, Component: req.Component, Icon: req.Icon,
		Sort: req.Sort, Status: req.Status, ShowMenu: true,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Update 更新。
func (s *MenuService) Update(ctx context.Context, m *model.Menu) error {
	return s.repo.Update(ctx, m)
}

// Delete 级联删除：删自身 + 所有子孙。
func (s *MenuService) Delete(ctx context.Context, id uint) error {
	all, err := s.repo.GetAllMenus(ctx)
	if err != nil {
		return err
	}
	// 找 id 及所有后代
	ids := []uint{id}
	collectDescendants(all, id, &ids)
	return s.repo.DeleteByIDs(ctx, ids)
}

// Sort 拖拽排序。inner=变 parent；before/after=同级插入目标前后，重排整数化。
// 纯 int 逻辑（model.Sort 是 int，不用 0.5 float）。
func (s *MenuService) Sort(ctx context.Context, req *MenuSortReq) error {
	dragging, err := s.repo.FindByID(ctx, req.DraggingID)
	if err != nil {
		return apperr.NotFound("拖拽节点不存在")
	}
	target, err := s.repo.FindByID(ctx, req.TargetID)
	if err != nil {
		return apperr.NotFound("目标节点不存在")
	}
	if req.Position == "inner" {
		dragging.ParentID = target.ID
	} else {
		dragging.ParentID = target.ParentID
	}
	// 取同级（含 dragging 新 parentID 下）所有菜单，按 sort 升序
	all, err := s.repo.GetAllMenus(ctx)
	if err != nil {
		return err
	}
	siblings := []model.Menu{}
	for _, m := range all {
		if m.ParentID == dragging.ParentID {
			if m.ID == dragging.ID {
				siblings = append(siblings, *dragging)
			} else {
				siblings = append(siblings, m)
			}
		}
	}
	sort.Slice(siblings, func(i, j int) bool { return siblings[i].Sort < siblings[j].Sort })
	// 从 siblings 移除 dragging，按 position 插入 target 前/后
	dragIdx := -1
	targetIdx := -1
	for i := range siblings {
		if siblings[i].ID == dragging.ID {
			dragIdx = i
		}
		if siblings[i].ID == target.ID {
			targetIdx = i
		}
	}
	if dragIdx >= 0 {
		siblings = append(siblings[:dragIdx], siblings[dragIdx+1:]...)
		// 重新找 target 位置（移除后索引可能变）
		targetIdx = -1
		for i := range siblings {
			if siblings[i].ID == target.ID {
				targetIdx = i
				break
			}
		}
	}
	if req.Position == "before" {
		// 插入 target 前
		siblings = append(siblings, model.Menu{})
		copy(siblings[targetIdx+1:], siblings[targetIdx:])
		siblings[targetIdx] = *dragging
	} else if req.Position == "after" {
		// 插入 target 后
		siblings = append(siblings, model.Menu{})
		copy(siblings[targetIdx+2:], siblings[targetIdx+1:])
		siblings[targetIdx+1] = *dragging
	} else {
		// inner: dragging 变 target 子节点，追加到同级末尾后重排（写回新 ParentID）
		siblings = append(siblings, *dragging)
	}
	// 重排整数化
	for i := range siblings {
		siblings[i].Sort = i
	}
	return s.repo.UpdateSorts(ctx, siblings)
}

// parentIDVal *uint→uint（nil→0=根）。
func parentIDVal(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

// collectDescendants 递归收集 id 的所有后代。
func collectDescendants(all []model.Menu, id uint, ids *[]uint) {
	for _, m := range all {
		if m.ParentID == id {
			*ids = append(*ids, m.ID)
			collectDescendants(all, m.ID, ids)
		}
	}
}

// buildMenuInfoTree 扁平菜单→MenuInfo 树。
// 三遍法（同 buildMenuTree 模式）：转 DTO 入 map → 挂 children → 收集 roots。
func buildMenuInfoTree(menus []model.Menu) []MenuInfo {
	byID := make(map[uint]*MenuInfo, len(menus))
	// 第一遍：全部转 MenuInfo 入 map
	for i := range menus {
		m := &menus[i]
		info := toMenuInfo(m)
		byID[m.ID] = &info
	}
	// 第二遍：挂载 children（操作 map 中的指针）
	for i := range menus {
		m := &menus[i]
		if m.ParentID == 0 || byID[m.ParentID] == nil {
			continue
		}
		byID[m.ParentID].Children = append(byID[m.ParentID].Children, *byID[m.ID])
	}
	// 第三遍：收集 roots
	roots := make([]MenuInfo, 0)
	for i := range menus {
		m := &menus[i]
		if m.ParentID == 0 || byID[m.ParentID] == nil {
			roots = append(roots, *byID[m.ID])
		}
	}
	return roots
}

// toMenuInfo Menu→MenuInfo。ParentID 0→nil（JSON null）。
func toMenuInfo(m *model.Menu) MenuInfo {
	var pid *uint
	if m.ParentID != 0 {
		p := m.ParentID
		pid = &p
	}
	return MenuInfo{
		ID: m.ID, ParentID: pid, Name: m.Name, Title: m.Title, Path: m.Path,
		Component: m.Component, Icon: m.Icon, Sort: m.Sort, Status: m.Status,
	}
}
