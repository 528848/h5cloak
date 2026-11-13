package mysql

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cloak/internal/domain/decisionlog"
)

// LogRepo 决策日志、归因、访客、计量、API 日志。
type LogRepo struct{ db *gorm.DB }

func NewLogRepo(db *gorm.DB) *LogRepo { return &LogRepo{db: db} }

// InsertDecisions 批量写入，重复 request_id 忽略（回放时幂等）。
func (r *LogRepo) InsertDecisions(ctx context.Context, events []decisionlog.Event) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}
	rows := make([]decisionlog.Record, 0, len(events))
	for i := range events {
		rows = append(rows, toRecord(&events[i]))
	}
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, 200)
	return res.RowsAffected, res.Error
}

func toRecord(e *decisionlog.Event) decisionlog.Record {
	qp, _ := json.Marshal(nz(e.QueryParamNames))
	mp, _ := json.Marshal(nz(e.MatchedParams))
	ve, _ := json.Marshal(e.ValidationErrors)
	tr, _ := json.Marshal(e.Trace)
	return decisionlog.Record{
		CreatedAt: e.CreatedAt, RequestID: e.RequestID, MerchantID: e.MerchantID, SiteID: e.SiteID,
		Entry: e.Entry, Method: e.Method, Host: e.Host, Path: trunc(e.Path, 1000), QueryParamNames: qp,
		TouchType: e.TouchType, Content: e.Content, Attributed: e.Attributed, Confidence: e.Confidence,
		InAppBrowser: e.InAppBrowser, DeviceFamily: e.DeviceFamily, HandoffKind: e.HandoffKind, HandoffFromRequestID: e.HandoffFromRequestID,
		Channel: trunc(e.Channel, 128), CodeID: e.CodeID, Code: e.Code, MatchedParams: mp, ValidationErrors: ve,
		VisitorID: e.VisitorID, AttributionID: e.AttributionID, AttributionExpiresAt: e.AttributionExpiresAt,
		IPHash: e.IPHash, IPPrefix: e.IPPrefix, Country: e.Country, OS: e.OS,
		UASummary: trunc(e.UASummary, 255), RefererSummary: trunc(e.RefererSummary, 255), PathRule: trunc(e.PathRule, 255),
		OriginKind: e.OriginKind, OriginStatus: uint16(e.OriginStatus), Fallback: e.Fallback,
		LatencyMs: uint32(e.LatencyMs), Degraded: e.Degraded, Trace: tr,
	}
}

// ApplyAttribution 按事件更新访客与归因：首次字段只在新建时写，最近字段每次更新。
func (r *LogRepo) ApplyAttribution(ctx context.Context, w *decisionlog.AttributionWrite) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		v := decisionlog.Visitor{
			SiteID: w.SiteID, VisitorID: w.VisitorID, MerchantID: w.MerchantID,
			FirstSeenAt: w.At, LastSeenAt: w.At, VisitCount: 1,
			FirstCountry: w.Country, FirstDeviceFamily: w.DeviceFamily, FirstInApp: w.InAppBrowser,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "site_id"}, {Name: "visitor_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"last_seen_at": w.At,
				"visit_count":  gorm.Expr("visit_count + 1"),
			}),
		}).Create(&v).Error; err != nil {
			return err
		}
		// 回访/交接不带码时，按已有归因的最近码累计回访数
		statCodeID := w.CodeID
		if statCodeID == 0 && w.TouchType != "first" {
			var prev decisionlog.Attribution
			if err := tx.Select("last_code_id").Where("site_id = ? AND visitor_id = ?", w.SiteID, w.VisitorID).First(&prev).Error; err == nil {
				statCodeID = prev.LastCodeID
			}
		}
		mp, _ := json.Marshal(nz(w.MatchedParams))
		a := decisionlog.Attribution{
			AttributionID: w.AttributionID, MerchantID: w.MerchantID, SiteID: w.SiteID, VisitorID: w.VisitorID,
			FirstChannel: w.Channel, FirstCodeID: w.CodeID, FirstParams: mp, FirstLandingPath: trunc(w.LandingPath, 1000),
			FirstRequestID: w.RequestID, FirstTouchAt: w.At,
			LastChannel: w.Channel, LastCodeID: w.CodeID, LastTouchAt: w.At, TouchCount: 1,
			Confidence: w.Confidence, ExpiresAt: w.ExpiresAt,
		}
		if w.TouchType == "handoff" {
			a.HandoffCount = 1
		}
		upd := map[string]any{
			"last_touch_at": w.At,
			"touch_count":   gorm.Expr("touch_count + 1"),
			"expires_at":    w.ExpiresAt,
		}
		if w.Channel != "" {
			upd["last_channel"] = w.Channel
			upd["last_code_id"] = w.CodeID
		}
		if w.TouchType == "handoff" {
			upd["handoff_count"] = gorm.Expr("handoff_count + 1")
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "site_id"}, {Name: "visitor_id"}},
			DoUpdates: clause.Assignments(upd),
		}).Create(&a).Error; err != nil {
			return err
		}
		// 只统计页面请求。样式、脚本、图片不累加首访和回访。
		if statCodeID > 0 && pageCountable(w.LandingPath) {
			col := "return_count"
			if w.TouchType == "first" {
				col = "first_pass_count"
			}
			setVisit := ""
			if w.IPHash != "" {
				res := tx.Exec(`INSERT IGNORE INTO promo_code_ips (code_id, ip_hash, first_seen_at) VALUES (?, ?, ?)`, statCodeID, w.IPHash, w.At)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected > 0 {
					setVisit = "visit_count = visit_count + 1, "
				}
			}
			if err := tx.Exec("UPDATE promo_codes SET "+setVisit+col+" = "+col+" + 1, last_visit_at = ? WHERE id = ?", w.At, statCodeID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ApplyConversion 写转化（幂等），关联归因，累计分销码转化数，并按邮箱哈希合并访客。
func (r *LogRepo) ApplyConversion(ctx context.Context, e *decisionlog.ConversionEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attrID, channel string
		var codeID uint64
		vid := e.VisitorID
		// 没有访客 ID 但有邮箱哈希：找同邮箱已知访客
		if vid == "" && e.EmailHash != "" {
			var ve struct{ VisitorID string }
			if err := tx.Table("visitor_emails").Select("visitor_id").
				Where("site_id = ? AND email_hash = ?", e.SiteID, e.EmailHash).Order("first_seen_at").Limit(1).Scan(&ve).Error; err == nil {
				vid = ve.VisitorID
			}
		}
		if vid != "" {
			var a decisionlog.Attribution
			if err := tx.Where("site_id = ? AND visitor_id = ?", e.SiteID, vid).First(&a).Error; err == nil {
				attrID, channel, codeID = a.AttributionID, a.LastChannel, a.LastCodeID
				if codeID == 0 {
					codeID = a.FirstCodeID
				}
			}
		}
		res := tx.Exec(`INSERT IGNORE INTO conversions
			(merchant_id, site_id, order_no, status, amount, currency, visitor_id, attribution_id, code_id, channel, email_hash, occurred_at, request_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.MerchantID, e.SiteID, e.OrderNo, e.Status, e.Amount, e.Currency, vid, attrID, codeID, channel, e.EmailHash, e.At, e.RequestID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // 重复回传
		}
		if codeID > 0 && e.Status == "paid" {
			if err := tx.Exec("UPDATE promo_codes SET conversion_count = conversion_count + 1 WHERE id = ?", codeID).Error; err != nil {
				return err
			}
		}
		// 邮箱 → 访客映射；若同邮箱已有其它访客，把当前访客的归因合并到最早的那个
		if e.EmailHash != "" && vid != "" {
			if err := tx.Exec("INSERT IGNORE INTO visitor_emails (site_id, email_hash, visitor_id, first_seen_at) VALUES (?, ?, ?, ?)",
				e.SiteID, e.EmailHash, vid, e.At).Error; err != nil {
				return err
			}
			var primary struct{ VisitorID string }
			if err := tx.Table("visitor_emails").Select("visitor_id").
				Where("site_id = ? AND email_hash = ?", e.SiteID, e.EmailHash).Order("first_seen_at").Limit(1).Scan(&primary).Error; err == nil &&
				primary.VisitorID != "" && primary.VisitorID != vid {
				_ = tx.Exec("UPDATE attributions SET merged_into_visitor_id = ? WHERE site_id = ? AND visitor_id = ? AND merged_into_visitor_id IS NULL",
					primary.VisitorID, e.SiteID, vid).Error
			}
		}
		return nil
	})
}

// AddUsage 月度计量累加。
func (r *LogRepo) AddUsage(ctx context.Context, merchantID uint64, metric, period string, delta uint64) error {
	if delta == 0 {
		return nil
	}
	row := decisionlog.UsageCounter{MerchantID: merchantID, Metric: metric, Period: period, Value: delta, UpdatedAt: time.Now()}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "merchant_id"}, {Name: "metric"}, {Name: "period"}},
		DoUpdates: clause.Assignments(map[string]any{"value": gorm.Expr("value + ?", delta), "updated_at": time.Now()}),
	}).Create(&row).Error
}

// Usage 读取月度计量。
func (r *LogRepo) Usage(ctx context.Context, merchantID uint64, metric, period string) (uint64, error) {
	var row decisionlog.UsageCounter
	err := r.db.WithContext(ctx).Where("merchant_id = ? AND metric = ? AND period = ?", merchantID, metric, period).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	return row.Value, err
}

// InsertAPILogs 批量写入接口日志。
func (r *LogRepo) InsertAPILogs(ctx context.Context, events []decisionlog.APILogEvent) error {
	if len(events) == 0 {
		return nil
	}
	rows := make([]decisionlog.APILog, 0, len(events))
	for _, e := range events {
		rows = append(rows, decisionlog.APILog{
			CreatedAt: e.CreatedAt, RequestID: e.RequestID, MerchantID: e.MerchantID, SiteID: e.SiteID,
			KeyPrefix: e.KeyPrefix, Endpoint: e.Endpoint, Status: uint16(e.Status), ErrorCode: e.ErrorCode,
			LatencyMs: uint32(e.LatencyMs), IP: e.IP,
		})
	}
	return r.db.WithContext(ctx).CreateInBatches(rows, 200).Error
}

// APILogFilter 接口日志筛选。MerchantID 为 0 表示全平台。
type APILogFilter struct {
	MerchantID uint64
	SiteID     uint64
	Endpoint   string
	From       *time.Time
	Page, Size int
}

// ListAPILogs 按时间倒序分页。
func (r *LogRepo) ListAPILogs(ctx context.Context, f APILogFilter) ([]decisionlog.APILog, int64, error) {
	q := r.db.WithContext(ctx).Model(&decisionlog.APILog{})
	if f.MerchantID > 0 {
		q = q.Where("merchant_id = ?", f.MerchantID)
	}
	if f.SiteID > 0 {
		q = q.Where("site_id = ?", f.SiteID)
	}
	if f.Endpoint != "" {
		q = q.Where("endpoint LIKE ?", "%"+f.Endpoint+"%")
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := f.Page, f.Size
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	var rows []decisionlog.APILog
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

// FindAPILog 按主键读取一条接口日志。
func (r *LogRepo) FindAPILog(ctx context.Context, id uint64) (*decisionlog.APILog, error) {
	var row decisionlog.APILog
	err := r.db.WithContext(ctx).First(&row, id).Error
	return wrap(&row, err)
}

// Purge 删除某商户早于 before 的决策日志，分批避免长事务；返回删除行数。
func (r *LogRepo) Purge(ctx context.Context, merchantID uint64, before time.Time, batch int) (int64, error) {
	var total int64
	for {
		res := r.db.WithContext(ctx).Exec("DELETE FROM decision_logs WHERE merchant_id = ? AND created_at < ? LIMIT ?", merchantID, before, batch)
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < int64(batch) {
			break
		}
	}
	for {
		res := r.db.WithContext(ctx).Exec("DELETE FROM cloak_access_logs WHERE merchant_id = ? AND created_at < ? LIMIT ?", merchantID, before, batch)
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < int64(batch) {
			return total, nil
		}
	}
}

// ---- 查询 ----

// LogFilter 决策日志筛选。
type LogFilter struct {
	MerchantID uint64 // 0 为全平台
	SiteID     uint64
	From, To   *time.Time
	Path       string
	TouchType  string
	Content    string
	Channel    string
	Code       string
	Reason     string // validation error reason
	Country    string
	InApp      string
	VisitorID  string
	RequestID  string
	PageOnly   bool // 只看网站页面路径，过滤页面内的文件请求
	Page, Size int
}

func (r *LogRepo) ListDecisions(ctx context.Context, f LogFilter) ([]decisionlog.Record, int64, error) {
	q := r.db.WithContext(ctx).Model(&decisionlog.Record{})
	// 只看页面路径时，排除静态资源（bypass）与页面内的文件请求；按 request_id 精确查时不过滤。
	if f.PageOnly && f.RequestID == "" {
		q = q.Where(pagePathSQL)
	}
	if f.MerchantID > 0 {
		q = q.Where("merchant_id = ?", f.MerchantID)
	}
	if f.SiteID > 0 {
		q = q.Where("site_id = ?", f.SiteID)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}
	if p := strings.TrimSpace(f.Path); p != "" {
		q = q.Where("path LIKE ?", escapeLike(p)+"%")
	}
	if f.TouchType != "" {
		q = q.Where("touch_type = ?", f.TouchType)
	}
	if f.Content != "" {
		q = q.Where("content = ?", f.Content)
	}
	if f.Channel != "" {
		q = q.Where("channel LIKE ?", escapeLike(f.Channel)+"%")
	}
	if f.Code != "" {
		q = q.Where("code = ?", f.Code)
	}
	if f.Reason != "" {
		q = q.Where("JSON_SEARCH(validation_errors, 'one', ?, NULL, '$[*].reason') IS NOT NULL", f.Reason)
	}
	if f.Country != "" {
		q = q.Where("country = ?", f.Country)
	}
	if f.InApp != "" {
		q = q.Where("in_app_browser = ?", f.InApp)
	}
	if f.VisitorID != "" {
		q = q.Where("visitor_id = ?", f.VisitorID)
	}
	if f.RequestID != "" {
		q = q.Where("request_id = ?", f.RequestID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []decisionlog.Record
	err := q.Order("created_at DESC, id DESC").Offset(offset(f.Page, f.Size)).Limit(limit(f.Size)).Find(&rows).Error
	return rows, total, err
}

func (r *LogRepo) FindDecision(ctx context.Context, requestID string) (*decisionlog.Record, error) {
	var row decisionlog.Record
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&row).Error
	return wrap(&row, err)
}

func (r *LogRepo) FindAttribution(ctx context.Context, siteID uint64, visitorID string) (*decisionlog.Attribution, error) {
	var row decisionlog.Attribution
	err := r.db.WithContext(ctx).Where("site_id = ? AND visitor_id = ?", siteID, visitorID).First(&row).Error
	return wrap(&row, err)
}

func (r *LogRepo) FindVisitor(ctx context.Context, siteID uint64, visitorID string) (*decisionlog.Visitor, error) {
	var row decisionlog.Visitor
	err := r.db.WithContext(ctx).Where("site_id = ? AND visitor_id = ?", siteID, visitorID).First(&row).Error
	return wrap(&row, err)
}

// LogStats 时间段内按页面 + IP 统计。同一页面、同一 IP 只计一次；有放行则不再计审核。样式和脚本不计入。
func (r *LogRepo) LogStats(ctx context.Context, merchantID, siteID uint64, from time.Time) (map[string]int64, error) {
	where := "created_at >= ? AND " + pagePathSQL
	args := []any{from}
	if merchantID > 0 {
		where += " AND merchant_id = ?"
		args = append(args, merchantID)
	}
	if siteID > 0 {
		where += " AND site_id = ?"
		args = append(args, siteID)
	}
	inner := `SELECT
		MAX(content = 'real') AS is_real,
		MAX(content = 'review') AS is_review,
		MAX(content = 'block') AS is_block,
		MAX(content = 'not_found') AS is_nf,
		MAX(content = 'real' AND touch_type = 'first') AS has_first,
		MAX(content = 'real' AND touch_type = 'return') AS has_return,
		MAX(content = 'real' AND touch_type = 'handoff') AS has_handoff
	FROM decision_logs
	WHERE ` + where + `
	GROUP BY site_id, ` + pageKeySQL + `, ` + visitIPSQL
	var out struct {
		Total    int64
		Real     int64 `gorm:"column:real_n"`
		Review   int64 `gorm:"column:review_n"`
		Block    int64 `gorm:"column:block_n"`
		NotFound int64 `gorm:"column:not_found_n"`
		First    int64 `gorm:"column:first_n"`
		ReturnN  int64 `gorm:"column:return_n"`
		Handoff  int64 `gorm:"column:handoff_n"`
	}
	err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) AS total,
		COALESCE(SUM(is_real),0) AS real_n,
		COALESCE(SUM(is_real = 0 AND is_review = 1),0) AS review_n,
		COALESCE(SUM(is_real = 0 AND is_review = 0 AND is_block = 1),0) AS block_n,
		COALESCE(SUM(is_real = 0 AND is_review = 0 AND is_block = 0 AND is_nf = 1),0) AS not_found_n,
		COALESCE(SUM(is_real = 1 AND has_first = 1),0) AS first_n,
		COALESCE(SUM(is_real = 1 AND has_first = 0 AND has_return = 1),0) AS return_n,
		COALESCE(SUM(is_real = 1 AND has_first = 0 AND has_return = 0 AND has_handoff = 1),0) AS handoff_n
		FROM (`+inner+`) v`, args...).Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return map[string]int64{
		"total": out.Total,
		"content:real": out.Real, "content:review": out.Review,
		"content:block": out.Block, "content:not_found": out.NotFound, "content:bypass": 0,
		"touch:first": out.First, "touch:return": out.ReturnN, "touch:handoff": out.Handoff,
		"touch:none": out.Review,
	}, nil
}

func nz(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func pageKey(path string) string {
	path = strings.TrimSpace(path)
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	if path == "" {
		return "/"
	}
	return path
}

func pageCountable(path string) bool {
	p := strings.ToLower(pageKey(path))
	if p == "" {
		return false
	}
	for _, dir := range []string{"/wp-content/", "/wp-includes/", "/assets/", "/static/", "/uploads/", "/fonts/", "/images/", "/img/", "/media/"} {
		if p == strings.TrimSuffix(dir, "/") || strings.HasPrefix(p, dir) {
			return false
		}
	}
	base := p
	if i := strings.LastIndex(p, "/"); i >= 0 {
		base = p[i+1:]
	}
	if strings.HasPrefix(base, ".") {
		return false
	}
	for _, ext := range []string{".css", ".js", ".mjs", ".map", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".svg", ".ico", ".bmp", ".woff", ".woff2", ".ttf", ".eot", ".otf", ".json", ".xml", ".txt", ".pdf", ".zip", ".gz", ".mp3", ".mp4", ".webm", ".wasm"} {
		if strings.HasSuffix(base, ext) {
			return false
		}
	}
	return true
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
