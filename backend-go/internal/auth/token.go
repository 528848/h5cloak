package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	goredis "github.com/redis/go-redis/v9"

	"cloak/internal/pkg/requestid"
)

// Claims JWT 载荷。
type Claims struct {
	UserID     uint64 `json:"uid"`
	MerchantID uint64 `json:"mid,omitempty"`
	Portal     string `json:"portal"`
	jwt.RegisteredClaims
}

// TokenManager 签发与校验 JWT，并用 Redis 维护会话以支持主动失效。
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	rdb    *goredis.Client
}

// NewTokenManager rdb 可为 nil（测试），此时不做会话校验。
func NewTokenManager(secret string, ttl time.Duration, rdb *goredis.Client) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, rdb: rdb}
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrSessionGone  = errors.New("session revoked")
)

// Issue 签发 token 并登记会话。
func (m *TokenManager) Issue(ctx context.Context, userID, merchantID uint64, portal string) (string, error) {
	now := time.Now()
	jti := requestid.New()
	claims := Claims{
		UserID: userID, MerchantID: merchantID, Portal: portal,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			Subject:   strconv.FormatUint(userID, 10),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", err
	}
	if m.rdb != nil {
		if err := m.rdb.Set(ctx, sessionKey(jti), userID, m.ttl).Err(); err != nil {
			return "", fmt.Errorf("store session: %w", err)
		}
		// 记录用户的会话集合，便于踢出该用户全部登录
		m.rdb.SAdd(ctx, userSessionsKey(userID), jti)
		m.rdb.Expire(ctx, userSessionsKey(userID), m.ttl)
	}
	return signed, nil
}

// Parse 校验签名、有效期与会话存在性。
func (m *TokenManager) Parse(ctx context.Context, raw string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !tok.Valid {
		return nil, ErrInvalidToken
	}
	if m.rdb != nil {
		exists, err := m.rdb.Exists(ctx, sessionKey(claims.ID)).Result()
		if err == nil && exists == 0 {
			return nil, ErrSessionGone
		}
		// Redis 故障时放行，依赖 JWT 自身有效期
	}
	return claims, nil
}

// Revoke 注销单个会话。
func (m *TokenManager) Revoke(ctx context.Context, claims *Claims) {
	if m.rdb == nil {
		return
	}
	m.rdb.Del(ctx, sessionKey(claims.ID))
	m.rdb.SRem(ctx, userSessionsKey(claims.UserID), claims.ID)
}

// RevokeUser 注销用户全部会话（禁用、改密、后台踢出）。
func (m *TokenManager) RevokeUser(ctx context.Context, userID uint64) {
	if m.rdb == nil {
		return
	}
	jtis, _ := m.rdb.SMembers(ctx, userSessionsKey(userID)).Result()
	for _, jti := range jtis {
		m.rdb.Del(ctx, sessionKey(jti))
	}
	m.rdb.Del(ctx, userSessionsKey(userID))
}

func sessionKey(jti string) string      { return "token:" + jti }
func userSessionsKey(uid uint64) string { return "user_sessions:" + strconv.FormatUint(uid, 10) }
