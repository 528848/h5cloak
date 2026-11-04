package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cloak/internal/domain/ops"
)

// OpsRepo 聚合、告警、通知、Webhook、导出。
type OpsRepo struct{ db *gorm.DB }

func NewOpsRepo(db *gorm.DB) *OpsRepo { return &OpsRepo{db: db} }

// ---- 聚合（从日志重算，幂等覆盖） ----

// RebuildHourly 重算 [from, to) 范围内每个站点每小时的指标。
func (r *OpsRepo) RebuildHourly(ctx context.Context, from, to time.Time) (int64, error) {
	sql := `
INSERT INTO metrics_hourly (merchant_id, site_id, hour, visits, real_total, first_real, return_real, handoff_real,
  review, block, not_found, bypass, validation_failed, in_app, degraded, origin_errors, latency_sum)
SELECT merchant_id, site_id, DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00') AS h,
  COUNT(*),
  SUM(content = 'real'),
  SUM(content = 'real' AND touch_type = 'first'),
  SUM(content = 'real' AND touch_type = 'return'),
  SUM(content = 'real' AND touch_type = 'handoff'),
  SUM(content = 'review'),
  SUM(content = 'block'),
  SUM(content = 'not_found'),
  SUM(content = 'bypass'),
  SUM(validation_errors IS NOT NULL AND JSON_LENGTH(validation_errors) > 0),
  SUM(in_app_browser <> ''),
  SUM(degraded = 1),
  SUM(origin_status >= 500 OR fallback = 1),
  SUM(latency_ms)
FROM decision_logs
WHERE created_at >= ? AND created_at < ?
GROUP BY merchant_id, site_id, h
ON DUPLICATE KEY UPDATE
  merchant_id = VALUES(merchant_id), visits = VALUES(visits), real_total = VALUES(real_total), first_real = VALUES(first_real),
  return_real = VALUES(return_real), handoff_real = VALUES(handoff_real), review = VALUES(review), block = VALUES(block),
  not_found = VALUES(not_found), bypass = VALUES(bypass), validation_failed = VALUES(validation_failed), in_app = VALUES(in_app),
  degraded = VALUES(degraded), origin_errors = VALUES(origin_errors), latency_sum = VALUES(latency_sum)`
	res := r.db.WithContext(ctx).Exec(sql, from, to)
	if res.Error != nil {
		return 0, res.Error
	}
	conv := `
INSERT INTO metrics_hourly (merchant_id, site_id, hour, conversions, conversion_amount)
SELECT merchant_id, site_id, DATE_FORMAT(occurred_at, '%Y-%m-%d %H:00:00') AS h, COUNT(*), SUM(amount)
FROM conversions WHERE occurred_at >= ? AND occurred_at < ? AND status = 'paid'
GROUP BY merchant_id, site_id, h
ON DUPLICATE KEY UPDATE conversions = VALUES(conversions), conversion_amount = VALUES(conversion_amount)`
	if err := r.db.WithContext(ctx).Exec(conv, from, to).Error; err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}

// RebuildDaily 重算某天的分销码与路径统计。
func (r *OpsRepo) RebuildDaily(ctx context.Context, day time.Time) error {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)
	codes := `
INSERT INTO code_stats_daily (site_id, code_id, day, merchant_id, code, channel, visits, first_visits, return_visits, handoff_visits, in_app)
SELECT site_id, code_id, ?, merchant_id, MAX(code), MAX(channel), COUNT(*),
  SUM(touch_type = 'first'), SUM(touch_type = 'return'), SUM(touch_type = 'handoff'), SUM(in_app_browser <> '')
FROM decision_logs
WHERE created_at >= ? AND created_at < ? AND code_id > 0 AND content = 'real'
GROUP BY site_id, code_id, merchant_id
ON DUPLICATE KEY UPDATE code = VALUES(code), channel = VALUES(channel), visits = VALUES(visits), first_visits = VALUES(first_visits),
  return_visits = VALUES(return_visits), handoff_visits = VALUES(handoff_visits), in_app = VALUES(in_app)`
	if err := r.db.WithContext(ctx).Exec(codes, start.Format("2006-01-02"), start, end).Error; err != nil {
		return err
	}
	conv := `
INSERT INTO code_stats_daily (site_id, code_id, day, merchant_id, conversions, conversion_amount)
SELECT site_id, code_id, ?, merchant_id, COUNT(*), SUM(amount)
FROM conversions WHERE occurred_at >= ? AND occurred_at < ? AND status = 'paid' AND code_id > 0
GROUP BY site_id, code_id, merchant_id
ON DUPLICATE KEY UPDATE conversions = VALUES(conversions), conversion_amount = VALUES(conversion_amount)`
	if err := r.db.WithContext(ctx).Exec(conv, start.Format("2006-01-02"), start, end).Error; err != nil {
		return err
	}
	paths := `
INSERT INTO path_stats_daily (site_id, day, path, merchant_id, visits, real_total, review, block, no_attribution)
SELECT site_id, ?, LEFT(path, 255), merchant_id, COUNT(*), SUM(content = 'real'), SUM(content = 'review'), SUM(content = 'block'),
  SUM(content = 'review' AND touch_type = 'none')
FROM decision_logs
WHERE created_at >= ? AND created_at < ? AND content <> 'bypass'
GROUP BY site_id, merchant_id, LEFT(path, 255)
ON DUPLICATE KEY UPDATE visits = VALUES(visits), real_total = VALUES(real_total), review = VALUES(review), block = VALUES(block),
  no_attribution = VALUES(no_attribution)`
	return r.db.WithContext(ctx).Exec(paths, start.Format("2006-01-02"), start, end).Error
}

// ---- 看板 / 报表查询 ----

// HourlyRange 读小时聚合。siteID=0 时按商户汇总（merchantID=0 全平台）。
func (r *OpsRepo) HourlyRange(ctx context.Context, merchantID, siteID uint64, from, to time.Time) ([]ops.MetricsHourly, error) {
	q := r.db.WithContext(ctx).Model(&ops.MetricsHourly{}).Where("hour >= ? AND hour < ?", from, to)
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		q = q.Where("site_id = ?", siteID)
	}
	var rows []ops.MetricsHourly
	err := q.Select(`hour, SUM(visits) visits, SUM(real_total) real_total, SUM(first_real) first_real, SUM(return_real) return_real,
		SUM(handoff_real) handoff_real, SUM(review) review, SUM(block) block, SUM(not_found) not_found, SUM(bypass) bypass,
		SUM(validation_failed) validation_failed, SUM(in_app) in_app, SUM(degraded) degraded, SUM(origin_errors) origin_errors,
		SUM(latency_sum) latency_sum, SUM(conversions) conversions, SUM(conversion_amount) conversion_amount`).
		Group("hour").Order("hour").Scan(&rows).Error
	return rows, err
}

// Breakdown 按日志某列分组计数（渠道、置信度、国家、内置浏览器、设备、校验失败原因等）。
func (r *OpsRepo) Breakdown(ctx context.Context, merchantID, siteID uint64, from, to time.Time, column string, onlyReal bool, limitN int) ([]KV, error) {
	allowed := map[string]string{
		"channel": "channel", "confidence": "confidence", "country": "country", "in_app_browser": "in_app_browser",
		"os": "os", "device_family": "device_family", "code": "code", "content": "content", "touch_type": "touch_type", "entry": "entry",
	}
	col, ok := allowed[column]
	if !ok {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Table("decision_logs").Where("created_at >= ? AND created_at < ?", from, to)
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		q = q.Where("site_id = ?", siteID)
	}
	if onlyReal {
		q = q.Where("content = 'real'")
	}
	if limitN <= 0 {
		limitN = 10
	}
	var rows []KV
	err := q.Select(col + " AS k, COUNT(*) AS v").Group(col).Order("v DESC").Limit(limitN).Scan(&rows).Error
	return rows, err
}

// KV 分组统计。
type KV struct {
	K string `json:"k"`
	V int64  `json:"v"`
}

// ValidationReasons 校验失败原因分布（展开 JSON）。
func (r *OpsRepo) ValidationReasons(ctx context.Context, merchantID, siteID uint64, from, to time.Time) ([]KV, error) {
	q := r.db.WithContext(ctx).Table("decision_logs d, JSON_TABLE(d.validation_errors, '$[*]' COLUMNS (reason VARCHAR(32) PATH '$.reason')) jt").
		Where("d.created_at >= ? AND d.created_at < ? AND d.validation_errors IS NOT NULL", from, to)
	if merchantID > 0 {
		q = q.Where("d.merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		q = q.Where("d.site_id = ?", siteID)
	}
	var rows []KV
	err := q.Select("jt.reason AS k, COUNT(*) AS v").Group("jt.reason").Order("v DESC").Scan(&rows).Error
	return rows, err
}

// CodeReport 分销码 / 活动码报表（按码汇总某区间）。
func (r *OpsRepo) CodeReport(ctx context.Context, merchantID, siteID uint64, codeType string, from, to time.Time, page, size int) ([]map[string]any, int64, error) {
	base := r.db.WithContext(ctx).Table("code_stats_daily s").Joins("JOIN promo_codes p ON p.id = s.code_id").
		Where("s.day >= ? AND s.day < ?", from.Format("2006-01-02"), to.Format("2006-01-02"))
	if merchantID > 0 {
		base = base.Where("s.merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		base = base.Where("s.site_id = ?", siteID)
	}
	if codeType != "" {
		base = base.Where("p.code_type = ?", codeType)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Select("COUNT(DISTINCT s.code_id)").Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []map[string]any
	err := base.Select(`s.code_id AS codeId, MAX(s.code) AS code, MAX(p.name) AS name, MAX(p.code_type) AS codeType, MAX(p.channel) AS channel,
		MAX(p.status) AS status, SUM(s.visits) AS visits, SUM(s.first_visits) AS firstVisits, SUM(s.return_visits) AS returnVisits,
		SUM(s.handoff_visits) AS handoffVisits, SUM(s.in_app) AS inApp, SUM(s.conversions) AS conversions, SUM(s.conversion_amount) AS conversionAmount`).
		Group("s.code_id").Order("visits DESC").Offset(offset(page, size)).Limit(limit(size)).Scan(&rows).Error
	return rows, total, err
}

// ChannelReport 渠道报表。
func (r *OpsRepo) ChannelReport(ctx context.Context, merchantID, siteID uint64, from, to time.Time) ([]map[string]any, error) {
	q := r.db.WithContext(ctx).Table("decision_logs").Where("created_at >= ? AND created_at < ? AND content = 'real'", from, to)
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		q = q.Where("site_id = ?", siteID)
	}
	var rows []map[string]any
	if err := q.Select(`channel, COUNT(*) AS visits, SUM(touch_type = 'first') AS firstVisits, SUM(touch_type = 'return') AS returnVisits,
		SUM(touch_type = 'handoff') AS handoffVisits, SUM(in_app_browser <> '') AS inApp, COUNT(DISTINCT visitor_id) AS visitors`).
		Group("channel").Order("visits DESC").Limit(200).Scan(&rows).Error; err != nil {
		return nil, err
	}
	// 转化按渠道
	cq := r.db.WithContext(ctx).Table("conversions").Where("occurred_at >= ? AND occurred_at < ? AND status = 'paid'", from, to)
	if merchantID > 0 {
		cq = cq.Where("merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		cq = cq.Where("site_id = ?", siteID)
	}
	var convs []struct {
		Channel string
		N       int64
		Amount  float64
	}
	_ = cq.Select("channel, COUNT(*) n, SUM(amount) amount").Group("channel").Scan(&convs).Error
	cm := map[string][2]float64{}
	for _, c := range convs {
		cm[c.Channel] = [2]float64{float64(c.N), c.Amount}
	}
	for _, row := range rows {
		ch, _ := row["channel"].(string)
		v := cm[ch]
		row["conversions"] = int64(v[0])
		row["conversionAmount"] = v[1]
	}
	return rows, nil
}

// PathReport 路径报表。
func (r *OpsRepo) PathReport(ctx context.Context, merchantID, siteID uint64, from, to time.Time, page, size int) ([]map[string]any, int64, error) {
	base := r.db.WithContext(ctx).Table("path_stats_daily").Where("day >= ? AND day < ?", from.Format("2006-01-02"), to.Format("2006-01-02"))
	if merchantID > 0 {
		base = base.Where("merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		base = base.Where("site_id = ?", siteID)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Select("COUNT(DISTINCT site_id, path)").Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []map[string]any
	err := base.Select(`site_id AS siteId, path, SUM(visits) AS visits, SUM(real_total) AS realTotal, SUM(review) AS review, SUM(block) AS block,
		SUM(no_attribution) AS noAttribution`).Group("site_id, path").Order("noAttribution DESC, visits DESC").
		Offset(offset(page, size)).Limit(limit(size)).Scan(&rows).Error
	return rows, total, err
}

// ConversionReport 转化明细。
func (r *OpsRepo) ConversionReport(ctx context.Context, merchantID, siteID uint64, from, to time.Time, page, size int) ([]map[string]any, int64, error) {
	base := r.db.WithContext(ctx).Table("conversions c").Where("c.occurred_at >= ? AND c.occurred_at < ?", from, to)
	if merchantID > 0 {
		base = base.Where("c.merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		base = base.Where("c.site_id = ?", siteID)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []map[string]any
	err := base.Select(`c.id, c.site_id AS siteId, c.order_no AS orderNo, c.status, c.amount, c.currency, c.visitor_id AS visitorId,
		c.channel, c.code_id AS codeId, p.code, c.occurred_at AS occurredAt, c.email_hash <> '' AS hasEmail`).
		Joins("LEFT JOIN promo_codes p ON p.id = c.code_id").
		Order("c.occurred_at DESC").Offset(offset(page, size)).Limit(limit(size)).Scan(&rows).Error
	return rows, total, err
}

// ConversionTotals 区间转化合计。
func (r *OpsRepo) ConversionTotals(ctx context.Context, merchantID, siteID uint64, from, to time.Time) (int64, float64, error) {
	q := r.db.WithContext(ctx).Table("conversions").Where("occurred_at >= ? AND occurred_at < ? AND status = 'paid'", from, to)
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		q = q.Where("site_id = ?", siteID)
	}
	var out struct {
		N      int64
		Amount float64
	}
	err := q.Select("COUNT(*) n, COALESCE(SUM(amount),0) amount").Scan(&out).Error
	return out.N, out.Amount, err
}

// LogCountSince 原始日志计数，供聚合对账与实时告警。
func (r *OpsRepo) LogCountSince(ctx context.Context, merchantID, siteID uint64, from, to time.Time) (map[string]int64, error) {
	q := r.db.WithContext(ctx).Table("decision_logs").Where("created_at >= ? AND created_at < ?", from, to)
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		q = q.Where("site_id = ?", siteID)
	}
	var out struct {
		Total    int64
		Real     int64 `gorm:"column:real_n"`
		Review   int64 `gorm:"column:review_n"`
		Block    int64 `gorm:"column:block_n"`
		Bypass   int64 `gorm:"column:bypass_n"`
		Failed   int64 `gorm:"column:failed_n"`
		Degraded int64 `gorm:"column:degraded_n"`
	}
	err := q.Select(`COUNT(*) AS total, COALESCE(SUM(content='real'),0) AS real_n, COALESCE(SUM(content='review'),0) AS review_n,
		COALESCE(SUM(content='block'),0) AS block_n, COALESCE(SUM(content='bypass'),0) AS bypass_n,
		COALESCE(SUM(validation_errors IS NOT NULL AND JSON_LENGTH(validation_errors) > 0),0) AS failed_n, COALESCE(SUM(degraded=1),0) AS degraded_n`).Scan(&out).Error
	return map[string]int64{"total": out.Total, "real": out.Real, "review": out.Review, "block": out.Block, "bypass": out.Bypass,
		"validationFailed": out.Failed, "degraded": out.Degraded}, err
}

// CodeHits 窗口内各码放行次数（code.abnormal）。
func (r *OpsRepo) CodeHits(ctx context.Context, merchantID uint64, from, to time.Time, minCount int64) ([]struct {
	SiteID uint64
	CodeID uint64
	Code   string
	N      int64
}, error) {
	var rows []struct {
		SiteID uint64
		CodeID uint64
		Code   string
		N      int64
	}
	q := r.db.WithContext(ctx).Table("decision_logs").Where("created_at >= ? AND created_at < ? AND code_id > 0", from, to)
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	err := q.Select("site_id, code_id, MAX(code) code, COUNT(*) n").Group("site_id, code_id").Having("n >= ?", minCount).
		Order("n DESC").Limit(50).Scan(&rows).Error
	return rows, err
}

// SiteMerchants 返回有日志的 (merchant, site) 对，供告警评估遍历。
func (r *OpsRepo) ActiveSites(ctx context.Context, from time.Time) ([]struct{ MerchantID, SiteID uint64 }, error) {
	var rows []struct{ MerchantID, SiteID uint64 }
	err := r.db.WithContext(ctx).Table("decision_logs").Where("created_at >= ?", from).
		Select("merchant_id, site_id").Group("merchant_id, site_id").Scan(&rows).Error
	return rows, err
}

// UpsertHourly 单行写入（测试与补数用）。
func (r *OpsRepo) UpsertHourly(ctx context.Context, m *ops.MetricsHourly) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(m).Error
}
