package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// SystemSetting 总后台单例配置，id 固定为 1。JSON 列为 NULL 时表示沿用代码内置值。
type SystemSetting struct {
	ID                 uint8 `gorm:"primaryKey"`
	EmergencyReview    bool
	EntryHosted        bool
	EntryAPIV2         bool `gorm:"column:entry_api_v2"`
	EntryAPIV1         bool `gorm:"column:entry_api_v1"`
	EntryJS            bool `gorm:"column:entry_js"`
	LogRetentionDays   int
	RatePerKey         int
	RatePerSite        int
	WebhookMaxAttempts int
	AdReferers         []byte `gorm:"column:ad_referers"`
	InAppMarkers       []byte `gorm:"column:inapp_markers"`
	DefaultPathRules   []byte `gorm:"column:default_path_rules"`
	UpdatedAt          time.Time
}

func (SystemSetting) TableName() string { return "system_settings" }

// PlatformRepo 系统配置读写。
type PlatformRepo struct{ db *gorm.DB }

func NewPlatformRepo(db *gorm.DB) *PlatformRepo { return &PlatformRepo{db: db} }

func (r *PlatformRepo) Settings(ctx context.Context) (*SystemSetting, error) {
	var row SystemSetting
	err := r.db.WithContext(ctx).First(&row, 1).Error
	return &row, err
}

func (r *PlatformRepo) SaveSettings(ctx context.Context, fields map[string]any) error {
	fields["updated_at"] = time.Now()
	return r.db.WithContext(ctx).Model(&SystemSetting{}).Where("id = ?", 1).Updates(fields).Error
}

func (r *PlatformRepo) RetentionDays(ctx context.Context) (int, error) {
	var n int
	err := r.db.WithContext(ctx).Model(&SystemSetting{}).Where("id = ?", 1).Select("log_retention_days").Scan(&n).Error
	return n, err
}

func (r *PlatformRepo) WebhookAttempts(ctx context.Context) (int, error) {
	var n int
	err := r.db.WithContext(ctx).Model(&SystemSetting{}).Where("id = ?", 1).Select("webhook_max_attempts").Scan(&n).Error
	return n, err
}
