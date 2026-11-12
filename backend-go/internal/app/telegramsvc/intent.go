package telegramsvc

import (
	"context"
	"fmt"
	"strings"

	"cloak/internal/app/opssvc"
)

// intent 识别出的问题。
type intent struct {
	kind     string // visits / conversion / alerts / report / help / unknown
	rangeKey string // today / yesterday / 7d
	siteID   uint64
	siteName string
}

// parseText 规则识别意图。sites 用于匹配站点名。
func (s *Service) parseText(text string, sites map[string]uint64) intent {
	t := normalize(text)
	in := intent{kind: "unknown", rangeKey: "today"}

	// 站点名匹配（在归一化文本里包含）
	for name, id := range sites {
		if name != "" && strings.Contains(t, name) {
			in.siteID = id
			in.siteName = name
			break
		}
	}

	// 命令优先
	switch {
	case strings.HasPrefix(t, "/help"), t == "帮助", t == "菜单", t == "help":
		in.kind = "help"
		return in
	case strings.HasPrefix(t, "/report"), t == "日报", strings.Contains(t, "来份日报"), strings.Contains(t, "发日报"):
		in.kind = "report"
		return in
	case strings.HasPrefix(t, "/alerts"):
		in.kind = "alerts"
		return in
	case strings.HasPrefix(t, "/today"):
		in.kind, in.rangeKey = "visits", "today"
		return in
	case strings.HasPrefix(t, "/yesterday"):
		in.kind, in.rangeKey = "visits", "yesterday"
		return in
	case strings.HasPrefix(t, "/7d"), strings.HasPrefix(t, "/week"):
		in.kind, in.rangeKey = "visits", "7d"
		return in
	}

	// 时间词
	switch {
	case strings.Contains(t, "昨天"), strings.Contains(t, "昨日"), strings.Contains(t, "yesterday"):
		in.rangeKey = "yesterday"
	case strings.Contains(t, "7天"), strings.Contains(t, "七天"), strings.Contains(t, "本周"), strings.Contains(t, "这周"), strings.Contains(t, "一周"), strings.Contains(t, "7d"), strings.Contains(t, "week"):
		in.rangeKey = "7d"
	case strings.Contains(t, "今天"), strings.Contains(t, "今日"), strings.Contains(t, "today"):
		in.rangeKey = "today"
	}

	// 指标词
	switch {
	case strings.Contains(t, "告警"), strings.Contains(t, "报警"), strings.Contains(t, "alert"):
		in.kind = "alerts"
	case strings.Contains(t, "转化"), strings.Contains(t, "成交"), strings.Contains(t, "订单"), strings.Contains(t, "conversion"):
		in.kind = "conversion"
	case strings.Contains(t, "访问"), strings.Contains(t, "流量"), strings.Contains(t, "visit"), strings.Contains(t, "真实"), strings.Contains(t, "审核"), strings.Contains(t, "数据"), strings.Contains(t, "怎么样"), strings.Contains(t, "情况"):
		in.kind = "visits"
	}
	return in
}

// Answer 对一段文本给出回复（已绑定会话）。
func (s *Service) Answer(ctx context.Context, merchantID uint64, text string) string {
	sites := s.merchantSites(ctx, merchantID)
	in := s.parseText(text, sites)
	switch in.kind {
	case "help":
		return helpText()
	case "alerts":
		return s.answerAlerts(ctx, merchantID)
	case "conversion":
		return s.answerConversion(ctx, merchantID, in)
	case "report":
		return s.BuildDigest(ctx, merchantID, 0)
	case "visits":
		return s.answerVisits(ctx, merchantID, in)
	default:
		return "没太听懂～你可以这样问：\n" + helpText()
	}
}

func (s *Service) answerVisits(ctx context.Context, merchantID uint64, in intent) string {
	r := opssvc.ParseRange(in.rangeKey, "", "", s.now())
	d, err := s.ops.Digest(ctx, merchantID, in.siteID, r)
	if err != nil {
		return "查询出错了，请稍后再试。"
	}
	scope := rangeLabel(in.rangeKey)
	if in.siteName != "" {
		scope += " · " + in.siteName
	}
	var b strings.Builder
	fmt.Fprintf(&b, "📈 <b>%s</b>\n", scope)
	fmt.Fprintf(&b, "访问 %d　真实 %d　审核 %d　拦截 %d\n", d.Visits, d.Real, d.Review, d.Block)
	fmt.Fprintf(&b, "首访 %d　回访 %d　交接 %d\n", d.First, d.Return, d.Handoff)
	fmt.Fprintf(&b, "转化 %d　金额 %.2f", d.Conversions, d.Amount)
	if len(d.TopChannels) > 0 {
		b.WriteString("\nTop 渠道：" + joinKV(d.TopChannels))
	}
	return b.String()
}

func (s *Service) answerConversion(ctx context.Context, merchantID uint64, in intent) string {
	r := opssvc.ParseRange(in.rangeKey, "", "", s.now())
	d, err := s.ops.Digest(ctx, merchantID, in.siteID, r)
	if err != nil {
		return "查询出错了，请稍后再试。"
	}
	scope := rangeLabel(in.rangeKey)
	if in.siteName != "" {
		scope += " · " + in.siteName
	}
	out := fmt.Sprintf("💰 <b>%s 转化</b>\n成交 %d 笔　金额 %.2f", scope, d.Conversions, d.Amount)
	if len(d.TopCodes) > 0 {
		out += "\nTop 分销码：" + joinKV(d.TopCodes)
	}
	return out
}

func (s *Service) answerAlerts(ctx context.Context, merchantID uint64) string {
	rows, _, err := s.ops.Alerts(ctx, merchantID, opssvc.AlertListInput{Status: "open", Page: 1, Size: 10})
	if err != nil {
		return "查询出错了，请稍后再试。"
	}
	if len(rows) == 0 {
		return "✅ 当前没有未恢复的告警。"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "🔴 <b>未恢复告警 %d 条</b>\n", len(rows))
	for i, a := range rows {
		if i >= 10 {
			break
		}
		name, _ := a["typeName"].(string)
		site, _ := a["siteName"].(string)
		fired, _ := a["firedAt"].(string)
		line := "· " + esc(name)
		if site != "" {
			line += "（" + esc(site) + "）"
		}
		if fired != "" {
			line += " " + fired
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// BuildDigest 拼装昨日日报。siteID=0 表示汇总全部站点。
func (s *Service) BuildDigest(ctx context.Context, merchantID, siteID uint64) string {
	r := opssvc.ParseRange("yesterday", "", "", s.now())
	d, err := s.ops.Digest(ctx, merchantID, siteID, r)
	if err != nil {
		return "日报生成失败，请稍后再试。"
	}
	name := s.merchantName(ctx, merchantID)
	day := r.From.Format("2006-01-02")
	var b strings.Builder
	fmt.Fprintf(&b, "📊 <b>%s 昨日日报 %s</b>\n", esc(name), day)
	fmt.Fprintf(&b, "访问 %d　真实 %d　审核 %d　拦截 %d\n", d.Visits, d.Real, d.Review, d.Block)
	fmt.Fprintf(&b, "首访 %d　回访 %d　交接 %d\n", d.First, d.Return, d.Handoff)
	fmt.Fprintf(&b, "转化 %d　金额 %.2f\n", d.Conversions, d.Amount)
	if len(d.TopChannels) > 0 {
		fmt.Fprintf(&b, "Top 渠道：%s\n", joinKV(d.TopChannels))
	}
	if len(d.TopCodes) > 0 {
		fmt.Fprintf(&b, "Top 分销码：%s\n", joinKV(d.TopCodes))
	}
	if d.OpenAlerts > 0 {
		fmt.Fprintf(&b, "⚠️ 未恢复告警 %d 条\n", d.OpenAlerts)
	}
	if link := strings.TrimRight(s.cfg.ConsoleURL, "/"); link != "" {
		fmt.Fprintf(&b, `<a href="%s/merchant/dashboard">打开看板</a>`, link)
	}
	return strings.TrimRight(b.String(), "\n")
}

// merchantSites 商户站点名（归一化小写）→ ID。
func (s *Service) merchantSites(ctx context.Context, merchantID uint64) map[string]uint64 {
	out := map[string]uint64{}
	rows, _, err := s.sites.ListSites(ctx, merchantID, "", "", "", 1, 200)
	if err != nil {
		return out
	}
	for _, st := range rows {
		n := normalize(st.Name)
		if n != "" {
			out[n] = st.ID
		}
	}
	return out
}

func (s *Service) merchantName(ctx context.Context, merchantID uint64) string {
	if m, err := s.merchants.FindByID(ctx, merchantID); err == nil && m != nil {
		return m.Name
	}
	return "商户"
}

func helpText() string {
	return strings.Join([]string{
		"你可以直接问我（也支持命令）：",
		"· 今天多少访问 /today",
		"· 昨天转化多少",
		"· 近 7 天怎么样 /7d",
		"· 有没有告警 /alerts",
		"· 日报 /report",
		"多站点可带站点名，如「今天 站点A 多少访问」。",
	}, "\n")
}

func rangeLabel(key string) string {
	switch key {
	case "yesterday":
		return "昨日"
	case "7d":
		return "近 7 日"
	default:
		return "今日"
	}
}

func joinKV(list []opssvc.NameCount) string {
	parts := make([]string, 0, len(list))
	for _, kv := range list {
		parts = append(parts, fmt.Sprintf("%s %d", esc(kv.Name), kv.Count))
	}
	return strings.Join(parts, "，")
}

// normalize 归一化：去空格、小写、全角转半角。
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '　': // 全角空格
			continue
		case r >= '！' && r <= '～': // 全角 ASCII 区
			b.WriteRune(r - 0xFEE0)
		case r == ' ':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
