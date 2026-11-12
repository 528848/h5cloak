package opssvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/domain/ops"
	"cloak/internal/pkg/notifier"
)

// RuleDefault 每种告警类型的默认值与说明。
type RuleDefault struct {
	Type        string        `json:"type"`
	Name        string        `json:"name"`
	Desc        string        `json:"desc"`
	Severity    string        `json:"severity"`
	Window      int           `json:"windowMinutes"`
	Cooldown    int           `json:"cooldownMinutes"`
	Threshold   ops.Threshold `json:"threshold"`
	Fields      []string      `json:"fields"` // 可编辑的阈值字段
	Channels    []string      `json:"channels"`
	SiteScoped  bool          `json:"siteScoped"`
	AutoResolve bool          `json:"autoResolve"`
}

// RuleDefaults 默认规则（商户未自定义时按此评估）。
var RuleDefaults = []RuleDefault{
	{Type: ops.AlertOriginUnhealthy, Name: "源站不可达", Desc: "健康检查连续失败时触发，恢复后自动关闭", Severity: "critical", Window: 5, Cooldown: 30, Fields: nil, Channels: []string{"inapp", "email", "telegram", "webhook"}, SiteScoped: true, AutoResolve: true},
	{Type: ops.AlertCertExpiring, Name: "证书即将到期", Desc: "托管域名证书剩余天数低于阈值", Severity: "warning", Window: 60, Cooldown: 1440, Threshold: ops.Threshold{Days: 14}, Fields: []string{"days"}, Channels: []string{"inapp", "email", "webhook"}, SiteScoped: true, AutoResolve: true},
	{Type: ops.AlertDomainUnhealthy, Name: "域名解析异常", Desc: "域名健康检查失败（解析未指向网关或 HTTPS 不可用）", Severity: "critical", Window: 5, Cooldown: 60, Channels: []string{"inapp", "email", "telegram", "webhook"}, SiteScoped: true, AutoResolve: true},
	{Type: ops.AlertQuotaThreshold, Name: "额度阈值", Desc: "当月判定次数达到额度的百分比", Severity: "warning", Window: 15, Cooldown: 1440, Threshold: ops.Threshold{Pct: 80}, Fields: []string{"pct"}, Channels: []string{"inapp", "email", "webhook"}, SiteScoped: false, AutoResolve: false},
	{Type: ops.AlertReviewSpike, Name: "审核返回占比突增", Desc: "窗口内审核占比高于阈值且样本足够（排除放行为 0 的新站点）", Severity: "warning", Window: 15, Cooldown: 120, Threshold: ops.Threshold{Pct: 90, MinVisits: 100}, Fields: []string{"pct", "minVisits"}, Channels: []string{"inapp", "webhook"}, SiteScoped: true, AutoResolve: false},
	{Type: ops.AlertCodeAbnormal, Name: "分销码异常高频", Desc: "同一分销码在窗口内放行次数超过阈值", Severity: "warning", Window: 15, Cooldown: 120, Threshold: ops.Threshold{Count: 500}, Fields: []string{"count"}, Channels: []string{"inapp", "webhook"}, SiteScoped: true, AutoResolve: false},
	{Type: ops.AlertValidationSpike, Name: "校验失败突增", Desc: "窗口内参数校验失败次数超过阈值（可能是伪造参数扫描）", Severity: "info", Window: 15, Cooldown: 120, Threshold: ops.Threshold{Count: 200}, Fields: []string{"count"}, Channels: []string{"inapp"}, SiteScoped: true, AutoResolve: false},
	{Type: ops.AlertDegraded, Name: "判定降级率", Desc: "窗口内降级（快照缺失 / 存储异常）占比超过阈值", Severity: "critical", Window: 15, Cooldown: 60, Threshold: ops.Threshold{Pct: 5, MinVisits: 50}, Fields: []string{"pct", "minVisits"}, Channels: []string{"inapp", "email", "telegram", "webhook"}, SiteScoped: true, AutoResolve: false},
}

func defaultFor(typ string) *RuleDefault {
	for i := range RuleDefaults {
		if RuleDefaults[i].Type == typ {
			return &RuleDefaults[i]
		}
	}
	return nil
}

// ---- 规则 CRUD ----

// RuleView 规则输出。
type RuleView struct {
	ID              uint64        `json:"id"`
	SiteID          uint64        `json:"siteId"`
	Type            string        `json:"type"`
	Name            string        `json:"name"`
	Enabled         bool          `json:"enabled"`
	Severity        string        `json:"severity"`
	Threshold       ops.Threshold `json:"threshold"`
	WindowMinutes   int           `json:"windowMinutes"`
	CooldownMinutes int           `json:"cooldownMinutes"`
	Channels        []string      `json:"channels"`
	Custom          bool          `json:"custom"`
}

// Rules 返回商户规则：自定义行 + 未自定义类型的默认行。
func (s *Service) Rules(ctx context.Context, merchantID uint64) ([]RuleView, []RuleDefault, error) {
	rows, err := s.repo.ListRules(ctx, merchantID)
	if err != nil {
		return nil, nil, err
	}
	out := make([]RuleView, 0, len(RuleDefaults)+len(rows))
	seen := map[string]bool{}
	for _, r := range rows {
		v := ruleView(r)
		seen[strconv.FormatUint(r.SiteID, 10)+":"+r.Type] = true
		out = append(out, v)
	}
	for _, d := range RuleDefaults {
		if seen["0:"+d.Type] {
			continue
		}
		out = append(out, RuleView{Type: d.Type, Name: d.Name, Enabled: true, Severity: d.Severity, Threshold: d.Threshold,
			WindowMinutes: d.Window, CooldownMinutes: d.Cooldown, Channels: d.Channels})
	}
	return out, RuleDefaults, nil
}

func ruleView(r ops.AlertRule) RuleView {
	var th ops.Threshold
	_ = json.Unmarshal(r.Threshold, &th)
	name := r.Type
	if d := defaultFor(r.Type); d != nil {
		name = d.Name
	}
	return RuleView{ID: r.ID, SiteID: r.SiteID, Type: r.Type, Name: name, Enabled: r.Enabled, Severity: r.Severity, Threshold: th,
		WindowMinutes: r.WindowMinutes, CooldownMinutes: r.CooldownMinutes, Channels: jsonList(r.Channels), Custom: true}
}

// RuleInput 保存规则。
type RuleInput struct {
	SiteID          uint64
	Type            string
	Enabled         *bool
	Severity        string
	Threshold       ops.Threshold
	WindowMinutes   int
	CooldownMinutes int
	Channels        []string
}

func (s *Service) SaveRule(ctx context.Context, merchantID uint64, in RuleInput) (RuleView, error) {
	d := defaultFor(in.Type)
	if d == nil {
		return RuleView{}, errf(400, "未知告警类型")
	}
	if in.SiteID > 0 {
		site, err := s.sites.FindSite(ctx, in.SiteID)
		if err != nil || site.MerchantID != merchantID {
			return RuleView{}, errf(404, "站点不存在")
		}
	}
	sev := lower(in.Severity)
	if sev != "info" && sev != "warning" && sev != "critical" {
		sev = d.Severity
	}
	if in.WindowMinutes < 5 || in.WindowMinutes > 1440 {
		in.WindowMinutes = d.Window
	}
	if in.CooldownMinutes < 0 || in.CooldownMinutes > 10080 {
		in.CooldownMinutes = d.Cooldown
	}
	if in.Threshold.Pct < 0 || in.Threshold.Pct > 100 {
		in.Threshold.Pct = d.Threshold.Pct
	}
	if in.Threshold.Count < 0 {
		in.Threshold.Count = d.Threshold.Count
	}
	if in.Threshold.Days < 0 || in.Threshold.Days > 90 {
		in.Threshold.Days = d.Threshold.Days
	}
	if in.Threshold.MinVisits < 0 {
		in.Threshold.MinVisits = d.Threshold.MinVisits
	}
	chans := make([]string, 0, 4)
	for _, c := range in.Channels {
		c = lower(c)
		if c == ops.ChannelInApp || c == ops.ChannelEmail || c == ops.ChannelTelegram || c == ops.ChannelWebhook {
			chans = append(chans, c)
		}
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	th, _ := json.Marshal(in.Threshold)
	ch, _ := json.Marshal(chans)
	row := &ops.AlertRule{MerchantID: merchantID, SiteID: in.SiteID, Type: in.Type, Enabled: enabled, Severity: sev, Threshold: th,
		WindowMinutes: in.WindowMinutes, CooldownMinutes: in.CooldownMinutes, Channels: ch}
	if err := s.repo.UpsertRule(ctx, row); err != nil {
		return RuleView{}, err
	}
	rows, _ := s.repo.ListRules(ctx, merchantID)
	for _, r := range rows {
		if r.SiteID == in.SiteID && r.Type == in.Type {
			return ruleView(r), nil
		}
	}
	return ruleView(*row), nil
}

func (s *Service) DeleteRule(ctx context.Context, merchantID, id uint64) error {
	return s.repo.DeleteRule(ctx, merchantID, id)
}

// ---- 告警列表 ----

// AlertListInput 列表筛选。
type AlertListInput struct {
	SiteID     uint64
	Type       string
	Status     string
	Severity   string
	Page, Size int
}

func (s *Service) Alerts(ctx context.Context, merchantID uint64, in AlertListInput) ([]map[string]any, int64, error) {
	rows, total, err := s.repo.ListAlerts(ctx, mysql.AlertFilter{MerchantID: merchantID, SiteID: in.SiteID, Type: in.Type, Status: in.Status, Severity: in.Severity, Page: in.Page, Size: in.Size})
	if err != nil {
		return nil, 0, err
	}
	ids := map[uint64]bool{}
	for _, a := range rows {
		if a.SiteID > 0 {
			ids[a.SiteID] = true
		}
	}
	names := map[uint64]string{}
	if len(ids) > 0 {
		list := make([]uint64, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		names, _ = s.sites.SiteNames(ctx, list)
	}
	out := make([]map[string]any, 0, len(rows))
	for _, a := range rows {
		out = append(out, alertView(a, names[a.SiteID]))
	}
	return out, total, nil
}

func alertView(a ops.Alert, siteName string) map[string]any {
	var detail map[string]any
	_ = json.Unmarshal(a.Detail, &detail)
	var notified map[string]string
	_ = json.Unmarshal(a.Notified, &notified)
	name := a.Type
	if d := defaultFor(a.Type); d != nil {
		name = d.Name
	}
	v := map[string]any{
		"id": a.ID, "merchantId": a.MerchantID, "siteId": a.SiteID, "siteName": siteName, "type": a.Type, "typeName": name, "severity": a.Severity,
		"targetKey": a.TargetKey, "title": a.Title, "message": a.Message, "detail": detail, "status": a.Status,
		"firedAt": a.FiredAt.Format(layout), "lastSeenAt": a.LastSeenAt.Format(layout), "notified": notified,
	}
	if a.ResolvedAt != nil {
		v["resolvedAt"] = a.ResolvedAt.Format(layout)
	}
	if a.AckAt != nil {
		v["ackAt"] = a.AckAt.Format(layout)
	}
	return v
}

func (s *Service) AlertCounts(ctx context.Context, merchantID uint64) (map[string]int64, error) {
	return s.repo.AlertCounts(ctx, merchantID)
}

// ResolveAlert 手动关闭。
func (s *Service) ResolveAlert(ctx context.Context, merchantID, id, userID uint64) error {
	a, err := s.repo.FindAlert(ctx, merchantID, id)
	if err != nil {
		return errf(404, "告警不存在")
	}
	if a.Status == ops.AlertResolved {
		return nil
	}
	now := s.now()
	a.Status = ops.AlertResolved
	a.ResolvedAt = &now
	a.AckUserID = &userID
	a.AckAt = &now
	return s.repo.UpdateAlert(ctx, a, "Status", "ResolvedAt", "AckUserID", "AckAt")
}

// ---- 评估 ----

// candidate 一次评估产生的触发。
type candidate struct {
	merchantID uint64
	siteID     uint64
	typ        string
	target     string
	title      string
	message    string
	detail     map[string]any
}

// effectiveRule 取商户 + 站点最匹配的规则（站点行 > 商户全站行 > 默认）。
func effectiveRule(rules []ops.AlertRule, merchantID, siteID uint64, typ string) (RuleView, bool) {
	var site, all *ops.AlertRule
	for i := range rules {
		r := &rules[i]
		if r.MerchantID != merchantID || r.Type != typ {
			continue
		}
		if r.SiteID == siteID && siteID > 0 {
			site = r
		} else if r.SiteID == 0 {
			all = r
		}
	}
	if site != nil {
		return ruleView(*site), site.Enabled
	}
	if all != nil {
		return ruleView(*all), all.Enabled
	}
	d := defaultFor(typ)
	if d == nil {
		return RuleView{}, false
	}
	return RuleView{Type: typ, Name: d.Name, Enabled: true, Severity: d.Severity, Threshold: d.Threshold, WindowMinutes: d.Window, CooldownMinutes: d.Cooldown, Channels: d.Channels}, true
}

// Evaluate 执行一轮告警评估：生成/更新/恢复告警并发通知。供 worker 周期调用。
func (s *Service) Evaluate(ctx context.Context) (fired, resolved int) {
	now := s.now()
	rules, err := s.repo.ListRules(ctx, 0)
	if err != nil {
		s.log.Warn("alert rules", zap.Error(err))
		return
	}
	var cands []candidate

	// 1. 源站不可达
	if rows, err := s.repo.UnhealthyOrigins(ctx); err == nil {
		for _, o := range rows {
			id := toU64(o["id"])
			cands = append(cands, candidate{
				merchantID: toU64(o["merchantId"]), siteID: toU64(o["siteId"]), typ: ops.AlertOriginUnhealthy, target: "origin:" + strconv.FormatUint(id, 10),
				title:   fmt.Sprintf("源站不可达：%s（%s）", str(o["siteName"]), kindName(str(o["kind"]))),
				message: fmt.Sprintf("%s 健康检查失败：%s", str(o["url"]), str(o["message"])),
				detail:  map[string]any{"originId": id, "kind": o["kind"], "url": o["url"], "message": o["message"]},
			})
		}
	}
	// 2. 域名异常
	if rows, err := s.repo.UnhealthyDomains(ctx); err == nil {
		for _, d := range rows {
			cands = append(cands, candidate{
				merchantID: toU64(d["merchantId"]), siteID: toU64(d["siteId"]), typ: ops.AlertDomainUnhealthy, target: "domain:" + str(d["domain"]),
				title: "域名异常：" + str(d["domain"]), message: str(d["message"]),
				detail: map[string]any{"domain": d["domain"], "message": d["message"]},
			})
		}
	}
	// 3. 证书到期（按每商户阈值分别查，取最大天数一次查出再过滤）
	if rows, err := s.repo.ExpiringCerts(ctx, now.AddDate(0, 0, 90)); err == nil {
		for _, d := range rows {
			mid, sid := toU64(d["merchantId"]), toU64(d["siteId"])
			rv, on := effectiveRule(rules, mid, sid, ops.AlertCertExpiring)
			if !on {
				continue
			}
			exp, ok := d["certExpiresAt"].(time.Time)
			if !ok {
				continue
			}
			days := int(exp.Sub(now).Hours() / 24)
			if days > rv.Threshold.Days {
				continue
			}
			cands = append(cands, candidate{
				merchantID: mid, siteID: sid, typ: ops.AlertCertExpiring, target: "cert:" + str(d["domain"]),
				title: fmt.Sprintf("证书 %d 天后到期：%s", days, str(d["domain"])), message: "到期时间 " + exp.Format(layout),
				detail: map[string]any{"domain": d["domain"], "expiresAt": exp.Format(layout), "days": days},
			})
		}
	}
	// 4. 额度阈值
	if rows, err := s.repo.MerchantQuotas(ctx); err == nil {
		month := now.Format("200601")
		for _, m := range rows {
			if m.DecisionQuota == 0 {
				continue
			}
			rv, on := effectiveRule(rules, m.ID, 0, ops.AlertQuotaThreshold)
			if !on {
				continue
			}
			used := s.usage(ctx, m.ID)
			pct := float64(used) * 100 / float64(m.DecisionQuota)
			if pct < rv.Threshold.Pct {
				continue
			}
			cands = append(cands, candidate{
				merchantID: m.ID, typ: ops.AlertQuotaThreshold, target: "quota:" + month,
				title:   fmt.Sprintf("本月判定额度已用 %.0f%%", pct),
				message: fmt.Sprintf("已用 %d / %d，达到阈值 %.0f%%", used, m.DecisionQuota, rv.Threshold.Pct),
				detail:  map[string]any{"used": used, "quota": m.DecisionQuota, "pct": pct, "month": month},
			})
		}
	}
	// 5–8. 流量类：按活跃站点逐个窗口统计
	if sites, err := s.repo.ActiveSites(ctx, now.Add(-24*time.Hour)); err == nil {
		for _, st := range sites {
			cands = append(cands, s.trafficCandidates(ctx, rules, st.MerchantID, st.SiteID, now)...)
		}
	}

	// 写入 / 更新
	active := map[string]bool{}
	for _, c := range cands {
		rv, on := effectiveRule(rules, c.merchantID, c.siteID, c.typ)
		if !on {
			continue
		}
		key := strconv.FormatUint(c.merchantID, 10) + "|" + c.typ + "|" + c.target
		active[key] = true
		open, err := s.repo.OpenAlert(ctx, c.merchantID, c.typ, c.target)
		if err != nil {
			continue
		}
		if open != nil {
			open.LastSeenAt = now
			open.Message = c.message
			open.Detail, _ = json.Marshal(c.detail)
			_ = s.repo.UpdateAlert(ctx, open, "LastSeenAt", "Message", "Detail")
			continue
		}
		if last, _ := s.repo.LastAlertAt(ctx, c.merchantID, c.typ, c.target); last != nil && now.Sub(*last) < time.Duration(rv.CooldownMinutes)*time.Minute {
			continue
		}
		detail, _ := json.Marshal(c.detail)
		a := &ops.Alert{MerchantID: c.merchantID, SiteID: c.siteID, RuleID: rv.ID, Type: c.typ, Severity: rv.Severity, TargetKey: c.target,
			Title: c.title, Message: c.message, Detail: detail, Status: ops.AlertOpen, FiredAt: now, LastSeenAt: now}
		if err := s.repo.CreateAlert(ctx, a); err != nil {
			s.log.Warn("create alert", zap.Error(err))
			continue
		}
		fired++
		s.notify(ctx, a, rv.Channels, false)
	}

	// 自动恢复：open 的可恢复类型，目标不再在 active 中
	open, _, err := s.repo.ListAlerts(ctx, mysql.AlertFilter{Status: ops.AlertOpen, Page: 1, Size: 1000})
	if err == nil {
		for i := range open {
			a := open[i]
			d := defaultFor(a.Type)
			if d == nil || !d.AutoResolve {
				continue
			}
			if active[strconv.FormatUint(a.MerchantID, 10)+"|"+a.Type+"|"+a.TargetKey] {
				continue
			}
			t := now
			a.Status = ops.AlertResolved
			a.ResolvedAt = &t
			if err := s.repo.UpdateAlert(ctx, &a, "Status", "ResolvedAt"); err == nil {
				resolved++
				rv, _ := effectiveRule(rules, a.MerchantID, a.SiteID, a.Type)
				s.notify(ctx, &a, rv.Channels, true)
			}
		}
	}
	return
}

func (s *Service) trafficCandidates(ctx context.Context, rules []ops.AlertRule, mid, sid uint64, now time.Time) []candidate {
	var out []candidate
	// 审核占比 / 降级率 / 校验失败：共用一次窗口计数（窗口取各规则中的最大）
	rvReview, onReview := effectiveRule(rules, mid, sid, ops.AlertReviewSpike)
	rvDeg, onDeg := effectiveRule(rules, mid, sid, ops.AlertDegraded)
	rvVal, onVal := effectiveRule(rules, mid, sid, ops.AlertValidationSpike)
	rvCode, onCode := effectiveRule(rules, mid, sid, ops.AlertCodeAbnormal)
	if !onReview && !onDeg && !onVal && !onCode {
		return nil
	}
	windows := map[int]map[string]int64{}
	count := func(min int) map[string]int64 {
		if m, ok := windows[min]; ok {
			return m
		}
		m, err := s.repo.LogCountSince(ctx, mid, sid, now.Add(-time.Duration(min)*time.Minute), now)
		if err != nil {
			m = map[string]int64{}
		}
		windows[min] = m
		return m
	}
	siteKey := "site:" + strconv.FormatUint(sid, 10)
	if onReview {
		m := count(rvReview.WindowMinutes)
		base := m["total"] - m["bypass"]
		if base >= rvReview.Threshold.MinVisits && base > 0 {
			pct := float64(m["review"]) * 100 / float64(base)
			if pct >= rvReview.Threshold.Pct {
				out = append(out, candidate{merchantID: mid, siteID: sid, typ: ops.AlertReviewSpike, target: siteKey,
					title:   fmt.Sprintf("审核返回占比 %.0f%%", pct),
					message: fmt.Sprintf("近 %d 分钟 %d 次访问中 %d 次返回审核内容", rvReview.WindowMinutes, base, m["review"]),
					detail:  map[string]any{"pct": pct, "visits": base, "review": m["review"], "windowMinutes": rvReview.WindowMinutes}})
			}
		}
	}
	if onDeg {
		m := count(rvDeg.WindowMinutes)
		if m["total"] >= rvDeg.Threshold.MinVisits && m["total"] > 0 {
			pct := float64(m["degraded"]) * 100 / float64(m["total"])
			if pct >= rvDeg.Threshold.Pct {
				out = append(out, candidate{merchantID: mid, siteID: sid, typ: ops.AlertDegraded, target: siteKey,
					title:   fmt.Sprintf("判定降级率 %.1f%%", pct),
					message: fmt.Sprintf("近 %d 分钟 %d 次判定中 %d 次降级", rvDeg.WindowMinutes, m["total"], m["degraded"]),
					detail:  map[string]any{"pct": pct, "visits": m["total"], "degraded": m["degraded"], "windowMinutes": rvDeg.WindowMinutes}})
			}
		}
	}
	if onVal {
		m := count(rvVal.WindowMinutes)
		if m["validationFailed"] >= rvVal.Threshold.Count && rvVal.Threshold.Count > 0 {
			out = append(out, candidate{merchantID: mid, siteID: sid, typ: ops.AlertValidationSpike, target: siteKey,
				title:   fmt.Sprintf("校验失败 %d 次", m["validationFailed"]),
				message: fmt.Sprintf("近 %d 分钟参数校验失败 %d 次，阈值 %d", rvVal.WindowMinutes, m["validationFailed"], rvVal.Threshold.Count),
				detail:  map[string]any{"failed": m["validationFailed"], "windowMinutes": rvVal.WindowMinutes}})
		}
	}
	if onCode && rvCode.Threshold.Count > 0 {
		if rows, err := s.repo.CodeHits(ctx, mid, now.Add(-time.Duration(rvCode.WindowMinutes)*time.Minute), now, rvCode.Threshold.Count); err == nil {
			for _, r := range rows {
				if r.SiteID != sid {
					continue
				}
				out = append(out, candidate{merchantID: mid, siteID: sid, typ: ops.AlertCodeAbnormal, target: "code:" + strconv.FormatUint(r.CodeID, 10),
					title:   fmt.Sprintf("分销码 %s 短时高频：%d 次", r.Code, r.N),
					message: fmt.Sprintf("近 %d 分钟放行 %d 次，阈值 %d", rvCode.WindowMinutes, r.N, rvCode.Threshold.Count),
					detail:  map[string]any{"code": r.Code, "codeId": r.CodeID, "count": r.N, "windowMinutes": rvCode.WindowMinutes}})
			}
		}
	}
	return out
}

func kindName(k string) string {
	if k == "review" {
		return "审核源站"
	}
	return "真实源站"
}

func str(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case *string:
		if x != nil {
			return *x
		}
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

// ---- 通知 ----

// notify 按渠道分发；结果写回 alert.notified。
func (s *Service) notify(ctx context.Context, a *ops.Alert, channels []string, resolvedEvent bool) {
	results := map[string]string{}
	if len(a.Notified) > 0 {
		_ = json.Unmarshal(a.Notified, &results)
	}
	settings, _ := s.repo.GetNotifySettings(ctx, a.MerchantID)
	if settings == nil {
		settings = &ops.NotifySettings{MerchantID: a.MerchantID}
	}
	// 商户级禁用的渠道不发
	allowed := map[string]bool{}
	if mc := jsonList(settings.Channels); len(mc) > 0 {
		for _, c := range mc {
			allowed[c] = true
		}
	} else {
		for _, c := range []string{ops.ChannelInApp, ops.ChannelEmail, ops.ChannelTelegram, ops.ChannelWebhook} {
			allowed[c] = true
		}
	}
	suffix := ""
	if resolvedEvent {
		suffix = ":resolved"
	}
	title := a.Title
	kind := "alert"
	if resolvedEvent {
		title = "已恢复：" + a.Title
		kind = "alert_resolved"
	}
	link := strings.TrimRight(s.cfg.ConsoleURL, "/") + "/merchant/alert/list?id=" + strconv.FormatUint(a.ID, 10)
	for _, ch := range channels {
		if !allowed[ch] {
			results[ch+suffix] = "skipped:disabled"
			continue
		}
		var err error
		switch ch {
		case ops.ChannelInApp:
			refType := "alert"
			refID := a.ID
			sev := a.Severity
			if resolvedEvent {
				sev = "info"
			}
			err = s.repo.CreateNotification(ctx, &ops.Notification{MerchantID: a.MerchantID, Kind: kind, Severity: sev, Title: title,
				Content: a.Message, Link: &link, RefType: &refType, RefID: &refID, CreatedAt: s.now()})
		case ops.ChannelEmail:
			if !s.mailer.Configured() {
				results[ch+suffix] = "skipped:not_configured"
				continue
			}
			emails := jsonList(settings.Emails)
			if len(emails) == 0 {
				results[ch+suffix] = "skipped:no_recipients"
				continue
			}
			text := fmt.Sprintf("%s\n\n%s\n\n级别：%s\n时间：%s\n查看：%s", title, a.Message, a.Severity, a.FiredAt.Format(layout), link)
			html := fmt.Sprintf(`<h3>%s</h3><p>%s</p><p>级别：%s<br>时间：%s</p><p><a href="%s">查看详情</a></p>`, esc(title), esc(a.Message), a.Severity, a.FiredAt.Format(layout), link)
			err = s.mailer.Send(ctx, emails, "[斗篷告警] "+title, text, html)
		case ops.ChannelTelegram:
			icon := "⚠️"
			if a.Severity == "critical" {
				icon = "🔴"
			}
			if resolvedEvent {
				icon = "✅"
			}
			text := fmt.Sprintf("%s <b>%s</b>\n%s\n<a href=\"%s\">查看详情</a>", icon, esc(title), esc(a.Message), link)
			// 商户配了自己的机器人：从自有 Bot 发给所有绑定会话；否则回退平台 Bot + Chat ID
			if handled, status, e := s.sendMerchantTelegram(ctx, a.MerchantID, text); handled {
				if status != "" {
					results[ch+suffix] = status
					continue
				}
				err = e
			} else if !s.telegram.Configured() {
				results[ch+suffix] = "skipped:not_configured"
				continue
			} else if settings.TelegramChatID == "" {
				results[ch+suffix] = "skipped:no_chat_id"
				continue
			} else {
				err = s.telegram.Send(ctx, settings.TelegramChatID, text)
			}
		case ops.ChannelWebhook:
			event := a.Type
			if resolvedEvent {
				event = a.Type + ".resolved"
			}
			n, e := s.EnqueueWebhook(ctx, a.MerchantID, event, "alert-"+strconv.FormatUint(a.ID, 10)+suffix, map[string]any{
				"alertId": a.ID, "siteId": a.SiteID, "type": a.Type, "severity": a.Severity, "status": a.Status, "title": a.Title,
				"message": a.Message, "detail": json.RawMessage(nonEmpty(a.Detail)), "firedAt": a.FiredAt.UTC().Format(time.RFC3339),
			})
			if e != nil {
				err = e
			} else if n == 0 {
				results[ch+suffix] = "skipped:no_endpoint"
				continue
			}
		default:
			continue
		}
		if err != nil {
			results[ch+suffix] = "error:" + trunc(err.Error(), 120)
			s.log.Warn("notify", zap.String("channel", ch), zap.Uint64("alert", a.ID), zap.Error(err))
		} else {
			results[ch+suffix] = "ok"
		}
	}
	a.Notified, _ = json.Marshal(results)
	_ = s.repo.UpdateAlert(ctx, a, "Notified")
}

// sendMerchantTelegram 用商户自有 Bot 向其全部启用会话发送。
// 返回 handled=true 表示该商户配了自有机器人（无论发送成功与否，都不再回退平台 Bot）。
// status 非空时表示跳过原因（如没有绑定会话）。
func (s *Service) sendMerchantTelegram(ctx context.Context, merchantID uint64, html string) (handled bool, status string, err error) {
	if s.box == nil {
		return false, "", nil
	}
	bot, e := s.repo.GetTelegramBot(ctx, merchantID)
	if e != nil || bot == nil || bot.Status != ops.TelegramBotActive || len(bot.BotTokenEnc) == 0 {
		return false, "", nil
	}
	tok, e := s.box.Open(bot.BotTokenEnc)
	if e != nil {
		return true, "", e
	}
	chats, _ := s.repo.ActiveTelegramChats(ctx, merchantID)
	if len(chats) == 0 {
		return true, "skipped:no_chat", nil
	}
	tg := notifier.NewTelegramWithToken(s.cfg.Telegram.APIBase, string(tok))
	var firstErr error
	for _, c := range chats {
		if e := tg.Send(ctx, c.ChatID, html); e != nil && firstErr == nil {
			firstErr = e
		}
	}
	return true, "", firstErr
}

func nonEmpty(b []byte) []byte {
	if len(b) == 0 {
		return []byte("{}")
	}
	return b
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ---- 通知设置与站内通知 ----

// NotifySettingsView 输出。
type NotifySettingsView struct {
	Emails             []string `json:"emails"`
	TelegramChatID     string   `json:"telegramChatId"`
	Channels           []string `json:"channels"`
	EmailConfigured    bool     `json:"emailConfigured"`
	TelegramConfigured bool     `json:"telegramConfigured"`
}

func (s *Service) NotifySettings(ctx context.Context, merchantID uint64) (NotifySettingsView, error) {
	row, err := s.repo.GetNotifySettings(ctx, merchantID)
	if err != nil {
		return NotifySettingsView{}, err
	}
	chans := jsonList(row.Channels)
	if len(chans) == 0 {
		chans = []string{ops.ChannelInApp, ops.ChannelEmail, ops.ChannelTelegram, ops.ChannelWebhook}
	}
	return NotifySettingsView{Emails: jsonList(row.Emails), TelegramChatID: row.TelegramChatID, Channels: chans,
		EmailConfigured: s.mailer.Configured(), TelegramConfigured: s.telegram.Configured()}, nil
}

func (s *Service) SaveNotifySettings(ctx context.Context, merchantID uint64, emails []string, chatID string, channels []string) error {
	clean := make([]string, 0, len(emails))
	for _, e := range emails {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.Contains(e, "@") || len(e) > 255 {
			return errf(400, "邮箱格式不正确："+e)
		}
		clean = append(clean, e)
	}
	if len(clean) > 10 {
		return errf(400, "最多 10 个邮箱")
	}
	chs := make([]string, 0, 4)
	for _, c := range channels {
		c = lower(c)
		if c == ops.ChannelInApp || c == ops.ChannelEmail || c == ops.ChannelTelegram || c == ops.ChannelWebhook {
			chs = append(chs, c)
		}
	}
	em, _ := json.Marshal(clean)
	cj, _ := json.Marshal(chs)
	return s.repo.SaveNotifySettings(ctx, &ops.NotifySettings{MerchantID: merchantID, Emails: em, TelegramChatID: strings.TrimSpace(chatID), Channels: cj})
}

// TestChannel 发送测试通知。
func (s *Service) TestChannel(ctx context.Context, merchantID uint64, channel string) error {
	settings, err := s.repo.GetNotifySettings(ctx, merchantID)
	if err != nil {
		return err
	}
	msg := "这是一条来自斗篷系统的测试通知，发送时间 " + s.now().Format(layout)
	switch channel {
	case ops.ChannelEmail:
		if !s.mailer.Configured() {
			return errf(400, "平台尚未配置 SMTP")
		}
		emails := jsonList(settings.Emails)
		if len(emails) == 0 {
			return errf(400, "请先保存接收邮箱")
		}
		return s.mailer.Send(ctx, emails, "[斗篷] 测试通知", msg, "<p>"+msg+"</p>")
	case ops.ChannelTelegram:
		if handled, status, e := s.sendMerchantTelegram(ctx, merchantID, "✅ "+msg); handled {
			if status == "skipped:no_chat" {
				return errf(400, "请先在 Telegram 配置页绑定接收会话")
			}
			return e
		}
		if !s.telegram.Configured() {
			return errf(400, "平台尚未配置 Telegram Bot，或请改用你自己的机器人")
		}
		if settings.TelegramChatID == "" {
			return errf(400, "请先保存 Chat ID，或配置你自己的机器人")
		}
		return s.telegram.Send(ctx, settings.TelegramChatID, "✅ "+msg)
	case ops.ChannelInApp:
		return s.repo.CreateNotification(ctx, &ops.Notification{MerchantID: merchantID, Kind: "system", Severity: "info", Title: "测试通知", Content: msg, CreatedAt: s.now()})
	}
	return errf(400, "不支持的渠道")
}

func (s *Service) Notifications(ctx context.Context, merchantID uint64, unreadOnly bool, page, size int) ([]ops.Notification, int64, error) {
	return s.repo.ListNotifications(ctx, merchantID, unreadOnly, page, size)
}

func (s *Service) UnreadCount(ctx context.Context, merchantID uint64) (int64, error) {
	return s.repo.UnreadCount(ctx, merchantID)
}

func (s *Service) MarkRead(ctx context.Context, merchantID uint64, ids []uint64) error {
	return s.repo.MarkRead(ctx, merchantID, ids)
}
