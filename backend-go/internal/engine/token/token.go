// Package token 访客令牌：v1.<base64url(payload)>.<base64url(hmac_sha256)>。
package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cloak/internal/engine/snapshot"
)

const prefix = "v1"

var (
	ErrMalformed  = errors.New("token malformed")
	ErrUnknownKey = errors.New("token key version unknown")
	ErrSignature  = errors.New("token signature mismatch")
	ErrExpired    = errors.New("token expired")
	ErrSiteMatch  = errors.New("token site mismatch")
	ErrNoKey      = errors.New("site has no active key")
)

// Payload 令牌载荷。
type Payload struct {
	VisitorID     string `json:"vid"`
	SiteID        uint64 `json:"sid"`
	AttributionID string `json:"aid"`
	IssuedAt      int64  `json:"iat"`
	ExpiresAt     int64  `json:"exp"`
	KeyVersion    int    `json:"kv"`
}

// Expiry 到期时间。
func (p Payload) Expiry() time.Time { return time.Unix(p.ExpiresAt, 0) }

// Issue 用站点当前密钥签发令牌。
func Issue(site *snapshot.Site, p Payload) (string, error) {
	k := site.ActiveKey()
	if k == nil {
		return "", ErrNoKey
	}
	p.KeyVersion = k.Version
	p.SiteID = site.SiteID
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	sig := sign(k.Key, prefix+"."+body)
	return prefix + "." + body + "." + sig, nil
}

// Verify 验签并解析。过期返回 ErrExpired 但仍返回载荷，便于调用方判断是否续期。
func Verify(site *snapshot.Site, tok string, now time.Time) (*Payload, error) {
	if tok == "" {
		return nil, ErrMalformed
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 || parts[0] != prefix {
		return nil, ErrMalformed
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrMalformed
	}
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, ErrMalformed
	}
	k := site.KeyByVersion(p.KeyVersion)
	if k == nil {
		return nil, ErrUnknownKey
	}
	if k.ExpiresAt != nil && now.After(*k.ExpiresAt) {
		return nil, ErrUnknownKey
	}
	expect := sign(k.Key, parts[0]+"."+parts[1])
	if !hmac.Equal([]byte(expect), []byte(parts[2])) {
		return nil, ErrSignature
	}
	if p.SiteID != site.SiteID {
		return &p, ErrSiteMatch
	}
	if now.Unix() >= p.ExpiresAt {
		return &p, ErrExpired
	}
	return &p, nil
}

// NeedsReissue 令牌是否用旧密钥签发（轮换后下一次放行换发新令牌）。
func NeedsReissue(site *snapshot.Site, p *Payload) bool {
	k := site.ActiveKey()
	return k != nil && p != nil && p.KeyVersion != k.Version
}

func sign(hexKey, data string) string {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) == 0 {
		key = []byte(hexKey)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
