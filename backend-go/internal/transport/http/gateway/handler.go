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

	"cloak/internal/config"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
	"cloak/internal/pkg/requestid"
)

// Deps 网关依赖。Store / Engine 为空时只提供健康检查（I0 兼容与单测）。
type Deps struct {
	Cfg     *config.Config
	Log     *zap.Logger
	Metrics *prometheus.Registry
	Store   *snapshot.Store
	Engine  *engine.Engine
}

// Handler 网关主处理器。使用标准库而非 gin，为后续反向代理留出完全控制权。
type Handler struct {
	d         Deps
	requests  *prometheus.CounterVec
	decisions *prometheus.CounterVec
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
	d.Metrics.MustRegister(h.requests, h.decisions, h.latency)
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

	rc := h.requestContext(r, site, id)
	start := time.Now()
	d := h.d.Engine.Decide(r.Context(), rc)
	h.latency.Observe(time.Since(start).Seconds())
	h.decisions.WithLabelValues(d.Content, d.TouchType).Inc()

	h.writeCookie(w, r, site, &d)
	// I6 之前：只返回判定结果，不回源。用 JSON 方便验证热更新与判定链路。
	h.writeDecision(w, id, site, &d, time.Since(start))
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
