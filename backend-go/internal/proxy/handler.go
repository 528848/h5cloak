package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"cloak/internal/config"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
)

// Request 一次已判定的请求。
type Request struct {
	Site      *snapshot.Site
	Decision  *engine.Decision
	ClientIP  string
	Scheme    string // 访客看到的协议 http/https
	RequestID string
}

// Result 代理结果，供日志使用。
type Result struct {
	Kind         string // real / review / template / static / block / not_found / bypass / maintenance
	OriginStatus int    // 0 表示未回源或回源失败
	Fallback     bool   // 是否触发了失败兜底
	Error        string
}

// Proxy 反向代理。
type Proxy struct {
	cfg       config.Proxy
	log       *zap.Logger
	tr        *transports
	templates *templateCache
	maint     []byte
}

// New 创建代理。
func New(cfg config.Proxy, log *zap.Logger) *Proxy {
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 32 << 20
	}
	p := &Proxy{cfg: cfg, log: log, tr: newTransports(cfg), templates: newTemplateCache(), maint: []byte(maintenanceHTML)}
	if cfg.MaintenanceHTML != "" {
		if b, err := readFile(cfg.MaintenanceHTML); err == nil {
			p.maint = b
		}
	}
	return p
}

// Close 释放连接。
func (p *Proxy) Close() { p.tr.closeIdle() }

// Serve 按判定结果响应。
func (p *Proxy) Serve(w http.ResponseWriter, r *http.Request, req Request) Result {
	d := req.Decision
	switch d.Content {
	case engine.ContentBlock:
		p.writeHTML(w, http.StatusForbidden, []byte(blockHTML), true)
		return Result{Kind: "block"}
	case engine.ContentNotFound:
		p.writeHTML(w, http.StatusNotFound, []byte(notFoundHTML), true)
		return Result{Kind: "not_found"}
	case engine.ContentBypass, engine.ContentReal:
		return p.serveReal(w, r, req)
	case engine.ContentMaintenance:
		p.writeHTML(w, http.StatusServiceUnavailable, p.maint, true)
		return Result{Kind: "maintenance"}
	default:
		return p.serveReview(w, r, req)
	}
}

func (p *Proxy) serveReal(w http.ResponseWriter, r *http.Request, req Request) Result {
	origin := req.Site.Origin("real")
	if origin == nil || origin.URL == "" {
		// 没配真实源站：bypass 类请求直接 404，放行请求走兜底
		if req.Decision.Content == engine.ContentBypass {
			p.writeHTML(w, http.StatusNotFound, []byte(notFoundHTML), true)
			return Result{Kind: "not_found", Error: "no_real_origin"}
		}
		return p.fallback(w, r, req, "no_real_origin")
	}
	kind := "real"
	if req.Decision.Content == engine.ContentBypass {
		kind = "bypass"
	}
	res := p.forward(w, r, req, origin, kind)
	if res.Error != "" && res.OriginStatus == 0 && !res.headersSent {
		return p.fallback(w, r, req, res.Error)
	}
	return res.Result
}

func (p *Proxy) serveReview(w http.ResponseWriter, r *http.Request, req Request) Result {
	site := req.Site
	switch site.Review.Source {
	case "origin":
		origin := site.Origin("review")
		if origin == nil || origin.URL == "" {
			return p.serveTemplate(w, req)
		}
		res := p.forward(w, r, req, origin, "review")
		if res.Error != "" && res.OriginStatus == 0 && !res.headersSent {
			// 审核源站挂了：退回模板，不能把真实内容给无归因访客
			return p.serveTemplate(w, req)
		}
		return res.Result
	case "static":
		if serveStatic(w, r, site) {
			return Result{Kind: "static"}
		}
		return p.serveTemplate(w, req)
	default:
		return p.serveTemplate(w, req)
	}
}

func (p *Proxy) serveTemplate(w http.ResponseWriter, req Request) Result {
	body, err := p.templates.render(req.Site)
	if err != nil {
		p.log.Warn("render review template", zap.Uint64("site", req.Site.SiteID), zap.Error(err))
		body, _ = renderTemplate(defaultReviewHTML, req.Site.Review.TemplateVars)
	}
	p.writeHTML(w, http.StatusOK, body, true)
	return Result{Kind: "template"}
}

// fallback 真实源站不可达时按 fail_mode 处理。
func (p *Proxy) fallback(w http.ResponseWriter, r *http.Request, req Request, reason string) Result {
	p.log.Warn("origin unavailable", zap.Uint64("site", req.Site.SiteID), zap.String("reason", reason))
	var res Result
	switch req.Site.Attribution.FailMode {
	case "maintenance":
		p.writeHTML(w, http.StatusServiceUnavailable, p.maint, true)
		res = Result{Kind: "maintenance"}
	case "real":
		// 真实站本身不可达，退回维护页
		p.writeHTML(w, http.StatusBadGateway, p.maint, true)
		res = Result{Kind: "maintenance"}
	default:
		res = p.serveReview(w, r, req)
	}
	res.Fallback = true
	res.Error = reason
	return res
}

func (p *Proxy) writeHTML(w http.ResponseWriter, status int, body []byte, noIndex bool) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Length", strconv.Itoa(len(body)))
	if noIndex && p.cfg.ReviewNoIndex {
		h.Set("X-Robots-Tag", "noindex, nofollow")
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

type forwardResult struct {
	Result
	headersSent bool
}

// forward 回源。
func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, req Request, origin *snapshot.Origin, kind string) forwardResult {
	target, err := url.Parse(origin.URL)
	if err != nil || target.Host == "" {
		return forwardResult{Result: Result{Kind: kind, Error: "bad_origin_url"}}
	}
	if isWebSocket(r) {
		return p.forwardWebSocket(w, r, req, target, origin, kind)
	}

	// 请求体上限
	if r.ContentLength > p.cfg.MaxBodyBytes {
		http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		return forwardResult{Result: Result{Kind: kind, Error: "body_too_large"}, headersSent: true}
	}
	body := r.Body
	if body != nil && body != http.NoBody {
		body = http.MaxBytesReader(w, body, p.cfg.MaxBodyBytes)
	}

	timeout := time.Duration(origin.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout+5*time.Second)
	defer cancel()

	out := r.Clone(ctx)
	out.Body = body
	if r.ContentLength == 0 {
		out.Body = http.NoBody
	}
	d := req.Decision
	meta := map[string]string{
		"X-Cloak-Content":    d.Content,
		"X-Cloak-Touch":      d.TouchType,
		"X-Cloak-Request-Id": req.RequestID,
	}
	if d.VisitorID != "" {
		meta["X-Cloak-Visitor"] = d.VisitorID
	}
	// 用户关闭过引导条（会话 cookie _ck_hb）就不再注入引导条；链接延续不受影响
	bannerWanted := req.Site.Handoff.Banner && !bannerDismissed(r)
	injectHTMLNeeded := kind == "real" && d.InApp.InApp && d.HandoffToken != "" && (bannerWanted || req.Site.Handoff.LinkRewrite)
	rewriteRequest(out, r, target, origin.HostHeader, req.ClientIP, req.Scheme, meta)
	if injectHTMLNeeded {
		out.Header.Set("Accept-Encoding", "identity")
	}

	tr := p.tr.get(timeout)
	resp, err := tr.RoundTrip(out)
	if err != nil && isRetryable(r, err) {
		// 幂等方法且在拿到响应头前失败：换连接重试一次
		out2 := r.Clone(ctx)
		out2.Body = http.NoBody
		rewriteRequest(out2, r, target, origin.HostHeader, req.ClientIP, req.Scheme, meta)
		if injectHTMLNeeded {
			out2.Header.Set("Accept-Encoding", "identity")
		}
		resp, err = tr.RoundTrip(out2)
	}
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			return forwardResult{Result: Result{Kind: kind, Error: "body_too_large"}, headersSent: true}
		}
		return forwardResult{Result: Result{Kind: kind, Error: classify(err)}}
	}
	defer resp.Body.Close()

	return p.writeResponse(w, r, req, resp, target, origin, kind, injectHTMLNeeded)
}

func (p *Proxy) writeResponse(w http.ResponseWriter, r *http.Request, req Request, resp *http.Response, target *url.URL, origin *snapshot.Origin, kind string, inject bool) forwardResult {
	h := w.Header()
	for k, vv := range resp.Header {
		for _, v := range vv {
			h.Add(k, v)
		}
	}
	rewriteResponseHeaders(h, target, origin.HostHeader, r.Host, req.Scheme)
	if kind == "review" && p.cfg.ReviewNoIndex {
		h.Set("X-Robots-Tag", "noindex, nofollow")
	}
	if kind == "review" || kind == "real" {
		// 同一 URL 内容随访客变化，禁止中间缓存
		if cc := h.Get("Cache-Control"); cc == "" || !strings.Contains(strings.ToLower(cc), "private") {
			h.Set("Cache-Control", "private, no-store")
		}
		h.Add("Vary", "Cookie")
	}

	// 内置浏览器放行且为 HTML：改写响应体（引导条 / _ck 链接）
	if inject && isHTML(resp.Header.Get("Content-Type")) && resp.Header.Get("Content-Encoding") == "" {
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if err != nil {
			return forwardResult{Result: Result{Kind: kind, OriginStatus: resp.StatusCode, Error: "read_body"}, headersSent: false}
		}
		d := req.Decision
		current := *r.URL
		current.Scheme, current.Host = req.Scheme, r.Host
		raw = injectHTML(raw, req.Site, &current, d.HandoffToken, d.InApp.OS, req.Site.Handoff.LinkRewrite, !bannerDismissed(r))
		// 回源用了 identity，这里由网关自己压缩
		if acceptsGzip(r) && len(raw) > 1024 {
			if gz, err := gzipBytes(raw); err == nil {
				raw = gz
				h.Set("Content-Encoding", "gzip")
				h.Add("Vary", "Accept-Encoding")
			}
		}
		h.Del("Content-Length")
		h.Set("Content-Length", strconv.Itoa(len(raw)))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(raw)
		return forwardResult{Result: Result{Kind: kind, OriginStatus: resp.StatusCode}, headersSent: true}
	}

	w.WriteHeader(resp.StatusCode)
	// 流式转发，SSE 也能即时推送
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				break
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			break
		}
	}
	return forwardResult{Result: Result{Kind: kind, OriginStatus: resp.StatusCode}, headersSent: true}
}

func isRetryable(r *http.Request, err error) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
	default:
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return false // 超时不重试，避免放大
	}
	return true
}

func classify(err error) string {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "origin_timeout"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "origin_timeout"
	}
	return "origin_unreachable"
}
