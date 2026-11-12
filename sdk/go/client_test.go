package cloak

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSignGolden 跑跨语言共享的签名已知答案向量，确保 Go 与 PHP 等产出一致。
func TestSignGolden(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "sign-golden.json"))
	if err != nil {
		t.Fatalf("read sign golden: %v", err)
	}
	var g struct {
		Secret, Ts, Nonce, Method, Path, Body, Signature string
	}
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("parse sign golden: %v", err)
	}
	got := serverSign(g.Secret, g.Ts, g.Nonce, g.Method, g.Path, []byte(g.Body))
	if got != g.Signature {
		t.Fatalf("sign mismatch\n want %s\n got  %s", g.Signature, got)
	}
}

const testSecret = "sk_test_secret_key_1234567890"

// serverSign 复刻网关 internal/transport/http/openapi/auth.go 的 Sign，验证 SDK 签名一致。
func serverSign(secret, ts, nonce, method, path string, body []byte) string {
	sum := sha256.Sum256(body)
	msg := ts + "\n" + nonce + "\n" + strings.ToUpper(method) + "\n" + path + "\n" + hex.EncodeToString(sum[:])
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// decisionServer 模拟判定接口：校验 v2 签名后返回指定 content。
func decisionServer(t *testing.T, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != decisionPath {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		ts := r.Header.Get("X-Cloak-Timestamp")
		nonce := r.Header.Get("X-Cloak-Nonce")
		sig := r.Header.Get("X-Cloak-Signature")
		if r.Header.Get("X-Cloak-Key") != testSecret[:12] {
			t.Errorf("bad key prefix %q", r.Header.Get("X-Cloak-Key"))
		}
		want := serverSign(testSecret, ts, nonce, r.Method, r.URL.Path, body)
		if sig != want {
			t.Errorf("signature mismatch\n want %s\n got  %s", want, sig)
		}
		data := map[string]any{"content": content}
		if content == "real" {
			data["visitorToken"] = "vtok"
			data["cookie"] = map[string]any{"name": "_ck_vid", "value": "vtok", "maxAge": 3600, "path": "/", "sameSite": "Lax"}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "msg": "ok", "data": data})
	}))
}

func TestDecideAndSign(t *testing.T) {
	srv := decisionServer(t, "real")
	defer srv.Close()
	c := New(Config{APIBase: srv.URL, SecretKey: testSecret, ReviewOrigin: "https://review.com"})
	r := httptest.NewRequest(http.MethodGet, "https://shop.example/landing?x=1", nil)
	r.Header.Set("User-Agent", "UA")
	d, err := c.Decide(r.Context(), r)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if !d.Pass() {
		t.Fatalf("want pass, got content=%s", d.Content)
	}
	if d.Cookie == nil || d.Cookie.Value != "vtok" {
		t.Fatalf("want cookie vtok, got %+v", d.Cookie)
	}
}

func TestHandlePass(t *testing.T) {
	srv := decisionServer(t, "real")
	defer srv.Close()
	c := New(Config{APIBase: srv.URL, SecretKey: testSecret, ReviewOrigin: "https://review.com"})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("REAL-CONTENT"))
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://shop.example/p", nil)
	c.Handle(next).ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "REAL-CONTENT") {
		t.Fatalf("want real content, got %q", rec.Body.String())
	}
	if sc := rec.Result().Header.Get("Set-Cookie"); !strings.Contains(sc, "_ck_vid=vtok") {
		t.Fatalf("want cookie set, got %q", sc)
	}
}

func TestHandleReviewProxiesAndRewrites(t *testing.T) {
	// 审核站返回带自身域名的页面，经深度代理后应改成访客域名。
	review := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		host := r.Host // review 服务器收到的 Host（回源 Host）
		_, _ = w.Write([]byte(`<a href="https://` + host + `/buy">buy</a> at ` + host))
	}))
	defer review.Close()

	dsrv := decisionServer(t, "review")
	defer dsrv.Close()

	reviewHost := strings.TrimPrefix(review.URL, "http://")
	c := New(Config{
		APIBase: dsrv.URL, SecretKey: testSecret,
		ReviewOrigin: review.URL, VisitorHost: "shop.example",
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not run on review")
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://shop.example/landing", nil)
	req.Header.Set("Accept-Encoding", "identity")
	c.Handle(next).ServeHTTP(rec, req)

	out := rec.Body.String()
	if strings.Contains(out, reviewHost) {
		t.Fatalf("review host should be rewritten away, got %q", out)
	}
	if !strings.Contains(out, "shop.example/buy") || !strings.Contains(out, "at shop.example") {
		t.Fatalf("want rewritten to visitor host, got %q", out)
	}
	if rec.Result().Header.Get("X-Robots-Tag") == "" {
		t.Error("want noindex on review")
	}
}

func TestHandleFailClosedToReview(t *testing.T) {
	// 判定接口不可达，默认按审核走深度代理（这里 reviewOrigin 也不可达 → 502）。
	c := New(Config{APIBase: "http://127.0.0.1:1", SecretKey: testSecret, ReviewOrigin: "http://127.0.0.1:1"})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not run when fail-closed")
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://shop.example/p", nil)
	c.Handle(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("want 502, got %d", rec.Code)
	}
}
