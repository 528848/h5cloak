package engine

import (
	"context"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"cloak/internal/engine/snapshot"
	"cloak/internal/engine/token"
)

var (
	now     = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	ctx     = context.Background()
	iosFB   = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 [FBAN/FBIOS;FBAV/430.0.0;FBDV/iPhone15,2]"
	iosSaf  = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1"
	desktop = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0 Safari/537.36"
)

func testSite() *snapshot.Site {
	exp := now.Add(-time.Hour)
	future := now.Add(24 * time.Hour)
	s := &snapshot.Site{
		SiteID: 7, MerchantID: 1, Status: "enabled", MerchantActive: true,
		Domains: []snapshot.Domain{{Domain: "shop.example.com", Status: "bound"}},
		PathRules: []snapshot.PathRule{
			{Pattern: ".css", MatchType: "suffix", Action: "bypass", Sort: 10},
			{Pattern: "/privacy", MatchType: "prefix", Action: "always_real", Sort: 20},
			{Pattern: "/admin", MatchType: "prefix", Action: "block", Sort: 30},
			{Pattern: "/review-only", MatchType: "exact", Action: "always_review", Sort: 40},
		},
		Cookie:      snapshot.Cookie{Name: "_ck_vid"},
		Attribution: snapshot.Attribution{TTLDays: 30, RenewMode: "sliding", NoAttributionAction: "review", MinConfidence: "medium"},
		Handoff:     snapshot.Handoff{TokenTTLHours: 24, TokenMaxUses: 2, MatchEnabled: true, MatchMinutes: 15},
		Keys:        []snapshot.TokenKey{{Version: 1, Key: "0a0b0c0d0e0f", Status: "active"}},
		Params: []snapshot.Param{
			{Name: "fbclid", Type: "click", Platform: "facebook", MinLen: 20, MaxLen: 255, Pattern: `^[A-Za-z0-9_-]+$`, Sufficient: true, Enabled: true},
			{Name: "gclid", Type: "click", Platform: "google", MinLen: 20, MaxLen: 255, Sufficient: true, Enabled: true},
			{Name: "utm_source", Type: "source", MinLen: 1, MaxLen: 128, Sufficient: false, Enabled: true},
			{Name: "utm_medium", Type: "source", MinLen: 1, MaxLen: 128, Sufficient: false, Enabled: true},
			{Name: "ref", Type: "distributor", Aliases: []string{"r", "aff"}, MinLen: 2, MaxLen: 64, Sufficient: true, Enabled: true},
			{Name: "cid", Type: "campaign", MinLen: 2, MaxLen: 64, Sufficient: true, Enabled: true},
			{Name: "oldparam", Type: "click", MinLen: 1, MaxLen: 64, Sufficient: true, Enabled: false},
		},
		Rule: snapshot.Rule{CombineRule: "require_any", ValidateCodes: true, ValidateClickFormat: true, ClickReuseLimit: 0, ClickReuseWindowH: 24, ClickReuseAction: "downgrade"},
		Codes: []snapshot.Code{
			{ID: 1, Code: "ALICE", Type: "distributor", Name: "Alice", Channel: "kol", Enabled: true},
			{ID: 2, Code: "OFF", Type: "distributor", Enabled: false},
			{ID: 3, Code: "GONE", Type: "distributor", Enabled: true, ExpiresAt: &exp},
			{ID: 4, Code: "SOON", Type: "distributor", Enabled: true, StartsAt: &future},
			{ID: 5, Code: "LAND", Type: "distributor", Enabled: true, AllowedPaths: []string{"/landing"}},
			{ID: 6, Code: "SPRING", Type: "campaign", Enabled: true},
		},
	}
	if err := s.Compile(); err != nil {
		panic(err)
	}
	return s
}

func req(site *snapshot.Site, rawURL string) *RequestContext {
	u, _ := url.Parse(rawURL)
	return &RequestContext{Site: site, URL: u, Method: "GET", Now: now, UserAgent: desktop, IP: netip.MustParseAddr("203.0.113.10")}
}

func TestPathRules(t *testing.T) {
	e := New(nil)
	s := testSite()
	cases := map[string]string{
		"https://shop.example.com/assets/app.CSS": ContentBypass,
		"https://shop.example.com/privacy/policy": ContentReal,
		"https://shop.example.com/admin/login":    ContentBlock,
		"https://shop.example.com/review-only":    ContentReview,
	}
	for u, want := range cases {
		d := e.Decide(ctx, req(s, u))
		if d.Content != want || d.PathRule == nil {
			t.Errorf("%s: content=%s rule=%v want %s", u, d.Content, d.PathRule, want)
		}
	}
	// 精确匹配不吞子路径
	if d := e.Decide(ctx, req(s, "https://shop.example.com/review-only/sub")); d.PathRule != nil {
		t.Fatalf("exact rule must not match sub path: %v", d.PathRule)
	}
	// 顺序：同时命中 bypass(.css) 与 block(/admin) 时取 sort 小的
	if d := e.Decide(ctx, req(s, "https://shop.example.com/admin/x.css")); d.Content != ContentBypass {
		t.Fatalf("sort order: %+v", d)
	}
	// /privacy-ish 路径不被 /privacy 前缀以外的规则截获；/products 无规则 → 走判定
	d := e.Decide(ctx, req(s, "https://shop.example.com/products"))
	if d.PathRule != nil {
		t.Fatalf("unexpected rule %v", d.PathRule)
	}
}

func TestFirstVisitWithCode(t *testing.T) {
	e := New(nil)
	s := testSite()
	d := e.Decide(ctx, req(s, "https://shop.example.com/lp?ref=ALICE&utm_source=tiktok"))
	if d.Content != ContentReal || d.TouchType != TouchFirst || !d.Attributed {
		t.Fatalf("got %+v", d)
	}
	if d.Confidence != "high" || d.Channel != "distributor:ALICE" || d.Code == nil || d.Code.Name != "Alice" {
		t.Fatalf("confidence/channel wrong: %+v", d)
	}
	if d.VisitorToken == "" || !d.NewAttribution || d.CookieMaxAge(now) != 30*86400 {
		t.Fatalf("token not issued: %+v", d)
	}
	p, err := token.Verify(s, d.VisitorToken, now)
	if err != nil || p.VisitorID != d.VisitorID || p.AttributionID != d.AttributionID {
		t.Fatalf("token verify: %v %+v", err, p)
	}
}

func TestAliasAndCaseInsensitiveCode(t *testing.T) {
	e := New(nil)
	s := testSite()
	d := e.Decide(ctx, req(s, "https://shop.example.com/?aff=alice"))
	if d.Content != ContentReal || d.Code == nil || d.Code.Code != "ALICE" || d.MatchedParams[0] != "aff" {
		t.Fatalf("alias lookup failed: %+v", d)
	}
}

func TestCodeFailures(t *testing.T) {
	e := New(nil)
	s := testSite()
	cases := map[string]string{
		"ref=NOPE": "not_registered",
		"ref=OFF":  "disabled",
		"ref=GONE": "expired",
		"ref=SOON": "not_started",
		"ref=LAND": "path_not_allowed",
		"ref=x":    "not_registered", // 长度不够也先按未登记
		"cid=NOPE": "not_registered",
	}
	for q, reason := range cases {
		d := e.Decide(ctx, req(s, "https://shop.example.com/other?"+q))
		if d.Content != ContentReview || d.Attributed || d.TouchType != TouchNone {
			t.Errorf("%s: expected review/none, got %+v", q, d)
		}
		if len(d.ValidationErrors) != 1 || d.ValidationErrors[0].Reason != reason {
			t.Errorf("%s: errors=%v want %s", q, d.ValidationErrors, reason)
		}
	}
	// 允许路径匹配时放行
	d := e.Decide(ctx, req(s, "https://shop.example.com/landing/a?ref=LAND"))
	if d.Content != ContentReal {
		t.Fatalf("LAND on /landing should pass: %+v", d)
	}
}

func TestClickParamFormat(t *testing.T) {
	e := New(nil)
	s := testSite()
	short := e.Decide(ctx, req(s, "https://shop.example.com/?fbclid=short"))
	if short.Content != ContentReview || short.ValidationErrors[0].Reason != "format_invalid" {
		t.Fatalf("short fbclid: %+v", short)
	}
	bad := e.Decide(ctx, req(s, "https://shop.example.com/?fbclid=IwAR0abcdefghijklmnopqrstuvwxyz!!"))
	if bad.Content != ContentReview {
		t.Fatalf("bad charset fbclid: %+v", bad)
	}
	good := e.Decide(ctx, req(s, "https://shop.example.com/?fbclid=IwAR0abcdefghijklmnopqrstuvwxyz0123"))
	if good.Content != ContentReal || good.Confidence != "medium" || good.Channel != "click:facebook" {
		t.Fatalf("good fbclid: %+v", good)
	}
	// 禁用的参数不参与
	dis := e.Decide(ctx, req(s, "https://shop.example.com/?oldparam=abc"))
	if dis.Content != ContentReview || dis.ValidationErrors[0].Reason != "param_disabled" {
		t.Fatalf("disabled param: %+v", dis)
	}
}

func TestConfidence(t *testing.T) {
	e := New(nil)
	s := testSite()
	fb := "fbclid=IwAR0abcdefghijklmnopqrstuvwxyz0123"
	// 点击参数 + 内置浏览器 → high
	rc := req(s, "https://shop.example.com/?"+fb)
	rc.UserAgent = iosFB
	if d := e.Decide(ctx, rc); d.Confidence != "high" || !d.InApp.InApp || d.InApp.App != "facebook" {
		t.Fatalf("inapp click: %+v", d)
	}
	// 点击参数 + 广告 Referer → high
	rc = req(s, "https://shop.example.com/?"+fb)
	rc.Referer = "https://l.facebook.com/l.php?u=..."
	if d := e.Decide(ctx, rc); d.Confidence != "high" {
		t.Fatalf("ad referer click: %+v", d)
	}
	// 只有 utm_source，sufficient=false → 不放行
	if d := e.Decide(ctx, req(s, "https://shop.example.com/?utm_source=tiktok")); d.Content != ContentReview {
		t.Fatalf("utm_source alone should not pass: %+v", d)
	}
	// 站点把 utm_source 设为 sufficient → medium 放行
	s2 := testSite()
	for i := range s2.Params {
		if s2.Params[i].Name == "utm_source" {
			s2.Params[i].Sufficient = true
		}
	}
	_ = s2.Compile()
	if d := e.Decide(ctx, req(s2, "https://shop.example.com/?utm_source=tiktok")); d.Content != ContentReal || d.Confidence != "medium" || d.Channel != "source:tiktok" {
		t.Fatalf("sufficient utm_source: %+v", d)
	}
	// 最低置信度 high 时 medium 不放行
	s2.Attribution.MinConfidence = "high"
	if d := e.Decide(ctx, req(s2, "https://shop.example.com/?utm_source=tiktok")); d.Content != ContentReview {
		t.Fatalf("below min confidence should review: %+v", d)
	}
}

func TestAllowedSources(t *testing.T) {
	e := New(nil)
	s := testSite()
	for i := range s.Params {
		if s.Params[i].Name == "utm_source" {
			s.Params[i].Sufficient = true
		}
	}
	s.Rule.AllowedSources = []string{"facebook", "tiktok"}
	_ = s.Compile()
	if d := e.Decide(ctx, req(s, "https://shop.example.com/?utm_source=TikTok")); d.Content != ContentReal {
		t.Fatalf("allowed source: %+v", d)
	}
	d := e.Decide(ctx, req(s, "https://shop.example.com/?utm_source=spam"))
	if d.Content != ContentReview || d.ValidationErrors[0].Reason != "source_not_allowed" {
		t.Fatalf("disallowed source: %+v", d)
	}
}

func TestCombineRule(t *testing.T) {
	e := New(nil)
	s := testSite()
	for i := range s.Params {
		if s.Params[i].Name == "utm_source" {
			s.Params[i].Sufficient = true
		}
	}
	s.Rule.CombineRule = "require_click_or_code"
	_ = s.Compile()
	if d := e.Decide(ctx, req(s, "https://shop.example.com/?utm_source=tiktok")); d.Content != ContentReview {
		t.Fatalf("source only under click_or_code should review: %+v", d)
	}
	if d := e.Decide(ctx, req(s, "https://shop.example.com/?utm_source=tiktok&ref=ALICE")); d.Content != ContentReal {
		t.Fatalf("code under click_or_code should pass: %+v", d)
	}
}

func TestReturnVisit(t *testing.T) {
	e := New(nil)
	s := testSite()
	first := e.Decide(ctx, req(s, "https://shop.example.com/?ref=ALICE"))

	// 无参数 + 有效 cookie → return，sliding 续期
	later := now.Add(48 * time.Hour)
	rc := req(s, "https://shop.example.com/products")
	rc.CookieToken = first.VisitorToken
	rc.Now = later
	d := e.Decide(ctx, rc)
	if d.Content != ContentReal || d.TouchType != TouchReturn || d.VisitorID != first.VisitorID || d.NewAttribution {
		t.Fatalf("return: %+v", d)
	}
	if d.VisitorToken == "" || !d.TokenExpiresAt.Equal(later.Add(30*24*time.Hour)) {
		t.Fatalf("sliding renew expected: %+v", d)
	}

	// fixed 模式：不换发，保持原到期
	s.Attribution.RenewMode = "fixed"
	d = e.Decide(ctx, rc)
	if d.VisitorToken != "" || !d.TokenExpiresAt.Equal(first.TokenExpiresAt) {
		t.Fatalf("fixed renew: %+v", d)
	}

	// 过期 cookie → none
	rc.Now = now.Add(31 * 24 * time.Hour)
	d = e.Decide(ctx, rc)
	if d.Content != ContentReview || d.TouchType != TouchNone {
		t.Fatalf("expired token should review: %+v", d)
	}

	// 回访带新参数：仍是 return，归因刷新为新渠道，访客不变
	rc = req(s, "https://shop.example.com/?cid=SPRING")
	rc.CookieToken = first.VisitorToken
	d = e.Decide(ctx, rc)
	if d.TouchType != TouchReturn || d.VisitorID != first.VisitorID || d.Channel != "campaign:SPRING" || d.NewAttribution {
		t.Fatalf("return with params: %+v", d)
	}
}

func TestTokenTamperAndRotation(t *testing.T) {
	e := New(nil)
	s := testSite()
	first := e.Decide(ctx, req(s, "https://shop.example.com/?ref=ALICE"))

	rc := req(s, "https://shop.example.com/")
	rc.CookieToken = first.VisitorToken[:len(first.VisitorToken)-2] + "xx"
	if d := e.Decide(ctx, rc); d.Content != ContentReview {
		t.Fatalf("tampered token should review: %+v", d)
	}

	// 轮换：v2 active，v1 grace → 旧令牌仍可验签并换发 v2
	grace := now.Add(72 * time.Hour)
	s.Keys = []snapshot.TokenKey{
		{Version: 1, Key: "0a0b0c0d0e0f", Status: "grace", ExpiresAt: &grace},
		{Version: 2, Key: "ffeeddccbbaa", Status: "active"},
	}
	s.Attribution.RenewMode = "fixed"
	_ = s.Compile()
	rc.CookieToken = first.VisitorToken
	d := e.Decide(ctx, rc)
	if d.Content != ContentReal || d.VisitorToken == "" {
		t.Fatalf("grace key should verify and reissue: %+v", d)
	}
	p, err := token.Verify(s, d.VisitorToken, now)
	if err != nil || p.KeyVersion != 2 || !p.Expiry().Equal(first.TokenExpiresAt) {
		t.Fatalf("reissued under v2 with same exp: %v %+v", err, p)
	}
	// 宽限到期后旧令牌失效
	rc.Now = grace.Add(time.Second)
	if d := e.Decide(ctx, rc); d.Content != ContentReview {
		t.Fatalf("after grace should review: %+v", d)
	}
	// 另一站点的令牌不可用
	other := testSite()
	other.SiteID = 8
	_ = other.Compile()
	rc2 := req(other, "https://x/")
	rc2.CookieToken = first.VisitorToken
	if d := e.Decide(ctx, rc2); d.Content != ContentReview {
		t.Fatalf("cross-site token should review: %+v", d)
	}
}

func TestNoAttributionActions(t *testing.T) {
	e := New(nil)
	s := testSite()
	for action, want := range map[string]string{"review": ContentReview, "block": ContentBlock, "not_found": ContentNotFound} {
		s.Attribution.NoAttributionAction = action
		if d := e.Decide(ctx, req(s, "https://shop.example.com/products")); d.Content != want {
			t.Errorf("%s: got %s", action, d.Content)
		}
	}
	s.Status = "disabled"
	s.Attribution.NoAttributionAction = "not_found"
	if d := e.Decide(ctx, req(s, "https://shop.example.com/?ref=ALICE")); d.Content != ContentNotFound || d.Attributed {
		t.Fatalf("disabled site: %+v", d)
	}
}

func TestHandoffToken(t *testing.T) {
	store := NewMemStore()
	store.SetClock(func() time.Time { return now })
	e := New(store)
	s := testSite()

	// 内置浏览器首访：签发交接 token
	rc := req(s, "https://shop.example.com/lp?ref=ALICE")
	rc.UserAgent = iosFB
	first := e.Decide(ctx, rc)
	if first.HandoffToken == "" {
		t.Fatalf("expected handoff token: %+v", first)
	}
	// 系统浏览器不签发
	rc2 := req(s, "https://shop.example.com/lp?ref=ALICE")
	if d := e.Decide(ctx, rc2); d.HandoffToken != "" {
		t.Fatalf("system browser should not get handoff token")
	}

	// 新浏览器带 _ck，无 cookie → handoff:token，同一归因
	hu := "https://shop.example.com/lp?" + HandoffParam + "=" + first.HandoffToken
	h := req(s, hu)
	h.UserAgent = iosSaf
	d := e.Decide(ctx, h)
	if d.Content != ContentReal || d.TouchType != TouchHandoff || d.HandoffKind != "token" || d.AttributionID != first.AttributionID || d.VisitorToken == "" {
		t.Fatalf("handoff by token: %+v", d)
	}
	// max_uses=2：第二次可以，第三次不行
	if d := e.Decide(ctx, req(s, hu)); d.TouchType != TouchHandoff {
		t.Fatalf("second use should pass")
	}
	if d := e.Decide(ctx, req(s, hu)); d.TouchType != TouchNone || d.Content != ContentReview {
		t.Fatalf("third use should fail: %+v", d)
	}
	// 过期
	rc = req(s, "https://shop.example.com/lp?ref=ALICE")
	rc.UserAgent = iosFB
	f2 := e.Decide(ctx, rc)
	store.SetClock(func() time.Time { return now.Add(25 * time.Hour) })
	late := req(s, "https://shop.example.com/lp?"+HandoffParam+"="+f2.HandoffToken)
	late.Now = now.Add(25 * time.Hour)
	if d := e.Decide(ctx, late); d.TouchType != TouchNone {
		t.Fatalf("expired handoff token should fail: %+v", d)
	}
}

func TestHandoffMatch(t *testing.T) {
	store := NewMemStore()
	store.SetClock(func() time.Time { return now })
	e := New(store)
	s := testSite()

	rc := req(s, "https://shop.example.com/lp?ref=ALICE")
	rc.UserAgent = iosFB
	first := e.Decide(ctx, rc)

	// 同 IP /24、同设备族、系统浏览器、无参数无 cookie → handoff:match，一次性
	m := req(s, "https://shop.example.com/lp")
	m.UserAgent = iosSaf
	m.IP = netip.MustParseAddr("203.0.113.77")
	d := e.Decide(ctx, m)
	if d.Content != ContentReal || d.HandoffKind != "match" || d.Confidence != "low" || d.AttributionID != first.AttributionID {
		t.Fatalf("handoff by match: %+v", d)
	}
	if d := e.Decide(ctx, m); d.TouchType != TouchNone {
		t.Fatalf("candidate must be consumed once: %+v", d)
	}

	// 设备族不同不匹配
	rc.UserAgent = iosFB
	e.Decide(ctx, rc)
	other := req(s, "https://shop.example.com/lp")
	other.UserAgent = desktop
	if d := e.Decide(ctx, other); d.TouchType != TouchNone {
		t.Fatalf("different device family should not match: %+v", d)
	}
	// 窗口过期不匹配
	store.SetClock(func() time.Time { return now.Add(16 * time.Minute) })
	m.Now = now.Add(16 * time.Minute)
	if d := e.Decide(ctx, m); d.TouchType != TouchNone {
		t.Fatalf("outside window should not match: %+v", d)
	}
	// 关闭匹配开关
	s.Handoff.MatchEnabled = false
	store.SetClock(func() time.Time { return now })
	e.Decide(ctx, rc)
	m.Now = now
	if d := e.Decide(ctx, m); d.TouchType != TouchNone {
		t.Fatalf("match disabled: %+v", d)
	}
}

func TestClickReuse(t *testing.T) {
	store := NewMemStore()
	store.SetClock(func() time.Time { return now })
	e := New(store)
	s := testSite()
	s.Rule.ClickReuseLimit = 2
	s.Rule.ClickReuseAction = "downgrade"
	_ = s.Compile()
	u := "https://shop.example.com/?gclid=Cj0KCQjw_abcdefghijklmnopqrstuvwxyz"
	for i, ip := range []string{"198.51.100.1", "198.51.100.200", "203.0.113.5"} {
		rc := req(s, u)
		rc.IP = netip.MustParseAddr(ip)
		d := e.Decide(ctx, rc)
		if i < 2 && d.Confidence != "medium" {
			t.Fatalf("ip %s: expected medium, got %+v", ip, d)
		}
	}
	// 第 3 个不同 /24 前缀超过 limit=2 → 降为 low，仍放行（min=medium 时则不放行）
	rc := req(s, u)
	rc.IP = netip.MustParseAddr("192.0.2.9")
	d := e.Decide(ctx, rc)
	if d.Content != ContentReview {
		t.Fatalf("downgraded to low should fall below medium: %+v", d)
	}
	s.Attribution.MinConfidence = "low"
	if d := e.Decide(ctx, rc); d.Content != ContentReal || d.Confidence != "low" {
		t.Fatalf("low min should pass downgraded: %+v", d)
	}
	// review 动作：直接按无归因
	s.Rule.ClickReuseAction = "review"
	d = e.Decide(ctx, rc)
	if d.Content != ContentReview || d.ValidationErrors[0].Reason != "click_reused" {
		t.Fatalf("reuse review action: %+v", d)
	}
}

func TestHandoffOnReturnAndFromRequestID(t *testing.T) {
	store := NewMemStore()
	store.SetClock(func() time.Time { return now })
	e := New(store)
	s := testSite()

	// 内置浏览器首访：记录请求 ID
	rc := req(s, "https://shop.example.com/lp?ref=ALICE")
	rc.UserAgent = iosFB
	rc.RequestID = "req-first"
	first := e.Decide(ctx, rc)

	// 内置浏览器内翻到第二页（带 cookie 无参数）：也要拿到交接令牌
	second := req(s, "https://shop.example.com/second")
	second.UserAgent = iosFB
	second.CookieToken = first.VisitorToken
	second.RequestID = "req-second"
	d2 := e.Decide(ctx, second)
	if d2.TouchType != TouchReturn || d2.HandoffToken == "" {
		t.Fatalf("return in-app should issue handoff token: %+v", d2)
	}
	// 用第二页的令牌交接：来源请求是 req-second
	h := req(s, "https://shop.example.com/second?"+HandoffParam+"="+d2.HandoffToken)
	h.UserAgent = iosSaf
	dh := e.Decide(ctx, h)
	if dh.TouchType != TouchHandoff || dh.HandoffFromReqID != "req-second" || dh.AttributionID != first.AttributionID {
		t.Fatalf("handoff from request id: %+v", dh)
	}
	// 匹配交接：来源是首访请求
	s2 := testSite()
	rc2 := req(s2, "https://shop.example.com/lp?ref=ALICE")
	rc2.UserAgent = iosFB
	rc2.RequestID = "req-match-first"
	e.Decide(ctx, rc2)
	m := req(s2, "https://shop.example.com/lp")
	m.UserAgent = iosSaf
	dm := e.Decide(ctx, m)
	if dm.HandoffKind != "match" || dm.HandoffFromReqID != "req-match-first" {
		t.Fatalf("match from request id: %+v", dm)
	}
}

func TestHandoffMatchSystemOnlyToggle(t *testing.T) {
	store := NewMemStore()
	store.SetClock(func() time.Time { return now })
	e := New(store)
	s := testSite()
	s.Handoff.MatchSystemOnly = true
	rc := req(s, "https://shop.example.com/lp?ref=ALICE")
	rc.UserAgent = iosFB
	e.Decide(ctx, rc)
	// 新请求仍是内置浏览器：默认不匹配
	m := req(s, "https://shop.example.com/lp")
	m.UserAgent = iosFB
	if d := e.Decide(ctx, m); d.TouchType != TouchNone {
		t.Fatalf("in-app request must not match by default: %+v", d)
	}
	// 关闭限制后允许
	s.Handoff.MatchSystemOnly = false
	if d := e.Decide(ctx, m); d.HandoffKind != "match" {
		t.Fatalf("system-only off should match: %+v", d)
	}
}

func TestDryRunSkipsStore(t *testing.T) {
	store := NewMemStore()
	e := New(store)
	s := testSite()
	rc := req(s, "https://shop.example.com/?ref=ALICE")
	rc.UserAgent = iosFB
	rc.DryRun = true
	d := e.Decide(ctx, rc)
	if d.HandoffToken != "" || len(store.handoffs) != 0 {
		t.Fatalf("dry run must not write store")
	}
}
