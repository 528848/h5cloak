package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"cloak/internal/domain/cloak"
)

type CloakRepo struct{ db *gorm.DB }

func NewCloakRepo(db *gorm.DB) *CloakRepo { return &CloakRepo{db: db} }

type CloakFilter struct {
	MerchantID uint64
	Keyword    string
	DomainID   uint64
	Mode       string
	Status     string
	Page, Size int
}

type CloakRow struct {
	cloak.Cloak
	Domain string
}

func (r *CloakRepo) List(ctx context.Context, f CloakFilter) ([]CloakRow, int64, error) {
	q := r.db.WithContext(ctx).Table("cloaks c").
		Select("c.*, d.domain").
		Joins("JOIN domains d ON d.id = c.domain_id").
		Where("c.merchant_id = ?", f.MerchantID)
	if f.Keyword != "" {
		q = q.Where("c.name LIKE ?", "%"+f.Keyword+"%")
	}
	if f.DomainID > 0 {
		q = q.Where("c.domain_id = ?", f.DomainID)
	}
	if f.Mode != "" {
		q = q.Where("c.mode = ?", f.Mode)
	}
	if f.Status != "" {
		q = q.Where("c.status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	var rows []CloakRow
	err := q.Order("c.id DESC").Offset((f.Page - 1) * f.Size).Limit(f.Size).Scan(&rows).Error
	return rows, total, err
}

func (r *CloakRepo) CountStatus(ctx context.Context, merchantID uint64) (total, enabled, disabled int64, err error) {
	type row struct {
		Status string
		N      int64
	}
	var rows []row
	err = r.db.WithContext(ctx).Model(&cloak.Cloak{}).
		Select("status, COUNT(*) AS n").
		Where("merchant_id = ?", merchantID).
		Group("status").Scan(&rows).Error
	if err != nil {
		return 0, 0, 0, err
	}
	for _, item := range rows {
		total += item.N
		switch item.Status {
		case cloak.StatusEnabled:
			enabled = item.N
		case cloak.StatusDisabled:
			disabled = item.N
		}
	}
	return total, enabled, disabled, nil
}

func (r *CloakRepo) CountTodayJumps(ctx context.Context, merchantID uint64, start time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&cloak.AccessLog{}).
		Where("merchant_id = ? AND created_at >= ? AND final_action IN ?", merchantID, start, []string{"real", "review"}).
		Count(&n).Error
	return n, err
}

type HitStats struct {
	CloakID     uint64
	ReviewHits  int64
	RealHits    int64
	TodayReview int64
	TodayReal   int64
}

func (r *CloakRepo) HitStats(ctx context.Context, ids []uint64, today time.Time) (map[uint64]HitStats, error) {
	out := map[uint64]HitStats{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []HitStats
	err := r.db.WithContext(ctx).Raw(`
SELECT cloak_id,
  SUM(final_action = 'review') AS review_hits,
  SUM(final_action = 'real') AS real_hits,
  SUM(final_action = 'review' AND created_at >= ?) AS today_review,
  SUM(final_action = 'real' AND created_at >= ?) AS today_real
FROM cloak_access_logs
WHERE cloak_id IN ?
GROUP BY cloak_id`, today, today, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.CloakID] = row
	}
	return out, nil
}

func (r *CloakRepo) Find(ctx context.Context, merchantID, id uint64) (*CloakRow, error) {
	var row CloakRow
	err := r.db.WithContext(ctx).Table("cloaks c").
		Select("c.*, d.domain").
		Joins("JOIN domains d ON d.id = c.domain_id").
		Where("c.id = ? AND c.merchant_id = ?", id, merchantID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (r *CloakRepo) Create(ctx context.Context, row *cloak.Cloak) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *CloakRepo) Save(ctx context.Context, row *cloak.Cloak) error {
	return r.db.WithContext(ctx).Model(row).Select(
		"Name", "Status", "RealURLs", "ReviewMode", "ReviewURL", "ReviewTag", "Code", "CodeCustom",
		"AllowedSources", "Mode", "TargetRegions", "SpiderWhitelist", "DeviceWhitelist",
		"BlockEmptyLanguage", "BlockPC", "BlockProxy", "IntermediatePage", "IntermediateText", "Remark",
	).Updates(row).Error
}

func (r *CloakRepo) Delete(ctx context.Context, merchantID, id uint64) error {
	res := r.db.WithContext(ctx).Where("id = ? AND merchant_id = ?", id, merchantID).Delete(&cloak.Cloak{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CloakRepo) Count(ctx context.Context, merchantID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&cloak.Cloak{}).Where("merchant_id = ?", merchantID).Count(&n).Error
	return n, err
}

func (r *CloakRepo) CodeTaken(ctx context.Context, code string, exceptID uint64) (bool, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&cloak.Cloak{}).Where("code = ?", code)
	if exceptID > 0 {
		q = q.Where("id <> ?", exceptID)
	}
	err := q.Count(&n).Error
	return n > 0, err
}

type DomainOption struct {
	ID     uint64 `json:"id"`
	Domain string `json:"domain"`
}

// DomainOptions 可选域名：已绑定、站点不是托管或混合、没有被其他斗篷使用。
func (r *CloakRepo) DomainOptions(ctx context.Context, merchantID, keepID uint64, includeTaken bool) ([]DomainOption, error) {
	q := `
SELECT d.id, d.domain
FROM domains d
JOIN sites s ON s.id = d.site_id
WHERE d.merchant_id = ? AND d.status = 'bound' AND s.site_type = 'server'`
	args := []any{merchantID}
	if !includeTaken {
		q += ` AND NOT EXISTS (SELECT 1 FROM cloaks c WHERE c.domain_id = d.id AND c.id <> ?)`
		args = append(args, keepID)
	}
	q += ` ORDER BY d.domain`
	var rows []DomainOption
	err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error
	return rows, err
}

type DomainUse struct {
	ID         uint64
	Domain     string
	Status     string
	MerchantID uint64
	SiteType   string
}

func (r *CloakRepo) DomainUse(ctx context.Context, id uint64) (*DomainUse, error) {
	var row DomainUse
	err := r.db.WithContext(ctx).Raw(`
SELECT d.id, d.domain, d.status, d.merchant_id, s.site_type
FROM domains d
JOIN sites s ON s.id = d.site_id
WHERE d.id = ?`, id).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}

// GatewayCloak 网关内存表的一行。
type GatewayCloak struct {
	cloak.Cloak
	Domain         string
	DomainStatus   string
	MerchantStatus string
	ExpireAt       *time.Time
}

func (r *CloakRepo) ListForGateway(ctx context.Context) ([]GatewayCloak, error) {
	var rows []GatewayCloak
	err := r.db.WithContext(ctx).Raw(`
SELECT c.*, d.domain, d.status AS domain_status, m.status AS merchant_status, m.expire_at
FROM cloaks c
JOIN domains d ON d.id = c.domain_id
JOIN merchants m ON m.id = c.merchant_id`).Scan(&rows).Error
	return rows, err
}

func (r *CloakRepo) InsertLog(ctx context.Context, row *cloak.AccessLog) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *CloakRepo) PurgeLogs(ctx context.Context, merchantID uint64, before time.Time, batch int) (int64, error) {
	var total int64
	for {
		res := r.db.WithContext(ctx).Exec(
			"DELETE FROM cloak_access_logs WHERE merchant_id = ? AND created_at < ? LIMIT ?",
			merchantID, before, batch,
		)
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < int64(batch) {
			return total, nil
		}
	}
}
