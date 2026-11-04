package openapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"cloak/internal/engine/snapshot"
	"cloak/internal/pkg/secretbox"
)

const testSecret = "sk_0123456789abcdef0123456789abcdef0123456789abcdef"

func testAuth(t *testing.T, allow []string) (*Authenticator, *snapshot.Site) {
	box, _ := secretbox.New("master")
	enc, _ := box.Seal([]byte(testSecret))
	site := &snapshot.Site{SiteID: 9, Status: "enabled", MerchantActive: true, SecretPrefix: testSecret[:12], SecretEnc: enc, SecretAllow: allow}
	store := snapshot.NewStore(nil)
	store.Put(site)
	a := NewAuthenticator(store, box, nil)
	a.now = func() time.Time { return time.Unix(1_762_000_000, 0) }
	return a, site
}

func signed(body []byte, secret string, ts int64, nonce string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/open/v2/decision", bytes.NewReader(body))
	tsS := strconv.FormatInt(ts, 10)
	r.Header.Set(HeaderKey, secret[:12])
	r.Header.Set(HeaderTimestamp, tsS)
	r.Header.Set(HeaderNonce, nonce)
	r.Header.Set(HeaderSignature, Sign(secret, tsS, nonce, "POST", "/api/open/v2/decision", body))
	return r
}

func TestV2Signature(t *testing.T) {
	a, _ := testAuth(t, nil)
	ctx := context.Background()
	ip := netip.MustParseAddr("203.0.113.1")
	body := []byte(`{"url":"https://shop.test/"}`)

	p, aerr := a.Authenticate(ctx, signed(body, testSecret, 1_762_000_000, "nonce-0001"), ip)
	if aerr != nil || p.Version != "v2" || string(p.Body) != string(body) {
		t.Fatalf("valid: %v", aerr)
	}
	// 过期
	if _, aerr := a.Authenticate(ctx, signed(body, testSecret, 1_762_000_000-400, "nonce-0002"), ip); aerr == nil || aerr.Code != "timestamp_expired" {
		t.Fatalf("expired: %v", aerr)
	}
	// 篡改体
	r := signed(body, testSecret, 1_762_000_000, "nonce-0003")
	r.Body = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{"url":"https://evil/"}`))).Body
	if _, aerr := a.Authenticate(ctx, r, ip); aerr == nil || aerr.Code != "bad_signature" {
		t.Fatalf("tampered: %v", aerr)
	}
	// 错密钥
	if _, aerr := a.Authenticate(ctx, signed(body, testSecret[:12]+"wrongwrongwrongwrongwrongwrongwrongw", 1_762_000_000, "nonce-0004"), ip); aerr == nil || aerr.Code != "bad_signature" {
		t.Fatalf("wrong secret: %v", aerr)
	}
	// 未知前缀
	r = signed(body, "sk_zzzzzzzzz000000000000000000000000000000000000000", 1_762_000_000, "nonce-0005")
	if _, aerr := a.Authenticate(ctx, r, ip); aerr == nil || aerr.Code != "unknown_key" {
		t.Fatalf("unknown: %v", aerr)
	}
	// 缺头
	r = httptest.NewRequest(http.MethodPost, "/api/open/v2/decision", bytes.NewReader(body))
	r.Header.Set(HeaderKey, testSecret[:12])
	if _, aerr := a.Authenticate(ctx, r, ip); aerr == nil || aerr.Code != "missing_signature" {
		t.Fatalf("missing: %v", aerr)
	}
	// nonce 太短
	if _, aerr := a.Authenticate(ctx, signed(body, testSecret, 1_762_000_000, "abc"), ip); aerr == nil || aerr.Code != "bad_nonce" {
		t.Fatalf("short nonce: %v", aerr)
	}
}

func TestV1KeyAndAllowlist(t *testing.T) {
	a, _ := testAuth(t, []string{"203.0.113.0/24"})
	ctx := context.Background()
	body := []byte(`{}`)
	r := httptest.NewRequest(http.MethodPost, "/api/open/decision", bytes.NewReader(body))
	r.Header.Set(HeaderV1Key, testSecret)
	p, aerr := a.Authenticate(ctx, r, netip.MustParseAddr("203.0.113.9"))
	if aerr != nil || p.Version != "v1" {
		t.Fatalf("v1 key: %v", aerr)
	}
	// Bearer 形式
	r = httptest.NewRequest(http.MethodPost, "/api/open/decision", bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+testSecret)
	if _, aerr := a.Authenticate(ctx, r, netip.MustParseAddr("203.0.113.9")); aerr != nil {
		t.Fatalf("bearer: %v", aerr)
	}
	// 白名单外
	r = httptest.NewRequest(http.MethodPost, "/api/open/decision", bytes.NewReader(body))
	r.Header.Set(HeaderV1Key, testSecret)
	if _, aerr := a.Authenticate(ctx, r, netip.MustParseAddr("198.51.100.1")); aerr == nil || aerr.Code != "ip_not_allowed" {
		t.Fatalf("allowlist: %v", aerr)
	}
	// 错 key
	r = httptest.NewRequest(http.MethodPost, "/api/open/decision", bytes.NewReader(body))
	r.Header.Set(HeaderV1Key, testSecret[:12]+"x")
	if _, aerr := a.Authenticate(ctx, r, netip.MustParseAddr("203.0.113.9")); aerr == nil || aerr.Code != "invalid_api_key" {
		t.Fatalf("wrong key: %v", aerr)
	}
	// 无凭证
	r = httptest.NewRequest(http.MethodPost, "/api/open/decision", bytes.NewReader(body))
	if _, aerr := a.Authenticate(ctx, r, netip.MustParseAddr("203.0.113.9")); aerr == nil || aerr.Code != "missing_credentials" {
		t.Fatalf("none: %v", aerr)
	}
}

func TestDisabledSiteAndNoSecret(t *testing.T) {
	a, site := testAuth(t, nil)
	ctx := context.Background()
	body := []byte(`{}`)
	site.Status = "disabled"
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	r.Header.Set(HeaderV1Key, testSecret)
	if _, aerr := a.Authenticate(ctx, r, netip.Addr{}); aerr == nil || aerr.Code != "site_disabled" {
		t.Fatalf("disabled: %v", aerr)
	}
	site.Status = "enabled"
	site.SecretEnc = nil
	r = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	r.Header.Set(HeaderV1Key, testSecret)
	if _, aerr := a.Authenticate(ctx, r, netip.Addr{}); aerr == nil || aerr.Code != "secret_unavailable" {
		t.Fatalf("no enc: %v", aerr)
	}
}

func TestLimiterNilRedis(t *testing.T) {
	l := NewLimiter(nil, 0, 0)
	ok, err := l.Allow(context.Background(), &snapshot.Site{})
	if !ok || err == nil {
		t.Fatal("nil redis should allow and report")
	}
}
