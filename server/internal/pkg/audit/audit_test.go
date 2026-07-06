package audit

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/testutil"

	"github.com/stretchr/testify/require"
)

// auditItem 测试专用实体（嵌入 model.Model 获得审计字段）。
type auditItem struct {
	model.Model
	Name string `gorm:"size:64"`
}

func (auditItem) TableName() string { return "audit_items" }

func TestUserIDFrom(t *testing.T) {
	ctx := WithUserID(context.Background(), 7)
	uid, ok := UserIDFrom(ctx)
	require.True(t, ok)
	require.Equal(t, uint(7), uid)

	_, ok = UserIDFrom(context.Background())
	require.False(t, ok)
}

func TestRegister_AutoFillsCreatedByUpdatedBy(t *testing.T) {
	db := testutil.NewTestDB(t)
	require.NoError(t, db.AutoMigrate(&auditItem{}))
	Register(db)

	// 带 userID 的 ctx：Create 自动写 CreatedBy + UpdatedBy
	ctx := WithUserID(context.Background(), 42)
	e := &auditItem{Name: "x"}
	require.NoError(t, db.WithContext(ctx).Create(e).Error)
	require.Equal(t, uint(42), e.CreatedBy)
	require.Equal(t, uint(42), e.UpdatedBy)

	// Update（Save）刷新 UpdatedBy
	e.Name = "y"
	require.NoError(t, db.WithContext(ctx).Save(e).Error)
	require.Equal(t, uint(42), e.UpdatedBy)

	// 无 userID 的 ctx：不注入，字段保持零值
	e2 := &auditItem{Name: "no-user"}
	require.NoError(t, db.WithContext(context.Background()).Create(e2).Error)
	require.Equal(t, uint(0), e2.CreatedBy)
}
