// Package openapi 商户服务端开放接口：判定 v2、归因、转化、分销码、日志，以及 V1 兼容判定。
// 运行在网关进程内，站点与密钥来自快照，不查数据库即可完成鉴权与判定。
package openapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"cloak/internal/engine/snapshot"
	"cloak/internal/pkg/secretbox"
)

// 请求头
const (
	HeaderKey       = "X-Cloak-Key"
	HeaderTimestamp = "X-Cloak-Timestamp"
	HeaderNonce     = "X-Cloak-Nonce"
	HeaderSignature = "X-Cloak-Signature"
	HeaderV1Key     = "X-API-Key"

	maxBody     = 256 << 10
	clockSkew   = 300 * time.Second
	nonceWindow = 5 * time.Minute
)

// AuthError 鉴权失败。
type AuthError struct {
	Status int
	Code   string
	Msg    string
}

func (e *AuthError) Error() string { return e.Code + ": " + e.Msg }

func authErr(status int, code, msg string) *AuthError {
	return &AuthError{Status: status, Code: code, Msg: msg}
}

// Authenticator 解析站点并校验签名 / 密钥。
type Authenticator struct {
	store *snapshot.Store
	box   *secretbox.Box
	rdb   *goredis.Client
	now   func() time.Time
}

func NewAuthenticator(store *snapshot.Store, box *secretbox.Box, rdb *goredis.Client) *Authenticator {
	return &Authenticator{store: store, box: box, rdb: rdb, now: time.Now}
}

// Principal 鉴权结果。
type Principal struct {
	Site     *snapshot.Site
	Version  string // v2 / v1
	Body     []byte
	ClientIP netip.Addr
}

// Authenticate 读取请求体并按 v2 签名或 v1 密钥鉴权。
func (a *Authenticator) Authenticate(ctx context.Context, r *http.Request, clientIP netip.Addr) (*Principal, *AuthError) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return nil, authErr(http.StatusBadRequest, "bad_request", "读取请求体失败")
	}
	if len(body) > maxBody {
		return nil, authErr(http.StatusRequestEntityTooLarge, "body_too_large", "请求体超过 256KB")
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	if prefix := r.Header.Get(HeaderKey); prefix != "" {
		return a.authV2(ctx, r, body, prefix, clientIP)
	}
	key := r.Header.Get(HeaderV1Key)
	if key == "" {
		if bt := r.Header.Get("Authorization"); strings.HasPrefix(bt, "Bearer ") {
			key = strings.TrimPrefix(bt, "Bearer ")
		}
	}
	if key != "" {
		return a.authV1(ctx, key, body, clientIP)
	}
	return nil, authErr(http.StatusUnauthorized, "missing_credentials", "缺少 X-Cloak-Key 签名头或 X-API-Key")
}

// authV2 HMAC-SHA256(secret, ts \n nonce \n METHOD \n path \n sha256hex(body))。
func (a *Authenticator) authV2(ctx context.Context, r *http.Request, body []byte, prefix string, ip netip.Addr) (*Principal, *AuthError) {
	site, secret, aerr := a.lookup(ctx, prefix, ip)
	if aerr != nil {
		return nil, aerr
	}
	tsRaw, nonce, sig := r.Header.Get(HeaderTimestamp), r.Header.Get(HeaderNonce), r.Header.Get(HeaderSignature)
	if tsRaw == "" || nonce == "" || sig == "" {
		return nil, authErr(http.StatusUnauthorized, "missing_signature", "缺少 X-Cloak-Timestamp / X-Cloak-Nonce / X-Cloak-Signature")
	}
	ts, err := strconv.ParseInt(tsRaw, 10, 64)
	if err != nil {
		return nil, authErr(http.StatusUnauthorized, "bad_timestamp", "时间戳须为 Unix 秒")
	}
	if d := a.now().Sub(time.Unix(ts, 0)); d > clockSkew || d < -clockSkew {
		return nil, authErr(http.StatusUnauthorized, "timestamp_expired", "时间戳与服务器相差超过 300 秒")
	}
	if len(nonce) < 8 || len(nonce) > 64 {
		return nil, authErr(http.StatusUnauthorized, "bad_nonce", "nonce 长度须在 8-64 之间")
	}
	expect := Sign(secret, tsRaw, nonce, r.Method, r.URL.Path, body)
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(sig)), []byte(expect)) != 1 {
		return nil, authErr(http.StatusUnauthorized, "bad_signature", "签名不匹配")
	}
	if a.rdb != nil {
		ok, err := a.rdb.SetNX(ctx, "nonce:"+strconv.FormatUint(site.SiteID, 10)+":"+nonce, 1, nonceWindow).Result()
		if err == nil && !ok {
			return nil, authErr(http.StatusUnauthorized, "nonce_replayed", "nonce 已使用")
		}
	}
	return &Principal{Site: site, Version: "v2", Body: body, ClientIP: ip}, nil
}

// authV1 直接比对密钥明文（V1 插件 X-API-Key / Bearer）。
func (a *Authenticator) authV1(ctx context.Context, key string, body []byte, ip netip.Addr) (*Principal, *AuthError) {
	key = strings.TrimSpace(key)
	if len(key) < 12 {
		return nil, authErr(http.StatusUnauthorized, "invalid_api_key", "API Key 无效")
	}
	site, secret, aerr := a.lookup(ctx, key[:12], ip)
	if aerr != nil {
		if aerr.Code == "unknown_key" {
			aerr.Code, aerr.Msg = "invalid_api_key", "API Key 无效"
		}
		return nil, aerr
	}
	if subtle.ConstantTimeCompare([]byte(key), []byte(secret)) != 1 {
		return nil, authErr(http.StatusUnauthorized, "invalid_api_key", "API Key 无效")
	}
	return &Principal{Site: site, Version: "v1", Body: body, ClientIP: ip}, nil
}

func (a *Authenticator) lookup(ctx context.Context, prefix string, ip netip.Addr) (*snapshot.Site, string, *AuthError) {
	site, ok := a.store.ByPrefix(ctx, prefix)
	if !ok {
		return nil, "", authErr(http.StatusUnauthorized, "unknown_key", "密钥不存在")
	}
	if !site.Serving() {
		return nil, "", authErr(http.StatusForbidden, "site_disabled", "站点已停用")
	}
	if len(site.SecretEnc) == 0 || a.box == nil {
		return nil, "", authErr(http.StatusForbidden, "secret_unavailable", "该站点密钥不支持服务端接口，请在后台重置一次密钥")
	}
	plain, err := a.box.Open(site.SecretEnc)
	if err != nil {
		return nil, "", authErr(http.StatusForbidden, "secret_unavailable", "密钥解密失败，请检查 auth.secret_key 配置是否一致")
	}
	if len(site.SecretAllow) > 0 && !ipAllowed(ip, site.SecretAllow) {
		return nil, "", authErr(http.StatusForbidden, "ip_not_allowed", "来源 IP 不在白名单内")
	}
	return site, string(plain), nil
}

// Sign 计算签名（导出供文档示例与测试复用）。
func Sign(secret, ts, nonce, method, path string, body []byte) string {
	sum := sha256.Sum256(body)
	msg := ts + "\n" + nonce + "\n" + strings.ToUpper(method) + "\n" + path + "\n" + hex.EncodeToString(sum[:])
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func ipAllowed(ip netip.Addr, allow []string) bool {
	if !ip.IsValid() {
		return false
	}
	for _, a := range allow {
		if p, err := netip.ParsePrefix(a); err == nil && p.Contains(ip) {
			return true
		}
		if x, err := netip.ParseAddr(a); err == nil && x == ip {
			return true
		}
	}
	return false
}

var errNoRedis = errors.New("redis unavailable")

// Limiter 两级限流：每密钥每秒、每站点每分钟（固定窗口，Redis INCR）。
type Limiter struct {
	rdb       *goredis.Client
	perKeySec atomic.Int64
	perSiteMn atomic.Int64
}

func NewLimiter(rdb *goredis.Client, perKeyPerSecond, perSitePerMinute int64) *Limiter {
	if perKeyPerSecond <= 0 {
		perKeyPerSecond = 50
	}
	if perSitePerMinute <= 0 {
		perSitePerMinute = 2000
	}
	l := &Limiter{rdb: rdb}
	l.perKeySec.Store(perKeyPerSecond)
	l.perSiteMn.Store(perSitePerMinute)
	return l
}

// SetLimits 热更新两级限额。非正数保持原值。
func (l *Limiter) SetLimits(perKeyPerSecond, perSitePerMinute int64) {
	if l == nil {
		return
	}
	if perKeyPerSecond > 0 {
		l.perKeySec.Store(perKeyPerSecond)
	}
	if perSitePerMinute > 0 {
		l.perSiteMn.Store(perSitePerMinute)
	}
}

// Allow 返回是否放行；Redis 不可用时放行。
func (l *Limiter) Allow(ctx context.Context, site *snapshot.Site) (bool, error) {
	if l == nil || l.rdb == nil {
		return true, errNoRedis
	}
	now := time.Now()
	k1 := "rl:key:" + site.SecretPrefix + ":" + strconv.FormatInt(now.Unix(), 10)
	k2 := "rl:site:" + strconv.FormatUint(site.SiteID, 10) + ":" + now.Format("200601021504")
	pipe := l.rdb.Pipeline()
	c1 := pipe.Incr(ctx, k1)
	pipe.Expire(ctx, k1, 2*time.Second)
	c2 := pipe.Incr(ctx, k2)
	pipe.Expire(ctx, k2, 2*time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, err
	}
	return c1.Val() <= l.perKeySec.Load() && c2.Val() <= l.perSiteMn.Load(), nil
}
