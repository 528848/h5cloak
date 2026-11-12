package cloak

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// 回源时删除的逐跳头与源站指纹头。
var (
	hopHeaders = []string{
		"Connection", "Proxy-Connection", "Keep-Alive", "Transfer-Encoding",
		"TE", "Trailer", "Upgrade", "Proxy-Authenticate", "Proxy-Authorization",
	}
	stripResponseHeaders = []string{
		"Server", "X-Powered-By", "Via", "X-AspNet-Version", "X-AspNetMvc-Version",
		"Content-Security-Policy", "Content-Security-Policy-Report-Only",
		"Strict-Transport-Security", "Public-Key-Pins", "Report-To", "NEL",
		"Content-Encoding", "Content-Length",
	}
	maxProxyBody int64 = 8 << 20 // 8MB 内才做正文改写
)

// Handle 返回一个把判定 + 分流包进去的中间件。
// 放行走 next（客户自己的真实内容）；审核在本机深度代理审核站或直接输出平台模板。
func (c *Client) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.shouldSkip(r) {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), c.cfg.Timeout)
		defer cancel()
		d, err := c.Decide(ctx, r)
		if err != nil {
			if c.cfg.FailOpen {
				next.ServeHTTP(w, r)
				return
			}
			d = &Decision{Content: "review"} // 兜底按审核
		}
		c.Apply(w, r, d, next)
	})
}

// Apply 按判定结果分流。调用方可用它拼自己的框架中间件。
func (c *Client) Apply(w http.ResponseWriter, r *http.Request, d *Decision, next http.Handler) {
	if d.Cookie != nil && d.Cookie.Value != "" {
		setCookie(w, r, d.Cookie)
	}
	switch d.Content {
	case "real", "bypass":
		next.ServeHTTP(w, r)
	case "block":
		http.Error(w, "Forbidden", http.StatusForbidden)
	case "not_found":
		http.NotFound(w, r)
	default: // review
		if d.RenderHint != nil && d.RenderHint.Type == "html" && d.RenderHint.HTML != "" {
			h := w.Header()
			h.Set("Content-Type", "text/html; charset=utf-8")
			h.Set("Cache-Control", "no-store")
			h.Set("X-Robots-Tag", "noindex, nofollow")
			writeBody(w, r, []byte(d.RenderHint.HTML), http.StatusOK)
			return
		}
		if err := c.ProxyReview(w, r); err != nil {
			http.Error(w, "Service Unavailable", http.StatusBadGateway)
		}
	}
}

// ProxyReview 在客户服务器内深度代理审核站：回源 → 改写头与正文（审核站域名→客户域名）→ 输出。
// 只代理配置的 ReviewOrigin，访客无法通过 path 控制目标主机（防 SSRF）。
func (c *Client) ProxyReview(w http.ResponseWriter, r *http.Request) error {
	origin, err := url.Parse(strings.TrimRight(c.cfg.ReviewOrigin, "/"))
	if err != nil || origin.Host == "" {
		return errBadOrigin
	}
	target := *origin
	target.Path = singleJoin(origin.Path, r.URL.Path)
	target.RawQuery = r.URL.RawQuery

	ctx, cancel := context.WithTimeout(r.Context(), c.cfg.Timeout)
	defer cancel()
	method := r.Method
	if method == "" {
		method = http.MethodGet
	}
	var reqBody io.Reader
	if method != http.MethodGet && method != http.MethodHead {
		reqBody = r.Body
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), reqBody)
	if err != nil {
		return err
	}
	// 回源 Host / SNI：源站用 IP + 自定义 Host 时
	hostHeader := c.cfg.ReviewHost
	if hostHeader == "" {
		hostHeader = origin.Host
	}
	req.Host = stripPort(hostHeader)
	req.Header.Set("Accept-Encoding", "identity") // 便于改写
	if ua := r.UserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	if al := r.Header.Get("Accept-Language"); al != "" {
		req.Header.Set("Accept-Language", al)
	}
	if ac := r.Header.Get("Accept"); ac != "" {
		req.Header.Set("Accept", ac)
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if ip := c.clientIP(r); ip != "" {
		req.Header.Set("X-Forwarded-For", ip)
		req.Header.Set("X-Real-IP", ip)
	}
	req.Header.Set("X-Forwarded-Proto", requestScheme(r))

	hc := c.hc
	if sni := tlsServerName(target.Hostname(), hostHeader); sni != "" {
		hc = c.clientWithSNI(sni)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	pub := c.visitorHost(r)
	scheme := requestScheme(r)
	hosts := []string{stripPort(target.Hostname())}
	if h := stripPort(hostHeader); h != "" && !strings.EqualFold(h, hosts[0]) {
		hosts = append(hosts, h)
	}

	rewriteResponseHeaders(resp.Header, origin, hostHeader, pub, scheme)
	out := w.Header()
	for k, vs := range resp.Header {
		for _, v := range vs {
			out.Add(k, v)
		}
	}
	out.Set("Cache-Control", "no-store")
	out.Set("X-Robots-Tag", "noindex, nofollow")

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProxyBody+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > maxProxyBody {
		// 过大不改写，原样透传（流式）
		out.Del("Content-Length")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		_, _ = io.Copy(w, resp.Body)
		return nil
	}
	ct := resp.Header.Get("Content-Type")
	needRewrite := RewritableContentType(ct)
	if ct == "" {
		needRewrite = LooksLikeHTML(body)
	}
	if needRewrite {
		body = RewriteBodyHost(body, hosts, pub, scheme)
	}
	writeBody(w, r, body, resp.StatusCode)
	return nil
}

// rewriteResponseHeaders 删敏感/逐跳头，改写 Location 与 Set-Cookie Domain。
func rewriteResponseHeaders(h http.Header, origin *url.URL, originHost, publicHost, scheme string) {
	for _, k := range stripResponseHeaders {
		h.Del(k)
	}
	for _, k := range hopHeaders {
		h.Del(k)
	}
	if loc := h.Get("Location"); loc != "" {
		h.Set("Location", rewriteLocation(loc, origin, originHost, publicHost, scheme))
	}
	if cookies := h.Values("Set-Cookie"); len(cookies) > 0 {
		h.Del("Set-Cookie")
		for _, ck := range cookies {
			h.Add("Set-Cookie", rewriteCookieDomain(ck, origin.Hostname(), originHost, publicHost))
		}
	}
}

func rewriteLocation(loc string, origin *url.URL, originHost, publicHost, scheme string) string {
	u, err := url.Parse(loc)
	if err != nil || u.Host == "" {
		return loc
	}
	if hostEq(u.Host, origin.Host) || hostEq(u.Host, originHost) {
		u.Scheme = scheme
		u.Host = publicHost
		if p := strings.TrimSuffix(origin.Path, "/"); p != "" && strings.HasPrefix(u.Path, p) {
			u.Path = strings.TrimPrefix(u.Path, p)
			if u.Path == "" {
				u.Path = "/"
			}
		}
		return u.String()
	}
	return loc
}

func rewriteCookieDomain(c, originHostname, originHost, publicHost string) string {
	parts := strings.Split(c, ";")
	out := parts[:0]
	for _, p := range parts {
		kv := strings.TrimSpace(p)
		if strings.HasPrefix(strings.ToLower(kv), "domain=") {
			d := strings.TrimPrefix(kv[7:], ".")
			pub := stripPort(publicHost)
			if hostEq(d, originHostname) || hostEq(d, stripPort(originHost)) || (!hostEq(d, pub) && !strings.HasSuffix(pub, "."+d)) {
				continue
			}
		}
		out = append(out, p)
	}
	return strings.Join(out, ";")
}

func hostEq(a, b string) bool {
	return a != "" && b != "" && strings.EqualFold(stripPort(a), stripPort(b))
}

// tlsServerName 回源 Host 是另一个域名（非 IP）且与 URL 主机不同时，用它做 SNI。
func tlsServerName(urlHost, hostHeader string) string {
	name := stripPort(strings.TrimSpace(hostHeader))
	name = strings.TrimSuffix(name, ".")
	if name == "" || net.ParseIP(name) != nil {
		return ""
	}
	if strings.EqualFold(name, stripPort(urlHost)) {
		return ""
	}
	return name
}

func (c *Client) clientWithSNI(sni string) *http.Client {
	base, _ := c.hc.Transport.(*http.Transport)
	var tr *http.Transport
	if base != nil {
		tr = base.Clone()
	} else {
		tr = http.DefaultTransport.(*http.Transport).Clone()
	}
	if tr.TLSClientConfig == nil {
		tr.TLSClientConfig = &tls.Config{}
	} else {
		tr.TLSClientConfig = tr.TLSClientConfig.Clone()
	}
	tr.TLSClientConfig.ServerName = sni
	return &http.Client{Timeout: c.cfg.Timeout, Transport: tr}
}

func setCookie(w http.ResponseWriter, r *http.Request, ck *Cookie) {
	out := &http.Cookie{
		Name: ck.Name, Value: ck.Value, Path: firstNonEmptyStr(ck.Path, "/"),
		MaxAge: ck.MaxAge, Domain: ck.Domain, Secure: ck.Secure, HttpOnly: ck.HTTPOnly,
	}
	switch strings.ToLower(ck.SameSite) {
	case "strict":
		out.SameSite = http.SameSiteStrictMode
	case "none":
		out.SameSite = http.SameSiteNoneMode
	default:
		out.SameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, out)
}

// writeBody 按访客 Accept-Encoding 可选 gzip 后输出。
func writeBody(w http.ResponseWriter, r *http.Request, body []byte, status int) {
	h := w.Header()
	if acceptsGzip(r) && len(body) > 1024 {
		if gz, err := gzipBytes(body); err == nil {
			body = gz
			h.Set("Content-Encoding", "gzip")
			h.Add("Vary", "Accept-Encoding")
		}
	}
	h.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip")
}

func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func singleJoin(a, b string) string {
	a = strings.TrimSuffix(a, "/")
	if b == "" {
		if a == "" {
			return "/"
		}
		return a
	}
	if !strings.HasPrefix(b, "/") {
		b = "/" + b
	}
	return a + b
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

var errBadOrigin = &proxyError{"cloak: invalid ReviewOrigin"}

type proxyError struct{ msg string }

func (e *proxyError) Error() string { return e.msg }
