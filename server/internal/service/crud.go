// Package service 通用 CRUD 示例业务（脚手架范例）。
// 展示新模块的最简业务层：复用 repository.CrudRepository（底层由 GenericRepository 提供），
// 仅做 NotFound 翻译与 Normalize。无关联/无权限缓存/无唯一约束，是新增模块最干净的起点。
package service

import (
	"context"
	"errors"

	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"

	"gorm.io/gorm"
)

// CrudService 通用 CRUD 示例业务。
type CrudService struct {
	repo repository.CrudRepository
}

// NewCrudService 构造 crud 服务。
func NewCrudService(repo repository.CrudRepository) *CrudService {
	return &CrudService{repo: repo}
}

// List 分页列表。repo.List 已返回对齐前端的 Result，这里仅 Normalize + 回填 current/size。
func (s *CrudService) List(ctx context.Context, q pagination.Query) (pagination.Result[model.CrudItem], error) {
	q.Normalize()
	res, err := s.repo.List(ctx, q)
	if err != nil {
		return pagination.Result[model.CrudItem]{}, err
	}
	res.Current = q.Page
	res.Size = q.Size
	return res, nil
}

// Get 详情。不存在返回 404。
func (s *CrudService) Get(ctx context.Context, id uint) (*model.CrudItem, error) {
	e, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("记录不存在")
		}
		return nil, err
	}
	return e, nil
}

// Create 创建。CrudItem 无唯一约束，直接落库（CreatedBy/UpdatedBy 由审计回调注入）。
func (s *CrudService) Create(ctx context.Context, e *model.CrudItem) error {
	return s.repo.Create(ctx, e)
}

// Update 全量更新（UpdatedBy 由审计回调注入）。
func (s *CrudService) Update(ctx context.Context, e *model.CrudItem) error {
	return s.repo.Update(ctx, e)
}

// Delete 软删。先查再删，不存在→404。
func (s *CrudService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("记录不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

// BatchDelete 批量软删。空 ids 由 repo 兜底返回 nil。
func (s *CrudService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.repo.BatchDelete(ctx, ids)
}
