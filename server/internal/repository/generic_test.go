package repository

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/pagination"
	"gva/internal/testutil"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// testItem 泛型基类测试用 GORM 实体：嵌入 model.Model + 软删，模拟真实业务实体结构。
type testItem struct {
	model.Model
	Name   string `gorm:"size:64;not null"`
	Status string `gorm:"size:16;default:active"`
	Code   string `gorm:"size:32"`
	gorm.DeletedAt
}

func (testItem) TableName() string { return "test_items" }

// newGenericTestRepo 构造一个基于 testItem 的泛型仓储，并迁移测试表。
func newGenericTestRepo(t *testing.T) *GenericRepository[testItem] {
	t.Helper()
	db := testutil.NewTestDB(t)
	require.NoError(t, db.AutoMigrate(&testItem{}))
	return NewGenericRepository[testItem](db)
}

func TestGenericRepository_CreateAndFindByID(t *testing.T) {
	repo := newGenericTestRepo(t)
	ctx := context.Background()

	e := &testItem{Name: "alpha", Status: "active", Code: "A"}
	require.NoError(t, repo.Create(ctx, e))
	require.NotZero(t, e.ID)

	got, err := repo.FindByID(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, "alpha", got.Name)
	require.Equal(t, "A", got.Code)
}

func TestGenericRepository_FindByID_NotFound(t *testing.T) {
	repo := newGenericTestRepo(t)
	_, err := repo.FindByID(context.Background(), 9999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestGenericRepository_List_PaginationAndFilter(t *testing.T) {
	repo := newGenericTestRepo(t)
	ctx := context.Background()
	for _, n := range []string{"alpha", "beta", "alpha-2", "gamma"} {
		require.NoError(t, repo.Create(ctx, &testItem{Name: n, Status: "active"}))
	}

	// filter 演示模块特有过滤：name LIKE 'alpha%'
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("name LIKE ?", "alpha%")
	}
	res, err := repo.List(ctx, pagination.Query{Page: 1, Size: 10}, filter)
	require.NoError(t, err)
	require.Equal(t, int64(2), res.Total)
	require.Len(t, res.Records, 2)

	// 无 filter 全量分页
	all, err := repo.List(ctx, pagination.Query{Page: 1, Size: 10}, nil)
	require.NoError(t, err)
	require.Equal(t, int64(4), all.Total)
	require.Len(t, all.Records, 4)

	// 第二页（size=2）应只返回 2 条，total 仍为 4
	page2, err := repo.List(ctx, pagination.Query{Page: 2, Size: 2}, nil)
	require.NoError(t, err)
	require.Equal(t, int64(4), page2.Total)
	require.Len(t, page2.Records, 2)
}

func TestGenericRepository_ListAll(t *testing.T) {
	repo := newGenericTestRepo(t)
	ctx := context.Background()
	for _, n := range []string{"a", "b", "c"} {
		require.NoError(t, repo.Create(ctx, &testItem{Name: n, Status: "active"}))
	}
	all, err := repo.ListAll(ctx, nil)
	require.NoError(t, err)
	require.Len(t, all, 3)

	// ListAll 带 filter
	filtered, err := repo.ListAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", "b")
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
}

func TestGenericRepository_Update(t *testing.T) {
	repo := newGenericTestRepo(t)
	ctx := context.Background()
	e := &testItem{Name: "old", Status: "active"}
	require.NoError(t, repo.Create(ctx, e))

	e.Name = "new"
	e.Status = "inactive"
	require.NoError(t, repo.Update(ctx, e))

	got, err := repo.FindByID(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, "new", got.Name)
	require.Equal(t, "inactive", got.Status)
}

func TestGenericRepository_Delete_SoftDelete(t *testing.T) {
	repo := newGenericTestRepo(t)
	ctx := context.Background()
	e := &testItem{Name: "del", Status: "active"}
	require.NoError(t, repo.Create(ctx, e))

	require.NoError(t, repo.Delete(ctx, e.ID))

	// 软删后默认查询过滤，FindByID 应返回 NotFound
	_, err := repo.FindByID(ctx, e.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestGenericRepository_BatchDelete(t *testing.T) {
	repo := newGenericTestRepo(t)
	ctx := context.Background()
	var ids []uint
	for _, n := range []string{"x", "y", "z"} {
		e := &testItem{Name: n, Status: "active"}
		require.NoError(t, repo.Create(ctx, e))
		ids = append(ids, e.ID)
	}

	require.NoError(t, repo.BatchDelete(ctx, ids))

	all, err := repo.ListAll(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, all)
}

func TestGenericRepository_BatchDelete_EmptyIDs(t *testing.T) {
	repo := newGenericTestRepo(t)
	// 空切片必须直接返回 nil，避免 IN () 语法错误
	require.NoError(t, repo.BatchDelete(context.Background(), nil))
	require.NoError(t, repo.BatchDelete(context.Background(), []uint{}))
}

// TestGenericRepository_Delete_InjectsDeletedBy 验证软删时 DeletedBy 从 ctx 注入。
// GORM 软删 UPDATE 无法经回调注入 deleted_by（见 audit 包），故 repository 层用 Updates 双写。
func TestGenericRepository_Delete_InjectsDeletedBy(t *testing.T) {
	repo := newGenericTestRepo(t)
	ctx := audit.WithUserID(context.Background(), 42)
	e := &testItem{Name: "del", Status: "active"}
	require.NoError(t, repo.Create(ctx, e))

	require.NoError(t, repo.Delete(ctx, e.ID))

	// 软删后默认查询过滤，FindByID 应 NotFound
	_, err := repo.FindByID(ctx, e.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Unscoped 查软删记录验证 deleted_at + deleted_by
	var got testItem
	require.NoError(t, repo.DB(context.Background()).Unscoped().First(&got, e.ID).Error)
	require.True(t, got.Valid, "deleted_at 应已设置")
	require.Equal(t, uint(42), got.DeletedBy, "Delete 应从 ctx 注入 DeletedBy")

	// 无 userID 的 ctx：deleted_by 保持零值
	e2 := &testItem{Name: "no-user", Status: "active"}
	require.NoError(t, repo.Create(context.Background(), e2))
	require.NoError(t, repo.Delete(context.Background(), e2.ID))
	var got2 testItem
	require.NoError(t, repo.DB(context.Background()).Unscoped().First(&got2, e2.ID).Error)
	require.Equal(t, uint(0), got2.DeletedBy, "无 userID 时 deleted_by 应为零值")
}

func TestGenericRepository_BatchDelete_InjectsDeletedBy(t *testing.T) {
	repo := newGenericTestRepo(t)
	ctx := audit.WithUserID(context.Background(), 7)
	var ids []uint
	for _, n := range []string{"x", "y"} {
		e := &testItem{Name: n, Status: "active"}
		require.NoError(t, repo.Create(ctx, e))
		ids = append(ids, e.ID)
	}

	require.NoError(t, repo.BatchDelete(ctx, ids))

	for _, id := range ids {
		var got testItem
		require.NoError(t, repo.DB(context.Background()).Unscoped().First(&got, id).Error)
		require.True(t, got.Valid)
		require.Equal(t, uint(7), got.DeletedBy, "BatchDelete 应注入 DeletedBy")
	}
}
