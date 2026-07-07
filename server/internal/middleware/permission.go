package middleware

import (
	"context"
	"sync"
	"time"

	"gva/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

// PermissionReader 中间件仅需"按 userID 查权限码集合"，最小接口避免依赖完整 repo。
// repository.PermissionRepository 已实现该方法，赋值时按接口隐式满足。
type PermissionReader interface {
	GetUserPermissionCodes(ctx context.Context, userID uint) ([]string, error)
}

// permissionCacheTTL 权限缓存存活时长。权限变更低频，5 分钟可接受，配合 InvalidateAll 主动失效。
const permissionCacheTTL = 5 * time.Minute

// cacheEntry 单个用户的权限码集合与过期时间。
type cacheEntry struct {
	codes    map[string]struct{}
	expireAt time.Time
}

// permCache 包级权限缓存：userID -> 权限码集合。读多写少，配 RWMutex。
var (
	permCache   = make(map[uint]cacheEntry)
	permCacheMu sync.RWMutex
)

// RequirePermission 需任一权限码命中（hasAny）即放行；超管 "*" 短路放行。
// repo 查询走缓存（TTL 内复用），permission CRUD 后应调 InvalidateAll 失效。
func RequirePermission(repo PermissionReader, codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidAny, _ := c.Get(ContextKeyUserID)
		uid, ok := uidAny.(uint)
		if !ok {
			apperr.Write(c, apperr.Unauthorized("未授权"))
			return
		}
		codeset, err := loadPermissions(c.Request.Context(), repo, uid)
		if err != nil {
			apperr.Write(c, apperr.Forbidden("禁止访问"))
			return
		}
		// 超管短路
		if _, isSuper := codeset["*"]; isSuper {
			c.Next()
			return
		}
		// hasAny：任一码命中即放行
		for _, code := range codes {
			if _, ok := codeset[code]; ok {
				c.Next()
				return
			}
		}
		apperr.Write(c, apperr.Forbidden("禁止访问"))
	}
}

// RequireAllPermissions 需全部权限码命中（hasAll）才放行；超管 "*" 短路放行。
func RequireAllPermissions(repo PermissionReader, codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidAny, _ := c.Get(ContextKeyUserID)
		uid, ok := uidAny.(uint)
		if !ok {
			apperr.Write(c, apperr.Unauthorized("未授权"))
			return
		}
		codeset, err := loadPermissions(c.Request.Context(), repo, uid)
		if err != nil {
			apperr.Write(c, apperr.Forbidden("禁止访问"))
			return
		}
		// 超管短路
		if _, isSuper := codeset["*"]; isSuper {
			c.Next()
			return
		}
		// hasAll：缺任一码即拒绝
		for _, code := range codes {
			if _, ok := codeset[code]; !ok {
				apperr.Write(c, apperr.Forbidden("禁止访问"))
				return
			}
		}
		c.Next()
	}
}

// loadPermissions 取指定用户的权限码集合（缓存优先，TTL 到期重查）。
// 读缓存用 RLock；未命中时持写锁查 repo 并回填（双重检查锁定），
// 消除缓存过期瞬间多个 goroutine 同时打 DB 的击穿/惊群。
func loadPermissions(ctx context.Context, repo PermissionReader, uid uint) (map[string]struct{}, error) {
	permCacheMu.RLock()
	if e, ok := permCache[uid]; ok && time.Now().Before(e.expireAt) {
		permCacheMu.RUnlock()
		return e.codes, nil
	}
	permCacheMu.RUnlock()

	permCacheMu.Lock()
	defer permCacheMu.Unlock()
	// 双重检查：排队拿到写锁后，可能已被前一个 goroutine 回填，直接复用
	if e, ok := permCache[uid]; ok && time.Now().Before(e.expireAt) {
		return e.codes, nil
	}
	codes, err := repo.GetUserPermissionCodes(ctx, uid)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		set[c] = struct{}{}
	}
	permCache[uid] = cacheEntry{codes: set, expireAt: time.Now().Add(permissionCacheTTL)}
	return set, nil
}

// InvalidateAll 清全量权限缓存。权限 CRUD（增删改）后调用，保证一致。
func InvalidateAll() {
	permCacheMu.Lock()
	permCache = make(map[uint]cacheEntry)
	permCacheMu.Unlock()
}
