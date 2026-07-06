// Package service 登录日志业务：查询/删除/清空（日志由 AuthService.Login 自动生成，无 Create/Update）。
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

// LoginLogService 登录日志业务。
type LoginLogService struct {
	repo repository.LoginLogRepository
}

func NewLoginLogService(repo repository.LoginLogRepository) *LoginLogService {
	return &LoginLogService{repo: repo}
}

// List 分页查询（支持 keyword/status/时间范围）。
func (s *LoginLogService) List(ctx context.Context, q repository.LoginLogQuery) (pagination.Result[model.LoginLog], error) {
	q.Normalize()
	res, err := s.repo.List(ctx, q)
	if err != nil {
		return pagination.Result[model.LoginLog]{}, err
	}
	res.Current, res.Size = q.Page, q.Size
	return res, nil
}

// Get 详情。不存在→404。
func (s *LoginLogService) Get(ctx context.Context, id uint) (*model.LoginLog, error) {
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
func (s *LoginLogService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("日志不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

// BatchDelete 批量删除。
func (s *LoginLogService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.repo.BatchDelete(ctx, ids)
}

// Clear 清空全部登录日志（硬删）。
func (s *LoginLogService) Clear(ctx context.Context) error {
	return s.repo.Clear(ctx)
}
