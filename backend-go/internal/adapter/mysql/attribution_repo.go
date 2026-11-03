package mysql

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cloak/internal/domain/attribution"
)

// AttributionRepo 参数字典、站点规则、分销码。
type AttributionRepo struct{ db *gorm.DB }

func NewAttributionRepo(db *gorm.DB) *AttributionRepo { return &AttributionRepo{db: db} }

// ---- 参数字典 ----

// ListParams 返回内置（site_id=0）与指定站点的参数；siteID 为 0 只返回内置。
func (r *AttributionRepo) ListParams(ctx context.Context, siteID uint64) ([]attribution.Param, error) {
	var rows []attribution.Param
	q := r.db.WithContext(ctx)
	if siteID > 0 {
		q = q.Where("site_id IN ?", []uint64{0, siteID})
	} else {
		q = q.Where("site_id = 0")
	}
	err := q.Order("sort ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *AttributionRepo) FindParam(ctx context.Context, id uint64) (*attribution.Param, error) {
	var p attribution.Param
	err := r.db.WithContext(ctx).First(&p, id).Error
	return wrap(&p, err)
}

func (r *AttributionRepo) FindParamByName(ctx context.Context, siteID uint64, name string) (*attribution.Param, error) {
	var p attribution.Param
	err := r.db.WithContext(ctx).Where("site_id = ? AND name = ?", siteID, name).First(&p).Error
	return wrap(&p, err)
}

func (r *AttributionRepo) CreateParam(ctx context.Context, p *attribution.Param) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *AttributionRepo) UpdateParam(ctx context.Context, p *attribution.Param, fields ...string) error {
	q := r.db.WithContext(ctx).Model(p)
	if len(fields) > 0 {
		q = q.Select(fields)
	}
	return q.Updates(p).Error
}

func (r *AttributionRepo) DeleteParam(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&attribution.Param{}, id).Error
}

// ---- 站点规则 ----

func (r *AttributionRepo) FindRule(ctx context.Context, siteID uint64) (*attribution.Rule, error) {
	var rule attribution.Rule
	err := r.db.WithContext(ctx).Where("site_id = ?", siteID).First(&rule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return attribution.DefaultRule(siteID), nil
	}
	return wrap(&rule, err)
}

func (r *AttributionRepo) SaveRule(ctx context.Context, rule *attribution.Rule) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "site_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"combine_rule", "validate_codes", "validate_click_format", "allowed_sources",
			"click_reuse_limit", "click_reuse_window_hours", "click_reuse_action", "code_revoke_mode",
		}),
	}).Create(rule).Error
}

// ---- 分销码 ----

// CodeFilter 分销码列表筛选。
type CodeFilter struct {
	MerchantID uint64
	SiteID     uint64
	Keyword    string
	CodeType   string
	Status     string // enabled / disabled / expired
	Channel    string
	Page, Size int
}

func (r *AttributionRepo) ListCodes(ctx context.Context, f CodeFilter) ([]attribution.Code, int64, error) {
	q := r.codeQuery(ctx, f)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []attribution.Code
	err := q.Order("id DESC").Offset(offset(f.Page, f.Size)).Limit(limit(f.Size)).Find(&rows).Error
	return rows, total, err
}

// AllCodes 导出用，最多 50000 条。
func (r *AttributionRepo) AllCodes(ctx context.Context, f CodeFilter) ([]attribution.Code, error) {
	var rows []attribution.Code
	err := r.codeQuery(ctx, f).Order("id ASC").Limit(50000).Find(&rows).Error
	return rows, err
}

func (r *AttributionRepo) codeQuery(ctx context.Context, f CodeFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&attribution.Code{})
	if f.MerchantID > 0 {
		q = q.Where("merchant_id = ?", f.MerchantID)
	}
	if f.SiteID > 0 {
		q = q.Where("site_id = ?", f.SiteID)
	}
	if k := strings.TrimSpace(f.Keyword); k != "" {
		like := "%" + escapeLike(k) + "%"
		q = q.Where("code LIKE ? OR name LIKE ? OR channel LIKE ?", like, like, like)
	}
	if f.CodeType != "" {
		q = q.Where("code_type = ?", f.CodeType)
	}
	if f.Channel != "" {
		q = q.Where("channel = ?", f.Channel)
	}
	now := time.Now()
	switch f.Status {
	case attribution.StatusEnabled:
		q = q.Where("status = ? AND (expires_at IS NULL OR expires_at > ?)", attribution.StatusEnabled, now)
	case attribution.StatusDisabled:
		q = q.Where("status = ?", attribution.StatusDisabled)
	case "expired":
		q = q.Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", attribution.StatusEnabled, now)
	}
	return q
}

func (r *AttributionRepo) FindCode(ctx context.Context, id uint64) (*attribution.Code, error) {
	var c attribution.Code
	err := r.db.WithContext(ctx).First(&c, id).Error
	return wrap(&c, err)
}

func (r *AttributionRepo) FindCodeByValue(ctx context.Context, siteID uint64, code string) (*attribution.Code, error) {
	var c attribution.Code
	err := r.db.WithContext(ctx).Where("site_id = ? AND code = ?", siteID, code).First(&c).Error
	return wrap(&c, err)
}

func (r *AttributionRepo) CreateCode(ctx context.Context, c *attribution.Code) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// CreateCodes 批量插入，忽略已存在的码；返回实际插入条数。
func (r *AttributionRepo) CreateCodes(ctx context.Context, rows []attribution.Code) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, 500)
	return res.RowsAffected, res.Error
}

func (r *AttributionRepo) UpdateCode(ctx context.Context, c *attribution.Code, fields ...string) error {
	q := r.db.WithContext(ctx).Model(c)
	if len(fields) > 0 {
		q = q.Select(fields)
	}
	return q.Updates(c).Error
}

func (r *AttributionRepo) DeleteCode(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&attribution.Code{}, id).Error
}

// DeleteCodes 批量删除，限定商户。
func (r *AttributionRepo) DeleteCodes(ctx context.Context, merchantID uint64, ids []uint64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Where("merchant_id = ? AND id IN ?", merchantID, ids).Delete(&attribution.Code{})
	return res.RowsAffected, res.Error
}

// UpdateCodesStatus 批量启停，限定商户。
func (r *AttributionRepo) UpdateCodesStatus(ctx context.Context, merchantID uint64, ids []uint64, status string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Model(&attribution.Code{}).
		Where("merchant_id = ? AND id IN ?", merchantID, ids).Update("status", status)
	return res.RowsAffected, res.Error
}

func (r *AttributionRepo) CountCodes(ctx context.Context, merchantID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&attribution.Code{}).Where("merchant_id = ?", merchantID).Count(&n).Error
	return n, err
}

// ExistingCodes 返回给定码值中已存在的集合。
func (r *AttributionRepo) ExistingCodes(ctx context.Context, siteID uint64, codes []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(codes) == 0 {
		return out, nil
	}
	var rows []string
	if err := r.db.WithContext(ctx).Model(&attribution.Code{}).
		Where("site_id = ? AND code IN ?", siteID, codes).Pluck("code", &rows).Error; err != nil {
		return nil, err
	}
	for _, c := range rows {
		out[c] = true
	}
	return out, nil
}

// CodeStats 按站点（或商户）统计。
type CodeStats struct {
	Total    int64
	Enabled  int64
	Disabled int64
	Expired  int64
	Visits   int64
}

func (r *AttributionRepo) CodeStats(ctx context.Context, merchantID, siteID uint64) (*CodeStats, error) {
	base := func() *gorm.DB {
		q := r.db.WithContext(ctx).Model(&attribution.Code{}).Where("merchant_id = ?", merchantID)
		if siteID > 0 {
			q = q.Where("site_id = ?", siteID)
		}
		return q
	}
	now := time.Now()
	s := &CodeStats{}
	if err := base().Count(&s.Total).Error; err != nil {
		return nil, err
	}
	if err := base().Where("status = ? AND (expires_at IS NULL OR expires_at > ?)", attribution.StatusEnabled, now).Count(&s.Enabled).Error; err != nil {
		return nil, err
	}
	if err := base().Where("status = ?", attribution.StatusDisabled).Count(&s.Disabled).Error; err != nil {
		return nil, err
	}
	if err := base().Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", attribution.StatusEnabled, now).Count(&s.Expired).Error; err != nil {
		return nil, err
	}
	var visits struct{ N int64 }
	if err := base().Select("COALESCE(SUM(visit_count),0) AS n").Scan(&visits).Error; err != nil {
		return nil, err
	}
	s.Visits = visits.N
	return s, nil
}

// Channels 商户下出现过的渠道标签。
func (r *AttributionRepo) Channels(ctx context.Context, merchantID uint64) ([]string, error) {
	var rows []string
	err := r.db.WithContext(ctx).Model(&attribution.Code{}).
		Where("merchant_id = ? AND channel IS NOT NULL AND channel <> ''", merchantID).
		Distinct().Order("channel").Pluck("channel", &rows).Error
	return rows, err
}
