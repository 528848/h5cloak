// Package attribution 定义归因参数字典、站点校验规则与分销码。
package attribution

import "time"

const (
	TypeClick       = "click"
	TypeSource      = "source"
	TypeDistributor = "distributor"
	TypeCampaign    = "campaign"
	TypeFree        = "free"

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"

	CombineAny         = "require_any"
	CombineClickOrCode = "require_click_or_code"

	ReuseDowngrade = "downgrade"
	ReuseReview    = "review"

	RevokeWindow    = "window"
	RevokeImmediate = "immediate"

	CodeDistributor = "distributor"
	CodeCampaign    = "campaign"
)

// ValidTypes 参数类型集合。
var ValidTypes = []string{TypeClick, TypeSource, TypeDistributor, TypeCampaign, TypeFree}

// Param 参数定义。SiteID 为 0 表示平台内置。
type Param struct {
	ID         uint64 `gorm:"primaryKey"`
	SiteID     uint64
	Name       string
	ParamType  string
	Platform   *string
	Aliases    []byte `gorm:"type:json"`
	MinLen     uint32
	MaxLen     uint32
	Pattern    *string
	Sufficient bool
	Status     string
	Sort       uint32
	Remark     *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Param) TableName() string { return "attribution_params" }

// Rule 站点级校验规则。
type Rule struct {
	SiteID                uint64 `gorm:"primaryKey"`
	CombineRule           string
	ValidateCodes         bool
	ValidateClickFormat   bool
	AllowedSources        []byte `gorm:"type:json"`
	ClickReuseLimit       uint32
	ClickReuseWindowHours uint32
	ClickReuseAction      string
	CodeRevokeMode        string
	UpdatedAt             time.Time
}

func (Rule) TableName() string { return "site_attribution_rules" }

// DefaultRule 新站点的默认规则。
func DefaultRule(siteID uint64) *Rule {
	return &Rule{
		SiteID: siteID, CombineRule: CombineAny, ValidateCodes: true, ValidateClickFormat: true,
		AllowedSources: []byte("[]"), ClickReuseLimit: 0, ClickReuseWindowHours: 24,
		ClickReuseAction: ReuseDowngrade, CodeRevokeMode: RevokeWindow,
	}
}

// Code 分销码或活动码。
type Code struct {
	ID              uint64 `gorm:"primaryKey"`
	MerchantID      uint64
	SiteID          uint64
	Code            string
	CodeType        string
	Name            *string
	Channel         *string
	Status          string
	StartsAt        *time.Time
	ExpiresAt       *time.Time
	AllowedPaths    []byte `gorm:"type:json"`
	Remark          *string
	CreatedBy       *uint64
	VisitCount      uint64
	FirstPassCount  uint64
	ReturnCount     uint64
	ConversionCount uint64
	LastVisitAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (Code) TableName() string { return "promo_codes" }

// EffectiveStatus 结合有效期得到展示状态：enabled / disabled / expired / pending。
func (c *Code) EffectiveStatus(now time.Time) string {
	if c.Status == StatusDisabled {
		return StatusDisabled
	}
	if c.ExpiresAt != nil && now.After(*c.ExpiresAt) {
		return "expired"
	}
	if c.StartsAt != nil && now.Before(*c.StartsAt) {
		return "pending"
	}
	return StatusEnabled
}
