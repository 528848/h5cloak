package opssvc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/domain/ops"
)

// ExportKinds 支持的导出类型。
var ExportKinds = map[string]string{
	"decision_logs": "决策日志", "report_channel": "渠道报表", "report_code": "分销码报表", "report_campaign": "活动码报表",
	"report_path": "路径报表", "report_conversion": "转化报表",
}

// ExportParams 导出参数。
type ExportParams struct {
	SiteID   uint64 `json:"siteId"`
	RangeKey string `json:"rangeKey"`
	From     string `json:"from"`
	To       string `json:"to"`
	// 决策日志筛选
	Content   string `json:"content,omitempty"`
	TouchType string `json:"touchType,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Code      string `json:"code,omitempty"`
	Path      string `json:"path,omitempty"`
}

// CreateExport 新建异步导出任务。
func (s *Service) CreateExport(ctx context.Context, merchantID, userID uint64, kind string, p ExportParams) (map[string]any, error) {
	if _, ok := ExportKinds[kind]; !ok {
		return nil, errf(400, "不支持的导出类型")
	}
	pj, _ := json.Marshal(p)
	row := &ops.Export{MerchantID: merchantID, UserID: userID, Kind: kind, Params: pj, Status: ops.ExportPending, CreatedAt: s.now()}
	if err := s.repo.CreateExport(ctx, row); err != nil {
		return nil, err
	}
	return exportView(*row), nil
}

func (s *Service) Exports(ctx context.Context, merchantID uint64, page, size int) ([]map[string]any, int64, error) {
	rows, total, err := s.repo.ListExports(ctx, merchantID, page, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, exportView(r))
	}
	return out, total, nil
}

func exportView(e ops.Export) map[string]any {
	v := map[string]any{"id": e.ID, "kind": e.Kind, "kindName": ExportKinds[e.Kind], "status": e.Status, "fileSize": e.FileSize,
		"rowCount": e.RowCount, "error": e.Error, "createdAt": e.CreatedAt.Format(layout)}
	var p ExportParams
	_ = json.Unmarshal(e.Params, &p)
	v["params"] = p
	if e.FinishedAt != nil {
		v["finishedAt"] = e.FinishedAt.Format(layout)
	}
	if e.ExpiresAt != nil {
		v["expiresAt"] = e.ExpiresAt.Format(layout)
	}
	return v
}

// DownloadToken 签名下载凭证：id.exp.sig。
func (s *Service) DownloadToken(id uint64, exp time.Time, secret string) string {
	msg := strconv.FormatUint(id, 10) + "." + strconv.FormatInt(exp.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return msg + "." + hex.EncodeToString(mac.Sum(nil))
}

// VerifyDownloadToken 校验并返回导出 ID。
func VerifyDownloadToken(token, secret string, now time.Time) (uint64, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, false
	}
	id, err1 := strconv.ParseUint(parts[0], 10, 64)
	exp, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil || now.Unix() > exp {
		return 0, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	return id, hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(parts[2]))
}

// ExportFile 返回可下载文件路径（merchantID=0 不校验归属）。
func (s *Service) ExportFile(ctx context.Context, merchantID, id uint64) (*ops.Export, string, error) {
	e, err := s.repo.FindExport(ctx, merchantID, id)
	if err != nil {
		return nil, "", errf(404, "导出不存在")
	}
	if e.Status != ops.ExportDone || e.FilePath == "" {
		return nil, "", errf(400, "文件尚未生成")
	}
	if e.ExpiresAt != nil && s.now().After(*e.ExpiresAt) {
		return nil, "", errf(410, "下载链接已过期")
	}
	return e, filepath.Join(s.exportDir, e.FilePath), nil
}

// RunPendingExports worker：领取并执行待处理导出。返回处理数。
func (s *Service) RunPendingExports(ctx context.Context, maxN int) int {
	n := 0
	for n < maxN {
		e, err := s.repo.ClaimExport(ctx)
		if err != nil || e == nil {
			break
		}
		s.runExport(ctx, e)
		n++
	}
	return n
}

func (s *Service) runExport(ctx context.Context, e *ops.Export) {
	var p ExportParams
	_ = json.Unmarshal(e.Params, &p)
	r := ParseRange(p.RangeKey, p.From, p.To, s.now())
	name := fmt.Sprintf("%d/%s-%d-%s.csv", e.MerchantID, e.Kind, e.ID, s.now().Format("20060102-150405"))
	full := filepath.Join(s.exportDir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		s.failExport(ctx, e, err)
		return
	}
	f, err := os.Create(full)
	if err != nil {
		s.failExport(ctx, e, err)
		return
	}
	_, _ = f.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM，Excel 识别 UTF-8
	w := csv.NewWriter(f)
	var rows uint64
	switch e.Kind {
	case "decision_logs":
		rows, err = s.exportLogs(ctx, w, e.MerchantID, p, r)
	case "report_channel":
		rows, err = s.exportChannel(ctx, w, e.MerchantID, p.SiteID, r)
	case "report_code", "report_campaign":
		ct := "distributor"
		if e.Kind == "report_campaign" {
			ct = "campaign"
		}
		rows, err = s.exportCodes(ctx, w, e.MerchantID, p.SiteID, ct, r)
	case "report_path":
		rows, err = s.exportPaths(ctx, w, e.MerchantID, p.SiteID, r)
	case "report_conversion":
		rows, err = s.exportConversions(ctx, w, e.MerchantID, p.SiteID, r)
	}
	w.Flush()
	if err == nil {
		err = w.Error()
	}
	_ = f.Close()
	if err != nil {
		_ = os.Remove(full)
		s.failExport(ctx, e, err)
		return
	}
	st, _ := os.Stat(full)
	now := s.now()
	exp := now.Add(24 * time.Hour)
	e.Status, e.FilePath, e.RowCount, e.FinishedAt, e.ExpiresAt = ops.ExportDone, filepath.ToSlash(name), rows, &now, &exp
	if st != nil {
		e.FileSize = uint64(st.Size())
	}
	_ = s.repo.UpdateExport(ctx, e)
	link := strings.TrimRight(s.cfg.ConsoleURL, "/") + "/merchant/report?tab=export"
	refType, refID := "export", e.ID
	_ = s.repo.CreateNotification(ctx, &ops.Notification{MerchantID: e.MerchantID, Kind: "export", Severity: "info",
		Title: ExportKinds[e.Kind] + "导出完成", Content: fmt.Sprintf("共 %d 行，24 小时内可下载", rows), Link: &link, RefType: &refType, RefID: &refID, CreatedAt: now})
	_, _ = s.EnqueueWebhook(ctx, e.MerchantID, "export.done", "export-"+strconv.FormatUint(e.ID, 10), map[string]any{"exportId": e.ID, "kind": e.Kind, "rows": rows})
}

func (s *Service) failExport(ctx context.Context, e *ops.Export, err error) {
	now := s.now()
	e.Status, e.Error, e.FinishedAt = ops.ExportFailed, trunc(err.Error(), 500), &now
	_ = s.repo.UpdateExport(ctx, e)
	s.log.Warn("export failed", zap.Uint64("id", e.ID), zap.Error(err))
}

const exportMaxRows = 200_000

func (s *Service) exportLogs(ctx context.Context, w *csv.Writer, merchantID uint64, p ExportParams, r Range) (uint64, error) {
	_ = w.Write([]string{"时间", "请求ID", "站点ID", "入口", "方法", "主机", "路径", "触点", "内容", "置信度", "渠道", "分销码", "内置浏览器", "设备族", "国家", "访客ID", "校验失败", "耗时ms", "降级"})
	var n uint64
	for page := 1; ; page++ {
		from, to := r.From, r.To
		rows, _, err := s.logs.ListDecisions(ctx, mysql.LogFilter{MerchantID: merchantID, SiteID: p.SiteID, From: &from, To: &to, Content: p.Content,
			TouchType: p.TouchType, Channel: p.Channel, Code: p.Code, Path: p.Path, Page: page, Size: 2000})
		if err != nil {
			return n, err
		}
		for _, x := range rows {
			_ = w.Write([]string{x.CreatedAt.Format(layout), x.RequestID, strconv.FormatUint(x.SiteID, 10), x.Entry, x.Method, x.Host, x.Path, x.TouchType,
				x.Content, x.Confidence, x.Channel, x.Code, x.InAppBrowser, x.DeviceFamily, x.Country, x.VisitorID, string(nonEmpty(x.ValidationErrors)),
				strconv.FormatUint(uint64(x.LatencyMs), 10), boolStr(x.Degraded)})
			n++
		}
		if len(rows) < 2000 || n >= exportMaxRows {
			break
		}
	}
	return n, nil
}

func (s *Service) exportChannel(ctx context.Context, w *csv.Writer, merchantID, siteID uint64, r Range) (uint64, error) {
	rows, err := s.repo.ChannelReport(ctx, merchantID, siteID, r.From, r.To)
	if err != nil {
		return 0, err
	}
	_ = w.Write([]string{"渠道", "放行", "首访", "回访", "交接", "内置浏览器", "访客数", "转化数", "转化金额"})
	for _, x := range rows {
		_ = w.Write([]string{str(x["channel"]), str(x["visits"]), str(x["firstVisits"]), str(x["returnVisits"]), str(x["handoffVisits"]), str(x["inApp"]), str(x["visitors"]), str(x["conversions"]), str(x["conversionAmount"])})
	}
	return uint64(len(rows)), nil
}

func (s *Service) exportCodes(ctx context.Context, w *csv.Writer, merchantID, siteID uint64, codeType string, r Range) (uint64, error) {
	_ = w.Write([]string{"码", "名称", "类型", "渠道", "状态", "放行", "首访", "回访", "交接", "内置浏览器", "转化数", "转化金额"})
	var n uint64
	for page := 1; ; page++ {
		rows, _, err := s.repo.CodeReport(ctx, merchantID, siteID, codeType, r.From, r.To, page, 1000)
		if err != nil {
			return n, err
		}
		for _, x := range rows {
			_ = w.Write([]string{str(x["code"]), str(x["name"]), str(x["codeType"]), str(x["channel"]), str(x["status"]), str(x["visits"]), str(x["firstVisits"]),
				str(x["returnVisits"]), str(x["handoffVisits"]), str(x["inApp"]), str(x["conversions"]), str(x["conversionAmount"])})
			n++
		}
		if len(rows) < 1000 {
			break
		}
	}
	return n, nil
}

func (s *Service) exportPaths(ctx context.Context, w *csv.Writer, merchantID, siteID uint64, r Range) (uint64, error) {
	_ = w.Write([]string{"站点ID", "路径", "访问", "放行", "审核", "拦截", "无归因访问"})
	var n uint64
	for page := 1; ; page++ {
		rows, _, err := s.repo.PathReport(ctx, merchantID, siteID, r.From, r.To, page, 1000)
		if err != nil {
			return n, err
		}
		for _, x := range rows {
			_ = w.Write([]string{str(x["siteId"]), str(x["path"]), str(x["visits"]), str(x["realTotal"]), str(x["review"]), str(x["block"]), str(x["noAttribution"])})
			n++
		}
		if len(rows) < 1000 {
			break
		}
	}
	return n, nil
}

func (s *Service) exportConversions(ctx context.Context, w *csv.Writer, merchantID, siteID uint64, r Range) (uint64, error) {
	_ = w.Write([]string{"时间", "站点ID", "订单号", "状态", "金额", "币种", "渠道", "码", "访客ID"})
	var n uint64
	for page := 1; ; page++ {
		rows, _, err := s.repo.ConversionReport(ctx, merchantID, siteID, r.From, r.To, page, 1000)
		if err != nil {
			return n, err
		}
		for _, x := range rows {
			t := ""
			if ts, ok := x["occurredAt"].(time.Time); ok {
				t = ts.Format(layout)
			}
			_ = w.Write([]string{t, str(x["siteId"]), str(x["orderNo"]), str(x["status"]), str(x["amount"]), str(x["currency"]), str(x["channel"]), str(x["code"]), str(x["visitorId"])})
			n++
		}
		if len(rows) < 1000 || n >= exportMaxRows {
			break
		}
	}
	return n, nil
}

// PurgeExpiredExports 删除过期文件。
func (s *Service) PurgeExpiredExports(ctx context.Context) int {
	rows, err := s.repo.ExpiredExports(ctx, s.now())
	if err != nil {
		return 0
	}
	n := 0
	for i := range rows {
		e := rows[i]
		_ = os.Remove(filepath.Join(s.exportDir, e.FilePath))
		e.FilePath = ""
		if err := s.repo.UpdateExport(ctx, &e); err == nil {
			n++
		}
	}
	return n
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// ---- 聚合入口（worker） ----

// Aggregate 重算最近若干小时与今天/昨天的日统计。
func (s *Service) Aggregate(ctx context.Context, hoursBack int) error {
	now := s.now()
	from := now.Truncate(time.Hour).Add(-time.Duration(hoursBack) * time.Hour)
	if _, err := s.repo.RebuildHourly(ctx, from, now.Truncate(time.Hour).Add(time.Hour)); err != nil {
		return err
	}
	days := map[string]time.Time{}
	for d := from; !d.After(now); d = d.Add(24 * time.Hour) {
		days[d.Format("2006-01-02")] = d
	}
	days[now.Format("2006-01-02")] = now
	for _, d := range days {
		if err := s.repo.RebuildDaily(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// AggregateRange 手动重算任意区间（补数 / 对账）。
func (s *Service) AggregateRange(ctx context.Context, from, to time.Time) error {
	if _, err := s.repo.RebuildHourly(ctx, from.Truncate(time.Hour), to); err != nil {
		return err
	}
	for d := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location()); d.Before(to); d = d.AddDate(0, 0, 1) {
		if err := s.repo.RebuildDaily(ctx, d); err != nil {
			return err
		}
	}
	return nil
}
