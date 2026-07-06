// Package service 操作日志业务：查询/删除/清空（日志由中间件自动生成，无 Create/Update）。
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

// OperationLogService 操作日志业务。
type OperationLogService struct {
	repo repository.OperationLogRepository
}

func NewOperationLogService(repo repository.OperationLogRepository) *OperationLogService {
	return &OperationLogService{repo: repo}
}

// List 分页查询（支持 keyword/status/时间范围）。
func (s *OperationLogService) List(ctx context.Context, q repository.OperationLogQuery) (pagination.Result[model.OperationLog], error) {
	q.Normalize()
	res, err := s.repo.List(ctx, q)
	if err != nil {
		return pagination.Result[model.OperationLog]{}, err
	}
	res.Current, res.Size = q.Page, q.Size
	return res, nil
}

// Get 详情。不存在→404。
func (s *OperationLogService) Get(ctx context.Context, id uint) (*model.OperationLog, error) {
	e, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("日志不存在")
		}
		return nil, err
	}
	return e, nil
}

// Delete 单条删除。
func (s *OperationLogService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("日志不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

// BatchDelete 批量删除。
func (s *OperationLogService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.repo.BatchDelete(ctx, ids)
}

// Clear 清空全部操作日志（硬删）。
func (s *OperationLogService) Clear(ctx context.Context) error {
	return s.repo.Clear(ctx)
}
