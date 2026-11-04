package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cloak/internal/domain/ops"
)

// ---- 告警规则 ----

func (r *OpsRepo) ListRules(ctx context.Context, merchantID uint64) ([]ops.AlertRule, error) {
	var rows []ops.AlertRule
	q := r.db.WithContext(ctx).Order("site_id, id")
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	return rows, q.Find(&rows).Error
}

func (r *OpsRepo) AllEnabledRules(ctx context.Context) ([]ops.AlertRule, error) {
	var rows []ops.AlertRule
	return rows, r.db.WithContext(ctx).Where("enabled = 1").Find(&rows).Error
}

func (r *OpsRepo) UpsertRule(ctx context.Context, rule *ops.AlertRule) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "merchant_id"}, {Name: "site_id"}, {Name: "type"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "severity", "threshold", "window_minutes", "cooldown_minutes", "channels"}),
	}).Create(rule).Error
}

func (r *OpsRepo) DeleteRule(ctx context.Context, merchantID, id uint64) error {
	return r.db.WithContext(ctx).Where("merchant_id = ? AND id = ?", merchantID, id).Delete(&ops.AlertRule{}).Error
}

// ---- 告警实例 ----

// OpenAlert 查找同目标未恢复的告警。
func (r *OpsRepo) OpenAlert(ctx context.Context, merchantID uint64, typ, target string) (*ops.Alert, error) {
	var row ops.Alert
	err := r.db.WithContext(ctx).Where("merchant_id = ? AND type = ? AND target_key = ? AND status = ?", merchantID, typ, target, ops.AlertOpen).
		Order("id DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// LastAlertAt 最近一次同目标告警时间（冷却用）。
func (r *OpsRepo) LastAlertAt(ctx context.Context, merchantID uint64, typ, target string) (*time.Time, error) {
	var row ops.Alert
	err := r.db.WithContext(ctx).Select("fired_at").Where("merchant_id = ? AND type = ? AND target_key = ?", merchantID, typ, target).
		Order("id DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row.FiredAt, nil
}

func (r *OpsRepo) CreateAlert(ctx context.Context, a *ops.Alert) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *OpsRepo) UpdateAlert(ctx context.Context, a *ops.Alert, cols ...string) error {
	return r.db.WithContext(ctx).Model(a).Select(cols).Updates(a).Error
}

func (r *OpsRepo) FindAlert(ctx context.Context, merchantID, id uint64) (*ops.Alert, error) {
	var row ops.Alert
	q := r.db.WithContext(ctx).Where("id = ?", id)
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	return wrap(&row, q.First(&row).Error)
}

// AlertFilter 告警列表筛选。
type AlertFilter struct {
	MerchantID uint64
	SiteID     uint64
	Type       string
	Status     string
	Severity   string
	Page, Size int
}

func (r *OpsRepo) ListAlerts(ctx context.Context, f AlertFilter) ([]ops.Alert, int64, error) {
	q := r.db.WithContext(ctx).Model(&ops.Alert{})
	if f.MerchantID > 0 {
		q = q.Where("merchant_id = ?", f.MerchantID)
	}
	if f.SiteID > 0 {
		q = q.Where("site_id = ?", f.SiteID)
	}
	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Severity != "" {
		q = q.Where("severity = ?", f.Severity)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ops.Alert
	err := q.Order("status = 'open' DESC, fired_at DESC").Offset(offset(f.Page, f.Size)).Limit(limit(f.Size)).Find(&rows).Error
	return rows, total, err
}

// AlertCounts 按状态/级别计数。
func (r *OpsRepo) AlertCounts(ctx context.Context, merchantID uint64) (map[string]int64, error) {
	q := r.db.WithContext(ctx).Model(&ops.Alert{})
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	var rows []struct {
		Status   string
		Severity string
		N        int64
	}
	if err := q.Select("status, severity, COUNT(*) n").Group("status, severity").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, x := range rows {
		out[x.Status] += x.N
		if x.Status == ops.AlertOpen {
			out["open:"+x.Severity] += x.N
		}
	}
	return out, nil
}

// ---- 通知 ----

func (r *OpsRepo) CreateNotification(ctx context.Context, n *ops.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *OpsRepo) ListNotifications(ctx context.Context, merchantID uint64, unreadOnly bool, page, size int) ([]ops.Notification, int64, error) {
	q := r.db.WithContext(ctx).Model(&ops.Notification{}).Where("merchant_id = ?", merchantID)
	if unreadOnly {
		q = q.Where("read_at IS NULL")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ops.Notification
	err := q.Order("id DESC").Offset(offset(page, size)).Limit(limit(size)).Find(&rows).Error
	return rows, total, err
}

func (r *OpsRepo) UnreadCount(ctx context.Context, merchantID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&ops.Notification{}).Where("merchant_id = ? AND read_at IS NULL", merchantID).Count(&n).Error
	return n, err
}

func (r *OpsRepo) MarkRead(ctx context.Context, merchantID uint64, ids []uint64) error {
	q := r.db.WithContext(ctx).Model(&ops.Notification{}).Where("merchant_id = ? AND read_at IS NULL", merchantID)
	if len(ids) > 0 {
		q = q.Where("id IN ?", ids)
	}
	return q.Update("read_at", time.Now()).Error
}

// ---- 通知设置 ----

func (r *OpsRepo) GetNotifySettings(ctx context.Context, merchantID uint64) (*ops.NotifySettings, error) {
	var row ops.NotifySettings
	err := r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &ops.NotifySettings{MerchantID: merchantID}, nil
	}
	return &row, err
}

func (r *OpsRepo) SaveNotifySettings(ctx context.Context, s *ops.NotifySettings) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(s).Error
}

// ---- Webhook ----

func (r *OpsRepo) ListEndpoints(ctx context.Context, merchantID uint64) ([]ops.WebhookEndpoint, error) {
	var rows []ops.WebhookEndpoint
	return rows, r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).Order("id").Find(&rows).Error
}

func (r *OpsRepo) EnabledEndpoints(ctx context.Context, merchantID uint64) ([]ops.WebhookEndpoint, error) {
	var rows []ops.WebhookEndpoint
	return rows, r.db.WithContext(ctx).Where("merchant_id = ? AND enabled = 1", merchantID).Find(&rows).Error
}

func (r *OpsRepo) FindEndpoint(ctx context.Context, merchantID, id uint64) (*ops.WebhookEndpoint, error) {
	var row ops.WebhookEndpoint
	return wrap(&row, r.db.WithContext(ctx).Where("merchant_id = ? AND id = ?", merchantID, id).First(&row).Error)
}

func (r *OpsRepo) SaveEndpoint(ctx context.Context, e *ops.WebhookEndpoint) error {
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *OpsRepo) DeleteEndpoint(ctx context.Context, merchantID, id uint64) error {
	return r.db.WithContext(ctx).Where("merchant_id = ? AND id = ?", merchantID, id).Delete(&ops.WebhookEndpoint{}).Error
}

func (r *OpsRepo) CreateDelivery(ctx context.Context, d *ops.WebhookDelivery) error {
	return r.db.WithContext(ctx).Create(d).Error
}

// DueDeliveries 取到期待投递的记录。
func (r *OpsRepo) DueDeliveries(ctx context.Context, now time.Time, limitN int) ([]ops.WebhookDelivery, error) {
	var rows []ops.WebhookDelivery
	err := r.db.WithContext(ctx).Where("status = ? AND next_attempt_at <= ?", ops.DeliveryPending, now).
		Order("next_attempt_at").Limit(limitN).Find(&rows).Error
	return rows, err
}

func (r *OpsRepo) UpdateDelivery(ctx context.Context, d *ops.WebhookDelivery) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *OpsRepo) FindDelivery(ctx context.Context, merchantID, id uint64) (*ops.WebhookDelivery, error) {
	var row ops.WebhookDelivery
	return wrap(&row, r.db.WithContext(ctx).Where("merchant_id = ? AND id = ?", merchantID, id).First(&row).Error)
}

func (r *OpsRepo) ListDeliveries(ctx context.Context, merchantID, endpointID uint64, status string, page, size int) ([]ops.WebhookDelivery, int64, error) {
	q := r.db.WithContext(ctx).Model(&ops.WebhookDelivery{}).Where("merchant_id = ?", merchantID)
	if endpointID > 0 {
		q = q.Where("endpoint_id = ?", endpointID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ops.WebhookDelivery
	err := q.Order("id DESC").Offset(offset(page, size)).Limit(limit(size)).Find(&rows).Error
	return rows, total, err
}

func (r *OpsRepo) UpdateEndpointStatus(ctx context.Context, id uint64, status int, ok bool, at time.Time) error {
	upd := map[string]any{"last_status": status, "last_delivered_at": at}
	if ok {
		upd["failure_count"] = 0
	} else {
		upd["failure_count"] = gorm.Expr("failure_count + 1")
	}
	return r.db.WithContext(ctx).Model(&ops.WebhookEndpoint{}).Where("id = ?", id).Updates(upd).Error
}

// PurgeDeliveries 删除超期投递记录。
func (r *OpsRepo) PurgeDeliveries(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&ops.WebhookDelivery{})
	return res.RowsAffected, res.Error
}

// ---- 导出 ----

func (r *OpsRepo) CreateExport(ctx context.Context, e *ops.Export) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *OpsRepo) UpdateExport(ctx context.Context, e *ops.Export) error {
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *OpsRepo) FindExport(ctx context.Context, merchantID, id uint64) (*ops.Export, error) {
	var row ops.Export
	q := r.db.WithContext(ctx).Where("id = ?", id)
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	return wrap(&row, q.First(&row).Error)
}

func (r *OpsRepo) ListExports(ctx context.Context, merchantID uint64, page, size int) ([]ops.Export, int64, error) {
	q := r.db.WithContext(ctx).Model(&ops.Export{}).Where("merchant_id = ?", merchantID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ops.Export
	err := q.Order("id DESC").Offset(offset(page, size)).Limit(limit(size)).Find(&rows).Error
	return rows, total, err
}

// ClaimExport 取一个待处理导出并标记为 running。
func (r *OpsRepo) ClaimExport(ctx context.Context) (*ops.Export, error) {
	var row ops.Export
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", ops.ExportPending).Order("id").First(&row).Error; err != nil {
			return err
		}
		row.Status = ops.ExportRunning
		return tx.Model(&row).Update("status", ops.ExportRunning).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// ExpiredExports 到期的导出（删文件用）。
func (r *OpsRepo) ExpiredExports(ctx context.Context, now time.Time) ([]ops.Export, error) {
	var rows []ops.Export
	err := r.db.WithContext(ctx).Where("status = ? AND expires_at IS NOT NULL AND expires_at < ? AND file_path <> ''", ops.ExportDone, now).Limit(100).Find(&rows).Error
	return rows, err
}

// ---- 健康相关读取（告警评估） ----

// UnhealthyOrigins 不健康源站。
func (r *OpsRepo) UnhealthyOrigins(ctx context.Context) ([]map[string]any, error) {
	var rows []map[string]any
	err := r.db.WithContext(ctx).Table("site_origins o").Joins("JOIN sites s ON s.id = o.site_id").
		Where("o.health_status = 'unhealthy' AND s.status = 'enabled'").
		Select("o.id, o.site_id AS siteId, s.merchant_id AS merchantId, s.name AS siteName, o.kind, o.url, o.health_message AS message, o.last_health_at AS lastHealthAt").
		Scan(&rows).Error
	return rows, err
}

// HealthyOriginIDs 当前健康的源站 ID（用于自动恢复）。
func (r *OpsRepo) OriginHealth(ctx context.Context, id uint64) (string, error) {
	var status string
	err := r.db.WithContext(ctx).Table("site_origins").Where("id = ?", id).Select("health_status").Scan(&status).Error
	return status, err
}

// ExpiringCerts 到期证书。
func (r *OpsRepo) ExpiringCerts(ctx context.Context, before time.Time) ([]map[string]any, error) {
	var rows []map[string]any
	err := r.db.WithContext(ctx).Table("domains").Where("cert_expires_at IS NOT NULL AND cert_expires_at < ? AND status = 'bound'", before).
		Select("id, merchant_id AS merchantId, site_id AS siteId, domain, cert_expires_at AS certExpiresAt").Scan(&rows).Error
	return rows, err
}

// UnhealthyDomains 解析/校验异常域名。
func (r *OpsRepo) UnhealthyDomains(ctx context.Context) ([]map[string]any, error) {
	var rows []map[string]any
	err := r.db.WithContext(ctx).Table("domains").Where("health_status = 'unhealthy' AND status = 'bound'").
		Select("id, merchant_id AS merchantId, site_id AS siteId, domain, verify_message AS message").Scan(&rows).Error
	return rows, err
}

// DomainHealth 域名当前健康状态。
func (r *OpsRepo) DomainHealth(ctx context.Context, domain string) (string, error) {
	var status string
	err := r.db.WithContext(ctx).Table("domains").Where("domain = ?", domain).Select("health_status").Scan(&status).Error
	return status, err
}

// MerchantQuotas 所有启用商户的判定额度。
func (r *OpsRepo) MerchantQuotas(ctx context.Context) ([]struct {
	ID            uint64
	Name          string
	DecisionQuota uint64
	Status        string
}, error) {
	var rows []struct {
		ID            uint64
		Name          string
		DecisionQuota uint64
		Status        string
	}
	err := r.db.WithContext(ctx).Table("merchants").Where("status = 'enabled'").Select("id, name, decision_quota, status").Scan(&rows).Error
	return rows, err
}

// PlatformCounts 总后台计数。
func (r *OpsRepo) PlatformCounts(ctx context.Context) (map[string]int64, error) {
	out := map[string]int64{}
	type kv struct {
		K string
		V int64
	}
	var rows []kv
	if err := r.db.WithContext(ctx).Raw(`
SELECT 'merchants' k, COUNT(*) v FROM merchants
UNION ALL SELECT 'merchants_enabled', COUNT(*) FROM merchants WHERE status = 'enabled'
UNION ALL SELECT 'sites', COUNT(*) FROM sites
UNION ALL SELECT 'sites_enabled', COUNT(*) FROM sites WHERE status = 'enabled'
UNION ALL SELECT 'domains', COUNT(*) FROM domains
UNION ALL SELECT 'domains_bound', COUNT(*) FROM domains WHERE status = 'bound'
UNION ALL SELECT 'domains_unhealthy', COUNT(*) FROM domains WHERE health_status = 'unhealthy'
UNION ALL SELECT 'origins_unhealthy', COUNT(*) FROM site_origins WHERE health_status = 'unhealthy'
UNION ALL SELECT 'certs_expiring', COUNT(*) FROM domains WHERE cert_expires_at IS NOT NULL AND cert_expires_at < DATE_ADD(NOW(), INTERVAL 14 DAY)
UNION ALL SELECT 'codes', COUNT(*) FROM promo_codes
UNION ALL SELECT 'alerts_open', COUNT(*) FROM alerts WHERE status = 'open'`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.K] = x.V
	}
	return out, nil
}

// MerchantHealth 商户看板健康块。
func (r *OpsRepo) MerchantHealth(ctx context.Context, merchantID uint64) (map[string]any, error) {
	out := map[string]any{}
	var domains []map[string]any
	if err := r.db.WithContext(ctx).Table("domains").Where("merchant_id = ?", merchantID).
		Select("domain, status, health_status AS healthStatus, cert_status AS certStatus, cert_expires_at AS certExpiresAt").Scan(&domains).Error; err != nil {
		return nil, err
	}
	var origins []map[string]any
	if err := r.db.WithContext(ctx).Table("site_origins o").Joins("JOIN sites s ON s.id = o.site_id").Where("s.merchant_id = ?", merchantID).
		Select("o.site_id AS siteId, s.name AS siteName, o.kind, o.health_status AS healthStatus, o.health_message AS message, o.last_health_at AS lastHealthAt").
		Scan(&origins).Error; err != nil {
		return nil, err
	}
	out["domains"] = domains
	out["origins"] = origins
	return out, nil
}
