package identity

import "encoding/json"

// Features 商户功能开关。
type Features struct {
	Hosted       bool `json:"hosted"`
	ServerAPI    bool `json:"server_api"`
	JSSDK        bool `json:"js_sdk"`
	Conversion   bool `json:"conversion"`
	Webhook      bool `json:"webhook"`
	Enhancements bool `json:"enhancements"`
}

// DefaultFeatures 开通商户时的默认值（D3）。
func DefaultFeatures() Features {
	return Features{Hosted: true, ServerAPI: true, JSSDK: true, Conversion: true, Webhook: true, Enhancements: false}
}

// FeaturesOf 解析商户的 features 字段，解析失败返回默认值。
func (m *Merchant) FeaturesOf() Features {
	f := DefaultFeatures()
	if len(m.Features) > 0 {
		_ = json.Unmarshal(m.Features, &f)
	}
	return f
}

// SetFeatures 写回 features。
func (m *Merchant) SetFeatures(f Features) {
	b, _ := json.Marshal(f)
	m.Features = b
}

// Quota 商户使用额度。
type Quota struct {
	DecisionQuota    uint64 `json:"decisionQuota"`
	SiteQuota        uint32 `json:"siteQuota"`
	DomainQuota      uint32 `json:"domainQuota"`
	CodeQuota        uint32 `json:"codeQuota"`
	CloakQuota       uint32 `json:"cloakQuota"`
	UserQuota        uint32 `json:"userQuota"`
	LogRetentionDays uint32 `json:"logRetentionDays"`
	QuotaPolicy      string `json:"quotaPolicy"` // alert, review_all, deny
}

// DefaultQuota 开通商户时的默认额度（D2）。
func DefaultQuota() Quota {
	return Quota{DecisionQuota: 1_000_000, SiteQuota: 5, DomainQuota: 20, CodeQuota: 1000, CloakQuota: 20, UserQuota: 10, LogRetentionDays: 90, QuotaPolicy: "alert"}
}

// QuotaOf 读取额度。
func (m *Merchant) QuotaOf() Quota {
	return Quota{
		DecisionQuota: m.DecisionQuota, SiteQuota: m.SiteQuota, DomainQuota: m.DomainQuota,
		CodeQuota: m.CodeQuota, CloakQuota: m.CloakQuota, UserQuota: m.UserQuota, LogRetentionDays: m.LogRetentionDays, QuotaPolicy: m.QuotaPolicy,
	}
}

// SetQuota 写回额度。
func (m *Merchant) SetQuota(q Quota) {
	m.DecisionQuota, m.SiteQuota, m.DomainQuota = q.DecisionQuota, q.SiteQuota, q.DomainQuota
	if q.CloakQuota == 0 {
		q.CloakQuota = m.CloakQuota
	}
	if q.CloakQuota == 0 {
		q.CloakQuota = 20
	}
	m.CodeQuota, m.CloakQuota, m.UserQuota, m.LogRetentionDays, m.QuotaPolicy = q.CodeQuota, q.CloakQuota, q.UserQuota, q.LogRetentionDays, q.QuotaPolicy
}

// ValidQuotaPolicies 允许的额度策略。
var ValidQuotaPolicies = map[string]bool{"alert": true, "review_all": true, "deny": true}
