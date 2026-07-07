package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gva/internal/middleware"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/csvutil"
	"gva/internal/pkg/datascope"
	"gva/internal/pkg/gormx"
	"gva/internal/pkg/hash"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"

	"gorm.io/gorm"
)

// UserInfo 用户响应 DTO，对齐前端 UserInfo 多角色契约。
type UserInfo struct {
	ID            uint     `json:"id"`
	Username      string   `json:"username"`
	RealName      string   `json:"realName"`
	Email         string   `json:"email"`
	Phone         string   `json:"phone"`
	Roles         []string `json:"roles"`
	Status        string   `json:"status"`
	Avatar        string   `json:"avatar"`
	CreateTime    string   `json:"createTime"`
	LastLoginTime string   `json:"lastLoginTime"`
	LoginCount    int      `json:"loginCount"`
}

// UserCreateReq 创建用户请求。
type UserCreateReq struct {
	Username string   `json:"username" binding:"required,min=3,max=64"`
	RealName string   `json:"realName" binding:"max=64"`
	Email    string   `json:"email" binding:"omitempty,email,max=128"`
	Phone    string   `json:"phone" binding:"max=32"`
	Roles    []string `json:"roles" binding:"required,min=1"`
	Status   string   `json:"status" binding:"required,oneof=active inactive"`
	Password string   `json:"password" binding:"required,min=6,max=72"`
}

// UserUpdateReq 更新用户请求。指针字段 nil=不改，传值=改；Roles 切片 nil=不改。
type UserUpdateReq struct {
	RealName *string  `json:"realName"`
	Email    *string  `json:"email"`
	Phone    *string  `json:"phone"`
	Roles    []string `json:"roles"`
	Status   *string  `json:"status"`
	Password *string  `json:"password"`
}

// UserService 用户业务：CRUD + list + export，支持多角色、防自删/自禁。
// resolver 用于 List/Export 时按当前用户角色推导数据范围；nil 时不过滤（全量，供测试/未启用场景）。
type UserService struct {
	repo     repository.UserRepository
	resolver *datascope.Resolver
}

// NewUserService 构造用户业务。resolver 可为 nil（关闭数据范围过滤）。
func NewUserService(repo repository.UserRepository, resolver *datascope.Resolver) *UserService {
	return &UserService{repo: repo, resolver: resolver}
}

// resolveScope 取当前用户的数据范围。resolver 为 nil 或 ctx 无 userID → All（全量）。
func (s *UserService) resolveScope(ctx context.Context) datascope.Scope {
	if s.resolver == nil {
		return datascope.Scope{All: true}
	}
	uid, ok := audit.UserIDFrom(ctx)
	if !ok || uid == 0 {
		return datascope.Scope{All: true}
	}
	scope, err := s.resolver.Resolve(ctx, uid)
	if err != nil {
		// 解析异常保守退化为仅本人（Resolver 内部已兜底，此处双保险）。
		return datascope.Scope{Self: true, UserID: uid}
	}
	return scope
}

// toUserInfo User→UserInfo 转换：Roles→code 数组，LastLoginAt→RFC3339 字符串（空则 ""）。
func toUserInfo(u *model.User) UserInfo {
	roles := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, r.Code)
	}
	lastLogin := ""
	if u.LastLoginAt.Valid {
		lastLogin = u.LastLoginAt.Time.Format(time.RFC3339)
	}
	return UserInfo{
		ID:            u.ID,
		Username:      u.Username,
		RealName:      u.RealName,
		Email:         u.Email,
		Phone:         u.Phone,
		Roles:         roles,
		Status:        u.Status,
		Avatar:        u.Avatar,
		CreateTime:    u.CreatedAt.Format(time.RFC3339),
		LastLoginTime: lastLogin,
		LoginCount:    u.LoginCount,
	}
}

// List 分页列表（含 roles code）。按当前用户数据范围过滤。
func (s *UserService) List(ctx context.Context, q pagination.Query, roleCode string) (pagination.Result[UserInfo], error) {
	q.Normalize()
	scope := s.resolveScope(ctx)
	users, total, err := s.repo.List(ctx, q, roleCode, scope)
	if err != nil {
		return pagination.Result[UserInfo]{}, err
	}
	infos := make([]UserInfo, 0, len(users))
	for i := range users {
		infos = append(infos, toUserInfo(&users[i]))
	}
	return pagination.Result[UserInfo]{Records: infos, Total: total, Current: q.Page, Size: q.Size}, nil
}

// Get 详情。不存在→404。
func (s *UserService) Get(ctx context.Context, id uint) (UserInfo, error) {
	u, err := s.repo.FindByIDWithRoles(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserInfo{}, apperr.NotFound("用户不存在")
		}
		return UserInfo{}, err
	}
	return toUserInfo(u), nil
}

// Create 创建。username 重复→409，未知角色→404，密码 bcrypt 哈希；
// 成功后替换角色、失效权限缓存、重查返回完整 UserInfo。
func (s *UserService) Create(ctx context.Context, username, realName, email, phone string, roleCodes []string, status, password string) (UserInfo, error) {
	roleIDs, err := s.resolveRoleIDs(ctx, roleCodes)
	if err != nil {
		return UserInfo{}, err
	}
	hashed, err := hash.Hash(password)
	if err != nil {
		return UserInfo{}, err
	}
	u := &model.User{
		Username: username,
		Password: hashed,
		RealName: realName,
		Email:    email,
		Phone:    phone,
		Status:   status,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		if gormx.IsDuplicateKey(err) {
			return UserInfo{}, apperr.Conflict("用户名已存在")
		}
		return UserInfo{}, err
	}
	if err := s.repo.ReplaceRoles(ctx, u.ID, roleIDs); err != nil {
		return UserInfo{}, err
	}
	middleware.InvalidateAll()
	// 重查带 Roles 的完整数据返回
	full, err := s.repo.FindByIDWithRoles(ctx, u.ID)
	if err != nil {
		return UserInfo{}, err
	}
	return toUserInfo(full), nil
}

// Update 更新。operatorID 用于防自禁；指针字段 nil 不改。
// 改角色或状态后失效权限缓存。
func (s *UserService) Update(ctx context.Context, id, operatorID uint, req *UserUpdateReq) (UserInfo, error) {
	u, err := s.repo.FindByIDWithRoles(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserInfo{}, apperr.NotFound("用户不存在")
		}
		return UserInfo{}, err
	}
	// 已知技术债：FindByIDWithRoles 与 Update(Save) 之间存在 TOCTOU 窗口，
	// 并发删+更可能复活软删记录。与 M3.1/M3.2 同类，后续统一用乐观锁解决。
	// 防自禁
	if req.Status != nil && *req.Status == "inactive" && id == operatorID {
		return UserInfo{}, apperr.Conflict("不能禁用自己")
	}
	if req.RealName != nil {
		u.RealName = *req.RealName
	}
	if req.Email != nil {
		u.Email = *req.Email
	}
	if req.Phone != nil {
		u.Phone = *req.Phone
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	if req.Password != nil && *req.Password != "" {
		hashed, err := hash.Hash(*req.Password)
		if err != nil {
			return UserInfo{}, err
		}
		u.Password = hashed
	}
	if err := s.repo.Update(ctx, u); err != nil {
		if gormx.IsDuplicateKey(err) {
			return UserInfo{}, apperr.Conflict("用户名已存在")
		}
		return UserInfo{}, err
	}
	if req.Roles != nil {
		roleIDs, err := s.resolveRoleIDs(ctx, req.Roles)
		if err != nil {
			return UserInfo{}, err
		}
		if err := s.repo.ReplaceRoles(ctx, u.ID, roleIDs); err != nil {
			return UserInfo{}, err
		}
		middleware.InvalidateAll()
	}
	full, err := s.repo.FindByIDWithRoles(ctx, id)
	if err != nil {
		return UserInfo{}, err
	}
	return toUserInfo(full), nil
}

// Delete 软删+清关联。防自删。
func (s *UserService) Delete(ctx context.Context, id, operatorID uint) error {
	if id == operatorID {
		return apperr.Conflict("不能删除自己")
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("用户不存在")
		}
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	middleware.InvalidateAll()
	return nil
}

// BatchDelete 批量软删。任一 id==operatorID→409。
func (s *UserService) BatchDelete(ctx context.Context, ids []uint, operatorID uint) error {
	for _, id := range ids {
		if id == operatorID {
			return apperr.Conflict("不能删除自己")
		}
	}
	if err := s.repo.BatchDelete(ctx, ids); err != nil {
		return err
	}
	middleware.InvalidateAll()
	return nil
}

// Export 生成 CSV。用 ListAll 取全量不分页，按当前用户数据范围过滤，避免漏数据或越权。
func (s *UserService) Export(ctx context.Context, q pagination.Query, roleCode string) (string, error) {
	scope := s.resolveScope(ctx)
	users, err := s.repo.ListAll(ctx, q, roleCode, scope)
	if err != nil {
		return "", err
	}
	rows := make([]map[string]any, 0, len(users))
	for i := range users {
		u := &users[i]
		codes := make([]string, 0, len(u.Roles))
		for _, r := range u.Roles {
			codes = append(codes, r.Code)
		}
		lastLogin := ""
		if u.LastLoginAt.Valid {
			lastLogin = u.LastLoginAt.Time.Format(time.RFC3339)
		}
		rows = append(rows, map[string]any{
			"username":      u.Username,
			"realName":      u.RealName,
			"email":         u.Email,
			"phone":         u.Phone,
			"roles":         joinCodes(codes),
			"status":        u.Status,
			"createTime":    u.CreatedAt.Format(time.RFC3339),
			"lastLoginTime": lastLogin,
			"loginCount":    strconv.Itoa(u.LoginCount),
		})
	}
	return csvutil.Build(rows, []string{"username", "realName", "email", "phone", "roles", "status", "createTime", "lastLoginTime", "loginCount"}), nil
}

// joinCodes 角色码用分号拼接（CSV 单列）。
func joinCodes(codes []string) string {
	return strings.Join(codes, ";")
}

// resolveRoleIDs 角色 code→id，严格校验：未知 code→404。
func (s *UserService) resolveRoleIDs(ctx context.Context, codes []string) ([]uint, error) {
	m, err := s.repo.FindRoleIDsByCodes(ctx, codes)
	if err != nil {
		return nil, err
	}
	for _, c := range codes {
		if _, ok := m[c]; !ok {
			return nil, apperr.NotFound("角色 " + c + " 不存在")
		}
	}
	ids := make([]uint, 0, len(codes))
	for _, c := range codes {
		ids = append(ids, m[c])
	}
	return ids, nil
}
