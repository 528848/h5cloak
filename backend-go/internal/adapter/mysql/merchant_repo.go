package mysql

import (
	"context"

	"gorm.io/gorm"

	"cloak/internal/domain/identity"
)

// MerchantRepo 实现 identity.MerchantRepository。
type MerchantRepo struct{ db *gorm.DB }

func NewMerchantRepo(db *gorm.DB) *MerchantRepo { return &MerchantRepo{db: db} }

func (r *MerchantRepo) FindByID(ctx context.Context, id uint64) (*identity.Merchant, error) {
	var m identity.Merchant
	err := r.db.WithContext(ctx).First(&m, id).Error
	return wrap(&m, err)
}

func (r *MerchantRepo) Create(ctx context.Context, m *identity.Merchant) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *MerchantRepo) Update(ctx context.Context, m *identity.Merchant, fields ...string) error {
	q := r.db.WithContext(ctx).Model(m)
	if len(fields) > 0 {
		q = q.Select(fields)
	}
	return q.Updates(m).Error
}

func (r *MerchantRepo) List(ctx context.Context, f identity.MerchantFilter) ([]identity.Merchant, int64, error) {
	q := r.db.WithContext(ctx).Model(&identity.Merchant{})
	if f.Keyword != "" {
		like := "%" + escapeLike(f.Keyword) + "%"
		q = q.Where("name LIKE ? OR contact_name LIKE ? OR contact_email LIKE ?", like, like, like)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []identity.Merchant
	err := q.Order("id DESC").Offset(offset(f.Page, f.Size)).Limit(limit(f.Size)).Find(&rows).Error
	return rows, total, err
}

func (r *MerchantRepo) CountByStatus(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Status string
		C      int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).Model(&identity.Merchant{}).
		Select("status, COUNT(*) AS c").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, x := range rows {
		out[x.Status] = x.C
	}
	return out, nil
}

func (r *MerchantRepo) NameExists(ctx context.Context, name string, excludeID uint64) (bool, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&identity.Merchant{}).Where("name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&n).Error
	return n > 0, err
}

func escapeLike(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '%' || r == '_' || r == '\\' {
			out = append(out, '\\')
		}
		out = append(out, r)
	}
	return string(out)
}

func offset(page, size int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * limit(size)
}

func limit(size int) int {
	if size <= 0 {
		return 10
	}
	if size > 200 {
		return 200
	}
	return size
}
