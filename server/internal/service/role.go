// Package service 角色业务：CRUD + 分页列表 + CSV 导出 + 权限分配。
// 依赖 repository.RoleRepository 接口，便于单测用 mock 替换 DB。
// isDuplicateKey 为同包 permission.go 已定义的复用函数（不在此重复定义）。
package service

import (
	"context"
	"errors"
	"strconv"

	"gva/internal/middleware"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/csvutil"
	"gva/internal/pkg/datascope"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"

	"gorm.io/gorm"
)

// RoleService 角色业务。
type RoleService struct {
	repo repository.RoleRepository
}

// NewRoleService 构造角色服务。
func NewRoleService(repo repository.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

// List 分页列表，返回对齐前端 {records,total,current,size} 的结果。
func (s *RoleService) List(ctx context.Context, q pagination.Query) (pagination.Result[model.Role], error) {
	q.Normalize()
	roles, total, err := s.repo.List(ctx, q)
	if err != nil {
		return pagination.Result[model.Role]{}, err
	}
	return pagination.Result[model.Role]{
		Records: roles, Total: total, Current: q.Page, Size: q.Size,
	}, nil
}

// Get 详情。不存在返回 404。
func (s *RoleService) Get(ctx context.Context, id uint) (*model.Role, error) {
	r, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("角色不存在")
		}
		return nil, err
	}
	return r, nil
}

// Create 创建。code 唯一约束冲突返回 409。dataScope 空串兜底为 all（与 model 默认一致）。
func (s *RoleService) Create(ctx context.Context, name, code, description, status, dataScope string) (*model.Role, error) {
	if dataScope == "" {
		dataScope = datascope.ScopeAll
	}
	r := &model.Role{Name: name, Code: code, Description: description, Status: status, DataScope: dataScope}
	if err := s.repo.Create(ctx, r); err != nil {
		if isDuplicateKey(err) { // 复用 permission.go 中的 isDuplicateKey
			return nil, apperr.Conflict("角色编码已存在")
		}
		return nil, err
	}
	return r, nil
}

// Update 更新。code 唯一约束冲突返回 409。
func (s *RoleService) Update(ctx context.Context, r *model.Role) error {
	if err := s.repo.Update(ctx, r); err != nil {
		if isDuplicateKey(err) {
			return apperr.Conflict("角色编码已存在")
		}
		return err
	}
	return nil
}

// Delete 软删角色 + 解除关联（role_permissions/user_roles），并失效权限缓存。
// 先 FindByID 确认存在（不存在→404），与 permission service 一致。
// GORM 的 db.Delete(&model.Role{}, id) 在记录不存在时通常不返回错误，仅看 RowsAffected，故先查再删。
func (s *RoleService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("角色不存在")
		}
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	middleware.InvalidateAll()
	return nil
}

// BatchDelete 批量软删 + 解除关联，并失效权限缓存。
func (s *RoleService) BatchDelete(ctx context.Context, ids []uint) error {
	if err := s.repo.BatchDelete(ctx, ids); err != nil {
		return err
	}
	middleware.InvalidateAll()
	return nil
}

// Export 生成 CSV 文本（含表头），字段含转义。
// 用 ListAll 取全量（不分页），避免分页漏数据。
// id 列用 strconv.FormatUint 转为字符串，对齐前端 mock 的 string id 契约。
func (s *RoleService) Export(ctx context.Context, q pagination.Query) (string, error) {
	roles, err := s.repo.ListAll(ctx, q)
	if err != nil {
		return "", err
	}
	rows := make([]map[string]any, 0, len(roles))
	for _, r := range roles {
		rows = append(rows, map[string]any{
			"id":          strconv.FormatUint(uint64(r.ID), 10),
			"name":        r.Name,
			"code":        r.Code,
			"description": r.Description,
			"status":      r.Status,
			"createTime":  r.CreatedAt,
			"updateTime":  r.UpdatedAt,
		})
	}
	return csvutil.Build(rows, []string{"id", "name", "code", "description", "status", "createTime", "updateTime"}), nil
}

// GetPermissions 查角色已分配的权限 code 数组。角色不存在返回 404。
func (s *RoleService) GetPermissions(ctx context.Context, roleID uint) ([]string, error) {
	if _, err := s.repo.FindByID(ctx, roleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("角色不存在")
		}
		return nil, err
	}
	return s.repo.GetRolePermissionCodes(ctx, roleID)
}

// SetPermissions 全量替换角色权限（API 用 code，表存 id），并失效权限缓存。
// 严格校验：FindPermissionIDsByCodes 返回存在的 code→id 映射，
// 任一传入 code 不在映射中→404，全部存在才转 id 调 ReplaceRolePermissions。
func (s *RoleService) SetPermissions(ctx context.Context, roleID uint, codes []string) error {
	if _, err := s.repo.FindByID(ctx, roleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("角色不存在")
		}
		return err
	}
	codeToID, err := s.repo.FindPermissionIDsByCodes(ctx, codes)
	if err != nil {
		return err
	}
	// 严格校验：未知 code 报 404，禁止静默丢弃。
	for _, code := range codes {
		if _, ok := codeToID[code]; !ok {
			return apperr.NotFound("权限 " + code + " 不存在")
		}
	}
	ids := make([]uint, 0, len(codes))
	for _, code := range codes {
		ids = append(ids, codeToID[code])
	}
	if err := s.repo.ReplaceRolePermissions(ctx, roleID, ids); err != nil {
		return err
	}
	middleware.InvalidateAll()
	return nil
}
