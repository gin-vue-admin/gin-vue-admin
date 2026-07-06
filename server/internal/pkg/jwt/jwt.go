// Package jwt 提供 access/refresh 双 token 签发与解析。
// 首期不落库（YAGNI）：refresh token 同为 JWT，靠过期与轮换保证语义；吊销/黑名单留待后续。
package jwt

import (
	"errors"
	"time"

	"gva/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType token 用途标识。
type TokenType string

const (
	// TypeAccess 访问令牌，用于 API 鉴权。
	TypeAccess TokenType = "access"
	// TypeRefresh 刷新令牌，用于换取新的访问令牌。
	TypeRefresh TokenType = "refresh"
)

// Claims 自定义 JWT 声明。
type Claims struct {
	UserID   uint      `json:"uid"`
	Username string    `json:"usr"`
	Type     TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// Manager token 签发/解析器。
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

// NewManager 依据配置构造 Manager。
func NewManager(cfg config.JWTConfig) *Manager {
	return &Manager{
		secret:     []byte(cfg.Secret),
		accessTTL:  time.Duration(cfg.AccessTTL) * time.Second,
		refreshTTL: time.Duration(cfg.RefreshTTL) * time.Second,
		issuer:     cfg.Issuer,
	}
}

// AccessTTLSeconds 返回 access token 有效期（秒），供响应 expiresIn 使用。
func (m *Manager) AccessTTLSeconds() int {
	return int(m.accessTTL.Seconds())
}

// issue 签发指定类型的 token。
func (m *Manager) issue(userID uint, username string, typ TokenType, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Type:     typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(m.secret)
}

// GenerateAccess 签发 access token。
func (m *Manager) GenerateAccess(userID uint, username string) (string, error) {
	return m.issue(userID, username, TypeAccess, m.accessTTL)
}

// GenerateRefresh 签发 refresh token。
func (m *Manager) GenerateRefresh(userID uint, username string) (string, error) {
	return m.issue(userID, username, TypeRefresh, m.refreshTTL)
}

// SetAccessTTL 动态设置 access token 有效期（秒）。供 sys_config token_expire_seconds 接入。
func (m *Manager) SetAccessTTL(seconds int) {
	if seconds > 0 {
		m.accessTTL = time.Duration(seconds) * time.Second
	}
}

// AccessTTL 返回当前 access token 有效期（秒）。
func (m *Manager) AccessTTL() int {
	return int(m.accessTTL / time.Second)
}

// ErrInvalidToken token 无效或过期。
var ErrInvalidToken = errors.New("invalid token")

// Parse 解析并校验 token，返回 claims。
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !tok.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
