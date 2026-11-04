package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/config"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
	"cloak/internal/pkg/requestid"
)

func bare() *Handler {
	return New(Deps{Cfg: &config.Config{}, Log: zap.NewNop(), Metrics: prometheus.NewRegistry()})
}

func withSite() (*Handler, *snapshot.Site) {
	site := &snapshot.Site{
		SiteID: 3, Status: "enabled", MerchantActive: true,
		Domains:     []snapshot.Domain{{Domain: "shop.test", Status: "bound"}},
		Cookie:      snapshot.Cookie{Name: "_ck_vid"},
		Attribution: snapshot.Attribution{TTLDays: 30, RenewMode: "sliding", NoAttributionAction: "review", MinConfidence: "medium"},
		Keys:        []snapshot.TokenKey{{Version: 1, Key: "abcd", Status: "active"}},
		Params:      []snapshot.Param{{Name: "ref", Type: "distributor", MinLen: 2, MaxLen: 32, Sufficient: true, Enabled: true}},
		Rule:        snapshot.Rule{CombineRule: "require_any", ValidateCodes: true},
		Codes:       []snapshot.Code{{ID: 1, Code: "OK", Type: "distributor", Enabled: true}},
		PathRules:   []snapshot.PathRule{{Pattern: "/admin", MatchType: "prefix", Action: "block", Sort: 1}},
	}
	store := snapshot.NewStore(nil)
	store.Put(site)
	cfg := &config.Config{}
	cfg.Gateway.TrustedProxies = []string{"10.0.0.0/8"}
	h := New(Deps{Cfg: cfg, Log: zap.NewNop(), Metrics: prometheus.NewRegistry(), Store: store, Engine: engine.New(engine.NewMemStore())})
	return h, site
}

func TestHealth(t *testing.T) {
	h := bare()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_cloak/health", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("unexpected %d %q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get(requestid.HeaderName) == "" {
		t.Fatal("missing request id")
	}
}

func TestUnmatchedIs404WithoutServerHeader(t *testing.T) {
	h := bare()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/anything", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Server") != "" {
		t.Fatal("server header must not be exposed")
	}
}

func TestDecisionFlow(t *testing.T) {
	h, _ := withSite()

	// 未知域名 404
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://other.test/", nil)
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown host: %d", rec.Code)
	}

	// 带码首访：200、real、Set-Cookie
	rec = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "http://shop.test/lp?ref=OK", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK || rec.Header().Get("X-Cloak-Content") != "real" {
		t.Fatalf("first visit: %d %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "_ck_vid" || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].MaxAge <= 0 {
		t.Fatalf("cookie: %+v", cookies)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["touchType"] != "first" || body["attributed"] != true {
		t.Fatalf("body: %v", body)
	}

	// 回访：带 cookie 无参数 → real/return
	rec = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "http://shop.test/products", nil)
	r.AddCookie(cookies[0])
	h.ServeHTTP(rec, r)
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["content"] != "real" || body["touchType"] != "return" {
		t.Fatalf("return: %v", body)
	}

	// 无归因 → review 200
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://shop.test/products", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("X-Cloak-Content") != "review" {
		t.Fatalf("review: %d", rec.Code)
	}

	// 路径 block → 403
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://shop.test/admin/x", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("block: %d", rec.Code)
	}

	// 状态端点
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_cloak/status", nil))
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusOK || body["snapshots"] == nil {
		t.Fatalf("status: %d %v", rec.Code, body)
	}
}

func TestClientIP(t *testing.T) {
	h, _ := withSite()
	r := httptest.NewRequest(http.MethodGet, "http://shop.test/", nil)
	r.RemoteAddr = "10.1.1.1:5000"
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.2.2.2")
	if ip := h.clientIP(r); ip.String() != "203.0.113.9" {
		t.Fatalf("xff from trusted proxy: %s", ip)
	}
	r.RemoteAddr = "198.51.100.1:5000"
	if ip := h.clientIP(r); ip.String() != "198.51.100.1" {
		t.Fatalf("untrusted remote must ignore xff: %s", ip)
	}
	r.RemoteAddr = "10.1.1.1:5000"
	r.Header.Del("X-Forwarded-For")
	r.Header.Set("X-Real-IP", "192.0.2.4")
	if ip := h.clientIP(r); ip.String() != "192.0.2.4" {
		t.Fatalf("x-real-ip: %s", ip)
	}
	r.Header.Del("X-Real-IP")
	if ip := h.clientIP(r); ip.String() != "10.1.1.1" {
		t.Fatalf("fallback remote: %s", ip)
	}
	r.RemoteAddr = "garbage"
	if ip := h.clientIP(r); ip.IsValid() {
		t.Fatal("garbage remote")
	}
}
