package openapi

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
)

//go:embed sdk/cloak.js
var sdkJS []byte

// SDKPath 网关上 SDK 脚本的路径。
const SDKPath = "/_cloak/sdk.js"

// ServeSDK 返回 JS SDK（任意 Host 可取）。
func ServeSDK(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(sdkJS)
}

type jsCaptureReq struct {
	Key      string `json:"key"`
	URL      string `json:"url"`
	Referrer string `json:"referrer"`
	Visitor  string `json:"visitorToken"`
}

// jsCapture JS SDK 上报：公钥 + Origin 校验，不需要签名；返回判定与令牌，cookie 由页面脚本自己写。
func (h *Handler) jsCapture(w http.ResponseWriter, r *http.Request, rid string, ip netip.Addr, start time.Time) {
	origin := r.Header.Get("Origin")
	w.Header().Set("Vary", "Origin")
	if r.Method == http.MethodOptions {
		h.cors(w, origin)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		h.fail(w, rid, http.StatusMethodNotAllowed, "method_not_allowed", "方法不支持")
		return
	}
	var in jsCaptureReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil || in.Key == "" || in.URL == "" {
		h.cors(w, origin)
		h.fail(w, rid, http.StatusBadRequest, "bad_request", "缺少 key 或 url")
		return
	}
	site, ok := h.d.Auth.store.ByPublishableKey(r.Context(), in.Key)
	if !ok || !site.Serving() {
		h.cors(w, origin)
		h.apiLog(nil, rid, "v2/js/capture", 401, "unknown_key", start, ip)
		h.fail(w, rid, http.StatusUnauthorized, "unknown_key", "公钥无效")
		return
	}
	// Origin 必须是站点已绑定域名（本地开发允许 localhost）
	if !originAllowed(origin, site) {
		h.cors(w, origin)
		h.apiLog(site, rid, "v2/js/capture", 403, "origin_not_allowed", start, ip)
		h.fail(w, rid, http.StatusForbidden, "origin_not_allowed", "来源域名未绑定到该站点")
		return
	}
	h.cors(w, origin)
	if ok, _ := h.d.Limit.Allow(r.Context(), site); !ok {
		h.apiLog(site, rid, "v2/js/capture", 429, "rate_limited", start, ip)
		h.fail(w, rid, http.StatusTooManyRequests, "rate_limited", "请求过于频繁")
		return
	}
	u, err := url.Parse(in.URL)
	if err != nil || u.Host == "" {
		h.fail(w, rid, http.StatusBadRequest, "bad_request", "url 无效")
		return
	}
	p := &Principal{Site: site, Version: "js", ClientIP: ip}
	req := &decisionReq{URL: in.URL, VisitorToken: in.Visitor, UserAgent: r.UserAgent(), Referer: in.Referrer}
	rc, d, _ := h.decide(r.Context(), p, req, u, ip, rid, "js")
	data := map[string]any{
		"content": d.Content, "touchType": d.TouchType, "attributed": d.Attributed, "confidence": d.Confidence,
		"channel": d.Channel, "code": d.Code, "inAppBrowser": d.InApp, "latencyMs": time.Since(start).Milliseconds(),
	}
	if d.VisitorToken != "" {
		data["visitorToken"] = d.VisitorToken
		data["cookie"] = map[string]any{"name": site.Cookie.Name, "maxAge": d.CookieMaxAge(rc.Now), "domain": site.Cookie.Domain}
	}
	if d.HandoffToken != "" {
		hu := *rc.URL
		q := hu.Query()
		q.Set(engine.HandoffParam, d.HandoffToken)
		hu.RawQuery = q.Encode()
		data["handoffUrl"] = hu.String()
		data["handoffBanner"] = map[string]any{"enabled": site.Handoff.Banner, "text": site.Handoff.BannerText}
	}
	h.emit(rc, &d, start)
	h.apiLog(site, rid, "v2/js/capture", 200, "", start, ip)
	h.write(w, 200, envelope{Code: 200, Msg: "ok", Data: data, RequestID: rid})
}

func (h *Handler) cors(w http.ResponseWriter, origin string) {
	if origin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "600")
}

func originAllowed(origin string, site *snapshot.Site) bool {
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" {
		return true
	}
	return site.HasDomain(host)
}
