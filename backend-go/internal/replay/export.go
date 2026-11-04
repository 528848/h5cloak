package replay

import (
	"context"
	"time"

	"gorm.io/gorm"

	"cloak/internal/domain/decisionlog"
)

// ExportFilter 从 decision_logs 导出的筛选条件。
type ExportFilter struct {
	SiteID     uint64
	MerchantID uint64
	Entry      string // 默认 hosted；其他入口没有可直连的域名，回放会报 host_unbound
	From, To   *time.Time
	Limit      int
}

// Export 读取决策日志生成用例，返回用例与被跳过的条数（没有 Host 的记录）。
func Export(ctx context.Context, db *gorm.DB, f ExportFilter) ([]Case, int, error) {
	q := db.WithContext(ctx).Model(&decisionlog.Record{})
	if f.SiteID > 0 {
		q = q.Where("site_id = ?", f.SiteID)
	}
	if f.MerchantID > 0 {
		q = q.Where("merchant_id = ?", f.MerchantID)
	}
	if f.Entry != "" {
		q = q.Where("entry = ?", f.Entry)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at < ?", *f.To)
	}
	if f.Limit <= 0 {
		f.Limit = 10000
	}
	var rows []decisionlog.Record
	if err := q.Order("id ASC").Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]Case, 0, len(rows))
	skipped := 0
	for i := range rows {
		if rows[i].Host == "" {
			skipped++
			continue
		}
		out = append(out, FromRecord(&rows[i]))
	}
	return out, skipped, nil
}
