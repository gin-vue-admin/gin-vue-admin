package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gva/internal/model"
	"gva/internal/repository"
	"gva/internal/testutil"
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
	assert.Len(t, systemNode.Children, 2)                 // systemUser + systemRole
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
