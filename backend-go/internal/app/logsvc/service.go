// Package logsvc 决策日志查询：列表、链路详情、统计。
package logsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/domain/decisionlog"
)

type Service struct {
	repo  *mysql.LogRepo
	sites *mysql.SiteRepo
}

func New(repo *mysql.LogRepo, sites *mysql.SiteRepo) *Service {
	return &Service{repo: repo, sites: sites}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

const layout = "2006-01-02 15:04:05"

// ListInput 列表筛选。
type ListInput struct {
	SiteID    uint64
	From, To  string
	Path      string
	TouchType string
	Content   string
	Channel   string
	Code      string
	Reason    string
	Country   string
	InApp     string
	VisitorID string
	RequestID string
	Page      int
	Size      int
}

func (s *Service) List(ctx context.Context, merchantID uint64, in ListInput) ([]map[string]any, int64, error) {
	f := mysql.LogFilter{
		MerchantID: merchantID, SiteID: in.SiteID, Path: in.Path, TouchType: in.TouchType, Content: in.Content,
		Channel: in.Channel, Code: in.Code, Reason: in.Reason, Country: strings.ToUpper(in.Country), InApp: in.InApp,
		VisitorID: in.VisitorID, RequestID: in.RequestID, Page: in.Page, Size: in.Size,
	}
	var err error
	if f.From, err = parseTime(in.From); err != nil {
		return nil, 0, errf(400, "开始时间格式无效")
	}
	if f.To, err = parseTime(in.To); err != nil {
		return nil, 0, errf(400, "结束时间格式无效")
	}
	if f.From == nil && f.To == nil {
		t := time.Now().AddDate(0, 0, -7)
		f.From = &t
	}
	rows, total, err := s.repo.ListDecisions(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(rows))
	mids := make([]uint64, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].SiteID)
		mids = append(mids, rows[i].MerchantID)
	}
	siteNames, _ := s.sites.SiteNames(ctx, ids)
	merchantNames := map[uint64]string{}
	if merchantID == 0 {
		merchantNames, _ = s.sites.MerchantNames(ctx, mids)
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		row := brief(&rows[i])
		row["siteName"] = siteNames[rows[i].SiteID]
		if merchantID == 0 {
			row["merchantName"] = merchantNames[rows[i].MerchantID]
		}
		out = append(out, row)
	}
	return out, total, nil
}

// Detail 单条链路详情，附归因与访客信息。
func (s *Service) Detail(ctx context.Context, merchantID uint64, requestID string) (map[string]any, error) {
	rec, err := s.repo.FindDecision(ctx, requestID)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "日志不存在或已过期")
		}
		return nil, err
	}
	if merchantID > 0 && rec.MerchantID != merchantID {
		return nil, errf(404, "日志不存在或已过期")
	}
	row := brief(rec)
	var trace any
	_ = json.Unmarshal(rec.Trace, &trace)
	row["trace"] = trace
	var ve any
	_ = json.Unmarshal(rec.ValidationErrors, &ve)
	row["validationErrors"] = ve
	row["uaSummary"] = rec.UASummary
	row["refererSummary"] = rec.RefererSummary
	row["ipPrefix"] = rec.IPPrefix
	row["ipHash"] = rec.IPHash
	row["attributionId"] = rec.AttributionID
	row["originStatus"] = rec.OriginStatus
	row["fallback"] = rec.Fallback
	row["degraded"] = rec.Degraded
	if names, err := s.sites.SiteNames(ctx, []uint64{rec.SiteID}); err == nil {
		row["siteName"] = names[rec.SiteID]
	}
	if rec.VisitorID != "" {
		if a, err := s.repo.FindAttribution(ctx, rec.SiteID, rec.VisitorID); err == nil {
			var fp any
			_ = json.Unmarshal(a.FirstParams, &fp)
			row["attribution"] = map[string]any{
				"firstChannel": a.FirstChannel, "firstCodeId": a.FirstCodeID, "firstParams": fp,
				"firstLandingPath": a.FirstLandingPath, "firstRequestId": a.FirstRequestID, "firstTouchAt": a.FirstTouchAt.Format(layout),
				"lastChannel": a.LastChannel, "lastCodeId": a.LastCodeID, "lastTouchAt": a.LastTouchAt.Format(layout),
				"touchCount": a.TouchCount, "handoffCount": a.HandoffCount, "confidence": a.Confidence,
				"expiresAt": fmtTime(a.ExpiresAt),
			}
		}
		if v, err := s.repo.FindVisitor(ctx, rec.SiteID, rec.VisitorID); err == nil {
			row["visitor"] = map[string]any{
				"firstSeenAt": v.FirstSeenAt.Format(layout), "lastSeenAt": v.LastSeenAt.Format(layout), "visitCount": v.VisitCount,
				"firstCountry": v.FirstCountry, "firstDeviceFamily": v.FirstDeviceFamily, "firstInApp": v.FirstInApp,
			}
		}
	}
	return row, nil
}

// Stats 最近 24 小时统计。
func (s *Service) Stats(ctx context.Context, merchantID, siteID uint64) (map[string]any, error) {
	m, err := s.repo.LogStats(ctx, merchantID, siteID, time.Now().Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"total": m["total"], "real": m["content:real"], "review": m["content:review"],
		"block": m["content:block"] + m["content:not_found"], "bypass": m["content:bypass"],
		"first": m["touch:first"], "return": m["touch:return"], "handoff": m["touch:handoff"], "none": m["touch:none"],
	}, nil
}

// Usage 商户当月判定用量（MySQL 计量）。
func (s *Service) Usage(ctx context.Context, merchantID uint64) (uint64, error) {
	return s.repo.Usage(ctx, merchantID, decisionlog.MetricDecisions, time.Now().Format("200601"))
}

func brief(r *decisionlog.Record) map[string]any {
	var mp []string
	_ = json.Unmarshal(r.MatchedParams, &mp)
	if mp == nil {
		mp = []string{}
	}
	var ve []map[string]any
	_ = json.Unmarshal(r.ValidationErrors, &ve)
	reasons := make([]string, 0, len(ve))
	for _, e := range ve {
		if s, ok := e["reason"].(string); ok {
			reasons = append(reasons, s)
		}
	}
	return map[string]any{
		"requestId": r.RequestID, "createdAt": r.CreatedAt.Format("2006-01-02 15:04:05.000"), "merchantId": r.MerchantID, "siteId": r.SiteID,
		"entry": r.Entry, "method": r.Method, "host": r.Host, "path": r.Path, "touchType": r.TouchType, "content": r.Content,
		"attributed": r.Attributed, "confidence": r.Confidence, "inAppBrowser": r.InAppBrowser, "deviceFamily": r.DeviceFamily,
		"handoffKind": r.HandoffKind, "channel": r.Channel, "codeId": r.CodeID, "code": r.Code, "matchedParams": mp,
		"errorReasons": reasons, "visitorId": r.VisitorID, "attributionExpiresAt": fmtTime(r.AttributionExpiresAt),
		"country": r.Country, "os": r.OS, "pathRule": r.PathRule, "originKind": r.OriginKind, "originStatus": r.OriginStatus,
		"fallback": r.Fallback, "latencyMs": r.LatencyMs, "degraded": r.Degraded,
	}
}

func parseTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	for _, l := range []string{layout, "2006-01-02 15:04", "2006-01-02", time.RFC3339} {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("bad time")
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(layout)
}
