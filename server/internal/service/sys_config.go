// Package service 系统参数配置业务。
//
// 双重职责：
//   - 编程 API（GetValue/GetBool/GetInt/Set）：供其他模块运行时读配置，走内存缓存，零打库。
//   - 后台 CRUD：管理端维护配置项。
//
// 缓存策略：LoadAll 启动加载全量到 map；每个写操作后 reload 刷新。
// 配置数据量小（通常几十条），全量重载成本可忽略，换取实现简单与一致性。
package service

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/jwt"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"

	"gorm.io/gorm"
)

// SysConfigService 系统参数配置服务。
type SysConfigService struct {
	repo   repository.SysConfigRepository
	jwtMgr *jwt.Manager // 可选：注入后 token_expire_seconds 实时同步（applyRuntime）
	mu     sync.RWMutex
	cache  map[string]string // configKey → configValue
}

// NewSysConfigService 构造服务。缓存初始化为空 map，需在 seed 后调 LoadAll 预热。
func NewSysConfigService(repo repository.SysConfigRepository) *SysConfigService {
	return &SysConfigService{repo: repo, cache: make(map[string]string)}
}

// SetJWTManager 注入 JWT 管理器，使 token_expire_seconds 配置变更实时生效
// （applyRuntime 同步到 jwtMgr，需在 LoadAll 前调用）。
func (s *SysConfigService) SetJWTManager(m *jwt.Manager) {
	s.jwtMgr = m
}

// LoadAll 全量加载配置进缓存。main.go 启动时（seed 后）调用一次。
func (s *SysConfigService) LoadAll(ctx context.Context) error {
	all, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.cache = make(map[string]string, len(all))
	for _, c := range all {
		s.cache[c.ConfigKey] = c.ConfigValue
	}
	s.mu.Unlock()
	s.applyRuntime() // 同步配置到运行时组件（如分页默认条数）
	return nil
}

// applyRuntime 把配置同步到运行时组件。配置变更（Set/LoadAll）后调用。
// 接入：
//   - default_page_size → pagination.DefaultSize（列表默认每页条数）
//   - token_expire_seconds → jwtMgr.SetAccessTTL（JWT 有效期，注入 jwtMgr 后实时）
func (s *SysConfigService) applyRuntime() {
	s.mu.RLock()
	pageSize := s.cache["default_page_size"]
	tokenTTL := s.cache["token_expire_seconds"]
	s.mu.RUnlock()
	if n, err := strconv.Atoi(pageSize); err == nil && n > 0 && n <= 100 {
		pagination.DefaultSize = n
	}
	if s.jwtMgr != nil {
		if ttl, err := strconv.Atoi(tokenTTL); err == nil && ttl > 0 {
			s.jwtMgr.SetAccessTTL(ttl)
		}
	}
}

// publicConfigKeys 公开配置白名单（无需鉴权即可下发给前端启动期拉取）。
var publicConfigKeys = []string{"site_title", "login_captcha_enabled", "default_page_size"}

// GetPublic 返回白名单公开配置（前端启动拉取 site_title 等）。
func (s *SysConfigService) GetPublic() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(publicConfigKeys))
	for _, k := range publicConfigKeys {
		if v, ok := s.cache[k]; ok {
			out[k] = v
		}
	}
	return out
}

// reload 写后刷新缓存。失败忽略（下次写再刷），缓存与库短暂不一致可接受。
func (s *SysConfigService) reload(ctx context.Context) {
	_ = s.LoadAll(ctx)
}

// --- 编程 API（走缓存）---

// GetValue 取字符串值；不存在返回空串。
func (s *SysConfigService) GetValue(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache[key]
}

// GetBool 取布尔值。"1"/"true"/"yes"（大小写不敏感）为真，其余为假。
func (s *SysConfigService) GetBool(key string) bool {
	switch s.GetValue(key) {
	case "1", "true", "True", "TRUE", "yes", "Yes", "YES":
		return true
	default:
		return false
	}
}

// GetInt 取整数值；不存在或解析失败返回 fallback。
func (s *SysConfigService) GetInt(key string, fallback int) int {
	v := s.GetValue(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// Set 更新某 key 的值（运营一键改配置，写库 + 刷新缓存）。
func (s *SysConfigService) Set(ctx context.Context, key, value string) error {
	c, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("配置项不存在")
		}
		return err
	}
	c.ConfigValue = value
	if err := s.repo.Update(ctx, c); err != nil {
		return err
	}
	s.reload(ctx)
	return nil
}

// --- 后台 CRUD ---

// List 分页列表。
func (s *SysConfigService) List(ctx context.Context, q pagination.Query) (pagination.Result[model.SysConfig], error) {
	q.Normalize()
	res, err := s.repo.List(ctx, q)
	if err != nil {
		return pagination.Result[model.SysConfig]{}, err
	}
	res.Current, res.Size = q.Page, q.Size
	return res, nil
}

// Get 详情（按 id）。不存在→404。
func (s *SysConfigService) Get(ctx context.Context, id uint) (*model.SysConfig, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("配置项不存在")
		}
		return nil, err
	}
	return c, nil
}

// GetByKey 详情（按 key）。不存在→404。
func (s *SysConfigService) GetByKey(ctx context.Context, key string) (*model.SysConfig, error) {
	c, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("配置项不存在")
		}
		return nil, err
	}
	return c, nil
}

// Create 新建配置；写后刷新缓存。
func (s *SysConfigService) Create(ctx context.Context, c *model.SysConfig) error {
	if err := s.repo.Create(ctx, c); err != nil {
		return err
	}
	s.reload(ctx)
	return nil
}

// Update 更新配置；写后刷新缓存。
func (s *SysConfigService) Update(ctx context.Context, c *model.SysConfig) error {
	if err := s.repo.Update(ctx, c); err != nil {
		return err
	}
	s.reload(ctx)
	return nil
}

// Delete 软删；先查再删，不存在→404；删后刷新缓存。
func (s *SysConfigService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("配置项不存在")
		}
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.reload(ctx)
	return nil
}
