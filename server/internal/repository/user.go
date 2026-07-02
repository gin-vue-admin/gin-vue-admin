// Package repository 封装数据访问。仅对含分支业务逻辑的领域抽接口（为单测可注入 mock）。
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gva/internal/model"
)

// UserRepository 用户数据访问接口。
// 抽接口目的：auth service 含登录/刷新等分支逻辑，需在单测中用 mock 替换 DB。
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByID(ctx context.Context, id uint) (*model.User, error)
	FindByIDWithRoles(ctx context.Context, id uint) (*model.User, error)
	UpdateLoginStats(ctx context.Context, id uint) error
}

// userRepository gorm 实现。
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByIDWithRoles 预加载 Roles 及其 Permissions（双层），供 GetProfile 构造 profile。
func (r *userRepository) FindByIDWithRoles(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).
		Preload("Roles.Permissions").
		First(&u, id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateLoginStats 更新最后登录时间与登录次数。失败仅由调用方告警，不影响登录主流程。
func (r *userRepository) UpdateLoginStats(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).
		Updates(map[string]any{
			"last_login_at": time.Now(),
			"login_count":   gorm.Expr("login_count + 1"),
		}).Error
}
