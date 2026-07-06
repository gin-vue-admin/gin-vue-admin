package repository

import (
	"context"

	"gva/internal/model"

	"gorm.io/gorm"
)

// DeptRepository 部门数据访问。
// 嵌入 GenericRepository[model.Dept] 复用 FindByID/Create/Update/BatchDelete；
// 仅新增 GetAll（全量扁平，按 parent_id/sort 排序，供 service 构建树）。
type DeptRepository interface {
	GetAll(ctx context.Context) ([]model.Dept, error)
	FindByID(ctx context.Context, id uint) (*model.Dept, error)
	Create(ctx context.Context, e *model.Dept) error
	Update(ctx context.Context, e *model.Dept) error
	BatchDelete(ctx context.Context, ids []uint) error
}

// deptRepository 嵌入 GenericRepository[model.Dept]：FindByID/Create/Update/BatchDelete 全部复用基类。
type deptRepository struct {
	*GenericRepository[model.Dept]
}

// NewDeptRepository 构造部门仓储。
func NewDeptRepository(db *gorm.DB) DeptRepository {
	return &deptRepository{GenericRepository: NewGenericRepository[model.Dept](db)}
}

// GetAll 全量扁平（按 parent_id, sort 排序），供 service 构建树与级联删遍历。
func (r *deptRepository) GetAll(ctx context.Context) ([]model.Dept, error) {
	var depts []model.Dept
	err := r.DB(ctx).Order("parent_id, sort").Find(&depts).Error
	return depts, err
}
