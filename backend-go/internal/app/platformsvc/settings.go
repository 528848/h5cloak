// Package platformsvc 总后台监管：全局开关、模板、账号角色菜单、模拟登录。
package platformsvc

import (
	"context"
	"encoding/json"
	"strings"

	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/auth"
	"cloak/internal/domain/identity"
	domainsite "cloak/internal/domain/site"
	"cloak/internal/engine/globalcfg"
	"cloak/internal/engine/params"
	"cloak/internal/engine/pathrule"
	"cloak/internal/engine/ua"
)

type Service struct {
	settings   *mysql.PlatformRepo
	id         *mysql.IdentityRepo
	sites      *mysql.SiteRepo
	merchants  identity.MerchantRepository
	tokens     *auth.TokenManager
	rdb        *goredis.Client
	consoleURL string
	dataDir    string
	republish  func([]uint64)
}

func New(settings *mysql.PlatformRepo, id *mysql.IdentityRepo, sites *mysql.SiteRepo, merchants identity.MerchantRepository, tokens *auth.TokenManager, rdb *goredis.Client, consoleURL string) *Service {
	return &Service{settings: settings, id: id, sites: sites, merchants: merchants, tokens: tokens, rdb: rdb, consoleURL: consoleURL, dataDir: "./uploads"}
}

// SetDataDir 静态模板解压目录的根。
func (s *Service) SetDataDir(dir string) {
	if dir != "" {
		s.dataDir = dir
	}
}

// OnRepublish 模板变更后重建引用它的站点快照。
func (s *Service) OnRepublish(fn func([]uint64)) { s.republish = fn }

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

type pathRuleIn struct {
	Pattern   string `json:"pattern"`
	MatchType string `json:"matchType"`
	Action    string `json:"action"`
	Sort      int    `json:"sort"`
	Remark    string `json:"remark"`
}

// SettingsInput 系统配置保存。Custom 为 false 时对应 JSON 列写回 NULL，运行时用内置表。
type SettingsInput struct {
	EmergencyReview        bool            `json:"emergencyReview"`
	Entries                map[string]bool `json:"entries"`
	LogRetentionDays       int             `json:"logRetentionDays"`
	RatePerKey             int             `json:"ratePerKey"`
	RatePerSite            int             `json:"ratePerSite"`
	WebhookMaxAttempts     int             `json:"webhookMaxAttempts"`
	AdReferers             []string        `json:"adReferers"`
	AdReferersCustom       bool            `json:"adReferersCustom"`
	InAppMarkers           []ua.Marker     `json:"inAppMarkers"`
	InAppMarkersCustom     bool            `json:"inAppMarkersCustom"`
	DefaultPathRules       []pathRuleIn    `json:"defaultPathRules"`
	DefaultPathRulesCustom bool            `json:"defaultPathRulesCustom"`
}

func (s *Service) GetSettings(ctx context.Context) (map[string]any, error) {
	row, err := s.settings.Settings(ctx)
	if err != nil {
		return nil, err
	}
	refs, refCustom := decodeStrings(row.AdReferers)
	if !refCustom {
		refs = params.BuiltinAdReferers()
	}
	markers, markerCustom := decodeMarkers(row.InAppMarkers)
	if !markerCustom {
		markers = ua.BuiltinMarkers()
	}
	rules, ruleCustom := decodeRules(row.DefaultPathRules)
	if !ruleCustom {
		rules = builtinRuleView()
	} else {
		rules = omitPlatformStatic(rules)
	}
	updated := ""
	if !row.UpdatedAt.IsZero() {
		updated = row.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return map[string]any{
		"emergencyReview": row.EmergencyReview,
		"entries": map[string]bool{
			"hosted": row.EntryHosted, "api_v2": row.EntryAPIV2, "api_v1": row.EntryAPIV1, "js": row.EntryJS,
		},
		"logRetentionDays": row.LogRetentionDays, "ratePerKey": row.RatePerKey, "ratePerSite": row.RatePerSite,
		"webhookMaxAttempts": row.WebhookMaxAttempts,
		"adReferers":         refs, "adReferersCustom": refCustom,
		"inAppMarkers": markers, "inAppMarkersCustom": markerCustom,
		"defaultPathRules": rules, "defaultPathRulesCustom": ruleCustom,
		"staticPathRules": pathrule.StaticView(),
		"updatedAt":       updated,
	}, nil
}

// SaveSettings 落库、应用到本进程，并发布 cfg:global。网关在数秒内跟上。
func (s *Service) SaveSettings(ctx context.Context, in SettingsInput) error {
	if in.LogRetentionDays < 7 || in.LogRetentionDays > 3650 {
		return errf(400, "日志保留天数需在 7 到 3650 之间")
	}
	if in.RatePerKey < 1 || in.RatePerKey > 10000 || in.RatePerSite < 1 || in.RatePerSite > 1000000 {
		return errf(400, "限流数值超出允许范围")
	}
	if in.WebhookMaxAttempts < 1 || in.WebhookMaxAttempts > 20 {
		return errf(400, "Webhook 重试次数需在 1 到 20 之间")
	}
	entries := map[string]bool{"hosted": true, "api_v2": true, "api_v1": true, "js": true}
	for k := range entries {
		if in.Entries != nil {
			if v, ok := in.Entries[k]; ok {
				entries[k] = v
			}
		}
	}
	refs, err := cleanHosts(in.AdReferers, in.AdReferersCustom)
	if err != nil {
		return err
	}
	markers, err := cleanMarkers(in.InAppMarkers, in.InAppMarkersCustom)
	if err != nil {
		return err
	}
	rules, err := cleanRules(omitPlatformStatic(in.DefaultPathRules), in.DefaultPathRulesCustom)
	if err != nil {
		return err
	}
	fields := map[string]any{
		"emergency_review": in.EmergencyReview,
		"entry_hosted":     entries["hosted"], "entry_api_v2": entries["api_v2"],
		"entry_api_v1": entries["api_v1"], "entry_js": entries["js"],
		"log_retention_days": in.LogRetentionDays, "rate_per_key": in.RatePerKey,
		"rate_per_site": in.RatePerSite, "webhook_max_attempts": in.WebhookMaxAttempts,
		"ad_referers":        jsonOrNull(refs, in.AdReferersCustom),
		"inapp_markers":      jsonOrNull(markers, in.InAppMarkersCustom),
		"default_path_rules": jsonOrNull(rules, in.DefaultPathRulesCustom),
	}
	if err := s.settings.SaveSettings(ctx, fields); err != nil {
		return err
	}
	return s.Publish(ctx)
}

// Publish 把当前配置应用到本进程并写入 Redis。
func (s *Service) Publish(ctx context.Context) error {
	row, err := s.settings.Settings(ctx)
	if err != nil {
		return err
	}
	refs, refCustom := decodeStrings(row.AdReferers)
	markers, markerCustom := decodeMarkers(row.InAppMarkers)
	rules, ruleCustom := decodeRules(row.DefaultPathRules)
	if ruleCustom {
		domainsite.SetDefaultPathRules(toPathRules(omitPlatformStatic(rules)))
	} else {
		domainsite.SetDefaultPathRules(nil)
	}
	cfg := &globalcfg.Config{
		EmergencyReview: row.EmergencyReview,
		Entries: map[string]bool{
			"hosted": row.EntryHosted, "api_v2": row.EntryAPIV2, "api_v1": row.EntryAPIV1, "js": row.EntryJS,
		},
		RatePerKey: row.RatePerKey, RatePerSite: row.RatePerSite,
		WebhookMaxAttempts: row.WebhookMaxAttempts, LogRetentionDays: row.LogRetentionDays,
	}
	if refCustom {
		cfg.AdReferers = refs
	}
	if markerCustom {
		cfg.InAppMarkers = markers
	}
	globalcfg.ApplyRuntime(cfg, nil)
	if s.rdb == nil {
		return nil
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, globalcfg.RedisKey, b, 0)
	pipe.Publish(ctx, globalcfg.ChangeChannel, globalcfg.ChangeMessage)
	_, err = pipe.Exec(ctx)
	return err
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func jsonOrNull(v any, custom bool) any {
	if !custom {
		return gorm.Expr("NULL")
	}
	b, _ := json.Marshal(v)
	return b
}

func decodeStrings(b []byte) ([]string, bool) {
	if len(b) == 0 {
		return nil, false
	}
	var out []string
	if json.Unmarshal(b, &out) != nil {
		return nil, false
	}
	return out, true
}

func decodeMarkers(b []byte) ([]ua.Marker, bool) {
	if len(b) == 0 {
		return nil, false
	}
	var out []ua.Marker
	if json.Unmarshal(b, &out) != nil {
		return nil, false
	}
	return out, true
}

func decodeRules(b []byte) ([]pathRuleIn, bool) {
	if len(b) == 0 {
		return nil, false
	}
	var out []pathRuleIn
	if json.Unmarshal(b, &out) != nil {
		return nil, false
	}
	return out, true
}

func builtinRuleView() []pathRuleIn {
	rules := domainsite.BuiltinPathRules()
	out := make([]pathRuleIn, 0, len(rules))
	for _, r := range rules {
		out = append(out, pathRuleIn{Pattern: r.Pattern, MatchType: r.MatchType, Action: r.Action, Sort: r.Sort, Remark: deref(r.Remark)})
	}
	return out
}

func toPathRules(in []pathRuleIn) []domainsite.PathRule {
	out := make([]domainsite.PathRule, 0, len(in))
	for _, r := range in {
		remark := r.Remark
		out = append(out, domainsite.PathRule{Pattern: r.Pattern, MatchType: r.MatchType, Action: r.Action, Sort: r.Sort, Remark: &remark})
	}
	return out
}

func cleanHosts(in []string, custom bool) ([]string, error) {
	if !custom {
		return nil, nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, h := range in {
		h = strings.ToLower(strings.TrimSpace(h))
		h = strings.TrimPrefix(h, "https://")
		h = strings.TrimPrefix(h, "http://")
		h = strings.TrimPrefix(h, "*.")
		if i := strings.IndexByte(h, '/'); i >= 0 {
			h = h[:i]
		}
		if h == "" || strings.ContainsAny(h, " \t") || len(h) > 200 {
			continue
		}
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	if len(out) == 0 {
		return nil, errf(400, "请填写广告 Referer 域名，或改回内置列表")
	}
	if len(out) > 100 {
		return nil, errf(400, "广告 Referer 域名最多 100 条")
	}
	return out, nil
}

func cleanMarkers(in []ua.Marker, custom bool) ([]ua.Marker, error) {
	if !custom {
		return nil, nil
	}
	out := make([]ua.Marker, 0, len(in))
	for _, m := range in {
		m.Marker = strings.TrimSpace(m.Marker)
		m.App = strings.TrimSpace(m.App)
		if m.Marker == "" || m.App == "" {
			continue
		}
		if len(m.Marker) > 64 || len(m.App) > 32 {
			return nil, errf(400, "浏览器特征过长")
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		return nil, errf(400, "请填写内置浏览器特征，或改回内置列表")
	}
	if len(out) > 200 {
		return nil, errf(400, "内置浏览器特征最多 200 条")
	}
	return out, nil
}

func omitPlatformStatic(in []pathRuleIn) []pathRuleIn {
	out := make([]pathRuleIn, 0, len(in))
	for _, r := range in {
		if pathrule.IsPlatformStatic(r.Pattern, r.MatchType, r.Action) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func cleanRules(in []pathRuleIn, custom bool) ([]pathRuleIn, error) {
	if !custom {
		return nil, nil
	}
	if len(in) == 0 || len(in) > 80 {
		return nil, errf(400, "默认路径规则需 1 到 80 条")
	}
	match := map[string]bool{domainsite.MatchPrefix: true, domainsite.MatchExact: true, domainsite.MatchSuffix: true}
	action := map[string]bool{domainsite.PathBypass: true, domainsite.PathAlwaysReal: true, domainsite.PathAlwaysReview: true, domainsite.PathBlock: true}
	out := make([]pathRuleIn, 0, len(in))
	for _, r := range in {
		r.Pattern = strings.TrimSpace(r.Pattern)
		if r.Pattern == "" || len(r.Pattern) > 200 || !match[r.MatchType] || !action[r.Action] {
			return nil, errf(400, "路径规则的匹配方式或动作无效")
		}
		out = append(out, r)
	}
	return out, nil
}
