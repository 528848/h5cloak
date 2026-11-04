// Package snapshot 定义站点快照：管理端构建、Redis 传递、网关内存常驻。
// 快照是纯数据结构（可 JSON 序列化），加载后由 Compile 预编译匹配器。
package snapshot

import (
	"regexp"
	"strings"
	"time"
)

// Version 快照格式版本，结构不兼容时递增。
const Version = 1

// Site 站点快照。
type Site struct {
	Format     int       `json:"format"`
	Revision   int64     `json:"revision"` // 构建时间戳（纳秒），用于比较新旧
	BuiltAt    time.Time `json:"builtAt"`
	SiteID     uint64    `json:"siteId"`
	MerchantID uint64    `json:"merchantId"`
	Name       string    `json:"name"`
	Status     string    `json:"status"` // enabled / disabled
	SiteType   string    `json:"siteType"`

	// 商户状态：商户停用/冻结/到期时站点也不服务
	MerchantActive bool  `json:"merchantActive"`
	Quota          Quota `json:"quota"`

	Domains   []Domain   `json:"domains"`
	Origins   []Origin   `json:"origins"`
	Review    Review     `json:"review"`
	PathRules []PathRule `json:"pathRules"`

	Cookie      Cookie      `json:"cookie"`
	Attribution Attribution `json:"attribution"`
	Handoff     Handoff     `json:"handoff"`
	Keys        []TokenKey  `json:"keys"`
	KeyVersion  int         `json:"keyVersion"`

	Params         []Param  `json:"params"`
	Rule           Rule     `json:"rule"`
	Codes          []Code   `json:"codes"`
	PublishableKey string   `json:"publishableKey"`
	SecretPrefix   string   `json:"secretPrefix"`
	SecretHash     string   `json:"secretHash"`
	SecretEnc      []byte   `json:"secretEnc,omitempty"` // AES-GCM 加密的服务端密钥，网关解密后验签
	SecretAllow    []string `json:"secretAllow"`

	compiled *compiled
}

// Quota 商户判定额度与达到后的策略。
type Quota struct {
	Decisions uint64 `json:"decisions"` // 每月判定次数，0 为不限
	Policy    string `json:"policy"`    // alert / review_all / deny
	Retention int    `json:"retentionDays"`
}

type Domain struct {
	Domain     string `json:"domain"`
	DomainType string `json:"domainType"`
	Status     string `json:"status"`
}

type Origin struct {
	Kind       string `json:"kind"` // real / review
	URL        string `json:"url"`
	HostHeader string `json:"hostHeader"`
	TimeoutMs  int    `json:"timeoutMs"`
	Healthy    bool   `json:"healthy"`
}

type Review struct {
	Source       string            `json:"source"` // origin / template / static
	TemplateHTML string            `json:"templateHtml,omitempty"`
	TemplateVars map[string]string `json:"templateVars,omitempty"`
	StaticDir    string            `json:"staticDir,omitempty"`
	StaticEntry  string            `json:"staticEntry,omitempty"`
}

type PathRule struct {
	Pattern   string `json:"pattern"`
	MatchType string `json:"matchType"` // prefix / exact / suffix
	Action    string `json:"action"`    // bypass / always_real / always_review / block
	Sort      int    `json:"sort"`
}

type Cookie struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type Attribution struct {
	TTLDays             int    `json:"ttlDays"`
	RenewMode           string `json:"renewMode"`           // fixed / sliding
	NoAttributionAction string `json:"noAttributionAction"` // review / block / not_found
	FailMode            string `json:"failMode"`            // review / real / maintenance
	MinConfidence       string `json:"minConfidence"`       // low / medium / high
}

type Handoff struct {
	Banner        bool   `json:"banner"`
	BannerText    string `json:"bannerText"`
	LinkRewrite   bool   `json:"linkRewrite"`
	TokenTTLHours int    `json:"tokenTtlHours"`
	TokenMaxUses  int    `json:"tokenMaxUses"`
	MatchEnabled  bool   `json:"matchEnabled"`
	MatchMinutes  int    `json:"matchMinutes"`
	// MatchSystemOnly 匹配时要求新请求来自系统浏览器（默认 true）
	MatchSystemOnly bool `json:"matchSystemOnly"`
}

// TokenKey 访客令牌签名密钥。Status active 为当前签发密钥；grace 为宽限期内仍可验签的旧密钥。
type TokenKey struct {
	Version   int        `json:"version"`
	Key       string     `json:"key"` // hex
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// Param 归因参数定义（已合并内置与站点覆盖）。
type Param struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"` // click / source / distributor / campaign / free
	Platform   string   `json:"platform"`
	Aliases    []string `json:"aliases"`
	MinLen     int      `json:"minLen"`
	MaxLen     int      `json:"maxLen"`
	Pattern    string   `json:"pattern"`
	Sufficient bool     `json:"sufficient"`
	Enabled    bool     `json:"enabled"`
}

type Rule struct {
	CombineRule         string   `json:"combineRule"` // require_any / require_click_or_code
	ValidateCodes       bool     `json:"validateCodes"`
	ValidateClickFormat bool     `json:"validateClickFormat"`
	AllowedSources      []string `json:"allowedSources"`
	ClickReuseLimit     int      `json:"clickReuseLimit"`
	ClickReuseWindowH   int      `json:"clickReuseWindowHours"`
	ClickReuseAction    string   `json:"clickReuseAction"` // downgrade / review
	CodeRevokeMode      string   `json:"codeRevokeMode"`   // window / immediate
}

// Code 分销码/活动码。
type Code struct {
	ID           uint64     `json:"id"`
	Code         string     `json:"code"`
	Type         string     `json:"type"` // distributor / campaign
	Name         string     `json:"name"`
	Channel      string     `json:"channel"`
	Enabled      bool       `json:"enabled"`
	StartsAt     *time.Time `json:"startsAt,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	AllowedPaths []string   `json:"allowedPaths,omitempty"`
}

// ---- 预编译 ----

type compiled struct {
	params   map[string]*CompiledParam // lowercase(name|alias) -> def
	codes    map[string]*Code          // code -> def（大小写敏感，与登记一致）
	codesCI  map[string]*Code          // lowercase(code) -> def
	sources  map[string]bool
	keys     map[int]*TokenKey
	active   *TokenKey
	rules    []PathRule // 按 sort 排序
	origins  map[string]*Origin
	domains  map[string]*Domain
	realHost string
}

// CompiledParam 带预编译正则的参数定义。
type CompiledParam struct {
	Param
	Re *regexp.Regexp
}

// Compile 预编译匹配器。加载快照后必须调用一次；重复调用安全。
func (s *Site) Compile() error {
	c := &compiled{
		params:  map[string]*CompiledParam{},
		codes:   map[string]*Code{},
		codesCI: map[string]*Code{},
		sources: map[string]bool{},
		keys:    map[int]*TokenKey{},
		origins: map[string]*Origin{},
		domains: map[string]*Domain{},
	}
	for i := range s.Params {
		p := &s.Params[i]
		cp := &CompiledParam{Param: *p}
		if p.Pattern != "" {
			re, err := regexp.Compile(p.Pattern)
			if err != nil {
				return err
			}
			cp.Re = re
		}
		c.params[strings.ToLower(p.Name)] = cp
		for _, a := range p.Aliases {
			if a != "" {
				c.params[strings.ToLower(a)] = cp
			}
		}
	}
	for i := range s.Codes {
		code := &s.Codes[i]
		c.codes[code.Code] = code
		c.codesCI[strings.ToLower(code.Code)] = code
	}
	for _, src := range s.Rule.AllowedSources {
		c.sources[strings.ToLower(src)] = true
	}
	for i := range s.Keys {
		k := &s.Keys[i]
		c.keys[k.Version] = k
		if k.Status == "active" {
			c.active = k
		}
	}
	if c.active == nil && len(s.Keys) > 0 {
		c.active = &s.Keys[len(s.Keys)-1]
	}
	c.rules = append([]PathRule(nil), s.PathRules...)
	sortRules(c.rules)
	for i := range s.Origins {
		c.origins[s.Origins[i].Kind] = &s.Origins[i]
	}
	for i := range s.Domains {
		c.domains[strings.ToLower(s.Domains[i].Domain)] = &s.Domains[i]
	}
	s.compiled = c
	return nil
}

func sortRules(rules []PathRule) {
	// 插入排序，规则数很少
	for i := 1; i < len(rules); i++ {
		for j := i; j > 0 && rules[j].Sort < rules[j-1].Sort; j-- {
			rules[j], rules[j-1] = rules[j-1], rules[j]
		}
	}
}

func (s *Site) c() *compiled {
	if s.compiled == nil {
		_ = s.Compile()
	}
	return s.compiled
}

// LookupParam 按名字或别名找参数定义。
func (s *Site) LookupParam(name string) *CompiledParam {
	return s.c().params[strings.ToLower(name)]
}

// LookupCode 按码值找定义，先精确后忽略大小写。
func (s *Site) LookupCode(code string) *Code {
	c := s.c()
	if v, ok := c.codes[code]; ok {
		return v
	}
	return c.codesCI[strings.ToLower(code)]
}

// SourceAllowed 判断 utm_source 是否在允许列表；列表为空表示不限制。
func (s *Site) SourceAllowed(src string) bool {
	c := s.c()
	if len(c.sources) == 0 {
		return true
	}
	return c.sources[strings.ToLower(src)]
}

// ActiveKey 当前签发密钥。
func (s *Site) ActiveKey() *TokenKey { return s.c().active }

// KeyByVersion 按版本找密钥（含宽限期密钥）。
func (s *Site) KeyByVersion(v int) *TokenKey { return s.c().keys[v] }

// SortedRules 按 sort 升序的路径规则。
func (s *Site) SortedRules() []PathRule { return s.c().rules }

// Origin 按类型取源站。
func (s *Site) Origin(kind string) *Origin { return s.c().origins[kind] }

// HasDomain 域名是否属于该站点。
func (s *Site) HasDomain(host string) bool {
	host = strings.ToLower(host)
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	_, ok := s.c().domains[host]
	return ok
}

// Serving 站点是否可提供判定服务。
func (s *Site) Serving() bool {
	return s.Status == "enabled" && s.MerchantActive
}

// MinConfidenceLevel 最低置信度数值。
func (s *Site) MinConfidenceLevel() int {
	return ConfidenceLevel(s.Attribution.MinConfidence)
}

// ConfidenceLevel 字符串置信度转数值：low=1 medium=2 high=3。
func ConfidenceLevel(c string) int {
	switch c {
	case "high":
		return 3
	case "low":
		return 1
	default:
		return 2
	}
}

// ConfidenceName 数值转字符串。
func ConfidenceName(level int) string {
	switch {
	case level >= 3:
		return "high"
	case level <= 1:
		return "low"
	default:
		return "medium"
	}
}
