// Package service 字典业务（三级：分类/字典/字典项）。
// 每级标准 CRUD + 分页 + CSV 导出，复用各自的泛型 Repository。
package service

import (
	"context"
	"errors"
	"strconv"

	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/csvutil"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"

	"gorm.io/gorm"
)

// exportAllSize 导出取全量用的页大小（绕过 Normalize 的 100 上限；Paginate 本身不限制）。
const exportAllSize = 100000

// ==================== Level 1: 字典分类 ====================

// DictCategoryUpsertReq 分类创建/更新请求。
type DictCategoryUpsertReq struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"required,oneof=active inactive"`
}

// DictCategoryService 字典分类业务服务。
type DictCategoryService struct {
	repo repository.DictCategoryRepository
}

func NewDictCategoryService(repo repository.DictCategoryRepository) *DictCategoryService {
	return &DictCategoryService{repo: repo}
}

func (s *DictCategoryService) List(ctx context.Context, q pagination.Query) (pagination.Result[model.DictCategory], error) {
	q.Normalize()
	res, err := s.repo.List(ctx, q)
	if err != nil {
		return pagination.Result[model.DictCategory]{}, err
	}
	res.Current, res.Size = q.Page, q.Size
	return res, nil
}

func (s *DictCategoryService) Get(ctx context.Context, id uint) (*model.DictCategory, error) {
	e, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("字典分类不存在")
		}
		return nil, err
	}
	return e, nil
}

func (s *DictCategoryService) Create(ctx context.Context, e *model.DictCategory) error {
	return s.repo.Create(ctx, e)
}
func (s *DictCategoryService) Update(ctx context.Context, e *model.DictCategory) error {
	return s.repo.Update(ctx, e)
}

func (s *DictCategoryService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("字典分类不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *DictCategoryService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.repo.BatchDelete(ctx, ids)
}

func (s *DictCategoryService) Export(ctx context.Context) (string, error) {
	res, err := s.repo.List(ctx, pagination.Query{Page: 1, Size: exportAllSize})
	if err != nil {
		return "", err
	}
	rows := make([]map[string]any, 0, len(res.Records))
	for _, c := range res.Records {
		rows = append(rows, map[string]any{
			"id": strconv.FormatUint(uint64(c.ID), 10), "name": c.Name, "code": c.Code,
			"description": c.Description, "status": c.Status, "createTime": c.CreatedAt, "updateTime": c.UpdatedAt,
		})
	}
	return csvutil.Build(rows, []string{"id", "name", "code", "description", "status", "createTime", "updateTime"}), nil
}

// ==================== Level 2: 字典 ====================

// DictUpsertReq 字典创建/更新请求。
type DictUpsertReq struct {
	CategoryID  uint   `json:"categoryId" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"required,oneof=active inactive"`
}

// DictService 字典业务服务。
type DictService struct {
	repo repository.DictRepository
}

func NewDictService(repo repository.DictRepository) *DictService { return &DictService{repo: repo} }

func (s *DictService) List(ctx context.Context, q pagination.Query, categoryID uint) (pagination.Result[model.Dict], error) {
	q.Normalize()
	res, err := s.repo.List(ctx, q, categoryID)
	if err != nil {
		return pagination.Result[model.Dict]{}, err
	}
	res.Current, res.Size = q.Page, q.Size
	return res, nil
}

func (s *DictService) Get(ctx context.Context, id uint) (*model.Dict, error) {
	e, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("字典不存在")
		}
		return nil, err
	}
	return e, nil
}

func (s *DictService) Create(ctx context.Context, e *model.Dict) error { return s.repo.Create(ctx, e) }
func (s *DictService) Update(ctx context.Context, e *model.Dict) error { return s.repo.Update(ctx, e) }
func (s *DictService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.repo.BatchDelete(ctx, ids)
}

func (s *DictService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("字典不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *DictService) Export(ctx context.Context) (string, error) {
	res, err := s.repo.List(ctx, pagination.Query{Page: 1, Size: exportAllSize}, 0)
	if err != nil {
		return "", err
	}
	rows := make([]map[string]any, 0, len(res.Records))
	for _, d := range res.Records {
		rows = append(rows, map[string]any{
			"id": strconv.FormatUint(uint64(d.ID), 10), "categoryId": d.CategoryID,
			"name": d.Name, "code": d.Code, "description": d.Description,
			"status": d.Status, "createTime": d.CreatedAt, "updateTime": d.UpdatedAt,
		})
	}
	return csvutil.Build(rows, []string{"id", "categoryId", "name", "code", "description", "status", "createTime", "updateTime"}), nil
}

// ==================== Level 3: 字典项 ====================

// DictItemUpsertReq 字典项创建/更新请求。
type DictItemUpsertReq struct {
	DictID uint   `json:"dictId" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Code   string `json:"code" binding:"required"`
	Value  string `json:"value"`
	Sort   int    `json:"sort"`
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

// DictItemService 字典项业务服务。
type DictItemService struct {
	repo repository.DictItemRepository
}

func NewDictItemService(repo repository.DictItemRepository) *DictItemService {
	return &DictItemService{repo: repo}
}

func (s *DictItemService) List(ctx context.Context, q pagination.Query, dictID uint) (pagination.Result[model.DictItem], error) {
	q.Normalize()
	res, err := s.repo.List(ctx, q, dictID)
	if err != nil {
		return pagination.Result[model.DictItem]{}, err
	}
	res.Current, res.Size = q.Page, q.Size
	return res, nil
}

func (s *DictItemService) Get(ctx context.Context, id uint) (*model.DictItem, error) {
	e, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("字典项不存在")
		}
		return nil, err
	}
	return e, nil
}

func (s *DictItemService) Create(ctx context.Context, e *model.DictItem) error {
	return s.repo.Create(ctx, e)
}
func (s *DictItemService) Update(ctx context.Context, e *model.DictItem) error {
	return s.repo.Update(ctx, e)
}
func (s *DictItemService) BatchDelete(ctx context.Context, ids []uint) error {
	return s.repo.BatchDelete(ctx, ids)
}

func (s *DictItemService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("字典项不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *DictItemService) Export(ctx context.Context) (string, error) {
	res, err := s.repo.List(ctx, pagination.Query{Page: 1, Size: exportAllSize}, 0)
	if err != nil {
		return "", err
	}
	rows := make([]map[string]any, 0, len(res.Records))
	for _, it := range res.Records {
		rows = append(rows, map[string]any{
			"id": strconv.FormatUint(uint64(it.ID), 10), "dictId": it.DictID,
			"name": it.Name, "code": it.Code, "value": it.Value, "sort": it.Sort,
			"status": it.Status, "createTime": it.CreatedAt, "updateTime": it.UpdatedAt,
		})
	}
	return csvutil.Build(rows, []string{"id", "dictId", "name", "code", "value", "sort", "status", "createTime", "updateTime"}), nil
}
