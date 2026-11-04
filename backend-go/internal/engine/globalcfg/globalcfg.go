// Package globalcfg 是网关进程内的全局开关。
// 配置写在 Redis cfg:global，网关订阅 cfg:changed 的 global 消息，并每 4 秒轮询兜底。
package globalcfg

import (
	"encoding/json"
	"sync/atomic"

	"cloak/internal/engine"
	"cloak/internal/engine/params"
	"cloak/internal/engine/ua"
)

const (
	RedisKey      = "cfg:global"
	ChangeChannel = "cfg:changed"
	ChangeMessage = "global"
)

// Config 网关需要热更新的部分。AdReferers / InAppMarkers 为 nil 时沿用内置表。
type Config struct {
	EmergencyReview    bool            `json:"emergencyReview"`
	Entries            map[string]bool `json:"entries"`
	RatePerKey         int             `json:"ratePerKey"`
	RatePerSite        int             `json:"ratePerSite"`
	WebhookMaxAttempts int             `json:"webhookMaxAttempts"`
	LogRetentionDays   int             `json:"logRetentionDays"`
	AdReferers         []string        `json:"adReferers"`
	InAppMarkers       []ua.Marker     `json:"inAppMarkers"`
}

// Holder 供请求路径无锁读取当前配置。
type Holder struct{ p atomic.Pointer[Config] }

func NewHolder() *Holder { return &Holder{} }

func (h *Holder) Get() *Config {
	if h == nil {
		return nil
	}
	return h.p.Load()
}

func (h *Holder) Store(c *Config) {
	if h != nil && c != nil {
		h.p.Store(c)
	}
}

// Defaults 未下发配置时的行为：各入口开启，不进入紧急审核。
func Defaults() *Config {
	return &Config{
		Entries: map[string]bool{
			"hosted": true, "api_v2": true, "api_v1": true, "js": true,
		},
		RatePerKey: 50, RatePerSite: 2000, WebhookMaxAttempts: 5, LogRetentionDays: 90,
	}
}

// Parse 解析 Redis 中的 cfg:global。缺省限额回落到默认值。
func Parse(b []byte) (*Config, error) {
	c := Defaults()
	if err := json.Unmarshal(b, c); err != nil {
		return nil, err
	}
	if c.RatePerKey <= 0 {
		c.RatePerKey = 50
	}
	if c.RatePerSite <= 0 {
		c.RatePerSite = 2000
	}
	if c.WebhookMaxAttempts <= 0 {
		c.WebhookMaxAttempts = 5
	}
	if c.LogRetentionDays <= 0 {
		c.LogRetentionDays = 90
	}
	if c.Entries == nil {
		c.Entries = Defaults().Entries
	}
	return c, nil
}

// EntryEnabled 未声明的入口视为开启。
func (c *Config) EntryEnabled(entry string) bool {
	if c == nil || c.Entries == nil {
		return true
	}
	on, ok := c.Entries[entry]
	if !ok {
		return true
	}
	return on
}

// Apply 把全局开关叠到判定结果上。
// 入口关闭或紧急审核时，原本放行的流量改为审核；bypass、维护页、未找到保持原样。
// 返回的 reason 为空表示没有改动。
func Apply(c *Config, content, entry string) (string, string) {
	if c == nil || content != engine.ContentReal {
		return content, ""
	}
	if !c.EntryEnabled(entry) {
		return engine.ContentReview, "entry_off"
	}
	if c.EmergencyReview {
		return engine.ContentReview, "emergency_review"
	}
	return content, ""
}

// Overlay 在改写放行结果时清掉访客令牌，并记下 trace。
func Overlay(c *Config, content, token, entry string, trace []engine.TraceStep) (string, string, []engine.TraceStep) {
	next, reason := Apply(c, content, entry)
	if reason == "" {
		return content, token, trace
	}
	return next, "", append(trace, engine.TraceStep{Step: "global", Result: reason})
}

// ApplyRuntime 把名单和限流同步到进程内的包级变量。
func ApplyRuntime(c *Config, setLimits func(perKey, perSite int64)) {
	if c == nil {
		return
	}
	params.SetAdReferers(c.AdReferers)
	ua.SetMarkers(c.InAppMarkers)
	if setLimits != nil && c.RatePerKey > 0 && c.RatePerSite > 0 {
		setLimits(int64(c.RatePerKey), int64(c.RatePerSite))
	}
}
