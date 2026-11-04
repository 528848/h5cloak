package openapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cloak/internal/adapter/geoip"
	"cloak/internal/app/attrsvc"
	"cloak/internal/app/authsvc"
	"cloak/internal/app/logsvc"
	"cloak/internal/domain/decisionlog"
	"cloak/internal/engine"
	"cloak/internal/engine/globalcfg"
	"cloak/internal/engine/snapshot"
	"cloak/internal/logpipe"
	"cloak/internal/proxy"
	"cloak/internal/quota"
)

// Deps 开放接口依赖。Attr / Logs 为 nil 时对应管理类接口返回 503。
type Deps struct {
	Log    *zap.Logger
	Auth   *Authenticator
	Limit  *Limiter
	Engine *engine.Engine
	Pipe   *logpipe.Pipe
	Quota  *quota.Checker
	Geo    geoip.Resolver
	Global *globalcfg.Holder
	Rdb    *goredis.Client
	Attr   *attrsvc.Service
	Logs   *logsvc.Service
	// ClientIP 由网关注入（已处理可信代理）
	ClientIP func(r *http.Request) netip.Addr
}

// Handler 开放接口路由。
type Handler struct{ d Deps }

func New(d Deps) *Handler { return &Handler{d: d} }

// Prefix 开放接口路径前缀。
const Prefix = "/api/open/"

// Matches 判断请求是否属于开放接口。
func Matches(path string) bool { return strings.HasPrefix(path, Prefix) }

type envelope struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

func (h *Handler) write(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *Handler) fail(w http.ResponseWriter, rid string, status int, code, msg string) {
	h.write(w, status, envelope{Code: status, Msg: msg, Data: map[string]string{"error": code}, RequestID: rid})
}

// ServeHTTP 路由分发。requestID 由网关写入 ctx 与响应头。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rid := w.Header().Get("X-Request-Id")
	path := strings.TrimPrefix(r.URL.Path, Prefix)
	ip := netip.Addr{}
	if h.d.ClientIP != nil {
		ip = h.d.ClientIP(r)
	}

	// 无需鉴权的探活
	if path == "v2/ping" {
		h.write(w, 200, envelope{Code: 200, Msg: "ok", Data: map[string]any{"time": time.Now().Unix()}, RequestID: rid})
		return
	}
	// JS SDK：公钥 + Origin 校验
	if path == "v2/js/capture" {
		h.jsCapture(w, r, rid, ip, start)
		return
	}

	p, aerr := h.d.Auth.Authenticate(r.Context(), r, ip)
	if aerr != nil {
		h.apiLog(nil, rid, path, aerr.Status, aerr.Code, start, ip)
		h.fail(w, rid, aerr.Status, aerr.Code, aerr.Msg)
		return
	}
	if ok, _ := h.d.Limit.Allow(r.Context(), p.Site); !ok {
		h.apiLog(p.Site, rid, path, 429, "rate_limited", start, ip)
		w.Header().Set("Retry-After", "1")
		h.fail(w, rid, http.StatusTooManyRequests, "rate_limited", "请求过于频繁")
		return
	}

	var status int
	var code string
	switch {
	case path == "decision" && r.Method == http.MethodPost: // V1 兼容
		status, code = h.decision(w, r, p, rid, start, true)
	case path == "v2/decision" && r.Method == http.MethodPost:
		status, code = h.decision(w, r, p, rid, start, false)
	case path == "v2/attribution/capture" && r.Method == http.MethodPost:
		status, code = h.capture(w, r, p, rid, start)
	case path == "v2/attribution/resolve" && r.Method == http.MethodPost:
		status, code = h.resolve(w, r, p, rid)
	case path == "v2/conversions" && r.Method == http.MethodPost:
		status, code = h.conversion(w, r, p, rid)
	case strings.HasPrefix(path, "v2/promo-codes"):
		status, code = h.promoCodes(w, r, p, rid, strings.TrimPrefix(path, "v2/promo-codes"))
	case path == "v2/decision-logs" && r.Method == http.MethodGet:
		status, code = h.decisionLogs(w, r, p, rid)
	default:
		status, code = http.StatusNotFound, "not_found"
		h.fail(w, rid, status, code, "接口不存在")
	}
	h.apiLog(p.Site, rid, path, status, code, start, ip)
}

func (h *Handler) apiLog(site *snapshot.Site, rid, endpoint string, status int, code string, start time.Time, ip netip.Addr) {
	if h.d.Pipe == nil {
		return
	}
	ev := decisionlog.APILogEvent{
		CreatedAt: start, RequestID: rid, Endpoint: endpoint, Status: status, ErrorCode: code,
		LatencyMs: int(time.Since(start).Milliseconds()),
	}
	if ip.IsValid() {
		ev.IP = engine.IPPrefix(ip)
	}
	if site != nil {
		ev.MerchantID, ev.SiteID, ev.KeyPrefix = site.MerchantID, site.SiteID, site.SecretPrefix
	}
	h.d.Pipe.Publish(decisionlog.StreamAPILogs, ev)
}

// ---- 判定 ----

type decisionReq struct {
	URL            string            `json:"url"`
	Method         string            `json:"method"`
	VisitorToken   string            `json:"visitorToken"`
	IP             string            `json:"ip"`
	UserAgent      string            `json:"userAgent"`
	Referer        string            `json:"referer"`
	AcceptLanguage string            `json:"acceptLanguage"`
	Headers        map[string]string `json:"headers"`
	// V1 插件字段
	Path  string `json:"path"`
	Query string `json:"query"`
	Host  string `json:"host"`
}

func (h *Handler) parseDecision(p *Principal) (*decisionReq, *url.URL, netip.Addr, string) {
	var in decisionReq
	if len(p.Body) > 0 {
		if err := json.Unmarshal(p.Body, &in); err != nil {
			return nil, nil, netip.Addr{}, "请求体不是合法 JSON"
		}
	}
	raw := strings.TrimSpace(in.URL)
	if raw == "" && in.Host != "" {
		raw = "https://" + in.Host + in.Path
		if in.Query != "" {
			raw += "?" + strings.TrimPrefix(in.Query, "?")
		}
	}
	if raw == "" {
		return nil, nil, netip.Addr{}, "缺少 url"
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, nil, netip.Addr{}, "url 无效"
	}
	ip, _ := netip.ParseAddr(strings.TrimSpace(in.IP))
	return &in, u, ip, ""
}

func (h *Handler) decide(ctx context.Context, p *Principal, in *decisionReq, u *url.URL, ip netip.Addr, rid, entry string) (*engine.RequestContext, engine.Decision, string) {
	if !ip.IsValid() {
		ip = p.ClientIP
	}
	rc := &engine.RequestContext{
		RequestID: rid, Entry: entry, Site: p.Site, Method: strings.ToUpper(firstNonEmpty(in.Method, "GET")), URL: u,
		CookieToken: strings.TrimSpace(in.VisitorToken), IP: ip, UserAgent: in.UserAgent, Referer: in.Referer, Now: time.Now(),
	}
	d := h.d.Engine.Decide(ctx, rc)
	quotaHit := ""
	if d.Content != engine.ContentBypass && h.d.Quota != nil {
		if h.d.Quota.Over(ctx, p.Site.MerchantID, p.Site.Quota.Decisions) {
			switch p.Site.Quota.Policy {
			case quota.PolicyReviewAll:
				if d.Content == engine.ContentReal {
					d.Content, d.VisitorToken, quotaHit = engine.ContentReview, "", "review_all"
				}
			case quota.PolicyDeny:
				d.Content, d.VisitorToken, quotaHit = engine.ContentBlock, "", "deny"
			default:
				quotaHit = "alert"
			}
			if quotaHit != "" {
				d.Trace = append(d.Trace, engine.TraceStep{Step: "quota", Result: "over:" + quotaHit})
			}
		}
		h.d.Quota.Count(p.Site.MerchantID)
	}
	if h.d.Global != nil {
		d.Content, d.VisitorToken, d.Trace = globalcfg.Overlay(h.d.Global.Get(), d.Content, d.VisitorToken, entry, d.Trace)
	}
	return rc, d, quotaHit
}

func (h *Handler) decision(w http.ResponseWriter, r *http.Request, p *Principal, rid string, start time.Time, v1 bool) (int, string) {
	in, u, ip, msg := h.parseDecision(p)
	if msg != "" {
		h.fail(w, rid, http.StatusBadRequest, "bad_request", msg)
		return 400, "bad_request"
	}
	entry := "api_v2"
	if v1 {
		entry = "api_v1"
	}
	rc, d, _ := h.decide(r.Context(), p, in, u, ip, rid, entry)
	hostKnown := p.Site.HasDomain(u.Host)
	data := h.decisionData(p.Site, rc, &d, start)
	data["hostKnown"] = hostKnown
	if v1 {
		// V1 插件读取 page / action
		page := "real"
		action := "black"
		if d.Content != engine.ContentReal && d.Content != engine.ContentBypass {
			page, action = "review", "white"
		}
		data["page"] = page
		data["action"] = action
		data["hitRule"] = strings.Join(d.MatchedParams, ",")
	}
	h.emit(rc, &d, start)
	h.write(w, 200, envelope{Code: 200, Msg: "ok", Data: data, RequestID: rid})
	return 200, ""
}

func (h *Handler) decisionData(site *snapshot.Site, rc *engine.RequestContext, d *engine.Decision, start time.Time) map[string]any {
	data := map[string]any{
		"content": d.Content, "touchType": d.TouchType, "attributed": d.Attributed, "confidence": d.Confidence,
		"channel": d.Channel, "code": d.Code, "matchedParams": d.MatchedParams, "validationErrors": d.ValidationErrors,
		"inAppBrowser": d.InApp, "handoffKind": d.HandoffKind, "pathRule": d.PathRule, "degraded": d.Degraded,
		"latencyMs": time.Since(start).Milliseconds(),
	}
	if d.VisitorToken != "" {
		secure := rc.URL.Scheme == "https"
		data["visitorToken"] = d.VisitorToken
		data["cookie"] = map[string]any{
			"name": site.Cookie.Name, "value": d.VisitorToken, "maxAge": d.CookieMaxAge(rc.Now),
			"domain": site.Cookie.Domain, "path": "/", "secure": secure, "httpOnly": true, "sameSite": "Lax",
		}
		data["attributionExpiresAt"] = d.TokenExpiresAt.UTC().Format(time.RFC3339)
	}
	if d.HandoffToken != "" {
		hu := *rc.URL
		q := hu.Query()
		q.Set(engine.HandoffParam, d.HandoffToken)
		hu.RawQuery = q.Encode()
		data["handoffUrl"] = hu.String()
		data["handoffBanner"] = map[string]any{"enabled": site.Handoff.Banner, "text": site.Handoff.BannerText}
	}
	if d.Content == engine.ContentReview {
		data["renderHint"] = h.renderHint(site)
	}
	return data
}

// renderHint 审核内容为模板时直接给 HTML，为静态包/源站时给提示。
func (h *Handler) renderHint(site *snapshot.Site) map[string]any {
	switch site.Review.Source {
	case "template":
		if html, err := proxy.RenderTemplate(site.Review.TemplateHTML, site.Review.TemplateVars); err == nil {
			return map[string]any{"type": "html", "html": string(html)}
		}
	case "origin":
		if o := site.Origin("review"); o != nil {
			return map[string]any{"type": "url", "url": o.URL}
		}
	}
	return map[string]any{"type": "none"}
}

func (h *Handler) capture(w http.ResponseWriter, r *http.Request, p *Principal, rid string, start time.Time) (int, string) {
	in, u, ip, msg := h.parseDecision(p)
	if msg != "" {
		h.fail(w, rid, http.StatusBadRequest, "bad_request", msg)
		return 400, "bad_request"
	}
	rc, d, _ := h.decide(r.Context(), p, in, u, ip, rid, "api_v2")
	data := map[string]any{
		"attributed": d.Attributed, "touchType": d.TouchType, "confidence": d.Confidence, "channel": d.Channel,
		"code": d.Code, "matchedParams": d.MatchedParams, "validationErrors": d.ValidationErrors,
		"visitorId": d.VisitorID, "attributionId": d.AttributionID,
	}
	if d.VisitorToken != "" {
		data["visitorToken"] = d.VisitorToken
		data["cookie"] = map[string]any{
			"name": p.Site.Cookie.Name, "value": d.VisitorToken, "maxAge": d.CookieMaxAge(rc.Now),
			"domain": p.Site.Cookie.Domain, "path": "/", "secure": rc.URL.Scheme == "https", "httpOnly": true, "sameSite": "Lax",
		}
	}
	h.emit(rc, &d, start)
	h.write(w, 200, envelope{Code: 200, Msg: "ok", Data: data, RequestID: rid})
	return 200, ""
}

// emit 判定日志与归因写入，与托管入口同一条管道。
func (h *Handler) emit(rc *engine.RequestContext, d *engine.Decision, start time.Time) {
	if h.d.Pipe == nil {
		return
	}
	site := rc.Site
	country := ""
	if h.d.Geo != nil {
		country = h.d.Geo.Country(rc.IP)
	}
	names := []string{}
	for k := range rc.URL.Query() {
		names = append(names, k)
	}
	ev := decisionlog.Event{
		CreatedAt: rc.Now, RequestID: rc.RequestID, MerchantID: site.MerchantID, SiteID: site.SiteID,
		Entry: rc.Entry, Method: rc.Method, Host: rc.URL.Host, Path: rc.URL.Path, QueryParamNames: names,
		TouchType: d.TouchType, Content: d.Content, Attributed: d.Attributed, Confidence: d.Confidence,
		InAppBrowser: d.InApp.App, DeviceFamily: d.InApp.DeviceFamily, HandoffKind: d.HandoffKind, HandoffFromRequestID: d.HandoffFromReqID,
		Channel: d.Channel, MatchedParams: d.MatchedParams, ValidationErrors: d.ValidationErrors,
		VisitorID: d.VisitorID, AttributionID: d.AttributionID,
		IPHash: engine.HashIP(rc.IP), IPPrefix: engine.IPPrefix(rc.IP), Country: country, OS: d.InApp.OS,
		UASummary: trunc(rc.UserAgent, 255), RefererSummary: trunc(rc.Referer, 255),
		OriginKind: "api", LatencyMs: int(time.Since(start).Milliseconds()), Degraded: d.Degraded, Trace: d.Trace,
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

// ---- 归因查询 ----

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request, p *Principal, rid string) (int, string) {
	var in struct {
		VisitorToken string `json:"visitorToken"`
		VisitorID    string `json:"visitorId"`
	}
	_ = json.Unmarshal(p.Body, &in)
	vid := strings.TrimSpace(in.VisitorID)
	if vid == "" && in.VisitorToken != "" {
		rc := &engine.RequestContext{Site: p.Site, CookieToken: in.VisitorToken, Now: time.Now(), URL: &url.URL{Path: "/"}}
		if pl, ok := engine.VerifyToken(rc); ok {
			vid = pl.VisitorID
		} else {
			h.fail(w, rid, http.StatusBadRequest, "invalid_token", "访客令牌无效或已过期")
			return 400, "invalid_token"
		}
	}
	if vid == "" {
		h.fail(w, rid, http.StatusBadRequest, "bad_request", "需要 visitorToken 或 visitorId")
		return 400, "bad_request"
	}
	data := map[string]any{"visitorId": vid, "found": false}
	if h.d.Rdb != nil {
		m, err := h.d.Rdb.HGetAll(r.Context(), "attr:"+strconv.FormatUint(p.Site.SiteID, 10)+":"+vid).Result()
		if err == nil && len(m) > 0 {
			data["found"] = true
			data["attributionId"] = m["aid"]
			data["channel"] = m["ch"]
			data["lastTouchType"] = m["touch"]
			data["lastTouchAt"] = m["at"]
			if cid, err := strconv.ParseUint(m["cid"], 10, 64); err == nil && cid > 0 {
				data["codeId"] = cid
				for i := range p.Site.Codes {
					if p.Site.Codes[i].ID == cid {
						c := p.Site.Codes[i]
						data["code"] = map[string]any{"id": c.ID, "code": c.Code, "type": c.Type, "name": c.Name, "channel": c.Channel}
					}
				}
			}
		}
	}
	h.write(w, 200, envelope{Code: 200, Msg: "ok", Data: data, RequestID: rid})
	return 200, ""
}

// ---- 转化 ----

func (h *Handler) conversion(w http.ResponseWriter, r *http.Request, p *Principal, rid string) (int, string) {
	var in struct {
		OrderNo      string  `json:"orderNo"`
		Status       string  `json:"status"`
		Amount       float64 `json:"amount"`
		Currency     string  `json:"currency"`
		VisitorToken string  `json:"visitorToken"`
		VisitorID    string  `json:"visitorId"`
		Email        string  `json:"email"`
		EmailHash    string  `json:"emailHash"`
		OccurredAt   string  `json:"occurredAt"`
	}
	if err := json.Unmarshal(p.Body, &in); err != nil || strings.TrimSpace(in.OrderNo) == "" {
		h.fail(w, rid, http.StatusBadRequest, "bad_request", "缺少 orderNo")
		return 400, "bad_request"
	}
	vid := strings.TrimSpace(in.VisitorID)
	if vid == "" && in.VisitorToken != "" {
		rc := &engine.RequestContext{Site: p.Site, CookieToken: in.VisitorToken, Now: time.Now(), URL: &url.URL{Path: "/"}}
		if pl, ok := engine.VerifyToken(rc); ok {
			vid = pl.VisitorID
		}
	}
	emailHash := strings.ToLower(strings.TrimSpace(in.EmailHash))
	if emailHash == "" && in.Email != "" {
		emailHash = engine.HashEmail(in.Email)
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if status == "" {
		status = "paid"
	}
	occurred := time.Now()
	if in.OccurredAt != "" {
		if t, err := time.Parse(time.RFC3339, in.OccurredAt); err == nil {
			occurred = t
		}
	}
	if h.d.Pipe == nil {
		h.fail(w, rid, http.StatusServiceUnavailable, "pipeline_unavailable", "日志管道不可用")
		return 503, "pipeline_unavailable"
	}
	h.d.Pipe.Publish(decisionlog.StreamConversions, decisionlog.ConversionEvent{
		At: occurred, RequestID: rid, MerchantID: p.Site.MerchantID, SiteID: p.Site.SiteID,
		OrderNo: strings.TrimSpace(in.OrderNo), Status: status, Amount: in.Amount, Currency: strings.ToUpper(firstNonEmpty(in.Currency, "USD")),
		VisitorID: vid, EmailHash: emailHash,
	})
	h.write(w, 202, envelope{Code: 200, Msg: "accepted", Data: map[string]any{"orderNo": in.OrderNo, "visitorId": vid, "queued": true}, RequestID: rid})
	return 202, ""
}

// ---- 分销码 ----

func (h *Handler) promoCodes(w http.ResponseWriter, r *http.Request, p *Principal, rid, rest string) (int, string) {
	if h.d.Attr == nil {
		h.fail(w, rid, http.StatusServiceUnavailable, "unavailable", "该接口在当前节点不可用")
		return 503, "unavailable"
	}
	ctx := r.Context()
	mid, sid := p.Site.MerchantID, p.Site.SiteID
	rest = strings.Trim(rest, "/")
	switch {
	case rest == "" && r.Method == http.MethodGet:
		q := r.URL.Query()
		rows, total, err := h.d.Attr.ListCodes(ctx, mid, attrsvc.CodeListInput{
			SiteID: sid, Keyword: q.Get("keyword"), CodeType: q.Get("type"), Status: q.Get("status"), Channel: q.Get("channel"),
			Page: atoi(q.Get("page"), 1), Size: atoi(q.Get("pageSize"), 50),
		})
		if err != nil {
			return h.svcErr(w, rid, err)
		}
		h.write(w, 200, envelope{Code: 200, Msg: "ok", Data: map[string]any{"rows": rows, "total": total}, RequestID: rid})
		return 200, ""
	case rest == "" && r.Method == http.MethodPost:
		var in attrsvc.CodeInput
		if err := json.Unmarshal(p.Body, &in); err != nil {
			h.fail(w, rid, 400, "bad_request", "请求体不是合法 JSON")
			return 400, "bad_request"
		}
		in.SiteID = sid
		row, err := h.d.Attr.CreateCode(ctx, mid, 0, in)
		if err != nil {
			return h.svcErr(w, rid, err)
		}
		h.write(w, 201, envelope{Code: 200, Msg: "created", Data: row, RequestID: rid})
		return 201, ""
	case rest == "batch" && r.Method == http.MethodPost:
		var in attrsvc.GenerateInput
		if err := json.Unmarshal(p.Body, &in); err != nil {
			h.fail(w, rid, 400, "bad_request", "请求体不是合法 JSON")
			return 400, "bad_request"
		}
		in.SiteID = sid
		row, err := h.d.Attr.GenerateCodes(ctx, mid, 0, in)
		if err != nil {
			return h.svcErr(w, rid, err)
		}
		h.write(w, 201, envelope{Code: 200, Msg: "created", Data: row, RequestID: rid})
		return 201, ""
	}
	id, err := strconv.ParseUint(rest, 10, 64)
	if err != nil || id == 0 {
		h.fail(w, rid, 404, "not_found", "接口不存在")
		return 404, "not_found"
	}
	switch r.Method {
	case http.MethodPut:
		var in attrsvc.CodeInput
		if err := json.Unmarshal(p.Body, &in); err != nil {
			h.fail(w, rid, 400, "bad_request", "请求体不是合法 JSON")
			return 400, "bad_request"
		}
		if err := h.d.Attr.UpdateCode(ctx, mid, id, in); err != nil {
			return h.svcErr(w, rid, err)
		}
		h.write(w, 200, envelope{Code: 200, Msg: "updated", RequestID: rid})
		return 200, ""
	case http.MethodDelete:
		n, err := h.d.Attr.DeleteCodes(ctx, mid, []uint64{id})
		if err != nil {
			return h.svcErr(w, rid, err)
		}
		if n == 0 {
			h.fail(w, rid, 404, "not_found", "分销码不存在")
			return 404, "not_found"
		}
		h.write(w, 200, envelope{Code: 200, Msg: "deleted", RequestID: rid})
		return 200, ""
	}
	h.fail(w, rid, 405, "method_not_allowed", "方法不支持")
	return 405, "method_not_allowed"
}

// ---- 决策日志 ----

func (h *Handler) decisionLogs(w http.ResponseWriter, r *http.Request, p *Principal, rid string) (int, string) {
	if h.d.Logs == nil {
		h.fail(w, rid, http.StatusServiceUnavailable, "unavailable", "该接口在当前节点不可用")
		return 503, "unavailable"
	}
	q := r.URL.Query()
	rows, total, err := h.d.Logs.List(r.Context(), p.Site.MerchantID, logsvc.ListInput{
		SiteID: p.Site.SiteID, From: q.Get("from"), To: q.Get("to"), Path: q.Get("path"), TouchType: q.Get("touchType"),
		Content: q.Get("content"), Channel: q.Get("channel"), Code: q.Get("code"), Reason: q.Get("reason"),
		VisitorID: q.Get("visitorId"), RequestID: q.Get("requestId"), Page: atoi(q.Get("page"), 1), Size: atoi(q.Get("pageSize"), 50),
	})
	if err != nil {
		return h.svcErr(w, rid, err)
	}
	h.write(w, 200, envelope{Code: 200, Msg: "ok", Data: map[string]any{"rows": rows, "total": total}, RequestID: rid})
	return 200, ""
}

func (h *Handler) svcErr(w http.ResponseWriter, rid string, err error) (int, string) {
	if e, ok := authsvc.AsError(err); ok {
		status := e.Code
		if status < 400 || status > 599 {
			status = 400
		}
		h.fail(w, rid, status, "request_failed", e.Msg)
		return status, "request_failed"
	}
	h.d.Log.Warn("openapi", zap.Error(err))
	h.fail(w, rid, 500, "internal_error", "系统繁忙，请稍后重试")
	return 500, "internal_error"
}

func atoi(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
