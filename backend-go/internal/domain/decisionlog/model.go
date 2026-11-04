// Package decisionlog 定义决策日志事件与归因写入事件：网关产生、Redis Stream 传递、worker 落库。
package decisionlog

import "time"

// 流名
const (
	StreamDecisions    = "decision_logs"
	StreamAttributions = "attribution_writes"
	StreamAPILogs      = "api_logs"
	ConsumerGroup      = "workers"
)

// Event 一次判定的完整记录（与技术方案 11.3 字段对应）。
type Event struct {
	CreatedAt            time.Time  `json:"t"`
	RequestID            string     `json:"rid"`
	MerchantID           uint64     `json:"mid"`
	SiteID               uint64     `json:"sid"`
	Entry                string     `json:"entry"`
	Method               string     `json:"method"`
	Host                 string     `json:"host"`
	Path                 string     `json:"path"`
	QueryParamNames      []string   `json:"qp,omitempty"`
	TouchType            string     `json:"touch"`
	Content              string     `json:"content"`
	Attributed           bool       `json:"attr"`
	Confidence           string     `json:"conf,omitempty"`
	InAppBrowser         string     `json:"inapp,omitempty"`
	DeviceFamily         string     `json:"dev,omitempty"`
	HandoffKind          string     `json:"handoff,omitempty"`
	Channel              string     `json:"ch,omitempty"`
	CodeID               uint64     `json:"cid,omitempty"`
	Code                 string     `json:"code,omitempty"`
	MatchedParams        []string   `json:"mp,omitempty"`
	ValidationErrors     any        `json:"ve,omitempty"`
	VisitorID            string     `json:"vid,omitempty"`
	AttributionID        string     `json:"aid,omitempty"`
	AttributionExpiresAt *time.Time `json:"aexp,omitempty"`
	IPHash               string     `json:"iph,omitempty"`
	IPPrefix             string     `json:"ipp,omitempty"`
	Country              string     `json:"cc,omitempty"`
	OS                   string     `json:"os,omitempty"`
	UASummary            string     `json:"ua,omitempty"`
	RefererSummary       string     `json:"ref,omitempty"`
	PathRule             string     `json:"rule,omitempty"`
	OriginKind           string     `json:"ok,omitempty"`
	OriginStatus         int        `json:"os_status,omitempty"`
	Fallback             bool       `json:"fb,omitempty"`
	LatencyMs            int        `json:"lat"`
	Degraded             bool       `json:"deg,omitempty"`
	Trace                any        `json:"trace,omitempty"`
}

// AttributionWrite 首访 / 回访 / 交接时的归因更新事件。
type AttributionWrite struct {
	At            time.Time  `json:"t"`
	RequestID     string     `json:"rid"`
	MerchantID    uint64     `json:"mid"`
	SiteID        uint64     `json:"sid"`
	VisitorID     string     `json:"vid"`
	AttributionID string     `json:"aid"`
	TouchType     string     `json:"touch"` // first / return / handoff
	HandoffKind   string     `json:"handoff,omitempty"`
	Channel       string     `json:"ch,omitempty"`
	CodeID        uint64     `json:"cid,omitempty"`
	MatchedParams []string   `json:"mp,omitempty"`
	LandingPath   string     `json:"path"`
	Confidence    string     `json:"conf,omitempty"`
	ExpiresAt     *time.Time `json:"exp,omitempty"`
	Country       string     `json:"cc,omitempty"`
	DeviceFamily  string     `json:"dev,omitempty"`
	InAppBrowser  string     `json:"inapp,omitempty"`
}

// APILogEvent 开放接口调用日志。
type APILogEvent struct {
	CreatedAt  time.Time `json:"t"`
	RequestID  string    `json:"rid"`
	MerchantID uint64    `json:"mid"`
	SiteID     uint64    `json:"sid"`
	KeyPrefix  string    `json:"kp"`
	Endpoint   string    `json:"ep"`
	Status     int       `json:"st"`
	ErrorCode  string    `json:"err,omitempty"`
	LatencyMs  int       `json:"lat"`
	IP         string    `json:"ip,omitempty"`
}

// StreamConversions 转化回传流。
const StreamConversions = "conversions"

// ConversionEvent 转化回传事件。
type ConversionEvent struct {
	At         time.Time `json:"t"`
	RequestID  string    `json:"rid"`
	MerchantID uint64    `json:"mid"`
	SiteID     uint64    `json:"sid"`
	OrderNo    string    `json:"order"`
	Status     string    `json:"status"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"cur"`
	VisitorID  string    `json:"vid,omitempty"`
	EmailHash  string    `json:"eh,omitempty"`
}

// Metric 计量名。
const MetricDecisions = "decisions"
