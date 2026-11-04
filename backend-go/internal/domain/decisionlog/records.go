package decisionlog

import "time"

// Record decision_logs 表行。
type Record struct {
	ID                   uint64 `gorm:"primaryKey"`
	CreatedAt            time.Time
	RequestID            string
	MerchantID           uint64
	SiteID               uint64
	Entry                string
	Method               string
	Host                 string
	Path                 string
	QueryParamNames      []byte `gorm:"type:json"`
	TouchType            string
	Content              string
	Attributed           bool
	Confidence           string
	InAppBrowser         string
	DeviceFamily         string
	HandoffKind          string
	Channel              string
	CodeID               uint64
	Code                 string
	MatchedParams        []byte `gorm:"type:json"`
	ValidationErrors     []byte `gorm:"type:json"`
	VisitorID            string
	AttributionID        string
	AttributionExpiresAt *time.Time
	IPHash               string
	IPPrefix             string
	Country              string
	OS                   string `gorm:"column:os"`
	UASummary            string `gorm:"column:ua_summary"`
	RefererSummary       string
	PathRule             string
	OriginKind           string
	OriginStatus         uint16
	Fallback             bool
	LatencyMs            uint32
	Degraded             bool
	Trace                []byte `gorm:"type:json"`
}

func (Record) TableName() string { return "decision_logs" }

// Visitor visitors 表行。
type Visitor struct {
	SiteID            uint64 `gorm:"primaryKey"`
	VisitorID         string `gorm:"primaryKey"`
	MerchantID        uint64
	FirstSeenAt       time.Time
	LastSeenAt        time.Time
	VisitCount        uint64
	FirstCountry      string
	FirstDeviceFamily string
	FirstInApp        string
}

func (Visitor) TableName() string { return "visitors" }

// Attribution attributions 表行。
type Attribution struct {
	ID                  uint64 `gorm:"primaryKey"`
	AttributionID       string
	MerchantID          uint64
	SiteID              uint64
	VisitorID           string
	FirstChannel        string
	FirstCodeID         uint64
	FirstParams         []byte `gorm:"type:json"`
	FirstLandingPath    string
	FirstRequestID      string
	FirstTouchAt        time.Time
	LastChannel         string
	LastCodeID          uint64
	LastTouchAt         time.Time
	TouchCount          uint64
	HandoffCount        uint64
	Confidence          string
	ExpiresAt           *time.Time
	MergedIntoVisitorID *string
}

func (Attribution) TableName() string { return "attributions" }

// UsageCounter usage_counters 表行。
type UsageCounter struct {
	MerchantID uint64 `gorm:"primaryKey"`
	Metric     string `gorm:"primaryKey"`
	Period     string `gorm:"primaryKey"`
	Value      uint64
	UpdatedAt  time.Time
}

func (UsageCounter) TableName() string { return "usage_counters" }

// APILog api_logs 表行。
type APILog struct {
	ID         uint64 `gorm:"primaryKey"`
	CreatedAt  time.Time
	RequestID  string
	MerchantID uint64
	SiteID     uint64
	KeyPrefix  string
	Endpoint   string
	Status     uint16
	ErrorCode  string
	LatencyMs  uint32
	IP         string `gorm:"column:ip"`
}

func (APILog) TableName() string { return "api_logs" }
