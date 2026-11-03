// Package site 定义站点、域名、源站、审核内容和路径规则。
package site

import "time"

const (
	TypeHosted = "hosted"
	TypeServer = "server"
	TypeHybrid = "hybrid"

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"

	ActionReview   = "review"
	ActionBlock    = "block"
	ActionNotFound = "not_found"
	ActionReal     = "real"
	ActionMaintain = "maintenance"

	RenewFixed   = "fixed"
	RenewSliding = "sliding"

	OriginReal   = "real"
	OriginReview = "review"

	ReviewOrigin   = "origin"
	ReviewTemplate = "template"
	ReviewStatic   = "static"

	DomainHosted = "hosted"
	DomainOwn    = "own"

	DomainPending  = "pending"
	DomainBound    = "bound"
	DomainAbnormal = "abnormal"
	DomainDisabled = "disabled"

	HealthUnknown   = "unknown"
	HealthHealthy   = "healthy"
	HealthUnhealthy = "unhealthy"

	MatchPrefix = "prefix"
	MatchExact  = "exact"
	MatchSuffix = "suffix"

	PathBypass       = "bypass"
	PathAlwaysReal   = "always_real"
	PathAlwaysReview = "always_review"
	PathBlock        = "block"
)

type Site struct {
	ID                   uint64 `gorm:"primaryKey"`
	MerchantID           uint64
	Name                 string
	SiteType             string
	Status               string
	Remark               *string
	PublishableKey       string
	SecretPrefix         string
	SecretHash           string
	SecretIPAllowlist    []byte `gorm:"column:secret_ip_allowlist;type:json"`
	SecretExpiresAt      *time.Time
	CookieName           string
	CookieDomain         *string
	AttributionTTLDays   int
	RenewMode            string
	NoAttributionAction  string
	FailMode             string
	MinConfidence        string
	HandoffBanner        bool
	HandoffBannerText    string
	HandoffLinkRewrite   bool
	HandoffTokenTTLHours int
	HandoffTokenMaxUses  int
	HandoffMatchEnabled  bool
	HandoffMatchMinutes  int
	TokenKeyVersion      int
	TokenKeys            []byte `gorm:"type:json"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (Site) TableName() string { return "sites" }

type Origin struct {
	ID            uint64 `gorm:"primaryKey"`
	SiteID        uint64
	Kind          string
	URL           string `gorm:"column:url"`
	HostHeader    *string
	TimeoutMs     int
	HealthStatus  string
	HealthMessage *string
	LastHealthAt  *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Origin) TableName() string { return "site_origins" }

type ReviewContent struct {
	ID              uint64 `gorm:"primaryKey"`
	SiteID          uint64
	Source          string
	TemplateID      *uint64
	TemplateVars    []byte `gorm:"type:json"`
	StaticPackageID *uint64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (ReviewContent) TableName() string { return "site_review_contents" }

type Template struct {
	ID          uint64 `gorm:"primaryKey"`
	Code        string
	Name        string
	Description *string
	HTML        string `gorm:"column:html"`
	CreatedAt   time.Time
}

func (Template) TableName() string { return "review_templates" }

type StaticPackage struct {
	ID        uint64 `gorm:"primaryKey"`
	SiteID    uint64
	Version   int
	ObjectKey string
	Entry     string
	Status    string
	CreatedAt time.Time
}

func (StaticPackage) TableName() string { return "static_packages" }

type Domain struct {
	ID            uint64 `gorm:"primaryKey"`
	MerchantID    uint64
	SiteID        uint64
	Domain        string
	DomainType    string
	VerifyType    string
	VerifyValue   string
	Status        string
	VerifyMessage *string
	LastCheckedAt *time.Time
	CertStatus    string
	CertExpiresAt *time.Time
	HealthStatus  string
	Remark        *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Domain) TableName() string { return "domains" }

type PathRule struct {
	ID        uint64    `gorm:"primaryKey" json:"-"`
	SiteID    uint64    `json:"-"`
	Pattern   string    `json:"pattern"`
	MatchType string    `json:"matchType"`
	Action    string    `json:"action"`
	Sort      int       `json:"sort"`
	Remark    *string   `json:"remark"`
	CreatedAt time.Time `json:"-"`
}

func (PathRule) TableName() string { return "site_path_rules" }

// DefaultPathRules 新建站点时的默认路径规则。
func DefaultPathRules() []PathRule {
	type item struct {
		pattern, match, action, remark string
	}
	items := []item{
		{".css", MatchSuffix, PathBypass, "静态资源"},
		{".js", MatchSuffix, PathBypass, "静态资源"},
		{".png", MatchSuffix, PathBypass, "静态资源"},
		{".jpg", MatchSuffix, PathBypass, "静态资源"},
		{".jpeg", MatchSuffix, PathBypass, "静态资源"},
		{".gif", MatchSuffix, PathBypass, "静态资源"},
		{".webp", MatchSuffix, PathBypass, "静态资源"},
		{".svg", MatchSuffix, PathBypass, "静态资源"},
		{".ico", MatchSuffix, PathBypass, "静态资源"},
		{".woff", MatchSuffix, PathBypass, "静态资源"},
		{".woff2", MatchSuffix, PathBypass, "静态资源"},
		{".map", MatchSuffix, PathBypass, "静态资源"},
		{"/wp-admin", MatchPrefix, PathBypass, "WordPress 后台"},
		{"/wp-login.php", MatchExact, PathBypass, "WordPress 登录"},
		{"/wp-json", MatchPrefix, PathBypass, "WordPress REST"},
		{"/wp-cron.php", MatchExact, PathBypass, "WordPress 定时任务"},
		{"/robots.txt", MatchExact, PathBypass, "爬虫协议"},
		{"/sitemap.xml", MatchExact, PathBypass, "站点地图"},
		{"/favicon.ico", MatchExact, PathBypass, "图标"},
		{"/privacy", MatchPrefix, PathAlwaysReal, "隐私政策"},
		{"/terms", MatchPrefix, PathAlwaysReal, "服务条款"},
	}
	out := make([]PathRule, 0, len(items))
	for i, it := range items {
		remark := it.remark
		out = append(out, PathRule{Pattern: it.pattern, MatchType: it.match, Action: it.action, Sort: (i + 1) * 10, Remark: &remark})
	}
	return out
}
