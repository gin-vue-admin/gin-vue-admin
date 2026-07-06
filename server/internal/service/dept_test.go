package service

import (
	"context"
	"testing"

	"gva/internal/repository"
	"gva/internal/testutil"

	"github.com/stretchr/testify/require"
)

// newDeptSvc 构造测试用 DeptService（NewTestDB 的 AutoMigrate 已含 Dept）。
func newDeptSvc(t *testing.T) *DeptService {
	t.Helper()
	return NewDeptService(repository.NewDeptRepository(testutil.NewTestDB(t)))
}

// TestDeptService_TreeBuildAndCascadeDelete 验证：树构建正确 + 级联删除整棵子树。
func TestDeptService_TreeBuildAndCascadeDelete(t *testing.T) {
	svc := newDeptSvc(t)
	ctx := context.Background()

	// 建树：总部 → 技术部 → 前端组；总部 → 市场部
	zb, err := svc.Create(ctx, &DeptUpsertReq{Name: "总部", Status: "active", Sort: 0})
	require.NoError(t, err)
	js, err := svc.Create(ctx, &DeptUpsertReq{ParentID: &zb.ID, Name: "技术部", Status: "active", Sort: 0})
	require.NoError(t, err)
	_, err = svc.Create(ctx, &DeptUpsertReq{ParentID: &js.ID, Name: "前端组", Status: "active"})
	require.NoError(t, err)
	_, err = svc.Create(ctx, &DeptUpsertReq{ParentID: &zb.ID, Name: "市场部", Status: "active", Sort: 1})
	require.NoError(t, err)

	// List 建树：根只有"总部"，下挂技术部+市场部，技术部下挂前端组
	tree, err := svc.List(ctx, "", "")
	require.NoError(t, err)
	require.Len(t, tree, 1)
	require.Equal(t, "总部", tree[0].Name)
	require.Len(t, tree[0].Children, 2)

	var jsInfo *DeptInfo
	for i := range tree[0].Children {
		if tree[0].Children[i].Name == "技术部" {
			jsInfo = &tree[0].Children[i]
		}
	}
	require.NotNil(t, jsInfo)
	require.Len(t, jsInfo.Children, 1)
	require.Equal(t, "前端组", jsInfo.Children[0].Name)

	// 级联删"总部"：4 个节点全部删除
	require.NoError(t, svc.Delete(ctx, zb.ID))
	tree2, _ := svc.List(ctx, "", "")
	require.Empty(t, tree2)
}

// TestDeptService_FilterOrphanPromotedToRoot 过滤产生的孤儿（父被过滤）提升为根。
func TestDeptService_FilterOrphanPromotedToRoot(t *testing.T) {
	svc := newDeptSvc(t)
	ctx := context.Background()
	zb, _ := svc.Create(ctx, &DeptUpsertReq{Name: "总部", Status: "active"})
	_, _ = svc.Create(ctx, &DeptUpsertReq{ParentID: &zb.ID, Name: "技术部", Status: "active"})

	// keyword="技术部" → "总部"被过滤，"技术部"成孤儿，提升为根
	tree, err := svc.List(ctx, "技术部", "")
	require.NoError(t, err)
	require.Len(t, tree, 1)
	require.Equal(t, "技术部", tree[0].Name)
}

// TestDeptService_StatusFilter 验证 status 过滤。
func TestDeptService_StatusFilter(t *testing.T) {
	svc := newDeptSvc(t)
	ctx := context.Background()
	svc.Create(ctx, &DeptUpsertReq{Name: "活跃部", Status: "active"})
	svc.Create(ctx, &DeptUpsertReq{Name: "停用部", Status: "inactive"})

	tree, err := svc.List(ctx, "", "inactive")
	require.NoError(t, err)
	require.Len(t, tree, 1)
	require.Equal(t, "停用部", tree[0].Name)
}

// TestDeptService_DeleteNotFound 删除不存在→404。
func TestDeptService_DeleteNotFound(t *testing.T) {
	svc := newDeptSvc(t)
	err := svc.Delete(context.Background(), 9999)
	require.Error(t, err)
}
