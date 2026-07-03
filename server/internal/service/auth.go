// Package service 承载业务逻辑，依赖 repository 接口与 pkg 工具，不依赖 HTTP 层。
// 本任务先实现 Seed + 骨架；Login/Refresh/GetProfile/Logout 在 Task 4/5 用 TDD 补全。
package service

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/async"
	"gva/internal/pkg/hash"
	"gva/internal/pkg/jwt"
	"gva/internal/pkg/log"
	"gva/internal/repository"
)

// AuthResult 登录/刷新返回，对齐前端 AuthResult。
type AuthResult struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

// UserProfile 当前用户信息，对齐前端 UserProfile / mock SafeUser。
type UserProfile struct {
	ID          uint     `json:"id"`
	Username    string   `json:"username"`
	Nickname    string   `json:"nickname"`
	Avatar      string   `json:"avatar"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// AuthService 认证业务。
// db 字段仅供 Seed 使用（多表种子操作不塞进 UserRepository 接口，避免臃肿）。
type AuthService struct {
	repo   repository.UserRepository
	db     *gorm.DB // 仅 Seed 使用
	jwtMgr *jwt.Manager
	runner async.Runner
}

// NewAuthService 构造认证服务。runner 用于派发登录统计等后台任务，
// 生产传 GoroutineRunner（异步），测试传 SyncRunner（同步执行，消除 goroutine 与 teardown 的竞态）。
func NewAuthService(repo repository.UserRepository, db *gorm.DB, jwtMgr *jwt.Manager, runner async.Runner) *AuthService {
	return &AuthService{repo: repo, db: db, jwtMgr: jwtMgr, runner: runner}
}

// 种子数据定义（对齐前端 mock USERS）。
var seedAdmin = struct {
	username, password, nickname string
}{
	username: "admin", password: "123456", nickname: "Admin",
}
var seedNormal = struct {
	username, password, nickname string
}{
	username: "user", password: "123456", nickname: "User",
}

// seedPermissionCodes M3.1 种入的权限码（对齐前端 mock permissionCodes）。
// 共 27 个：system/user/role/permission/dict/config/menu 七模块。
var seedPermissionCodes = []string{
	"system:setting", "system:log", "system:operation",
	"user:list", "user:create", "user:edit", "user:delete",
	"role:list", "role:create", "role:edit", "role:delete", "role:permission",
	"permission:list", "permission:create", "permission:edit", "permission:delete",
	"dict:list", "dict:create", "dict:edit", "dict:delete",
	"config:system", "config:parameter", "config:email",
	"menu:list", "menu:create", "menu:edit", "menu:delete", // 新增 M4.2
}

// seedMenus M4.1 种入的菜单数据（对齐前端 ALL_MENUS 核心结构）。
// ParentID=0 为根菜单（无 PermissionCode）；PermissionCode 非空为 system 子菜单，
// 其 ParentID 在 Seed 时动态指向 system 菜单的 ID。
var seedMenus = []model.Menu{
	{Name: "home", Title: "首页", Path: "/", Component: "dashboard/views/Home", Icon: "HomeFilled", Sort: 0, ShowMenu: true},
	{Name: "crud", Title: "增删改查", Path: "/crud", Component: "crud/views/List", Icon: "Document", Sort: 10, ShowMenu: true},
	{Name: "system", Title: "系统管理", Path: "/system", Icon: "Setting", Sort: 20, ShowMenu: true},
	{Name: "systemUser", Title: "用户管理", Path: "/system/user", Component: "system/user/views/List", Icon: "User", Sort: 0, ShowMenu: true, PermissionCode: "user:list"},
	{Name: "systemRole", Title: "角色管理", Path: "/system/role", Component: "system/role/views/List", Icon: "Avatar", Sort: 10, ShowMenu: true, PermissionCode: "role:list"},
	{Name: "systemPermission", Title: "权限管理", Path: "/system/permission", Component: "system/permission/views/List", Icon: "Key", Sort: 20, ShowMenu: true, PermissionCode: "permission:list"},
}

// Seed 幂等种入权限/角色/账户。用 FirstOrCreate，已有数据不清不删。
func (s *AuthService) Seed(ctx context.Context) error {
	// 1. 权限
	permAll := model.Permission{Code: "*", Name: "通配权限", Type: "api"}
	if err := firstOrCreatePerm(ctx, s.db, &permAll); err != nil {
		return err
	}
	permUserRead := model.Permission{Code: "user:read", Name: "用户查看", Type: "api"}
	if err := firstOrCreatePerm(ctx, s.db, &permUserRead); err != nil {
		return err
	}

	// M3.1: 批量种入权限定义（不分配给角色，M3.2 处理 role-permission 分配）
	for _, code := range seedPermissionCodes {
		p := model.Permission{Code: code, Name: code, Type: "api", Module: moduleOf(code), Status: "active"}
		if err := firstOrCreatePerm(ctx, s.db, &p); err != nil {
			return err
		}
	}

	// 2. 角色（带权限关联）
	if err := s.seedRole(ctx, "super_admin", "超级管理员", "超级管理员，拥有所有权限", []model.Permission{permAll}); err != nil {
		return err
	}
	if err := s.seedRole(ctx, "user", "普通用户", "普通用户，仅查看权限", []model.Permission{permUserRead}); err != nil {
		return err
	}

	// 3. 账户
	if err := s.seedUser(ctx, seedAdmin.username, seedAdmin.password, seedAdmin.nickname, "super_admin"); err != nil {
		return err
	}
	if err := s.seedUser(ctx, seedNormal.username, seedNormal.password, seedNormal.nickname, "user"); err != nil {
		return err
	}

	// M4.1: 菜单（根菜单先建拿 system 的 ID，子菜单 ParentID 指向 system）
	if err := s.seedMenus(ctx); err != nil {
		return err
	}
	return nil
}

// firstOrCreatePerm 按 code 查/建权限。
func firstOrCreatePerm(ctx context.Context, db *gorm.DB, p *model.Permission) error {
	return db.WithContext(ctx).Where(model.Permission{Code: p.Code}).FirstOrCreate(p).Error
}

// seedRole 按 code 查/建角色，并补齐角色-权限关联（Replace 去重）。
func (s *AuthService) seedRole(ctx context.Context, code, name, description string, perms []model.Permission) error {
	role := model.Role{Code: code, Name: name, Description: description, Status: "active"}
	if err := s.db.WithContext(ctx).Where(model.Role{Code: code}).FirstOrCreate(&role).Error; err != nil {
		return err
	}
	// 已存在角色补 Description（FirstOrCreate 不覆盖已存在记录的字段）
	if err := s.db.WithContext(ctx).Model(&role).Update("description", description).Error; err != nil {
		return err
	}
	// 关联权限：Replace 法（覆盖式，幂等去重，多次 Seed 不会重复关联）。
	if err := s.db.WithContext(ctx).Model(&role).Association("Permissions").Replace(perms); err != nil {
		return err
	}
	return nil
}

// seedUser 按 username 查/建用户并绑定角色。仅新建时对密码做 bcrypt 哈希。
func (s *AuthService) seedUser(ctx context.Context, username, password, nickname, roleCode string) error {
	// 查角色（必须先由 seedRole 建好）
	var role model.Role
	if err := s.db.WithContext(ctx).Where(model.Role{Code: roleCode}).First(&role).Error; err != nil {
		return err
	}
	// 查/建用户（仅在新建时哈希密码，避免对已存在用户重复哈希）
	var user model.User
	result := s.db.WithContext(ctx).Where(model.User{Username: username}).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		hashed, err := hash.Hash(password)
		if err != nil {
			return err
		}
		user = model.User{
			Username: username, Password: hashed, Nickname: nickname,
			Status: "active",
		}
		if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	}
	// 绑定角色（Replace 去重，多次 Seed 不会重复关联）
	if err := s.db.WithContext(ctx).Model(&user).Association("Roles").Replace(&role); err != nil {
		return err
	}
	return nil
}

// Login 校验凭据并签发 access/refresh token。
// 安全：用户不存在与密码错返回同一文案，防用户枚举。
func (s *AuthService) Login(ctx context.Context, username, password string) (*AuthResult, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.Unauthorized("用户名或密码错误")
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, apperr.Unauthorized("账户已禁用")
	}
	if err := hash.Compare(user.Password, password); err != nil {
		return nil, apperr.Unauthorized("用户名或密码错误")
	}

	access, err := s.jwtMgr.GenerateAccess(user.ID, user.Username)
	if err != nil {
		return nil, err
	}
	refresh, err := s.jwtMgr.GenerateRefresh(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	// 异步更新登录统计：通过 Runner 派发，生产为 goroutine、测试为同步（见 async 包）。
	// 失败仅告警，不影响登录结果。
	s.runner.Go(func() {
		if err := s.repo.UpdateLoginStats(context.Background(), user.ID); err != nil {
			log.S.Warnw("更新登录统计失败", "uid", user.ID, "err", err)
		}
	})

	return &AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    s.jwtMgr.AccessTTLSeconds(),
	}, nil
}

// Refresh 校验 refresh token 并签发新的 token 对。
// 纯 JWT 不落库：旧 refresh 在 TTL 内仍可解析，无法真正删除（接受权衡）。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := s.jwtMgr.Parse(refreshToken)
	if err != nil || claims.Type != jwt.TypeRefresh {
		return nil, apperr.Unauthorized("Invalid refresh token")
	}
	user, err := s.repo.FindByID(ctx, claims.UserID)
	if err != nil || user.Status != "active" {
		return nil, apperr.Unauthorized("Invalid refresh token")
	}
	access, err := s.jwtMgr.GenerateAccess(user.ID, user.Username)
	if err != nil {
		return nil, err
	}
	refresh, err := s.jwtMgr.GenerateRefresh(user.ID, user.Username)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    s.jwtMgr.AccessTTLSeconds(),
	}, nil
}

// GetProfile 构造当前用户信息，汇总角色码与权限码。
// 超管语义：权限集合含 "*" 时返回 ["*"]（前端 isSuperAdmin 短路）。
// 判断依据是权限码而非角色名，故任一角色拥有 "*" 即视为超管。
func (s *AuthService) GetProfile(ctx context.Context, userID uint) (*UserProfile, error) {
	user, err := s.repo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		// 仅在用户不存在时返回 401；DB 等故障透传，由 apperr.Write 兜底 500，不伪装成 401。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.Unauthorized("未授权")
		}
		return nil, err
	}
	roles := make([]string, 0, len(user.Roles))
	permSet := make(map[string]struct{})
	for _, r := range user.Roles {
		roles = append(roles, r.Code)
		for _, p := range r.Permissions {
			permSet[p.Code] = struct{}{}
		}
	}
	perms := make([]string, 0, len(permSet))
	if _, ok := permSet["*"]; ok {
		// 超管短路：权限码含 "*" 即返回 ["*"]，前端跳过权限校验。
		perms = []string{"*"}
	} else {
		for code := range permSet {
			perms = append(perms, code)
		}
	}
	return &UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Roles:       roles,
		Permissions: perms,
	}, nil
}

// Logout 纯 JWT 模式下为空操作（token 不落库，前端清本地 storage）。
func (s *AuthService) Logout(ctx context.Context) error {
	return nil
}

// seedMenus 种入菜单。幂等：FirstOrCreate by Name。
// 两步：根菜单（无 PermissionCode）先建，拿到 system 的 ID；
// 子菜单（有 PermissionCode）后建，ParentID 指向 system。
func (s *AuthService) seedMenus(ctx context.Context) error {
	// 1. 先种根菜单（home/crud/system），拿 system 的 ID
	rootMenus := []model.Menu{}
	for _, m := range seedMenus {
		if m.PermissionCode == "" { // 根菜单无 PermissionCode
			existing := model.Menu{}
			if err := s.db.WithContext(ctx).Where(model.Menu{Name: m.Name}).FirstOrCreate(&existing, m).Error; err != nil {
				return err
			}
			if m.Name == "system" {
				rootMenus = append(rootMenus, existing)
			}
		}
	}
	if len(rootMenus) == 0 {
		return nil
	}
	systemID := rootMenus[0].ID
	// 2. 种子菜单（有 PermissionCode），ParentID = systemID
	for _, m := range seedMenus {
		if m.PermissionCode != "" {
			m.ParentID = systemID
			existing := model.Menu{}
			if err := s.db.WithContext(ctx).Where(model.Menu{Name: m.Name}).FirstOrCreate(&existing, m).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// moduleOf 从权限码推断模块（code 形如 "user:list" → "user"）。
func moduleOf(code string) string {
	for i := 0; i < len(code); i++ {
		if code[i] == ':' {
			return code[:i]
		}
	}
	return code
}
