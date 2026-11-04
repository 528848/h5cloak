package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"cloak/internal/config"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
	"cloak/internal/engine/ua"
)

func newProxy() *Proxy {
	return New(config.Proxy{MaxBodyBytes: 1024, ReviewNoIndex: true}, zap.NewNop())
}

func site(realURL, reviewURL string) *snapshot.Site {
	s := &snapshot.Site{
		SiteID: 1, Status: "enabled", MerchantActive: true,
		Domains:     []snapshot.Domain{{Domain: "shop.test", Status: "bound"}},
		Cookie:      snapshot.Cookie{Name: "_ck_vid"},
		Review:      snapshot.Review{Source: "template", TemplateHTML: "<html><body>REVIEW {{.siteName}}</body></html>", TemplateVars: map[string]string{"siteName": "Demo"}},
		Handoff:     snapshot.Handoff{Banner: true, BannerText: "Open in browser", LinkRewrite: true},
		Attribution: snapshot.Attribution{FailMode: "review"},
	}
	if realURL != "" {
		s.Origins = append(s.Origins, snapshot.Origin{Kind: "real", URL: realURL, TimeoutMs: 2000})
	}
	if reviewURL != "" {
		s.Origins = append(s.Origins, snapshot.Origin{Kind: "review", URL: reviewURL, TimeoutMs: 2000})
	}
	_ = s.Compile()
	return s
}

func realDecision() *engine.Decision {
	return &engine.Decision{Content: engine.ContentReal, TouchType: engine.TouchFirst, VisitorID: "vid1", InApp: ua.Info{OS: "windows"}}
}

func do(p *Proxy, s *snapshot.Site, d *engine.Decision, r *http.Request) (*httptest.ResponseRecorder, Result) {
	rec := httptest.NewRecorder()
	res := p.Serve(rec, r, Request{Site: s, Decision: d, ClientIP: "203.0.113.9", Scheme: "https", RequestID: "req1"})
	return rec, res
}

func TestForwardRealRewritesHeaders(t *testing.T) {
	var got *http.Request
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		w.Header().Set("Server", "nginx")
		w.Header().Set("X-Powered-By", "PHP")
		w.Header().Set("Set-Cookie", "sess=1; Domain="+strings.TrimPrefix(r.Host, "")+"; Path=/")
		w.Header().Set("Location", "http://"+r.Host+"/next")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusFound)
		_, _ = w.Write([]byte("hello"))
	}))
	defer origin.Close()
	s := site(origin.URL, "")
	s.Origins[0].HostHeader = "origin.internal"
	_ = s.Compile()

	r := httptest.NewRequest(http.MethodGet, "https://shop.test/path?a=1", nil)
	r.Host = "shop.test"
	r.Header.Set("X-Cloak-Content", "forged")
	r.Header.Set("Connection", "keep-alive")
	rec, res := do(newProxy(), s, realDecision(), r)

	if res.Kind != "real" || res.OriginStatus != 302 || rec.Code != 302 {
		t.Fatalf("result %+v code %d", res, rec.Code)
	}
	if got.Host != "origin.internal" || got.URL.Path != "/path" || got.URL.RawQuery != "a=1" {
		t.Fatalf("origin request: host=%s url=%s", got.Host, got.URL)
	}
	if got.Header.Get("X-Cloak-Content") != "real" || got.Header.Get("X-Cloak-Touch") != "first" || got.Header.Get("X-Cloak-Visitor") != "vid1" {
		t.Fatalf("meta headers: %v", got.Header)
	}
	if got.Header.Get("X-Forwarded-For") != "203.0.113.9" || got.Header.Get("X-Forwarded-Proto") != "https" || got.Header.Get("X-Forwarded-Host") != "shop.test" {
		t.Fatalf("forward headers: %v", got.Header)
	}
	if rec.Header().Get("Server") != "" || rec.Header().Get("X-Powered-By") != "" {
		t.Fatal("server headers leaked")
	}
	if loc := rec.Header().Get("Location"); loc != "https://shop.test/next" {
		t.Fatalf("location: %s", loc)
	}
	if sc := rec.Header().Get("Set-Cookie"); strings.Contains(strings.ToLower(sc), "domain=") {
		t.Fatalf("origin cookie domain should be dropped: %s", sc)
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "no-store") || rec.Header().Get("Vary") != "Cookie" {
		t.Fatalf("cache headers: %v", rec.Header())
	}
	if rec.Body.String() != "hello" {
		t.Fatal("body")
	}
}

func TestReviewSources(t *testing.T) {
	review := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("REVIEW ORIGIN"))
	}))
	defer review.Close()
	p := newProxy()
	d := &engine.Decision{Content: engine.ContentReview, TouchType: engine.TouchNone}

	// 模板
	s := site("", "")
	rec, res := do(p, s, d, httptest.NewRequest(http.MethodGet, "https://shop.test/", nil))
	if res.Kind != "template" || !strings.Contains(rec.Body.String(), "REVIEW Demo") || rec.Header().Get("X-Robots-Tag") == "" {
		t.Fatalf("template: %+v %s", res, rec.Body.String())
	}
	// 审核源站
	s = site("", review.URL)
	s.Review.Source = "origin"
	rec, res = do(p, s, d, httptest.NewRequest(http.MethodGet, "https://shop.test/", nil))
	if res.Kind != "review" || rec.Body.String() != "REVIEW ORIGIN" || rec.Header().Get("X-Robots-Tag") == "" {
		t.Fatalf("review origin: %+v %s", res, rec.Body.String())
	}
	// 审核源站不可达 → 模板
	s = site("", "http://127.0.0.1:1")
	s.Review.Source = "origin"
	rec, res = do(p, s, d, httptest.NewRequest(http.MethodGet, "https://shop.test/", nil))
	if res.Kind != "template" || !strings.Contains(rec.Body.String(), "REVIEW Demo") {
		t.Fatalf("review origin down: %+v", res)
	}
	// 静态包
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>STATIC</h1>"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "a.css"), []byte("body{}"), 0o644)
	s = site("", "")
	s.Review = snapshot.Review{Source: "static", StaticDir: dir, StaticEntry: "index.html"}
	rec, res = do(p, s, d, httptest.NewRequest(http.MethodGet, "https://shop.test/a.css", nil))
	if res.Kind != "static" || rec.Body.String() != "body{}" {
		t.Fatalf("static file: %+v %s", res, rec.Body.String())
	}
	rec, res = do(p, s, d, httptest.NewRequest(http.MethodGet, "https://shop.test/deep/unknown", nil))
	if res.Kind != "static" || rec.Body.String() != "<h1>STATIC</h1>" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("static spa fallback: %+v %s", res, rec.Body.String())
	}
	rec, _ = do(p, s, d, httptest.NewRequest(http.MethodGet, "https://shop.test/../../etc/passwd", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "<h1>STATIC</h1>" {
		t.Fatalf("traversal must fall back to entry: %d", rec.Code)
	}
}

func TestFallbackModes(t *testing.T) {
	p := newProxy()
	for mode, want := range map[string]struct {
		kind   string
		status int
	}{
		"review":      {"template", 200},
		"maintenance": {"maintenance", 503},
		"real":        {"maintenance", 502},
	} {
		s := site("http://127.0.0.1:1", "")
		s.Attribution.FailMode = mode
		rec, res := do(p, s, realDecision(), httptest.NewRequest(http.MethodGet, "https://shop.test/", nil))
		if res.Kind != want.kind || rec.Code != want.status || !res.Fallback || res.Error != "origin_unreachable" {
			t.Errorf("%s: %+v code=%d", mode, res, rec.Code)
		}
	}
	// 没配真实源站
	s := site("", "")
	_, res := do(p, s, realDecision(), httptest.NewRequest(http.MethodGet, "https://shop.test/", nil))
	if res.Kind != "template" || res.Error != "no_real_origin" {
		t.Fatalf("no origin: %+v", res)
	}
	// bypass 没源站 → 404
	rec, res := do(p, s, &engine.Decision{Content: engine.ContentBypass}, httptest.NewRequest(http.MethodGet, "https://shop.test/a.css", nil))
	if rec.Code != 404 || res.Kind != "not_found" {
		t.Fatalf("bypass no origin: %+v", res)
	}
}

func TestBlockNotFoundMaintenance(t *testing.T) {
	p := newProxy()
	s := site("", "")
	for content, status := range map[string]int{engine.ContentBlock: 403, engine.ContentNotFound: 404, engine.ContentMaintenance: 503} {
		rec, _ := do(p, s, &engine.Decision{Content: content}, httptest.NewRequest(http.MethodGet, "https://shop.test/", nil))
		if rec.Code != status || !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Errorf("%s: %d", content, rec.Code)
		}
	}
}

func TestBodyLimit(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_, _ = w.Write([]byte(strings.Repeat("x", len(b))))
	}))
	defer origin.Close()
	s := site(origin.URL, "")
	p := newProxy()
	r := httptest.NewRequest(http.MethodPost, "https://shop.test/upload", strings.NewReader(strings.Repeat("a", 2048)))
	rec, res := do(p, s, realDecision(), r)
	if rec.Code != http.StatusRequestEntityTooLarge || res.Error != "body_too_large" {
		t.Fatalf("limit: %d %+v", rec.Code, res)
	}
	r = httptest.NewRequest(http.MethodPost, "https://shop.test/upload", strings.NewReader(strings.Repeat("a", 100)))
	rec, _ = do(p, s, realDecision(), r)
	if rec.Code != 200 || len(rec.Body.String()) != 100 {
		t.Fatalf("small body: %d %d", rec.Code, rec.Body.Len())
	}
}

func TestHTMLInjection(t *testing.T) {
	var gotAccept string
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept-Encoding")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><a href="/p?x=1">in</a><a href="https://other.com/">out</a><a href="#top">anchor</a><form action="/buy"></form></body></html>`))
	}))
	defer origin.Close()
	s := site(origin.URL, "")
	p := newProxy()
	d := realDecision()
	d.HandoffToken = "TOK123"
	d.InApp = ua.Info{InApp: true, App: "facebook", OS: "ios"}
	r := httptest.NewRequest(http.MethodGet, "https://shop.test/lp?ref=A", nil)
	r.Host = "shop.test"
	rec, res := do(p, s, d, r)
	body := rec.Body.String()
	if res.Kind != "real" || gotAccept != "identity" {
		t.Fatalf("accept-encoding: %q %+v", gotAccept, res)
	}
	if !strings.Contains(body, `id="ck-handoff"`) || !strings.Contains(body, "Open in browser") || !strings.Contains(body, "x-safari-https://shop.test/lp?") {
		t.Fatalf("banner missing: %s", body)
	}
	if !strings.Contains(body, `href="/p?_ck=TOK123&amp;x=1"`) || !strings.Contains(body, `action="/buy?_ck=TOK123"`) {
		t.Fatalf("link rewrite: %s", body)
	}
	if !strings.Contains(body, `href="https://other.com/"`) || !strings.Contains(body, `href="#top"`) {
		t.Fatalf("external/anchor must be untouched: %s", body)
	}
	if cl := rec.Header().Get("Content-Length"); cl == "" || cl == "0" {
		t.Fatal("content-length must be recomputed")
	}
	// 系统浏览器不注入
	d2 := realDecision()
	d2.HandoffToken = ""
	rec, _ = do(p, s, d2, httptest.NewRequest(http.MethodGet, "https://shop.test/lp", nil))
	if strings.Contains(rec.Body.String(), "ck-handoff") {
		t.Fatal("must not inject for system browser")
	}
}

func TestStreaming(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f := w.(http.Flusher)
		for i := 0; i < 3; i++ {
			_, _ = w.Write([]byte("data: tick\n\n"))
			f.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer origin.Close()
	s := site(origin.URL, "")
	rec, res := do(newProxy(), s, realDecision(), httptest.NewRequest(http.MethodGet, "https://shop.test/events", nil))
	if res.OriginStatus != 200 || strings.Count(rec.Body.String(), "data: tick") != 3 || !rec.Flushed {
		t.Fatalf("stream: %+v flushed=%v body=%q", res, rec.Flushed, rec.Body.String())
	}
}

func TestWebSocket(t *testing.T) {
	// 极简 WS 源站：接受升级后把收到的字节原样回写
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" || r.Header.Get("X-Cloak-Content") != "real" {
			http.Error(w, "bad", 400)
			return
		}
		hj := w.(http.Hijacker)
		conn, buf, _ := hj.Hijack()
		defer conn.Close()
		_, _ = buf.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = buf.Flush()
		line, _ := buf.ReadString('\n')
		_, _ = buf.WriteString("echo:" + line)
		_ = buf.Flush()
	}))
	defer origin.Close()
	s := site(origin.URL, "")
	p := newProxy()

	// 用真实 listener 承载代理，httptest.Recorder 不支持 Hijack
	front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.Serve(w, r, Request{Site: s, Decision: realDecision(), ClientIP: "1.1.1.1", Scheme: "http", RequestID: "ws"})
	}))
	defer front.Close()
	u, _ := url.Parse(front.URL)
	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = conn.Write([]byte("GET /ws HTTP/1.1\r\nHost: shop.test\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: x\r\nSec-WebSocket-Version: 13\r\n\r\n"))
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil || resp.StatusCode != 101 {
		t.Fatalf("upgrade: %v %v", err, resp)
	}
	_, _ = conn.Write([]byte("ping\n"))
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := br.ReadString('\n')
	if err != nil || line != "echo:ping\n" {
		t.Fatalf("echo: %q %v", line, err)
	}
}

func TestRewriteHelpers(t *testing.T) {
	origin, _ := url.Parse("https://origin.example.net/base/")
	if got := rewriteLocation("https://origin.example.net/base/x?y=1", origin, "", "shop.test", "https"); got != "https://shop.test/x?y=1" {
		t.Fatalf("location with prefix: %s", got)
	}
	if got := rewriteLocation("https://origin.example.net/base", origin, "", "shop.test", "https"); got != "https://shop.test/" {
		t.Fatalf("location root: %s", got)
	}
	if got := rewriteLocation("https://cdn.other/x", origin, "", "shop.test", "https"); got != "https://cdn.other/x" {
		t.Fatal("external location untouched")
	}
	if got := rewriteLocation("/relative", origin, "", "shop.test", "https"); got != "/relative" {
		t.Fatal("relative location untouched")
	}
	if got := rewriteLocation("https://hdr.host/x", origin, "hdr.host", "shop.test", "https"); got != "https://shop.test/x" {
		t.Fatal("host header location")
	}
	c := rewriteCookieDomain("a=1; Domain=.origin.example.net; Path=/; Secure", "origin.example.net", "", "shop.test")
	if strings.Contains(c, "Domain") || !strings.Contains(c, "Secure") {
		t.Fatalf("cookie: %s", c)
	}
	c = rewriteCookieDomain("a=1; Domain=shop.test", "origin.example.net", "", "shop.test")
	if !strings.Contains(c, "Domain=shop.test") {
		t.Fatalf("own domain kept: %s", c)
	}
	c = rewriteCookieDomain("a=1; Domain=.test", "origin.example.net", "", "shop.test")
	if !strings.Contains(c, "Domain=.test") {
		t.Fatalf("parent domain kept: %s", c)
	}
	if stripPort("[::1]:80") != "[::1]" || stripPort("a.b:1") != "a.b" || stripPort("a.b") != "a.b" {
		t.Fatal("stripPort")
	}
	if openInBrowserHref("https://s/x", "android") != "intent://s/x#Intent;scheme=https;action=android.intent.action.VIEW;end" {
		t.Fatal("android intent")
	}
	if openInBrowserHref("http://s/x", "ios") != "http://s/x" || openInBrowserHref("https://s/x", "windows") != "https://s/x" {
		t.Fatal("non-https ios / other os")
	}
}
