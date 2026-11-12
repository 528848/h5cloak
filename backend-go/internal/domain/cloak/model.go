// Package cloak 定义跳转斗篷，和站点反向代理分开。
package cloak

import "time"

const (
	ModeCloak    = "cloak"
	ModeGreen    = "green"
	ModeFullOpen = "full_open"
	ModeReview   = "review"

	ReviewExternal = "external"
	ReviewLocal    = "local_template"

	PageNone   = "none"
	PageSystem = "system_template"
	PageCustom = "custom_template"

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)

// Cloak 一条斗篷。
type Cloak struct {
	ID                 uint64 `gorm:"primaryKey"`
	MerchantID         uint64
	DomainID           uint64
	Name               string
	Status             string
	RealURLs           string `gorm:"column:real_urls"`
	ReviewMode         string
	ReviewURL          string `gorm:"column:review_url"`
	ReviewTag          string
	Code               string
	CodeCustom         bool
	AllowedSources     string
	Mode               string
	TargetRegions      string
	SpiderWhitelist    string
	DeviceWhitelist    string
	BlockEmptyLanguage bool
	BlockPC            bool
	BlockProxy         bool
	IntermediatePage   string
	IntermediateText   string
	Remark             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (Cloak) TableName() string { return "cloaks" }

// AccessLog 短链访问记录。
type AccessLog struct {
	ID          uint64 `gorm:"primaryKey"`
	MerchantID  *uint64
	CloakID     *uint64
	Code        string
	Country     string
	Device      string
	HitRule     string
	FinalAction string
	LatencyMs   uint32
	CreatedAt   time.Time
}

func (AccessLog) TableName() string { return "cloak_access_logs" }
