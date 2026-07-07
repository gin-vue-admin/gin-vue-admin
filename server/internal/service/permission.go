// Package service 权限业务：CRUD + 分页列表 + CSV 导出。
// 依赖 repository.PermissionRepository 接口，便于单测用 mock 替换 DB。
package service

import (
	"context"
	"errors"
	"strconv"

	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/csvutil"
	"gva/internal/pkg/gormx"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"

	"gorm.io/gorm"
)

// PermissionService 权限业务。
type PermissionService struct {
	repo repository.PermissionRepository
}

// NewPermissionService 构造权限服务。
func NewPermissionService(repo repository.PermissionRepository) *PermissionService {
	return &PermissionService{repo: repo}
}

// List 分页列表，返回对齐前端 {records,total,current,size} 的结果。
func (s *PermissionService) List(ctx context.Context, q pagination.Query, module string) (pagination.Result[model.Permission], error) {
	q.Normalize()
	perms, total, err := s.repo.List(ctx, q, module)
	if err != nil {
		return pagination.Result[model.Permission]{}, err
	}
	return pagination.Result[model.Permission]{
		Records: perms, Total: total, Current: q.Page, Size: q.Size,
	}, nil
}

// ListAll 全量列表（?all=true 场景），仍受 module/status/keyword 过滤。
func (s *PermissionService) ListAll(ctx context.Context, q pagination.Query, module string) ([]model.Permission, error) {
	q.Normalize()
	return s.repo.ListAll(ctx, q, module)
}

// Get 详情。不存在返回 404。
func (s *PermissionService) Get(ctx context.Context, id uint) (*model.Permission, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("权限不存在")
		}
		return nil, err
	}
	return p, nil
}

// Create 创建。code 唯一约束冲突返回 409。
func (s *PermissionService) Create(ctx context.Context, name, code, module, description, status string) (*model.Permission, error) {
	p := &model.Permission{
		Name: name, Code: code, Type: "api", Module: module,
		Description: description, Status: status,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		if gormx.IsDuplicateKey(err) {
			return nil, apperr.Conflict("权限编码已存在")
		}
		return nil, err
	}
	return p, nil
}

// Update 更新。code 唯一约束冲突返回 409。
func (s *PermissionService) Update(ctx context.Context, p *model.Permission) error {
	if err := s.repo.Update(ctx, p); err != nil {
		if gormx.IsDuplicateKey(err) {
			return apperr.Conflict("权限编码已存在")
		}
		return err
	}
	return nil
}

// Delete 软删除。先 FindByID 确认存在（不存在→404），再 Delete。
// GORM 的 db.Delete(&model.Permission{}, id) 在记录不存在时通常不返回错误，
// 仅看 RowsAffected，故在此先查再删，语义明确。
func (s *PermissionService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("权限不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

// BatchDelete 批量软删除。
func (s *PermissionService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.repo.BatchDelete(ctx, ids)
}

// Export 生成 CSV 文本（含表头），字段含转义。
// id 列用 strconv.FormatUint 转为字符串，对齐前端 mock 的 string id 契约。
func (s *PermissionService) Export(ctx context.Context, q pagination.Query, module string) (string, error) {
	perms, err := s.repo.ListAll(ctx, q, module)
	if err != nil {
		return "", err
	}
	rows := make([]map[string]any, 0, len(perms))
	for _, p := range perms {
		rows = append(rows, map[string]any{
			"id":          strconv.FormatUint(uint64(p.ID), 10),
			"name":        p.Name,
			"code":        p.Code,
			"module":      p.Module,
			"description": p.Description,
			"status":      p.Status,
			"createTime":  p.CreatedAt,
			"updateTime":  p.UpdatedAt,
		})
	}
	return csvutil.Build(rows, []string{"id", "name", "code", "module", "description", "status", "createTime", "updateTime"}), nil
}
