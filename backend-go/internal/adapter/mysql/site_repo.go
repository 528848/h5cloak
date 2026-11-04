package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"cloak/internal/domain/site"
)

// SiteRepo 站点及相关资源。
type SiteRepo struct{ db *gorm.DB }

func NewSiteRepo(db *gorm.DB) *SiteRepo { return &SiteRepo{db: db} }

func (r *SiteRepo) DB() *gorm.DB { return r.db }

func (r *SiteRepo) CreateSite(ctx context.Context, s *site.Site, rules []site.PathRule, review *site.ReviewContent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(s).Error; err != nil {
			return err
		}
		for i := range rules {
			rules[i].SiteID = s.ID
			rules[i].ID = 0
		}
		if len(rules) > 0 {
			if err := tx.Create(&rules).Error; err != nil {
				return err
			}
		}
		review.SiteID = s.ID
		return tx.Create(review).Error
	})
}

func (r *SiteRepo) UpdateSite(ctx context.Context, s *site.Site, fields ...string) error {
	q := r.db.WithContext(ctx).Model(s)
	if len(fields) > 0 {
		q = q.Select(fields)
	}
	return q.Updates(s).Error
}

func (r *SiteRepo) FindSite(ctx context.Context, id uint64) (*site.Site, error) {
	var s site.Site
	err := r.db.WithContext(ctx).First(&s, id).Error
	return wrap(&s, err)
}

func (r *SiteRepo) DeleteSite(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&site.Site{}, id).Error
}

func (r *SiteRepo) ListSites(ctx context.Context, merchantID uint64, keyword, status string, page, size int) ([]site.Site, int64, error) {
	q := r.db.WithContext(ctx).Model(&site.Site{})
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	if keyword != "" {
		q = q.Where("name LIKE ?", "%"+escapeLike(keyword)+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []site.Site
	err := q.Order("id DESC").Offset(offset(page, size)).Limit(limit(size)).Find(&rows).Error
	return rows, total, err
}

// SiteCounters 列表页附加信息：域名数、源站健康。
type SiteCounters struct {
	Domains      int64
	BoundDomains int64
	Origins      int64
	Unhealthy    int64
}

func (r *SiteRepo) SiteCounters(ctx context.Context, siteIDs []uint64) (map[uint64]*SiteCounters, error) {
	out := map[uint64]*SiteCounters{}
	if len(siteIDs) == 0 {
		return out, nil
	}
	get := func(id uint64) *SiteCounters {
		if out[id] == nil {
			out[id] = &SiteCounters{}
		}
		return out[id]
	}
	var drows []struct {
		SiteID uint64
		Status string
		N      int64
	}
	if err := r.db.WithContext(ctx).Model(&site.Domain{}).Select("site_id, status, COUNT(*) AS n").
		Where("site_id IN ?", siteIDs).Group("site_id, status").Scan(&drows).Error; err != nil {
		return nil, err
	}
	for _, d := range drows {
		c := get(d.SiteID)
		c.Domains += d.N
		if d.Status == site.DomainBound {
			c.BoundDomains += d.N
		}
	}
	var orows []struct {
		SiteID       uint64
		HealthStatus string
		N            int64
	}
	if err := r.db.WithContext(ctx).Model(&site.Origin{}).Select("site_id, health_status, COUNT(*) AS n").
		Where("site_id IN ?", siteIDs).Group("site_id, health_status").Scan(&orows).Error; err != nil {
		return nil, err
	}
	for _, o := range orows {
		c := get(o.SiteID)
		c.Origins += o.N
		if o.HealthStatus == site.HealthUnhealthy {
			c.Unhealthy += o.N
		}
	}
	return out, nil
}

// MerchantNames 批量取商户名称（总后台列表用）。
func (r *SiteRepo) MerchantNames(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID   uint64
		Name string
	}
	if err := r.db.WithContext(ctx).Table("merchants").Select("id, name").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, m := range rows {
		out[m.ID] = m.Name
	}
	return out, nil
}

// SiteStats 站点状态统计。
func (r *SiteRepo) SiteStats(ctx context.Context, merchantID uint64) (map[string]int64, error) {
	q := r.db.WithContext(ctx).Model(&site.Site{})
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	var rows []struct {
		Status string
		N      int64
	}
	if err := q.Select("status, COUNT(*) AS n").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Status] = r.N
	}
	return out, nil
}

// DomainStats 域名状态统计。
func (r *SiteRepo) DomainStats(ctx context.Context, merchantID uint64) (map[string]int64, error) {
	q := r.db.WithContext(ctx).Model(&site.Domain{})
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	var rows []struct {
		Status string
		N      int64
	}
	if err := q.Select("status, COUNT(*) AS n").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Status] = r.N
	}
	return out, nil
}

// SiteNames 站点 ID → 名称。
func (r *SiteRepo) SiteNames(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID   uint64
		Name string
	}
	if err := r.db.WithContext(ctx).Model(&site.Site{}).Select("id, name").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, s := range rows {
		out[s.ID] = s.Name
	}
	return out, nil
}

// SiteOptions 商户下的站点简表（下拉用）。
func (r *SiteRepo) SiteOptions(ctx context.Context, merchantID uint64) ([]site.Site, error) {
	var rows []site.Site
	err := r.db.WithContext(ctx).Select("id, name, status, site_type").Where("merchant_id = ?", merchantID).Order("id DESC").Find(&rows).Error
	return rows, err
}

func (r *SiteRepo) CountSites(ctx context.Context, merchantID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&site.Site{}).Where("merchant_id = ?", merchantID).Count(&n).Error
	return n, err
}

func (r *SiteRepo) FindByPrefix(ctx context.Context, prefix string) (*site.Site, error) {
	var s site.Site
	err := r.db.WithContext(ctx).Where("secret_prefix = ?", prefix).First(&s).Error
	return wrap(&s, err)
}

func (r *SiteRepo) UpsertOrigin(ctx context.Context, o *site.Origin) error {
	var existing site.Origin
	err := r.db.WithContext(ctx).Where("site_id = ? AND kind = ?", o.SiteID, o.Kind).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(o).Error
	}
	if err != nil {
		return err
	}
	o.ID = existing.ID
	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"url": o.URL, "host_header": o.HostHeader, "timeout_ms": o.TimeoutMs,
	}).Error
}

func (r *SiteRepo) ListOrigins(ctx context.Context, siteID uint64) ([]site.Origin, error) {
	var rows []site.Origin
	err := r.db.WithContext(ctx).Where("site_id = ?", siteID).Find(&rows).Error
	return rows, err
}

func (r *SiteRepo) UpdateOriginHealth(ctx context.Context, id uint64, status, message string, at interface{}) error {
	return r.db.WithContext(ctx).Model(&site.Origin{}).Where("id = ?", id).Updates(map[string]any{
		"health_status": status, "health_message": message, "last_health_at": at,
	}).Error
}

func (r *SiteRepo) ListOriginsForProbe(ctx context.Context, limitN int) ([]site.Origin, error) {
	var rows []site.Origin
	err := r.db.WithContext(ctx).Order("last_health_at IS NULL DESC, last_health_at ASC").Limit(limitN).Find(&rows).Error
	return rows, err
}

func (r *SiteRepo) SaveReview(ctx context.Context, c *site.ReviewContent) error {
	var existing site.ReviewContent
	err := r.db.WithContext(ctx).Where("site_id = ?", c.SiteID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(c).Error
	}
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"source": c.Source, "template_id": c.TemplateID, "template_vars": c.TemplateVars, "static_package_id": c.StaticPackageID,
	}).Error
}

func (r *SiteRepo) FindReview(ctx context.Context, siteID uint64) (*site.ReviewContent, error) {
	var c site.ReviewContent
	err := r.db.WithContext(ctx).Where("site_id = ?", siteID).First(&c).Error
	return wrap(&c, err)
}

func (r *SiteRepo) ListTemplates(ctx context.Context) ([]site.Template, error) {
	var rows []site.Template
	err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *SiteRepo) SaveTemplate(ctx context.Context, t *site.Template) error {
	if t.ID == 0 {
		return r.db.WithContext(ctx).Create(t).Error
	}
	return r.db.WithContext(ctx).Model(t).Select("Code", "Name", "Description", "HTML").Updates(t).Error
}

func (r *SiteRepo) TemplateInUse(ctx context.Context, id uint64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&site.ReviewContent{}).Where("template_id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *SiteRepo) DeleteTemplate(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&site.Template{}, id).Error
}

func (r *SiteRepo) FindTemplate(ctx context.Context, id uint64) (*site.Template, error) {
	var t site.Template
	err := r.db.WithContext(ctx).First(&t, id).Error
	return wrap(&t, err)
}

func (r *SiteRepo) NextPackageVersion(ctx context.Context, siteID uint64) (int, error) {
	var v int
	err := r.db.WithContext(ctx).Model(&site.StaticPackage{}).Where("site_id = ?", siteID).Select("COALESCE(MAX(version),0)").Scan(&v).Error
	return v + 1, err
}

func (r *SiteRepo) CreatePackage(ctx context.Context, p *site.StaticPackage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&site.StaticPackage{}).Where("site_id = ? AND status = ?", p.SiteID, "ready").Update("status", "archived").Error; err != nil {
			return err
		}
		return tx.Create(p).Error
	})
}

func (r *SiteRepo) FindPackage(ctx context.Context, id uint64) (*site.StaticPackage, error) {
	var p site.StaticPackage
	err := r.db.WithContext(ctx).First(&p, id).Error
	return wrap(&p, err)
}

func (r *SiteRepo) CreateDomain(ctx context.Context, d *site.Domain) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *SiteRepo) UpdateDomain(ctx context.Context, d *site.Domain, fields ...string) error {
	q := r.db.WithContext(ctx).Model(d)
	if len(fields) > 0 {
		q = q.Select(fields)
	}
	return q.Updates(d).Error
}

func (r *SiteRepo) FindDomain(ctx context.Context, id uint64) (*site.Domain, error) {
	var d site.Domain
	err := r.db.WithContext(ctx).First(&d, id).Error
	return wrap(&d, err)
}

func (r *SiteRepo) DeleteDomain(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&site.Domain{}, id).Error
}

func (r *SiteRepo) DomainExists(ctx context.Context, domain string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&site.Domain{}).Where("domain = ?", domain).Count(&n).Error
	return n > 0, err
}

func (r *SiteRepo) CountDomains(ctx context.Context, merchantID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&site.Domain{}).Where("merchant_id = ?", merchantID).Count(&n).Error
	return n, err
}

func (r *SiteRepo) ListDomains(ctx context.Context, merchantID, siteID uint64, keyword, status string, page, size int) ([]site.Domain, int64, error) {
	q := r.db.WithContext(ctx).Model(&site.Domain{})
	if merchantID > 0 {
		q = q.Where("merchant_id = ?", merchantID)
	}
	if siteID > 0 {
		q = q.Where("site_id = ?", siteID)
	}
	if keyword != "" {
		q = q.Where("domain LIKE ?", "%"+escapeLike(keyword)+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []site.Domain
	err := q.Order("id DESC").Offset(offset(page, size)).Limit(limit(size)).Find(&rows).Error
	return rows, total, err
}

func (r *SiteRepo) ListPathRules(ctx context.Context, siteID uint64) ([]site.PathRule, error) {
	var rows []site.PathRule
	err := r.db.WithContext(ctx).Where("site_id = ?", siteID).Order("sort ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *SiteRepo) ReplacePathRules(ctx context.Context, siteID uint64, rules []site.PathRule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id = ?", siteID).Delete(&site.PathRule{}).Error; err != nil {
			return err
		}
		if len(rules) == 0 {
			return nil
		}
		for i := range rules {
			rules[i].ID = 0
			rules[i].SiteID = siteID
		}
		return tx.Create(&rules).Error
	})
}
