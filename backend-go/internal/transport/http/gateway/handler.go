// Package gateway 是判定网关的 HTTP 入口。I5 接入快照仓库与判定引擎，
// 按 Host 找站点并给出判定；回源与模板渲染在 I6 接入。
package gateway

import (
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/adapter/geoip"
	"cloak/internal/config"
	"cloak/internal/domain/decisionlog"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
	"cloak/internal/logpipe"
	"cloak/internal/pkg/requestid"
	"cloak/internal/proxy"
	"cloak/internal/quota"
)

// Deps 网关依赖。Store / Engine 为空时只提供健康检查（I0 兼容与单测）。
// Proxy 为空时对站点请求返回判定 JSON（调试用）。
type Deps struct {
	Cfg     *config.Config
	Log     *zap.Logger
	Metrics *prometheus.Registry
	Store   *snapshot.Store
	Engine  *engine.Engine
	Proxy   *proxy.Proxy
	// RedirectHTTPS 为 true 时，明文请求（非 ACME 挑战）跳转到 https
	RedirectHTTPS bool
	// 以下可为 nil
	Pipe  *logpipe.Pipe
	Quota *quota.Checker
	Geo   geoip.Resolver
}

// DebugDecideHeader 请求头带此值时返回判定 JSON 而非回源，供排障。
const DebugDecideHeader = "X-Cloak-Debug"

// Handler 网关主处理器。使用标准库而非 gin，为后续反向代理留出完全控制权。
type Handler struct {
	d         Deps
	requests  *prometheus.CounterVec
	decisions *prometheus.CounterVec
	proxied   *prometheus.CounterVec
	latency   prometheus.Histogram
	trusted   []netip.Prefix
	started   time.Time
}

// New 创建处理器并注册指标。
func New(d Deps) *Handler {
	h := &Handler{d: d, started: time.Now()}
	h.requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gateway", Subsystem: "http", Name: "requests_total", Help: "Gateway requests by kind.",
	}, []string{"kind"})
	h.decisions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gateway", Subsystem: "engine", Name: "decisions_total", Help: "Decisions by content and touch type.",
	}, []string{"content", "touch"})
	h.latency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "gateway", Subsystem: "engine", Name: "decide_seconds", Help: "Decide() latency.",
		Buckets: []float64{.0005, .001, .002, .005, .01, .02, .05, .1},
	})
	h.proxied = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gateway", Subsystem: "proxy", Name: "responses_total", Help: "Proxy responses by kind and fallback.",
	}, []string{"kind", "fallback"})
	d.Metrics.MustRegister(h.requests, h.decisions, h.proxied, h.latency)
	if d.Cfg != nil {
		for _, c := range d.Cfg.Gateway.TrustedProxies {
			if p, err := netip.ParsePrefix(c); err == nil {
				h.trusted = append(h.trusted, p)
			} else if a, err := netip.ParseAddr(c); err == nil {
				h.trusted = append(h.trusted, netip.PrefixFrom(a, a.BitLen()))
			}
		}
	}
	return h
}

// ServeHTTP 分发请求。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get(requestid.HeaderName)
	if id == "" || len(id) > 64 {
		id = requestid.New()
	}
	w.Header().Set(requestid.HeaderName, id)
	r = r.WithContext(requestid.WithContext(r.Context(), id))

	switch r.URL.Path {
	case "/_cloak/health":
		h.requests.WithLabelValues("health").Inc()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	case "/_cloak/status":
		h.requests.WithLabelValues("status").Inc()
		h.status(w)
		return
	}

	if h.d.Store == nil || h.d.Engine == nil {
		h.notFound(w)
		return
	}

	site, ok := h.d.Store.ByHost(r.Context(), r.Host)
	if !ok {
		h.requests.WithLabelValues("unknown_host").Inc()
		h.notFound(w)
		return
	}
	h.requests.WithLabelValues("site").Inc()

	// 明文 → HTTPS 跳转（ACME 挑战路径在外层已被拦截）
	if h.d.RedirectHTTPS && r.TLS == nil && !strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}

	rc := h.requestContext(r, site, id)
	start := time.Now()
	d := h.d.Engine.Decide(r.Context(), rc)
	h.latency.Observe(time.Since(start).Seconds())

	// 额度：bypass 不计量；超额按商户策略覆盖判定
	quotaHit := ""
	if d.Content != engine.ContentBypass && h.d.Quota != nil {
		if h.d.Quota.Over(r.Context(), site.MerchantID, site.Quota.Decisions) {
			switch site.Quota.Policy {
			case quota.PolicyReviewAll:
				if d.Content == engine.ContentReal {
					d.Content = engine.ContentReview
					d.VisitorToken = ""
					quotaHit = "review_all"
				}
			case quota.PolicyDeny:
				d.Content = engine.ContentBlock
				d.VisitorToken = ""
				quotaHit = "deny"
			default:
				quotaHit = "alert"
			}
			if quotaHit != "" {
				d.Trace = append(d.Trace, engine.TraceStep{Step: "quota", Result: "over:" + quotaHit})
			}
		}
		h.d.Quota.Count(site.MerchantID)
	}
	h.decisions.WithLabelValues(d.Content, d.TouchType).Inc()

	// bypass 不写 cookie；其余按判定写
	if d.Content != engine.ContentBypass {
		h.writeCookie(w, r, site, &d)
	}

	var res proxy.Result
	if h.d.Proxy == nil || r.Header.Get(DebugDecideHeader) == "decide" {
		h.writeDecision(w, id, site, &d, time.Since(start))
		res = proxy.Result{Kind: "debug"}
	} else {
		res = h.d.Proxy.Serve(w, r, proxy.Request{
			Site: site, Decision: &d, ClientIP: ipString(rc.IP), Scheme: rc.URL.Scheme, RequestID: id,
		})
		h.proxied.WithLabelValues(res.Kind, boolLabel(res.Fallback)).Inc()
		if res.Error != "" {
			h.d.Log.Warn("proxy", zap.String("request_id", id), zap.Uint64("site", site.SiteID), zap.String("kind", res.Kind),
				zap.Int("origin_status", res.OriginStatus), zap.String("error", res.Error), zap.Bool("fallback", res.Fallback))
		}
	}
	h.emit(rc, &d, res, time.Since(start))
}

// emit 把判定写入日志管道（不阻塞）。
func (h *Handler) emit(rc *engine.RequestContext, d *engine.Decision, res proxy.Result, took time.Duration) {
	if h.d.Pipe == nil {
		return
	}
	site := rc.Site
	country := ""
	if h.d.Geo != nil {
		country = h.d.Geo.Country(rc.IP)
	}
	names := make([]string, 0, 4)
	for k := range rc.URL.Query() {
		names = append(names, k)
	}
	ev := decisionlog.Event{
		CreatedAt: rc.Now, RequestID: rc.RequestID, MerchantID: site.MerchantID, SiteID: site.SiteID,
		Entry: rc.Entry, Method: rc.Method, Host: rc.URL.Host, Path: rc.URL.Path, QueryParamNames: names,
		TouchType: d.TouchType, Content: d.Content, Attributed: d.Attributed, Confidence: d.Confidence,
		InAppBrowser: d.InApp.App, DeviceFamily: d.InApp.DeviceFamily, HandoffKind: d.HandoffKind,
		Channel: d.Channel, MatchedParams: d.MatchedParams, ValidationErrors: d.ValidationErrors,
		VisitorID: d.VisitorID, AttributionID: d.AttributionID,
		IPHash: engine.HashIP(rc.IP), IPPrefix: engine.IPPrefix(rc.IP), Country: country, OS: d.InApp.OS,
		UASummary: trunc(rc.UserAgent, 255), RefererSummary: trunc(rc.Referer, 255),
		OriginKind: res.Kind, OriginStatus: res.OriginStatus, Fallback: res.Fallback,
		LatencyMs: int(took.Milliseconds()), Degraded: d.Degraded, Trace: d.Trace,
	}
	if d.Code != nil {
		ev.CodeID, ev.Code = d.Code.ID, d.Code.Code
	}
	if d.PathRule != nil {
		ev.PathRule = d.PathRule.Action + " " + d.PathRule.Pattern
	}
	if !d.TokenExpiresAt.IsZero() {
		t := d.TokenExpiresAt
		ev.AttributionExpiresAt = &t
	}
	h.d.Pipe.Publish(decisionlog.StreamDecisions, ev)

	if d.Attributed && d.VisitorID != "" && d.Content == engine.ContentReal {
		aw := decisionlog.AttributionWrite{
			At: rc.Now, RequestID: rc.RequestID, MerchantID: site.MerchantID, SiteID: site.SiteID,
			VisitorID: d.VisitorID, AttributionID: d.AttributionID, TouchType: d.TouchType, HandoffKind: d.HandoffKind,
			Channel: d.Channel, MatchedParams: d.MatchedParams, LandingPath: rc.URL.Path, Confidence: d.Confidence,
			ExpiresAt: ev.AttributionExpiresAt, Country: country, DeviceFamily: d.InApp.DeviceFamily, InAppBrowser: d.InApp.App,
		}
		if d.Code != nil {
			aw.CodeID = d.Code.ID
		}
		h.d.Pipe.Publish(decisionlog.StreamAttributions, aw)
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func ipString(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	return ip.String()
}

func boolLabel(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func (h *Handler) requestContext(r *http.Request, site *snapshot.Site, id string) *engine.RequestContext {
	cookie := ""
	if c, err := r.Cookie(site.Cookie.Name); err == nil {
		cookie = c.Value
	}
	u := *r.URL
	u.Host = r.Host
	if u.Scheme == "" {
		u.Scheme = "https"
		if r.TLS == nil && !strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			u.Scheme = "http"
		}
	}
	return &engine.RequestContext{
		RequestID: id, Entry: "hosted", Site: site, Method: r.Method, URL: &u,
		CookieToken: cookie, IP: h.clientIP(r), UserAgent: r.UserAgent(), Referer: r.Referer(),
		Now: time.Now(),
	}
}

// clientIP 只在直连来源属于可信代理时信任 X-Forwarded-For。
func (h *Handler) clientIP(r *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remote, _ := netip.ParseAddr(host)
	if !remote.IsValid() {
		return remote
	}
	trusted := false
	for _, p := range h.trusted {
		if p.Contains(remote) {
			trusted = true
			break
		}
	}
	if !trusted {
		return remote
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			a, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
			if err != nil {
				continue
			}
			inTrusted := false
			for _, p := range h.trusted {
				if p.Contains(a) {
					inTrusted = true
					break
				}
			}
			if !inTrusted {
				return a
			}
		}
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		if a, err := netip.ParseAddr(strings.TrimSpace(rip)); err == nil {
			return a
		}
	}
	return remote
}

func (h *Handler) writeCookie(w http.ResponseWriter, r *http.Request, site *snapshot.Site, d *engine.Decision) {
	if d.VisitorToken == "" {
		return
	}
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	c := &http.Cookie{
		Name: site.Cookie.Name, Value: d.VisitorToken, Path: "/", HttpOnly: true, Secure: secure,
		SameSite: http.SameSiteLaxMode, MaxAge: d.CookieMaxAge(time.Now()),
	}
	if site.Cookie.Domain != "" {
		c.Domain = site.Cookie.Domain
	}
	http.SetCookie(w, c)
}

func (h *Handler) writeDecision(w http.ResponseWriter, id string, site *snapshot.Site, d *engine.Decision, took time.Duration) {
	status := http.StatusOK
	switch d.Content {
	case engine.ContentBlock:
		status = http.StatusForbidden
	case engine.ContentNotFound:
		status = http.StatusNotFound
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Cloak-Content", d.Content)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"requestId": id, "siteId": site.SiteID, "revision": site.Revision,
		"content": d.Content, "touchType": d.TouchType, "attributed": d.Attributed, "confidence": d.Confidence,
		"channel": d.Channel, "code": d.Code, "matchedParams": d.MatchedParams, "validationErrors": d.ValidationErrors,
		"handoffKind": d.HandoffKind, "handoffToken": d.HandoffToken, "inAppBrowser": d.InApp,
		"pathRule": d.PathRule, "trace": d.Trace, "degraded": d.Degraded, "latencyMs": took.Milliseconds(),
	})
}

func (h *Handler) status(w http.ResponseWriter) {
	out := map[string]any{"uptimeSec": int(time.Since(h.started).Seconds())}
	if h.d.Store != nil {
		out["snapshots"] = h.d.Store.Stats()
	}
	if h.d.Pipe != nil {
		out["logpipe"] = h.d.Pipe.Stats()
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) notFound(w http.ResponseWriter) {
	// 未知域名统一 404，不暴露平台特征。
	h.requests.WithLabelValues("unmatched").Inc()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Not Found"))
}
