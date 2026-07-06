package repository

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/pkg/pagination"
	"gva/internal/testutil"

	"github.com/stretchr/testify/require"
)

// newCrudRepo 构造测试用 CrudRepository（NewTestDB 的 AutoMigrate 已含 CrudItem）。
func newCrudRepo(t *testing.T) CrudRepository {
	t.Helper()
	return NewCrudRepository(testutil.NewTestDB(t))
}

func TestCrudRepository_CRUDAndList(t *testing.T) {
	repo := newCrudRepo(t)
	ctx := context.Background()

	// Create
	e := &model.CrudItem{Name: "王小虎", Province: "上海", City: "普陀区", Address: "金沙江路 1518 弄", Zip: 200333, Date: "2026-07-04"}
	require.NoError(t, repo.Create(ctx, e))
	require.NotZero(t, e.ID)

	// FindByID
	got, err := repo.FindByID(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, "王小虎", got.Name)
	require.Equal(t, "上海", got.Province)

	// Update
	got.City = "静安区"
	require.NoError(t, repo.Update(ctx, got))
	updated, _ := repo.FindByID(ctx, e.ID)
	require.Equal(t, "静安区", updated.City)

	// 再建一条以验证 List 总数
	require.NoError(t, repo.Create(ctx, &model.CrudItem{Name: "张三", Date: "2026-07-04"}))

	// List 全量
	res, err := repo.List(ctx, pagination.Query{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), res.Total)
	require.Len(t, res.Records, 2)

	// List keyword 过滤
	resWang, err := repo.List(ctx, pagination.Query{Keyword: "王小虎", Page: 1, Size: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), resWang.Total)

	// Delete（软删）
	require.NoError(t, repo.Delete(ctx, e.ID))
	_, err = repo.FindByID(ctx, e.ID)
	require.Error(t, err)
}

func TestCrudRepository_BatchDelete(t *testing.T) {
	repo := newCrudRepo(t)
	ctx := context.Background()
	var ids []uint
	for _, n := range []string{"a", "b", "c"} {
		e := &model.CrudItem{Name: n, Date: "2026-07-04"}
		require.NoError(t, repo.Create(ctx, e))
		ids = append(ids, e.ID)
	}
	require.NoError(t, repo.BatchDelete(ctx, ids))

	res, err := repo.List(ctx, pagination.Query{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Equal(t, int64(0), res.Total)
}
