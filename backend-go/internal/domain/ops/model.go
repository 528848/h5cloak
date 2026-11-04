// Package ops 看板聚合、告警、通知、Webhook、导出的领域模型。
package ops

import "time"

// MetricsHourly 站点小时聚合。
type MetricsHourly struct {
	MerchantID       uint64    `json:"merchantId"`
	SiteID           uint64    `json:"siteId"`
	Hour             time.Time `json:"hour"`
	Visits           uint64    `json:"visits"`
	RealTotal        uint64    `json:"realTotal"`
	FirstReal        uint64    `json:"firstReal"`
	ReturnReal       uint64    `json:"returnReal"`
	HandoffReal      uint64    `json:"handoffReal"`
	Review           uint64    `json:"review"`
	Block            uint64    `json:"block"`
	NotFound         uint64    `json:"notFound"`
	Bypass           uint64    `json:"bypass"`
	ValidationFailed uint64    `json:"validationFailed"`
	InApp            uint64    `json:"inApp"`
	Degraded         uint64    `json:"degraded"`
	OriginErrors     uint64    `json:"originErrors"`
	LatencySum       uint64    `json:"latencySum"`
	Conversions      uint64    `json:"conversions"`
	ConversionAmount float64   `json:"conversionAmount"`
}

func (MetricsHourly) TableName() string { return "metrics_hourly" }

// CodeStatsDaily 分销码 / 活动码日统计。
type CodeStatsDaily struct {
	SiteID           uint64    `json:"siteId"`
	CodeID           uint64    `json:"codeId"`
	Day              time.Time `json:"day"`
	MerchantID       uint64    `json:"merchantId"`
	Code             string    `json:"code"`
	Channel          string    `json:"channel"`
	Visits           uint64    `json:"visits"`
	FirstVisits      uint64    `json:"firstVisits"`
	ReturnVisits     uint64    `json:"returnVisits"`
	HandoffVisits    uint64    `json:"handoffVisits"`
	InApp            uint64    `json:"inApp"`
	Conversions      uint64    `json:"conversions"`
	ConversionAmount float64   `json:"conversionAmount"`
}

func (CodeStatsDaily) TableName() string { return "code_stats_daily" }

// PathStatsDaily 路径日统计。
type PathStatsDaily struct {
	SiteID        uint64    `json:"siteId"`
	Day           time.Time `json:"day"`
	Path          string    `json:"path"`
	MerchantID    uint64    `json:"merchantId"`
	Visits        uint64    `json:"visits"`
	RealTotal     uint64    `json:"realTotal"`
	Review        uint64    `json:"review"`
	Block         uint64    `json:"block"`
	NoAttribution uint64    `json:"noAttribution"`
}

func (PathStatsDaily) TableName() string { return "path_stats_daily" }

// 告警类型。
const (
	AlertOriginUnhealthy = "origin.unhealthy"
	AlertCertExpiring    = "certificate.expiring"
	AlertDomainUnhealthy = "domain.unhealthy"
	AlertQuotaThreshold  = "quota.threshold"
	AlertReviewSpike     = "traffic.review_spike"
	AlertCodeAbnormal    = "code.abnormal"
	AlertValidationSpike = "validation.spike"
	AlertDegraded        = "decision.degraded"
)

// AlertTypes 全部告警类型（顺序即展示顺序）。
var AlertTypes = []string{
	AlertOriginUnhealthy, AlertCertExpiring, AlertDomainUnhealthy, AlertQuotaThreshold,
	AlertReviewSpike, AlertCodeAbnormal, AlertValidationSpike, AlertDegraded,
}

// 通知渠道。
const (
	ChannelInApp    = "inapp"
	ChannelEmail    = "email"
	ChannelTelegram = "telegram"
	ChannelWebhook  = "webhook"
)

// AlertRule 告警规则（每商户每站点每类型一条；site_id=0 为全部站点）。
type AlertRule struct {
	ID              uint64    `json:"id" gorm:"primaryKey"`
	MerchantID      uint64    `json:"merchantId"`
	SiteID          uint64    `json:"siteId"`
	Type            string    `json:"type"`
	Enabled         bool      `json:"enabled"`
	Severity        string    `json:"severity"`
	Threshold       []byte    `json:"-" gorm:"type:json"`
	WindowMinutes   int       `json:"windowMinutes"`
	CooldownMinutes int       `json:"cooldownMinutes"`
	Channels        []byte    `json:"-" gorm:"type:json"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (AlertRule) TableName() string { return "alert_rules" }

// Threshold 规则阈值（按类型使用其中若干字段）。
type Threshold struct {
	Pct       float64 `json:"pct,omitempty"`       // quota.threshold / traffic.review_spike / decision.degraded 百分比
	Count     int64   `json:"count,omitempty"`     // code.abnormal / validation.spike 次数
	Days      int     `json:"days,omitempty"`      // certificate.expiring 天数
	MinVisits int64   `json:"minVisits,omitempty"` // 比例类告警的最低样本
}

// Alert 告警实例。
type Alert struct {
	ID         uint64     `json:"id" gorm:"primaryKey"`
	MerchantID uint64     `json:"merchantId"`
	SiteID     uint64     `json:"siteId"`
	RuleID     uint64     `json:"ruleId"`
	Type       string     `json:"type"`
	Severity   string     `json:"severity"`
	TargetKey  string     `json:"targetKey"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	Detail     []byte     `json:"-" gorm:"type:json"`
	Status     string     `json:"status"`
	FiredAt    time.Time  `json:"firedAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	ResolvedAt *time.Time `json:"resolvedAt"`
	Notified   []byte     `json:"-" gorm:"type:json"`
	AckUserID  *uint64    `json:"ackUserId"`
	AckAt      *time.Time `json:"ackAt"`
}

func (Alert) TableName() string { return "alerts" }

const (
	AlertOpen     = "open"
	AlertResolved = "resolved"
)

// Notification 站内通知。
type Notification struct {
	ID         uint64     `json:"id" gorm:"primaryKey"`
	MerchantID uint64     `json:"merchantId"`
	Kind       string     `json:"kind"`
	Severity   string     `json:"severity"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Link       *string    `json:"link"`
	RefType    *string    `json:"refType"`
	RefID      *uint64    `json:"refId"`
	ReadAt     *time.Time `json:"readAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (Notification) TableName() string { return "notifications" }

// NotifySettings 商户通知设置。
type NotifySettings struct {
	MerchantID     uint64    `json:"merchantId" gorm:"primaryKey"`
	Emails         []byte    `json:"-" gorm:"type:json"`
	TelegramChatID string    `json:"telegramChatId" gorm:"column:telegram_chat_id"`
	Channels       []byte    `json:"-" gorm:"type:json"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (NotifySettings) TableName() string { return "notify_settings" }

// WebhookEndpoint Webhook 目标。
type WebhookEndpoint struct {
	ID              uint64     `json:"id" gorm:"primaryKey"`
	MerchantID      uint64     `json:"merchantId"`
	Name            string     `json:"name"`
	URL             string     `json:"url" gorm:"column:url"`
	Secret          string     `json:"-"`
	Events          []byte     `json:"-" gorm:"type:json"`
	Enabled         bool       `json:"enabled"`
	LastStatus      int        `json:"lastStatus"`
	LastDeliveredAt *time.Time `json:"lastDeliveredAt"`
	FailureCount    uint32     `json:"failureCount"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (WebhookEndpoint) TableName() string { return "webhook_endpoints" }

// WebhookDelivery 一次投递。
type WebhookDelivery struct {
	ID             uint64     `json:"id" gorm:"primaryKey"`
	MerchantID     uint64     `json:"merchantId"`
	EndpointID     uint64     `json:"endpointId"`
	Event          string     `json:"event"`
	EventID        string     `json:"eventId"`
	Payload        []byte     `json:"-" gorm:"type:json"`
	Attempt        uint32     `json:"attempt"`
	MaxAttempts    uint32     `json:"maxAttempts"`
	Status         string     `json:"status"`
	ResponseStatus int        `json:"responseStatus"`
	ResponseBody   string     `json:"responseBody"`
	LastError      string     `json:"lastError"`
	NextAttemptAt  time.Time  `json:"nextAttemptAt"`
	DeliveredAt    *time.Time `json:"deliveredAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func (WebhookDelivery) TableName() string { return "webhook_deliveries" }

const (
	DeliveryPending = "pending"
	DeliverySuccess = "success"
	DeliveryFailed  = "failed"
)

// Export 异步导出任务。
type Export struct {
	ID         uint64     `json:"id" gorm:"primaryKey"`
	MerchantID uint64     `json:"merchantId"`
	UserID     uint64     `json:"userId"`
	Kind       string     `json:"kind"`
	Params     []byte     `json:"-" gorm:"type:json"`
	Status     string     `json:"status"`
	FilePath   string     `json:"-"`
	FileSize   uint64     `json:"fileSize"`
	RowCount   uint64     `json:"rowCount"`
	Error      string     `json:"error"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

func (Export) TableName() string { return "exports" }

const (
	ExportPending = "pending"
	ExportRunning = "running"
	ExportDone    = "done"
	ExportFailed  = "failed"
)
