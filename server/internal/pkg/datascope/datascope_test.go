package datascope

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeUserLoader 内存版 UserLoader，避免引入 repository 造成循环导入。
type fakeUserLoader struct{ users map[uint]*model.User }

func (f fakeUserLoader) FindByIDWithRoles(_ context.Context, id uint) (*model.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

type fakeDeptLister struct{ depts []model.Dept }

func (f fakeDeptLister) GetAll(context.Context) ([]model.Dept, error) { return f.depts, nil }

// 部门树：root(1) → child(2) → grand(3)。
func sampleDepts() []model.Dept {
	return []model.Dept{
		{Model: model.Model{ID: 1}, ParentID: 0, Name: "root"},
		{Model: model.Model{ID: 2}, ParentID: 1, Name: "child"},
		{Model: model.Model{ID: 3}, ParentID: 2, Name: "grand"},
	}
}

// mkUser 构造带角色（含 DataScope 与权限）的内存用户。
func mkUser(id, deptID uint, roles ...model.Role) *model.User {
	return &model.User{Model: model.Model{ID: id}, Username: "u", Roles: roles, DeptID: deptID}
}

func TestResolve_ZeroUserIDIsAll(t *testing.T) {
	r := NewResolver(fakeUserLoader{}, fakeDeptLister{})
	s, err := r.Resolve(context.Background(), 0)
	require.NoError(t, err)
	assert.True(t, s.All)
}

func TestResolve_SuperAdminWildcardIsAll(t *testing.T) {
	star := model.Permission{Code: "*"}
	super := model.Role{Model: model.Model{ID: 1}, Code: "super", DataScope: ScopeSelf, Permissions: []model.Permission{star}}
	u := mkUser(10, 1, super)
	r := NewResolver(fakeUserLoader{users: map[uint]*model.User{10: u}}, fakeDeptLister{sampleDepts()})
	s, err := r.Resolve(context.Background(), 10)
	require.NoError(t, err)
	assert.True(t, s.All)
}

func TestResolve_DataScopeAllRoleIsAll(t *testing.T) {
	all := model.Role{Model: model.Model{ID: 1}, Code: "all", DataScope: ScopeAll}
	u := mkUser(10, 0, all)
	r := NewResolver(fakeUserLoader{users: map[uint]*model.User{10: u}}, fakeDeptLister{})
	s, err := r.Resolve(context.Background(), 10)
	require.NoError(t, err)
	assert.True(t, s.All)
}

func TestResolve_DeptScope(t *testing.T) {
	dept := model.Role{Model: model.Model{ID: 1}, Code: "dept", DataScope: ScopeDept}
	u := mkUser(10, 2, dept) // 归属 child(2)
	r := NewResolver(fakeUserLoader{users: map[uint]*model.User{10: u}}, fakeDeptLister{sampleDepts()})
	s, err := r.Resolve(context.Background(), 10)
	require.NoError(t, err)
	assert.False(t, s.All)
	assert.False(t, s.Self)
	assert.ElementsMatch(t, []uint{2}, s.Depts)
}

func TestResolve_DeptAndSubScope(t *testing.T) {
	sub := model.Role{Model: model.Model{ID: 1}, Code: "sub", DataScope: ScopeDeptAndSub}
	u := mkUser(10, 2, sub) // 归属 child(2)：可见 child(2) + grand(3)
	r := NewResolver(fakeUserLoader{users: map[uint]*model.User{10: u}}, fakeDeptLister{sampleDepts()})
	s, err := r.Resolve(context.Background(), 10)
	require.NoError(t, err)
	assert.False(t, s.All)
	assert.ElementsMatch(t, []uint{2, 3}, s.Depts)
}

func TestResolve_SelfScope(t *testing.T) {
	self := model.Role{Model: model.Model{ID: 1}, Code: "self", DataScope: ScopeSelf}
	u := mkUser(10, 999, self)
	r := NewResolver(fakeUserLoader{users: map[uint]*model.User{10: u}}, fakeDeptLister{})
	s, err := r.Resolve(context.Background(), 10)
	require.NoError(t, err)
	assert.False(t, s.All)
	assert.True(t, s.Self)
	assert.Equal(t, uint(10), s.UserID)
	assert.Empty(t, s.Depts)
}

// TestResolve_DeptRoleWithoutDeptFallsBackToSelf：dept 角色但用户 DeptID=0 → 退化为 Self。
func TestResolve_DeptRoleWithoutDeptFallsBackToSelf(t *testing.T) {
	dept := model.Role{Model: model.Model{ID: 1}, Code: "dept", DataScope: ScopeDept}
	u := mkUser(10, 0, dept) // 无部门
	r := NewResolver(fakeUserLoader{users: map[uint]*model.User{10: u}}, fakeDeptLister{sampleDepts()})
	s, err := r.Resolve(context.Background(), 10)
	require.NoError(t, err)
	assert.False(t, s.All)
	assert.True(t, s.Self)
	assert.Equal(t, uint(10), s.UserID)
}

// TestResolve_MultiRoleUnion：dept 角色 + self 角色 → 同时可见本部门与本人。
func TestResolve_MultiRoleUnion(t *testing.T) {
	dept := model.Role{Model: model.Model{ID: 1}, Code: "dept", DataScope: ScopeDept}
	self := model.Role{Model: model.Model{ID: 2}, Code: "self", DataScope: ScopeSelf}
	u := mkUser(10, 2, dept, self)
	r := NewResolver(fakeUserLoader{users: map[uint]*model.User{10: u}}, fakeDeptLister{sampleDepts()})
	s, err := r.Resolve(context.Background(), 10)
	require.NoError(t, err)
	assert.False(t, s.All)
	assert.True(t, s.Self)
	assert.ElementsMatch(t, []uint{2}, s.Depts)
}

// TestResolve_UserNotFoundFallsBackToSelf：用户不存在 → 保守退化为仅本人。
func TestResolve_UserNotFoundFallsBackToSelf(t *testing.T) {
	r := NewResolver(fakeUserLoader{users: map[uint]*model.User{}}, fakeDeptLister{})
	s, err := r.Resolve(context.Background(), 999)
	require.NoError(t, err)
	assert.False(t, s.All)
	assert.True(t, s.Self)
	assert.Equal(t, uint(999), s.UserID)
}

// TestApply_OnRealDB 用真实 sqlite 验证 Apply 叠加的 WHERE 实际过滤效果。
// 建 items 表（id/dept_id/owner_id），插 3 行，分别用 All/Depts/Self/空 scope 计数。
func TestApply_OnRealDB(t *testing.T) {
	db := testutil.NewTestDB(t)
	require.NoError(t, db.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, dept_id INTEGER, owner_id INTEGER)").Error)
	ctx := context.Background()
	require.NoError(t, db.WithContext(ctx).Exec(
		"INSERT INTO items (id, dept_id, owner_id) VALUES (1,10,100),(2,20,100),(3,10,200)").Error)

	count := func(s Scope, deptCol, ownerCol string) int64 {
		var n int64
		q := s.Apply(db.WithContext(ctx).Table("items"), deptCol, ownerCol)
		require.NoError(t, q.Count(&n).Error)
		return n
	}

	assert.Equal(t, int64(3), count(Scope{All: true}, "dept_id", "owner_id"))                                  // 全量
	assert.Equal(t, int64(2), count(Scope{Depts: []uint{10}}, "dept_id", "owner_id"))                          // 仅 dept=10
	assert.Equal(t, int64(2), count(Scope{Self: true, UserID: 100}, "", "owner_id"))                           // owner=100
	assert.Equal(t, int64(3), count(Scope{Depts: []uint{10}, Self: true, UserID: 100}, "dept_id", "owner_id")) // dept=10 OR owner=100
	assert.Equal(t, int64(0), count(Scope{}, "", ""))                                                          // 拒绝全部
}
