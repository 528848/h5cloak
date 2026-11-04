// Package engine 判定引擎：输入 RequestContext 与站点快照，输出 Decision。
// 不依赖 HTTP 与数据库，Redis 相关能力通过 Store 接口注入，便于纯单测。
package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"cloak/internal/engine/params"
	"cloak/internal/engine/pathrule"
	"cloak/internal/engine/snapshot"
	"cloak/internal/engine/token"
	"cloak/internal/engine/ua"
)

// 内容类型
const (
	ContentReal        = "real"
	ContentReview      = "review"
	ContentBlock       = "block"
	ContentNotFound    = "not_found"
	ContentBypass      = "bypass"
	ContentMaintenance = "maintenance"
)

// 触点类型
const (
	TouchFirst   = "first"
	TouchReturn  = "return"
	TouchHandoff = "handoff"
	TouchNone    = "none"
)

// HandoffParam URL 中的交接 token 参数名。
const HandoffParam = "_ck"

// RequestContext 一次判定的输入。
type RequestContext struct {
	RequestID   string
	Entry       string // hosted / api_v2 / api_v1 / js / test
	Site        *snapshot.Site
	Method      string
	URL         *url.URL
	CookieToken string
	IP          netip.Addr
	UserAgent   string
	Referer     string
	Now         time.Time
	// DryRun 为 true 时不写 Redis（判定测试工具）
	DryRun bool
}

// CodeRef 命中的分销码/活动码。
type CodeRef struct {
	ID      uint64 `json:"id"`
	Code    string `json:"code"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Channel string `json:"channel"`
}

// TraceStep 判定过程记录。
type TraceStep struct {
	Step   string `json:"step"`
	Result string `json:"result"`
}

// Decision 判定输出。
type Decision struct {
	Content          string              `json:"content"`
	TouchType        string              `json:"touchType"`
	Attributed       bool                `json:"attributed"`
	Confidence       string              `json:"confidence"`
	Channel          string              `json:"channel,omitempty"`
	Code             *CodeRef            `json:"code,omitempty"`
	MatchedParams    []string            `json:"matchedParams"`
	ValidationErrors []params.ParamError `json:"validationErrors"`
	HandoffKind      string              `json:"handoffKind,omitempty"` // token / match
	InApp            ua.Info             `json:"inAppBrowser"`
	PathRule         *snapshot.PathRule  `json:"pathRule,omitempty"`
	Trace            []TraceStep         `json:"trace"`
	Degraded         bool                `json:"degraded"`

	// 令牌：VisitorToken 非空时应写 cookie；ClearCookie 为 true 时应删除 cookie
	VisitorToken   string    `json:"visitorToken,omitempty"`
	VisitorID      string    `json:"visitorId,omitempty"`
	AttributionID  string    `json:"attributionId,omitempty"`
	TokenExpiresAt time.Time `json:"attributionExpiresAt,omitempty"`
	ClearCookie    bool      `json:"-"`

	// HandoffToken 非空时网关应注入引导条并追加 _ck
	HandoffToken string `json:"handoffToken,omitempty"`
	// NewAttribution 为 true 表示本次是首访，应异步写归因
	NewAttribution bool `json:"-"`
}

// Store 引擎需要的外部状态（Redis）。全部允许失败：返回 error 时引擎降级继续。
type Store interface {
	// ClickSeen 记录点击参数值出现，返回该值在窗口内的不同 IP 前缀数
	ClickSeen(ctx context.Context, siteID uint64, value, ipPrefix string, window time.Duration) (int, error)
	// HandoffIssue 登记交接 token
	HandoffIssue(ctx context.Context, siteID uint64, tokenHash, attributionID, visitorID string, maxUses int, ttl time.Duration) error
	// HandoffConsume 消费交接 token；ok=false 表示不存在、超次或过期
	HandoffConsume(ctx context.Context, siteID uint64, tokenHash string) (attributionID, visitorID string, ok bool, err error)
	// HandoffCandidateAdd 登记跨浏览器匹配候选
	HandoffCandidateAdd(ctx context.Context, siteID uint64, ipHash, deviceFamily, attributionID, visitorID string, window time.Duration) error
	// HandoffCandidateTake 取出并消费窗口内最近一条候选
	HandoffCandidateTake(ctx context.Context, siteID uint64, ipHash, deviceFamily string, window time.Duration) (attributionID, visitorID string, ok bool, err error)
}

// Engine 判定引擎。
type Engine struct {
	store Store
}

// New 创建引擎；store 可为 nil（无 Redis，交接与复用检查全部跳过）。
func New(store Store) *Engine { return &Engine{store: store} }

// Decide 执行判定（技术方案 5.2）。
func (e *Engine) Decide(ctx context.Context, rc *RequestContext) Decision {
	site := rc.Site
	d := Decision{
		Content: ContentReview, TouchType: TouchNone, Confidence: "low",
		MatchedParams: []string{}, ValidationErrors: []params.ParamError{}, Trace: []TraceStep{},
	}
	if rc.Now.IsZero() {
		rc.Now = time.Now()
	}
	if rc.URL == nil {
		rc.URL = &url.URL{Path: "/"}
	}
	d.InApp = ua.Parse(rc.UserAgent)

	if !site.Serving() {
		d.Content = e.siteDisabledContent(site)
		d.trace("site", "not_serving:"+d.Content)
		return d
	}

	// 1. 路径规则
	if rule := pathrule.Match(site, rc.URL.Path); rule != nil {
		d.PathRule = rule
		d.trace("path_rule", rule.Action+" "+rule.Pattern)
		switch rule.Action {
		case "bypass":
			d.Content = ContentBypass
			return d
		case "always_real":
			d.Content = ContentReal
			return d
		case "always_review":
			d.Content = ContentReview
			return d
		case "block":
			d.Content = ContentBlock
			return d
		}
	}

	// 2. 令牌
	tok, tokErr := token.Verify(site, rc.CookieToken, rc.Now)
	switch {
	case rc.CookieToken == "":
		d.trace("token", "absent")
	case tokErr == nil:
		d.trace("token", "valid")
	default:
		d.trace("token", "invalid:"+tokErr.Error())
	}

	// 3. 参数提取与校验
	ms := params.Extract(site, rc.URL.Query())
	env := params.Env{Path: rc.URL.Path, Now: rc.Now, Referer: rc.Referer, InAppBrowser: d.InApp.InApp}
	if e.store != nil && site.Rule.ClickReuseLimit > 0 {
		env.ClickReused = func(value string) bool { return e.clickReused(ctx, rc, value) }
	}
	res := params.Validate(site, ms, env)
	d.MatchedParams = res.MatchedNames()
	d.ValidationErrors = res.Errors
	if len(ms) > 0 {
		d.trace("params", strings.Join(d.MatchedParams, ","))
	}

	// 4. 分支：首访 > 回访 > 交接 token > 交接匹配 > 无归因
	if res.Attributed && res.Confidence >= site.MinConfidenceLevel() {
		e.attributed(ctx, rc, &d, tok, res)
		return d
	}
	if res.Attributed {
		d.trace("confidence", "below_min:"+snapshot.ConfidenceName(res.Confidence))
	}
	hasToken := tokErr == nil && tok != nil
	if hasToken {
		e.returning(rc, &d, tok)
		return d
	}
	if hv := rc.URL.Query().Get(HandoffParam); hv != "" && e.handoffByToken(ctx, rc, &d, hv) {
		return d
	}
	if site.Handoff.MatchEnabled && len(ms) == 0 && e.handoffByMatch(ctx, rc, &d) {
		return d
	}
	e.unattributed(rc, &d, res)
	return d
}

func (e *Engine) siteDisabledContent(site *snapshot.Site) string {
	switch site.Attribution.NoAttributionAction {
	case "block":
		return ContentBlock
	case "not_found":
		return ContentNotFound
	default:
		return ContentReview
	}
}

func (e *Engine) attributed(ctx context.Context, rc *RequestContext, d *Decision, tok *token.Payload, res params.Result) {
	d.Content = ContentReal
	d.Attributed = true
	d.Confidence = snapshot.ConfidenceName(res.Confidence)
	d.Channel = res.Channel
	if res.Code != nil {
		d.Code = &CodeRef{ID: res.Code.ID, Code: res.Code.Code, Type: res.Code.Type, Name: res.Code.Name, Channel: res.Code.Channel}
	}
	// 有有效令牌且带参数：视为回访并刷新归因（最近渠道），不新建访客
	visitorID := ""
	if tok != nil {
		visitorID = tok.VisitorID
		d.TouchType = TouchReturn
		d.trace("decision", "return_with_params")
	} else {
		visitorID = newID()
		d.TouchType = TouchFirst
		d.NewAttribution = true
		d.trace("decision", "first")
	}
	d.VisitorID = visitorID
	d.AttributionID = newID()
	e.issue(rc, d, visitorID, d.AttributionID, nil)

	if d.InApp.InApp && e.store != nil && !rc.DryRun {
		e.issueHandoff(ctx, rc, d)
	}
}

func (e *Engine) returning(rc *RequestContext, d *Decision, tok *token.Payload) {
	site := rc.Site
	d.Content = ContentReal
	d.Attributed = true
	d.TouchType = TouchReturn
	d.Confidence = "high"
	d.VisitorID = tok.VisitorID
	d.AttributionID = tok.AttributionID
	d.trace("decision", "return")
	// 续期：sliding 每次重签；fixed 只在密钥轮换时换发
	if site.Attribution.RenewMode == "sliding" {
		e.issue(rc, d, tok.VisitorID, tok.AttributionID, nil)
	} else if token.NeedsReissue(site, tok) {
		exp := tok.Expiry()
		e.issue(rc, d, tok.VisitorID, tok.AttributionID, &exp)
	} else {
		d.TokenExpiresAt = tok.Expiry()
	}
}

func (e *Engine) unattributed(rc *RequestContext, d *Decision, res params.Result) {
	site := rc.Site
	d.TouchType = TouchNone
	d.Attributed = false
	switch site.Attribution.NoAttributionAction {
	case "block":
		d.Content = ContentBlock
	case "not_found":
		d.Content = ContentNotFound
	default:
		d.Content = ContentReview
	}
	if len(res.Errors) > 0 {
		d.trace("decision", "none:validation_failed")
	} else if len(res.Matched) > 0 {
		d.trace("decision", "none:insufficient")
	} else {
		d.trace("decision", "none:no_params")
	}
}

func (e *Engine) handoffByToken(ctx context.Context, rc *RequestContext, d *Decision, raw string) bool {
	if e.store == nil {
		return false
	}
	aid, vid, ok, err := e.store.HandoffConsume(ctx, rc.Site.SiteID, hashValue(raw))
	if err != nil {
		d.Degraded = true
		d.trace("handoff_token", "store_error")
		return false
	}
	if !ok {
		d.trace("handoff_token", "invalid_or_exhausted")
		return false
	}
	d.Content = ContentReal
	d.Attributed = true
	d.TouchType = TouchHandoff
	d.HandoffKind = "token"
	d.Confidence = "high"
	d.VisitorID = vid
	d.AttributionID = aid
	d.trace("decision", "handoff:token")
	e.issue(rc, d, vid, aid, nil)
	return true
}

func (e *Engine) handoffByMatch(ctx context.Context, rc *RequestContext, d *Decision) bool {
	if e.store == nil || !rc.IP.IsValid() {
		return false
	}
	// 新请求须是系统浏览器，避免 WebView 内互相匹配
	if d.InApp.InApp {
		return false
	}
	window := time.Duration(rc.Site.Handoff.MatchMinutes) * time.Minute
	aid, vid, ok, err := e.store.HandoffCandidateTake(ctx, rc.Site.SiteID, ipHash(rc.IP), d.InApp.DeviceFamily, window)
	if err != nil {
		d.Degraded = true
		d.trace("handoff_match", "store_error")
		return false
	}
	if !ok {
		d.trace("handoff_match", "no_candidate")
		return false
	}
	d.Content = ContentReal
	d.Attributed = true
	d.TouchType = TouchHandoff
	d.HandoffKind = "match"
	d.Confidence = "low"
	d.VisitorID = vid
	d.AttributionID = aid
	d.trace("decision", "handoff:match")
	e.issue(rc, d, vid, aid, nil)
	return true
}

func (e *Engine) issueHandoff(ctx context.Context, rc *RequestContext, d *Decision) {
	site := rc.Site
	raw := newID()
	ttl := time.Duration(site.Handoff.TokenTTLHours) * time.Hour
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	maxUses := site.Handoff.TokenMaxUses
	if maxUses <= 0 {
		maxUses = 3
	}
	if err := e.store.HandoffIssue(ctx, site.SiteID, hashValue(raw), d.AttributionID, d.VisitorID, maxUses, ttl); err != nil {
		d.Degraded = true
		return
	}
	d.HandoffToken = raw
	if site.Handoff.MatchEnabled && d.TouchType == TouchFirst && rc.IP.IsValid() {
		window := time.Duration(site.Handoff.MatchMinutes) * time.Minute
		_ = e.store.HandoffCandidateAdd(ctx, site.SiteID, ipHash(rc.IP), d.InApp.DeviceFamily, d.AttributionID, d.VisitorID, window)
	}
}

// issue 签发访客令牌；fixedExp 非 nil 时保持原到期时间。
func (e *Engine) issue(rc *RequestContext, d *Decision, visitorID, attributionID string, fixedExp *time.Time) {
	site := rc.Site
	exp := rc.Now.Add(time.Duration(site.Attribution.TTLDays) * 24 * time.Hour)
	if fixedExp != nil {
		exp = *fixedExp
	}
	tok, err := token.Issue(site, token.Payload{
		VisitorID: visitorID, AttributionID: attributionID,
		IssuedAt: rc.Now.Unix(), ExpiresAt: exp.Unix(),
	})
	if err != nil {
		d.Degraded = true
		d.trace("token_issue", err.Error())
		return
	}
	d.VisitorToken = tok
	d.TokenExpiresAt = exp
}

func (e *Engine) clickReused(ctx context.Context, rc *RequestContext, value string) bool {
	if rc.DryRun {
		return false
	}
	window := time.Duration(rc.Site.Rule.ClickReuseWindowH) * time.Hour
	if window <= 0 {
		window = 24 * time.Hour
	}
	n, err := e.store.ClickSeen(ctx, rc.Site.SiteID, hashValue(value), ipPrefix(rc.IP), window)
	if err != nil {
		return false
	}
	return n > rc.Site.Rule.ClickReuseLimit
}

func (d *Decision) trace(step, result string) {
	d.Trace = append(d.Trace, TraceStep{Step: step, Result: result})
}

// CookieMaxAge 令牌 cookie 的剩余秒数。
func (d *Decision) CookieMaxAge(now time.Time) int {
	if d.TokenExpiresAt.IsZero() {
		return 0
	}
	sec := int(d.TokenExpiresAt.Sub(now).Seconds())
	if sec < 0 {
		return 0
	}
	return sec
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hashValue(v string) string {
	return hex.EncodeToString(sha256Sum([]byte(v)))[:32]
}

func ipPrefix(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	if ip.Is4() {
		p, _ := ip.Prefix(24)
		return p.String()
	}
	p, _ := ip.Prefix(48)
	return p.String()
}

func ipHash(ip netip.Addr) string { return hashValue(ipPrefix(ip)) }
