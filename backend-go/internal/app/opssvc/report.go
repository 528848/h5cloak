// Package opssvc 看板、报表、导出、告警、通知、Webhook。
package opssvc

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/config"
	"cloak/internal/domain/ops"
	"cloak/internal/pkg/notifier"
	"cloak/internal/quota"
)

// Service 运营服务。
type Service struct {
	repo      *mysql.OpsRepo
	logs      *mysql.LogRepo
	sites     *mysql.SiteRepo
	merchants *mysql.MerchantRepo
	quota     *quota.Checker // 可为 nil（无 Redis 时用 usage_counters）
	cfg       config.Notify
	exportDir string
	mailer    *notifier.Mailer
	telegram  *notifier.Telegram
	webhook   *notifier.Webhook
	log       *zap.Logger
	now       func() time.Time
}

// Deps 依赖。
type Deps struct {
	Repo      *mysql.OpsRepo
	Logs      *mysql.LogRepo
	Sites     *mysql.SiteRepo
	Merchants *mysql.MerchantRepo
	Quota     *quota.Checker
	Notify    config.Notify
	ExportDir string
	Log       *zap.Logger
}

func New(d Deps) *Service {
	if d.Log == nil {
		d.Log = zap.NewNop()
	}
	return &Service{
		repo: d.Repo, logs: d.Logs, sites: d.Sites, merchants: d.Merchants, quota: d.Quota, cfg: d.Notify, exportDir: d.ExportDir,
		mailer: notifier.NewMailer(d.Notify.SMTP), telegram: notifier.NewTelegram(d.Notify.Telegram), webhook: notifier.NewWebhook(d.Notify.Webhook),
		log: d.Log, now: time.Now,
	}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

const layout = "2006-01-02 15:04:05"

// Range 时间范围解析：today / yesterday / 7d / 30d / 自定义 from,to（天粒度，含起不含止）。
type Range struct {
	From, To time.Time
	Key      string
}

// ParseRange 解析范围参数。
func ParseRange(key, from, to string, now time.Time) Range {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch key {
	case "yesterday":
		return Range{From: day.AddDate(0, 0, -1), To: day, Key: key}
	case "7d":
		return Range{From: day.AddDate(0, 0, -6), To: day.AddDate(0, 0, 1), Key: key}
	case "30d":
		return Range{From: day.AddDate(0, 0, -29), To: day.AddDate(0, 0, 1), Key: key}
	case "custom":
		f, e1 := time.ParseInLocation("2006-01-02", from, now.Location())
		t, e2 := time.ParseInLocation("2006-01-02", to, now.Location())
		if e1 == nil && e2 == nil && !t.Before(f) {
			if t.Sub(f) > 92*24*time.Hour {
				f = t.AddDate(0, 0, -92)
			}
			return Range{From: f, To: t.AddDate(0, 0, 1), Key: key}
		}
	}
	return Range{From: day, To: day.AddDate(0, 0, 1), Key: "today"}
}

// ---- 商户看板 ----

// Dashboard 商户工作台。
func (s *Service) Dashboard(ctx context.Context, merchantID, siteID uint64, r Range) (map[string]any, error) {
	now := s.now()
	hours, err := s.repo.HourlyRange(ctx, merchantID, siteID, r.From, r.To)
	if err != nil {
		return nil, err
	}
	sum := sumHourly(hours)
	curHour := now.Truncate(time.Hour)
	daily := r.To.Sub(r.From) > 48*time.Hour
	var delta map[string]int64
	// 当前小时用日志实时数替换已聚合的同一小时，避免重复累加
	if r.To.After(curHour) {
		if live, err := s.repo.LogCountSince(ctx, merchantID, siteID, maxTime(curHour, r.From), r.To); err == nil {
			delta = applyLiveHour(sum, hours, curHour, live)
		}
	}
	convN, convAmt, _ := s.repo.ConversionTotals(ctx, merchantID, siteID, r.From, r.To)

	trend := make([]map[string]any, 0, len(hours))
	if daily {
		trend = bucketDaily(hours)
	} else {
		for _, h := range hours {
			trend = append(trend, map[string]any{
				"t": h.Hour.Format("01-02 15:00"), "visits": h.Visits, "real": h.RealTotal, "review": h.Review,
				"first": h.FirstReal, "return": h.ReturnReal, "handoff": h.HandoffReal, "block": h.Block,
			})
		}
	}
	if delta != nil {
		trend = patchTrend(trend, curHour, daily, delta)
	}
	channels, _ := s.repo.Breakdown(ctx, merchantID, siteID, r.From, r.To, "channel", true, 10)
	codes, _ := s.repo.Breakdown(ctx, merchantID, siteID, r.From, r.To, "code", true, 10)
	confidence, _ := s.repo.Breakdown(ctx, merchantID, siteID, r.From, r.To, "confidence", true, 5)
	inApp, _ := s.repo.Breakdown(ctx, merchantID, siteID, r.From, r.To, "in_app_browser", false, 10)
	countries, _ := s.repo.Breakdown(ctx, merchantID, siteID, r.From, r.To, "country", false, 10)
	devices, _ := s.repo.Breakdown(ctx, merchantID, siteID, r.From, r.To, "os", false, 6)
	entries, _ := s.repo.Breakdown(ctx, merchantID, siteID, r.From, r.To, "entry", false, 6)
	reasons, _ := s.repo.ValidationReasons(ctx, merchantID, siteID, r.From, r.To)
	health, _ := s.repo.MerchantHealth(ctx, merchantID)
	alerts, _ := s.repo.AlertCounts(ctx, merchantID)

	out := map[string]any{
		"range":      map[string]any{"key": r.Key, "from": r.From.Format("2006-01-02"), "to": r.To.AddDate(0, 0, -1).Format("2006-01-02")},
		"updatedAt":  now.Format(layout),
		"summary":    sum,
		"trend":      trend,
		"trendDaily": daily,
		"channels":   channels, "codes": codes, "confidence": confidence, "inApp": inApp,
		"countries": countries, "devices": devices, "entries": entries, "validationReasons": reasons,
		"health": health,
		"alerts": alerts,
	}
	sum["conversions"] = convN
	out["conversionAmount"] = convAmt
	if m, err := s.merchants.FindByID(ctx, merchantID); err == nil {
		used := s.usage(ctx, merchantID)
		out["quota"] = map[string]any{"decisions": m.DecisionQuota, "used": used, "policy": m.QuotaPolicy, "expireAt": m.ExpireAt}
	}
	return out, nil
}

func (s *Service) usage(ctx context.Context, merchantID uint64) uint64 {
	if s.quota != nil {
		if n := s.quota.Usage(ctx, merchantID); n > 0 {
			return n
		}
	}
	n, _ := s.logs.Usage(ctx, merchantID, "decisions", s.now().Format("200601"))
	return n
}

func sumHourly(rows []ops.MetricsHourly) map[string]int64 {
	out := map[string]int64{"visits": 0, "realTotal": 0, "firstReal": 0, "returnReal": 0, "handoffReal": 0, "review": 0, "block": 0,
		"notFound": 0, "bypass": 0, "validationFailed": 0, "inApp": 0, "degraded": 0, "originErrors": 0, "latencySum": 0}
	for _, h := range rows {
		out["visits"] += int64(h.Visits)
		out["realTotal"] += int64(h.RealTotal)
		out["firstReal"] += int64(h.FirstReal)
		out["returnReal"] += int64(h.ReturnReal)
		out["handoffReal"] += int64(h.HandoffReal)
		out["review"] += int64(h.Review)
		out["block"] += int64(h.Block)
		out["notFound"] += int64(h.NotFound)
		out["bypass"] += int64(h.Bypass)
		out["validationFailed"] += int64(h.ValidationFailed)
		out["inApp"] += int64(h.InApp)
		out["degraded"] += int64(h.Degraded)
		out["originErrors"] += int64(h.OriginErrors)
		out["latencySum"] += int64(h.LatencySum)
	}
	return out
}

func bucketDaily(rows []ops.MetricsHourly) []map[string]any {
	var out []map[string]any
	var cur map[string]any
	curDay := ""
	for _, h := range rows {
		d := h.Hour.Format("01-02")
		if d != curDay {
			cur = map[string]any{"t": d, "visits": uint64(0), "real": uint64(0), "review": uint64(0), "first": uint64(0), "return": uint64(0), "handoff": uint64(0), "block": uint64(0)}
			out = append(out, cur)
			curDay = d
		}
		cur["visits"] = cur["visits"].(uint64) + h.Visits
		cur["real"] = cur["real"].(uint64) + h.RealTotal
		cur["review"] = cur["review"].(uint64) + h.Review
		cur["first"] = cur["first"].(uint64) + h.FirstReal
		cur["return"] = cur["return"].(uint64) + h.ReturnReal
		cur["handoff"] = cur["handoff"].(uint64) + h.HandoffReal
		cur["block"] = cur["block"].(uint64) + h.Block
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// applyLiveHour 用当前小时的实时日志替换聚合里同一小时的计数。
// 已聚合的当前小时先扣掉再加实时数，避免「聚合 + 全量实时」重复计算。
// 返回各展示字段的增量（实时 − 已聚合）；没有当前小时行时增量等于实时数。
func applyLiveHour(sum map[string]int64, hours []ops.MetricsHourly, cur time.Time, live map[string]int64) map[string]int64 {
	var agg ops.MetricsHourly
	for _, h := range hours {
		if sameHour(h.Hour, cur) {
			agg = h
			break
		}
	}
	delta := map[string]int64{
		"visits":           live["total"] - int64(agg.Visits),
		"real":             live["real"] - int64(agg.RealTotal),
		"review":           live["review"] - int64(agg.Review),
		"block":            live["block"] - int64(agg.Block),
		"bypass":           live["bypass"] - int64(agg.Bypass),
		"validationFailed": live["validationFailed"] - int64(agg.ValidationFailed),
		"degraded":         live["degraded"] - int64(agg.Degraded),
	}
	sum["visits"] += delta["visits"]
	sum["realTotal"] += delta["real"]
	sum["review"] += delta["review"]
	sum["block"] += delta["block"]
	sum["bypass"] += delta["bypass"]
	sum["validationFailed"] += delta["validationFailed"]
	sum["degraded"] += delta["degraded"]
	return delta
}

func sameHour(a, b time.Time) bool {
	return a.Truncate(time.Hour).Equal(b.Truncate(time.Hour))
}

// patchTrend 把当前小时的增量补进趋势点。按日聚合时加到当天，按小时时加到该小时；没有点则追加。
func patchTrend(trend []map[string]any, cur time.Time, daily bool, delta map[string]int64) []map[string]any {
	label := cur.Format("01-02 15:00")
	if daily {
		label = cur.Format("01-02")
	}
	keys := []string{"visits", "real", "review", "block"}
	for _, p := range trend {
		if p["t"] == label {
			for _, k := range keys {
				p[k] = num64(p[k]) + delta[k]
			}
			if v, ok := p["degraded"]; ok {
				p["degraded"] = num64(v) + delta["degraded"]
			}
			return trend
		}
	}
	if delta["visits"] == 0 && delta["real"] == 0 && delta["review"] == 0 && delta["block"] == 0 {
		return trend
	}
	return append(trend, map[string]any{
		"t": label, "visits": delta["visits"], "real": delta["real"], "review": delta["review"],
		"first": int64(0), "return": int64(0), "handoff": int64(0), "block": delta["block"],
	})
}

func num64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case uint64:
		return int64(x)
	case int:
		return int64(x)
	case float64:
		return int64(x)
	default:
		return 0
	}
}

// ---- 报表 ----

// ReportInput 报表参数。
type ReportInput struct {
	SiteID     uint64
	Range      Range
	CodeType   string
	Page, Size int
}

func (s *Service) ChannelReport(ctx context.Context, merchantID uint64, in ReportInput) ([]map[string]any, error) {
	return s.repo.ChannelReport(ctx, merchantID, in.SiteID, in.Range.From, in.Range.To)
}

func (s *Service) CodeReport(ctx context.Context, merchantID uint64, in ReportInput) ([]map[string]any, int64, error) {
	return s.repo.CodeReport(ctx, merchantID, in.SiteID, in.CodeType, in.Range.From, in.Range.To, in.Page, in.Size)
}

func (s *Service) PathReport(ctx context.Context, merchantID uint64, in ReportInput) ([]map[string]any, int64, error) {
	rows, total, err := s.repo.PathReport(ctx, merchantID, in.SiteID, in.Range.From, in.Range.To, in.Page, in.Size)
	if err != nil {
		return nil, 0, err
	}
	s.attachSiteNames(ctx, rows)
	return rows, total, nil
}

func (s *Service) ConversionReport(ctx context.Context, merchantID uint64, in ReportInput) ([]map[string]any, int64, error) {
	rows, total, err := s.repo.ConversionReport(ctx, merchantID, in.SiteID, in.Range.From, in.Range.To, in.Page, in.Size)
	if err != nil {
		return nil, 0, err
	}
	s.attachSiteNames(ctx, rows)
	return rows, total, nil
}

func (s *Service) attachSiteNames(ctx context.Context, rows []map[string]any) {
	ids := map[uint64]bool{}
	for _, r := range rows {
		if id := toU64(r["siteId"]); id > 0 {
			ids[id] = true
		}
	}
	if len(ids) == 0 {
		return
	}
	list := make([]uint64, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	names, err := s.sites.SiteNames(ctx, list)
	if err != nil {
		return
	}
	for _, r := range rows {
		r["siteName"] = names[toU64(r["siteId"])]
	}
}

func toU64(v any) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	case int64:
		return uint64(x)
	case int:
		return uint64(x)
	case uint32:
		return uint64(x)
	case int32:
		return uint64(x)
	case float64:
		return uint64(x)
	case []byte:
		var n uint64
		for _, c := range x {
			if c < '0' || c > '9' {
				return 0
			}
			n = n*10 + uint64(c-'0')
		}
		return n
	}
	return 0
}

// ---- 总后台看板 ----

// PlatformDashboard 全平台。
func (s *Service) PlatformDashboard(ctx context.Context, r Range, nodes []map[string]any) (map[string]any, error) {
	hours, err := s.repo.HourlyRange(ctx, 0, 0, r.From, r.To)
	if err != nil {
		return nil, err
	}
	sum := sumHourly(hours)
	curHour := s.now().Truncate(time.Hour)
	daily := r.To.Sub(r.From) > 48*time.Hour
	var delta map[string]int64
	if r.To.After(curHour) {
		if live, err := s.repo.LogCountSince(ctx, 0, 0, maxTime(curHour, r.From), r.To); err == nil {
			delta = applyLiveHour(sum, hours, curHour, live)
		}
	}
	counts, _ := s.repo.PlatformCounts(ctx)
	trend := bucketDaily(hours)
	if !daily {
		trend = trend[:0]
		for _, h := range hours {
			trend = append(trend, map[string]any{"t": h.Hour.Format("01-02 15:00"), "visits": h.Visits, "real": h.RealTotal, "review": h.Review, "block": h.Block,
				"degraded": h.Degraded, "originErrors": h.OriginErrors})
		}
	}
	if delta != nil {
		trend = patchTrend(trend, curHour, daily, delta)
	}
	entries, _ := s.repo.Breakdown(ctx, 0, 0, r.From, r.To, "entry", false, 6)
	alerts, _ := s.repo.AlertCounts(ctx, 0)
	unhealthy, _ := s.repo.UnhealthyOrigins(ctx)
	return map[string]any{
		"range":     map[string]any{"key": r.Key, "from": r.From.Format("2006-01-02"), "to": r.To.AddDate(0, 0, -1).Format("2006-01-02")},
		"updatedAt": s.now().Format(layout), "summary": sum, "counts": counts, "trend": trend, "entries": entries,
		"alerts": alerts, "unhealthyOrigins": unhealthy, "nodes": nodes,
	}, nil
}

// ---- 对账 ----

// Reconcile 比较某区间聚合与原始日志总数，返回差异。
func (s *Service) Reconcile(ctx context.Context, merchantID, siteID uint64, from, to time.Time) (map[string]any, error) {
	hours, err := s.repo.HourlyRange(ctx, merchantID, siteID, from, to)
	if err != nil {
		return nil, err
	}
	agg := sumHourly(hours)
	raw, err := s.repo.LogCountSince(ctx, merchantID, siteID, from, to)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"aggregated": map[string]int64{"visits": agg["visits"], "real": agg["realTotal"], "review": agg["review"], "block": agg["block"]},
		"raw":        map[string]int64{"visits": raw["total"], "real": raw["real"], "review": raw["review"], "block": raw["block"]},
		"match":      agg["visits"] == raw["total"] && agg["realTotal"] == raw["real"] && agg["review"] == raw["review"],
	}, nil
}

func jsonList(b []byte) []string {
	var out []string
	_ = json.Unmarshal(b, &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
