// Package datascope 实现按部门控制可见数据的数据权限（对标 RuoYi data scope）。
//
// 设计要点：
//   - Scope 是已解析的权限范围：All（全量）/ Depts（部门集合）/ Self（仅本人）。
//   - Resolver 由当前用户角色推导 Scope：任一角色含权限码 "*" 或 DataScope=all → All；
//     否则按各角色 DataScope（dept/dept_and_sub/self）并集取最宽。
//   - Apply 把 Scope 翻译成 SQL WHERE，供 repository 在 List/Count 时叠加。
//     deptCol 为实体的部门外键列（如 "dept_id"），creatorCol 为"本人"判定列
//     （User 实体传 "id"，业务实体传 "created_by"）。空串表示该实体无此列，跳过对应分支。
//
// 仅依赖 model + gorm，不引入 HTTP 层，便于在 service/repository 复用与单测。
package datascope

import (
	"context"
	"strings"

	"gva/internal/model"

	"gorm.io/gorm"
)

// 数据范围枚举（与 model.Role.DataScope 字段值对齐）。
const (
	ScopeAll        = "all"
	ScopeDept       = "dept"
	ScopeDeptAndSub = "dept_and_sub"
	ScopeSelf       = "self"
)

// Scope 已解析的数据权限范围。All=true 时其余字段忽略。
type Scope struct {
	All    bool
	Depts  []uint // 部门 id 集合（dept/dept_and_sub 并集）
	Self   bool   // 仅本人
	UserID uint   // Self 时所用（实体的"本人"列值）
}

// Apply 把 Scope 叠加到 gorm 查询。
//   - All → 不加条件
//   - Depts 非空且 deptCol != "" → deptCol IN ?
//   - Self 且 creatorCol != "" 且 UserID!=0 → creatorCol = ?
//   - 以上均无可应用分支 → WHERE 1=0（最小权限，拒绝全部）
//
// 多分支以 OR 组合（如：本部门 OR 本人）。
func (s Scope) Apply(db *gorm.DB, deptCol, creatorCol string) *gorm.DB {
	if s.All {
		return db
	}
	var ors []string
	var args []any
	if len(s.Depts) > 0 && deptCol != "" {
		ors = append(ors, deptCol+" IN ?")
		args = append(args, s.Depts)
	}
	if s.Self && creatorCol != "" && s.UserID != 0 {
		ors = append(ors, creatorCol+" = ?")
		args = append(args, s.UserID)
	}
	if len(ors) == 0 {
		return db.Where("1 = 0")
	}
	return db.Where(strings.Join(ors, " OR "), args...)
}

// UserLoader 解析所需的最小用户读取接口（repository.UserRepository 即满足）。
type UserLoader interface {
	FindByIDWithRoles(ctx context.Context, id uint) (*model.User, error)
}

// DeptLister 解析所需的最小部门读取接口（repository.DeptRepository 即满足）。
type DeptLister interface {
	GetAll(ctx context.Context) ([]model.Dept, error)
}

// Resolver 数据权限解析器。用 userLoader 取用户角色与部门，用 deptLister 推导子孙部门。
type Resolver struct {
	users UserLoader
	depts DeptLister
}

// NewResolver 构造解析器。
func NewResolver(users UserLoader, depts DeptLister) *Resolver {
	return &Resolver{users: users, depts: depts}
}

// Resolve 由 userID 推导其数据权限 Scope。
//   - userID==0（无认证身份，如系统/测试）→ All：受保护路由的中间件恒注入 userID，
//     缺失仅出现在非业务路径，默认放行避免误伤；生产不应由未鉴权请求触达。
//   - 用户任一角色权限码含 "*"（超管）→ All
//   - 否则并集各角色 DataScope：dept→{本人部门}；dept_and_sub→{本人部门+子孙}；self→Self=true
//   - 任一角色 all → All
//   - 推导结果 Depts 为空且非 Self → 退化为 Self（最小可见，避免无部门用户看到全部）
func (r *Resolver) Resolve(ctx context.Context, userID uint) (Scope, error) {
	if userID == 0 {
		return Scope{All: true}, nil
	}
	user, err := r.users.FindByIDWithRoles(ctx, userID)
	if err != nil {
		// 用户不存在等异常：保守退化为仅本人，不返回 All。
		return Scope{Self: true, UserID: userID}, nil
	}
	// 超管短路：任一角色权限码含 "*"
	if hasWildcard(user) {
		return Scope{All: true}, nil
	}

	scope := Scope{UserID: userID}
	deptSet := make(map[uint]struct{})
	needTree := false // 仅在有 dept_and_sub 时才加载部门树，避免无谓查询
	for _, role := range user.Roles {
		switch role.DataScope {
		case ScopeAll:
			return Scope{All: true}, nil
		case ScopeDept:
			if user.DeptID != 0 {
				deptSet[user.DeptID] = struct{}{}
			}
		case ScopeDeptAndSub:
			if user.DeptID != 0 {
				deptSet[user.DeptID] = struct{}{}
				needTree = true
			}
		case ScopeSelf:
			scope.Self = true
		}
	}

	if needTree {
		all, err := r.depts.GetAll(ctx)
		if err == nil {
			for _, d := range collectDescendants(all, user.DeptID) {
				deptSet[d] = struct{}{}
			}
		}
		// 加载失败不阻断查询：仅退化为本部门（已入 deptSet），记录由调用方负责
	}
	for d := range deptSet {
		scope.Depts = append(scope.Depts, d)
	}

	// 无任何可见分支 → 退化为仅本人，杜绝越权看到全量
	if !scope.Self && len(scope.Depts) == 0 {
		scope.Self = true
	}
	return scope, nil
}

// hasWildcard 检测用户的任一角色是否持有通配权限 "*"。
func hasWildcard(u *model.User) bool {
	for _, r := range u.Roles {
		for _, p := range r.Permissions {
			if p.Code == "*" {
				return true
			}
		}
	}
	return false
}

// collectDescendants 递归收集 parentID 的所有后代部门 id（不含 parentID 自身）。
// 与 service.collectDeptDescendants 同构；此处内联以保持包独立。
func collectDescendants(all []model.Dept, parentID uint) []uint {
	var out []uint
	var rec func(pid uint)
	rec = func(pid uint) {
		for _, d := range all {
			if d.ParentID == pid {
				out = append(out, d.ID)
				rec(d.ID)
			}
		}
	}
	rec(parentID)
	return out
}
