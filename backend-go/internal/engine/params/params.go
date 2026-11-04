// Package params 归因参数提取与校验：别名、格式、分销码登记、允许来源、组合规则、置信度。
package params

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"cloak/internal/engine/pathrule"
	"cloak/internal/engine/snapshot"
)

// 参数类型
const (
	TypeClick       = "click"
	TypeSource      = "source"
	TypeDistributor = "distributor"
	TypeCampaign    = "campaign"
	TypeFree        = "free"
)

// 校验失败类型
const (
	ErrNotRegistered    = "not_registered"
	ErrDisabled         = "disabled"
	ErrExpired          = "expired"
	ErrNotStarted       = "not_started"
	ErrPathNotAllowed   = "path_not_allowed"
	ErrFormatInvalid    = "format_invalid"
	ErrSourceNotAllowed = "source_not_allowed"
	ErrParamDisabled    = "param_disabled"
	ErrClickReused      = "click_reused"
)

// Matched 一个被识别的参数。
type Matched struct {
	Name   string // 字典中的规范名
	Raw    string // 请求中实际出现的键（可能是别名）
	Value  string
	Type   string
	Def    *snapshot.CompiledParam
	Valid  bool
	Error  string
	Weight int // 对放行的贡献：0 无，1 来源，2 点击，3 分销码/活动码
}

// ParamError 对外输出的校验错误。
type ParamError struct {
	Param  string `json:"param"`
	Value  string `json:"value,omitempty"`
	Reason string `json:"reason"`
}

// Result 校验结果。
type Result struct {
	Matched          []Matched
	Code             *snapshot.Code
	CodeParam        string
	HasClick         bool
	HasAllowedSource bool
	HasFree          bool
	Attributed       bool
	Confidence       int // 1 low 2 medium 3 high
	Channel          string
	Errors           []ParamError
	ClickValues      []string // 通过格式校验的点击参数值，供复用计数
}

// MatchedNames 命中的参数规范名（含未通过校验的）。
func (r *Result) MatchedNames() []string {
	out := make([]string, 0, len(r.Matched))
	for _, m := range r.Matched {
		out = append(out, m.Raw)
	}
	return out
}

// Extract 按字典与别名从 query 中提取参数。同一规范名出现多次取第一个非空值。
func Extract(site *snapshot.Site, q url.Values) []Matched {
	out := []Matched{}
	seen := map[string]bool{}
	for key, vals := range q {
		def := site.LookupParam(key)
		if def == nil {
			continue
		}
		name := strings.ToLower(def.Name)
		if seen[name] {
			continue
		}
		val := ""
		for _, v := range vals {
			if v = strings.TrimSpace(v); v != "" {
				val = v
				break
			}
		}
		if val == "" {
			continue
		}
		seen[name] = true
		out = append(out, Matched{Name: def.Name, Raw: key, Value: val, Type: def.Type, Def: def})
	}
	// 固定顺序：码 > 点击 > 来源 > 自由，便于日志与测试
	sortMatched(out)
	return out
}

func typeRank(t string) int {
	switch t {
	case TypeDistributor:
		return 0
	case TypeCampaign:
		return 1
	case TypeClick:
		return 2
	case TypeSource:
		return 3
	default:
		return 4
	}
}

func sortMatched(ms []Matched) {
	for i := 1; i < len(ms); i++ {
		for j := i; j > 0; j-- {
			a, b := ms[j-1], ms[j]
			if typeRank(a.Type) > typeRank(b.Type) || (typeRank(a.Type) == typeRank(b.Type) && a.Name > b.Name) {
				ms[j-1], ms[j] = ms[j], ms[j-1]
			} else {
				break
			}
		}
	}
}

// Env 校验时需要的请求环境。
type Env struct {
	Path         string
	Now          time.Time
	Referer      string
	InAppBrowser bool
	// ClickReused 由调用方（引擎，带 Redis）判断某点击参数值是否已被过多复用；nil 表示不检查
	ClickReused func(value string) bool
}

// Validate 校验提取出的参数，产出归因结论与置信度。
func Validate(site *snapshot.Site, ms []Matched, env Env) Result {
	res := Result{Matched: ms, Errors: []ParamError{}}
	rule := site.Rule
	anyPass := false
	clickReused := false

	for i := range ms {
		m := &ms[i]
		if !m.Def.Enabled {
			m.Error = ErrParamDisabled
			res.Errors = append(res.Errors, ParamError{Param: m.Raw, Reason: m.Error})
			continue
		}
		switch m.Type {
		case TypeDistributor, TypeCampaign:
			code := site.LookupCode(m.Value)
			if rule.ValidateCodes {
				if code == nil {
					m.Error = ErrNotRegistered
				} else if !code.Enabled {
					m.Error = ErrDisabled
				} else if code.ExpiresAt != nil && !env.Now.Before(*code.ExpiresAt) {
					m.Error = ErrExpired
				} else if code.StartsAt != nil && env.Now.Before(*code.StartsAt) {
					m.Error = ErrNotStarted
				} else if !pathrule.Allowed(env.Path, code.AllowedPaths) {
					m.Error = ErrPathNotAllowed
				}
			} else if !formatOK(m) {
				m.Error = ErrFormatInvalid
			}
			if m.Error != "" {
				res.Errors = append(res.Errors, ParamError{Param: m.Raw, Value: m.Value, Reason: m.Error})
				continue
			}
			m.Valid = true
			m.Weight = 3
			if res.Code == nil {
				if code == nil {
					// 关闭登记校验时构造一个临时码引用
					code = &snapshot.Code{Code: m.Value, Type: m.Type, Enabled: true}
				}
				res.Code = code
				res.CodeParam = m.Name
			}
			if m.Def.Sufficient {
				anyPass = true
			}
		case TypeClick:
			if rule.ValidateClickFormat && !formatOK(m) {
				m.Error = ErrFormatInvalid
				res.Errors = append(res.Errors, ParamError{Param: m.Raw, Value: m.Value, Reason: m.Error})
				continue
			}
			if env.ClickReused != nil && rule.ClickReuseLimit > 0 && env.ClickReused(m.Value) {
				clickReused = true
				if rule.ClickReuseAction == "review" {
					m.Error = ErrClickReused
					res.Errors = append(res.Errors, ParamError{Param: m.Raw, Reason: m.Error})
					continue
				}
			}
			m.Valid = true
			m.Weight = 2
			res.HasClick = true
			res.ClickValues = append(res.ClickValues, m.Value)
			if m.Def.Sufficient {
				anyPass = true
			}
		case TypeSource:
			if !formatOK(m) {
				m.Error = ErrFormatInvalid
				res.Errors = append(res.Errors, ParamError{Param: m.Raw, Value: m.Value, Reason: m.Error})
				continue
			}
			if strings.EqualFold(m.Name, "utm_source") && !site.SourceAllowed(m.Value) {
				m.Error = ErrSourceNotAllowed
				res.Errors = append(res.Errors, ParamError{Param: m.Raw, Value: m.Value, Reason: m.Error})
				continue
			}
			m.Valid = true
			m.Weight = 1
			if strings.EqualFold(m.Name, "utm_source") {
				res.HasAllowedSource = true
			}
			if m.Def.Sufficient {
				anyPass = true
			}
		default:
			if !formatOK(m) {
				m.Error = ErrFormatInvalid
				res.Errors = append(res.Errors, ParamError{Param: m.Raw, Value: m.Value, Reason: m.Error})
				continue
			}
			m.Valid = true
			res.HasFree = true
			if m.Def.Sufficient {
				anyPass = true
			}
		}
	}

	switch rule.CombineRule {
	case "require_click_or_code":
		res.Attributed = res.Code != nil || res.HasClick
	default:
		res.Attributed = anyPass
	}
	if !res.Attributed {
		res.Confidence = 1
		return res
	}

	// 置信度（技术方案 5.5）
	switch {
	case res.Code != nil:
		res.Confidence = 3
	case res.HasClick && (env.InAppBrowser || refererIsAd(env.Referer)):
		res.Confidence = 3
	case res.HasClick && !clickReused:
		res.Confidence = 2
	case res.HasAllowedSource:
		res.Confidence = 2
	default:
		res.Confidence = 1
	}
	if clickReused && res.Code == nil && res.Confidence > 1 {
		res.Confidence = 1
	}
	res.Channel = channel(res, ms)
	return res
}

func formatOK(m *Matched) bool {
	n := len(m.Value)
	if m.Def.MinLen > 0 && n < m.Def.MinLen {
		return false
	}
	if m.Def.MaxLen > 0 && n > m.Def.MaxLen {
		return false
	}
	if m.Def.Re != nil && !m.Def.Re.MatchString(m.Value) {
		return false
	}
	return true
}

// channel 渠道推断：分销码 > 活动码 > utm_source > 点击参数所属平台。
func channel(res Result, ms []Matched) string {
	if res.Code != nil {
		return res.Code.Type + ":" + res.Code.Code
	}
	for _, m := range ms {
		if m.Valid && m.Type == TypeSource && strings.EqualFold(m.Name, "utm_source") {
			return "source:" + strings.ToLower(m.Value)
		}
	}
	for _, m := range ms {
		if m.Valid && m.Type == TypeClick {
			if m.Def.Platform != "" {
				return "click:" + m.Def.Platform
			}
			return "click:" + m.Name
		}
	}
	return ""
}

func builtinAdReferers() []string {
	return []string{
		"facebook.com", "fb.com", "instagram.com", "l.facebook.com", "lm.facebook.com",
		"googleadservices.com", "googlesyndication.com", "doubleclick.net", "google.com",
		"tiktok.com", "ads.tiktok.com", "t.co", "twitter.com", "x.com",
		"bing.com", "snapchat.com", "linkedin.com", "lnkd.in", "kwai.com", "kwai-pro.com",
	}
}

var (
	adMu           sync.RWMutex
	adRefererHosts = builtinAdReferers()
)

// BuiltinAdReferers 返回内置广告 Referer 域名的副本。
func BuiltinAdReferers() []string { return builtinAdReferers() }

// SetAdReferers 用全局配置替换广告 Referer 域名表。空列表恢复内置默认。
func SetAdReferers(hosts []string) {
	adMu.Lock()
	defer adMu.Unlock()
	next := make([]string, 0, len(hosts))
	for _, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		h = strings.TrimPrefix(h, "*.")
		if h != "" {
			next = append(next, h)
		}
	}
	if len(next) == 0 {
		adRefererHosts = builtinAdReferers()
		return
	}
	adRefererHosts = next
}

func refererHosts() []string {
	adMu.RLock()
	defer adMu.RUnlock()
	out := make([]string, len(adRefererHosts))
	copy(out, adRefererHosts)
	return out
}

func refererIsAd(ref string) bool {
	if ref == "" {
		return false
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range refererHosts() {
		if host == h || strings.HasSuffix(host, "."+h) {
			return true
		}
	}
	return false
}

// RefererIsAd 导出给引擎做 Referer 推断渠道。
func RefererIsAd(ref string) bool { return refererIsAd(ref) }
