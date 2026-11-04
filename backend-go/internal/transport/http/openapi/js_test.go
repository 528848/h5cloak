package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
	"cloak/internal/pkg/secretbox"
)

func jsTestHandler(t *testing.T) (*Handler, *snapshot.Site) {
	t.Helper()
	site := &snapshot.Site{
		SiteID: 11, MerchantID: 2, Status: "enabled", MerchantActive: true, PublishableKey: "pk_test_js_0001",
		Domains:     []snapshot.Domain{{Domain: "shop.example.com", Status: "bound"}},
		Cookie:      snapshot.Cookie{Name: "_ck_vid"},
		Attribution: snapshot.Attribution{TTLDays: 30, RenewMode: "sliding", NoAttributionAction: "review", MinConfidence: "medium"},
		Handoff:     snapshot.Handoff{TokenTTLHours: 24, TokenMaxUses: 2, Banner: true, BannerText: "open in browser"},
		Keys:        []snapshot.TokenKey{{Version: 1, Key: "0a0b0c0d0e0f", Status: "active"}},
		Params: []snapshot.Param{
			{Name: "ref", Type: "distributor", MinLen: 2, MaxLen: 64, Sufficient: true, Enabled: true},
		},
		Rule:  snapshot.Rule{CombineRule: "require_any", ValidateCodes: true},
		Codes: []snapshot.Code{{ID: 1, Code: "ALICE", Type: "distributor", Enabled: true}},
	}
	if err := site.Compile(); err != nil {
		t.Fatal(err)
	}
	store := snapshot.NewStore(nil)
	store.Put(site)
	box, _ := secretbox.New("master")
	h := New(Deps{
		Auth:   NewAuthenticator(store, box, nil),
		Limit:  NewLimiter(nil, 0, 0),
		Engine: engine.New(engine.NewMemStore()),
	})
	return h, site
}

func jsPost(h *Handler, origin, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/open/v2/js/capture", strings.NewReader(body))
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	r.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestOriginAllowed(t *testing.T) {
	_, site := jsTestHandler(t)
	cases := map[string]bool{
		"https://shop.example.com":      true,
		"https://SHOP.example.com:8443": true,
		"http://localhost:5173":         true,
		"http://127.0.0.1":              true,
		"https://evil.example.com":      false,
		"https://shop.example.com.evil": false,
		"":                              false,
		"not a url":                     false,
	}
	for origin, want := range cases {
		if got := originAllowed(origin, site); got != want {
			t.Errorf("originAllowed(%q)=%v want %v", origin, got, want)
		}
	}
}

func TestJSCaptureFlow(t *testing.T) {
	h, _ := jsTestHandler(t)

	// 预检
	r := httptest.NewRequest(http.MethodOptions, "/api/open/v2/js/capture", nil)
	r.Header.Set("Origin", "https://shop.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "https://shop.example.com" {
		t.Fatalf("preflight: %d %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}

	// 未知公钥
	w = jsPost(h, "https://shop.example.com", `{"key":"pk_nope","url":"https://shop.example.com/"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unknown key: %d %s", w.Code, w.Body.String())
	}
	// Origin 不在站点域名
	w = jsPost(h, "https://evil.example.com", `{"key":"pk_test_js_0001","url":"https://shop.example.com/"}`)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "origin_not_allowed") {
		t.Fatalf("bad origin: %d %s", w.Code, w.Body.String())
	}
	// 缺字段
	w = jsPost(h, "https://shop.example.com", `{"key":"pk_test_js_0001"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing url: %d", w.Code)
	}

	// 带有效推广码：real + 令牌 + cookie 元数据
	w = jsPost(h, "https://shop.example.com", `{"key":"pk_test_js_0001","url":"https://shop.example.com/lp?ref=ALICE"}`)
	if w.Code != 200 {
		t.Fatalf("decide: %d %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			Content      string         `json:"content"`
			Attributed   bool           `json:"attributed"`
			VisitorToken string         `json:"visitorToken"`
			Cookie       map[string]any `json:"cookie"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Content != "real" || !env.Data.Attributed || env.Data.VisitorToken == "" || env.Data.Cookie["name"] != "_ck_vid" {
		t.Fatalf("attributed: %+v", env.Data)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://shop.example.com" {
		t.Fatal("cors header missing on success")
	}

	// 回访：带令牌无参数 → real
	w = jsPost(h, "https://shop.example.com", `{"key":"pk_test_js_0001","url":"https://shop.example.com/other","visitorToken":"`+env.Data.VisitorToken+`"}`)
	var back struct {
		Data struct {
			Content   string `json:"content"`
			TouchType string `json:"touchType"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &back)
	if w.Code != 200 || back.Data.Content != "real" || back.Data.TouchType != "return" {
		t.Fatalf("return: %d %+v", w.Code, back.Data)
	}

	// 无参数无令牌 → review
	w = jsPost(h, "http://localhost:5173", `{"key":"pk_test_js_0001","url":"https://shop.example.com/"}`)
	_ = json.Unmarshal(w.Body.Bytes(), &back)
	if w.Code != 200 || back.Data.Content != "review" {
		t.Fatalf("review: %d %+v", w.Code, back.Data)
	}

	// SDK 脚本
	r = httptest.NewRequest(http.MethodGet, SDKPath, nil)
	w = httptest.NewRecorder()
	ServeSDK(w, r)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), "javascript") || !strings.Contains(w.Body.String(), "js/capture") {
		t.Fatalf("sdk: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
}
