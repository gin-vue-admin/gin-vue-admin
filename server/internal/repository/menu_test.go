package repository

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMenuRepo_CRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewMenuRepository(db)
	ctx := context.Background()
	m := &model.Menu{Name: "test", Title: "测试", Path: "/t", Status: "active"}
	require.NoError(t, repo.Create(ctx, m))
	assert.NotZero(t, m.ID)

	got, err := repo.FindByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "test", got.Name)

	got.Title = "改"
	require.NoError(t, repo.Update(ctx, got))
	got2, _ := repo.FindByID(ctx, m.ID)
	assert.Equal(t, "改", got2.Title)
}

func TestMenuRepo_DeleteByIDs_Cascade(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewMenuRepository(db)
	ctx := context.Background()
	parent := &model.Menu{Name: "p", Title: "父", Path: "/p", Status: "active"}
	repo.Create(ctx, parent)
	child := &model.Menu{Name: "c", Title: "子", Path: "/c", ParentID: parent.ID, Status: "active"}
	repo.Create(ctx, child)
	// 级联删父子
	require.NoError(t, repo.DeleteByIDs(ctx, []uint{parent.ID, child.ID}))
	_, err := repo.FindByID(ctx, parent.ID)
	assert.Error(t, err)
	_, err = repo.FindByID(ctx, child.ID)
	assert.Error(t, err)
}

func TestMenuRepo_UpdateSorts(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewMenuRepository(db)
	ctx := context.Background()
	m1 := &model.Menu{Name: "m1", Title: "1", Path: "/1", Sort: 0, Status: "active"}
	m2 := &model.Menu{Name: "m2", Title: "2", Path: "/2", Sort: 1, Status: "active"}
	repo.Create(ctx, m1)
	repo.Create(ctx, m2)
	// 重排：m2.sort=0, m1.sort=1
	m1.Sort = 1
	m2.Sort = 0
	require.NoError(t, repo.UpdateSorts(ctx, []model.Menu{*m1, *m2}))
	got1, _ := repo.FindByID(ctx, m1.ID)
	assert.Equal(t, 1, got1.Sort)
	got2, _ := repo.FindByID(ctx, m2.ID)
	assert.Equal(t, 0, got2.Sort)
}
