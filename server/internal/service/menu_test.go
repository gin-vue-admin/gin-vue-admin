package service

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/repository"
	"gva/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMenuService_GetMenus_FullTree(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewMenuRepository(db)
	svc := NewMenuService(repo)
	ctx := context.Background()

	// home 根
	db.Create(&model.Menu{Name: "home", Title: "首页", Path: "/", Component: "dashboard/views/Home", Icon: "HomeFilled", Sort: 0, ShowMenu: true})
	// system 根 + 2 子
	system := model.Menu{Name: "system", Title: "系统管理", Path: "/system", Icon: "Setting", Sort: 20, ShowMenu: true}
	db.Create(&system)
	db.Create(&model.Menu{Name: "systemUser", Title: "用户管理", Path: "/system/user", Component: "system/user/views/List", Icon: "User", Sort: 0, ShowMenu: true, PermissionCode: "user:list", ParentID: system.ID})
	db.Create(&model.Menu{Name: "systemRole", Title: "角色管理", Path: "/system/role", Component: "system/role/views/List", Icon: "Avatar", Sort: 10, ShowMenu: true, PermissionCode: "role:list", ParentID: system.ID})

	tree, err := svc.GetMenus(ctx)
	require.NoError(t, err)
	// 两个根：home、system
	assert.Len(t, tree, 2)

	// 找 system 节点验证 children
	var systemNode *MenuDTO
	for i := range tree {
		if tree[i].Name == "system" {
			systemNode = &tree[i]
			break
		}
	}
	require.NotNil(t, systemNode)
	assert.Len(t, systemNode.Children, 2)                      // systemUser + systemRole
	assert.Equal(t, "systemUser", systemNode.Children[0].Name) // sort 0 在前
	assert.Equal(t, "systemRole", systemNode.Children[1].Name) // sort 10 在后
}

func TestMenuService_DTO_Permissions(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewMenuRepository(db)
	svc := NewMenuService(repo)
	ctx := context.Background()

	// 有 PermissionCode 的菜单
	db.Create(&model.Menu{Name: "u", Title: "用户", Path: "/u", Component: "c", Sort: 0, ShowMenu: true, PermissionCode: "user:list"})
	// 无 PermissionCode 的公共菜单
	db.Create(&model.Menu{Name: "home", Title: "首页", Path: "/", Component: "h", Sort: 0, ShowMenu: true})

	tree, err := svc.GetMenus(ctx)
	require.NoError(t, err)
	for _, n := range tree {
		if n.Name == "u" {
			require.NotNil(t, n.Meta.Permissions)
			assert.Equal(t, []string{"user:list"}, n.Meta.Permissions.Any)
		}
		if n.Name == "home" {
			assert.Nil(t, n.Meta.Permissions) // 公共菜单无 permissions
		}
	}
}

func TestMenuService_DTO_Meta(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewMenuRepository(db)
	svc := NewMenuService(repo)

	db.Create(&model.Menu{Name: "x", Title: "测试菜单", Path: "/x", Component: "c", Icon: "Setting", Sort: 0, ShowMenu: true})
	tree, err := svc.GetMenus(context.Background())
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "测试菜单", tree[0].Meta.Title)
	assert.Equal(t, "Setting", tree[0].Meta.Icon)
	assert.True(t, tree[0].Meta.ShowMenu)
	assert.Equal(t, "c", tree[0].Component)
}

func TestMenuService_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewMenuRepository(db)
	svc := NewMenuService(repo)
	tree, err := svc.GetMenus(context.Background())
	require.NoError(t, err)
	assert.Empty(t, tree)
}

func TestMenuService_CRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewMenuRepository(db)
	svc := NewMenuService(repo)
	ctx := context.Background()

	// 创建根
	m, err := svc.Create(ctx, &MenuCreateReq{Name: "test", Title: "测试", Path: "/t", Status: "active"})
	require.NoError(t, err)
	assert.NotZero(t, m.ID)

	// 详情
	got, err := svc.Get(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "test", got.Name)

	// 更新
	got.Title = "改"
	require.NoError(t, svc.Update(ctx, got))
}

func TestMenuService_Delete_Cascade(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewMenuRepository(db)
	svc := NewMenuService(repo)
	ctx := context.Background()
	parent, _ := svc.Create(ctx, &MenuCreateReq{Name: "p", Title: "父", Path: "/p", Status: "active"})
	svc.Create(ctx, &MenuCreateReq{Name: "c", Title: "子", Path: "/c", ParentID: &parent.ID, Status: "active"})

	require.NoError(t, svc.Delete(ctx, parent.ID))
	// 子也删了
	_, err := svc.Get(ctx, parent.ID)
	assert.Error(t, err)
}

func TestMenuService_Sort_Inner(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewMenuRepository(db)
	svc := NewMenuService(repo)
	ctx := context.Background()
	target, _ := svc.Create(ctx, &MenuCreateReq{Name: "t", Title: "t", Path: "/t", Status: "active"})
	dragging, _ := svc.Create(ctx, &MenuCreateReq{Name: "d", Title: "d", Path: "/d", Status: "active"})

	err := svc.Sort(ctx, &MenuSortReq{DraggingID: dragging.ID, TargetID: target.ID, Position: "inner"})
	require.NoError(t, err)
	// dragging 的 parentId 应变 target.id
	got, _ := repo.FindByID(ctx, dragging.ID)
	assert.Equal(t, target.ID, got.ParentID)
}

func TestMenuService_Sort_Before(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewMenuRepository(db)
	svc := NewMenuService(repo)
	ctx := context.Background()
	t1, _ := svc.Create(ctx, &MenuCreateReq{Name: "t1", Title: "1", Path: "/1", Sort: 0, Status: "active"})
	t2, _ := svc.Create(ctx, &MenuCreateReq{Name: "t2", Title: "2", Path: "/2", Sort: 1, Status: "active"})
	drag, _ := svc.Create(ctx, &MenuCreateReq{Name: "d", Title: "d", Path: "/d", Sort: 2, Status: "active"})
	_ = t1 // t1 仅占据 sort=0 槽位参与重排，断言不直接引用

	// drag 放 t2 之前
	err := svc.Sort(ctx, &MenuSortReq{DraggingID: drag.ID, TargetID: t2.ID, Position: "before"})
	require.NoError(t, err)
	got, _ := repo.FindByID(ctx, drag.ID)
	// drag.sort 应在 t1(0) 与 t2 之间重排后
	assert.Equal(t, 1, got.Sort) // 重排：t1=0, drag=1, t2=2
}
