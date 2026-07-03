package service

import (
	"context"

	"gva/internal/model"
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
