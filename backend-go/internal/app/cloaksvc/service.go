// Package cloaksvc 商户斗篷的增删改查。判定在网关。
package cloaksvc

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/domain/cloak"
	"cloak/internal/domain/identity"
	"cloak/internal/domain/site"
)

var codeRE = regexp.MustCompile(`^[a-z0-9]{4,32}$`)

type Service struct {
	repo      *mysql.CloakRepo
	merchants identity.MerchantRepository
	now       func() time.Time
}

func New(repo *mysql.CloakRepo, merchants identity.MerchantRepository) *Service {
	return &Service{repo: repo, merchants: merchants, now: time.Now}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

type Input struct {
	Name               string   `json:"name"`
	DomainID           uint64   `json:"domainId"`
	RealURLs           []string `json:"realUrls"`
	ReviewMode         string   `json:"reviewMode"`
	ReviewURL          string   `json:"reviewUrl"`
	ReviewTag          string   `json:"reviewTag"`
	ShortLinkType      string   `json:"shortLinkType"`
	ShortLinkURL       string   `json:"shortLinkUrl"`
	AllowedSources     []string `json:"allowedSources"`
	Mode               string   `json:"mode"`
	TargetRegions      []string `json:"targetRegions"`
	SpiderWhitelist    []string `json:"spiderWhitelist"`
	DeviceWhitelist    string   `json:"deviceWhitelist"`
	BlockEmptyLanguage bool     `json:"blockEmptyLanguage"`
	BlockPC            bool     `json:"blockPc"`
	BlockProxy         bool     `json:"blockProxy"`
	IntermediatePage   string   `json:"intermediatePage"`
	IntermediateText   string   `json:"intermediateText"`
	Status             string   `json:"status"`
	Remark             string   `json:"remark"`
}

type View struct {
	ID                  uint64   `json:"id"`
	Name                string   `json:"name"`
	DomainID            uint64   `json:"domainId"`
	Domain              string   `json:"domain"`
	RealURLs            []string `json:"realUrls"`
	RealURL             string   `json:"realUrl"`
	ReviewMode          string   `json:"reviewMode"`
	ReviewURL           string   `json:"reviewUrl"`
	ReviewTag           string   `json:"reviewTag"`
	ShortLink           string   `json:"shortLink"`
	Code                string   `json:"code"`
	CodeCustom          bool     `json:"codeCustom"`
	AllowedSources      []string `json:"allowedSources"`
	AllowedSourcesText  string   `json:"allowedSourcesText"`
	Mode                string   `json:"mode"`
	TargetRegions       []string `json:"targetRegions"`
	TargetRegionsText   string   `json:"targetRegionsText"`
	SpiderWhitelist     []string `json:"spiderWhitelist"`
	DeviceWhitelist     string   `json:"deviceWhitelist"`
	DeviceWhitelistText string   `json:"deviceWhitelistText"`
	BlockEmptyLanguage  bool     `json:"blockEmptyLanguage"`
	BlockPC             bool     `json:"blockPc"`
	BlockProxy          bool     `json:"blockProxy"`
	IntermediatePage    string   `json:"intermediatePage"`
	IntermediateText    string   `json:"intermediateText"`
	Status              string   `json:"status"`
	Remark              string   `json:"remark"`
	ReviewHits          int64    `json:"reviewHits"`
	RealHits            int64    `json:"realHits"`
	TodayReviewHits     int64    `json:"todayReviewHits"`
	TodayRealHits       int64    `json:"todayRealHits"`
	CreatedAt           string   `json:"createdAt"`
}

func (s *Service) Stats(ctx context.Context, merchantID uint64) (map[string]any, error) {
	total, enabled, disabled, err := s.repo.CountStatus(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	today, err := s.repo.CountTodayJumps(ctx, merchantID, dayStart(m.Timezone, s.now()))
	if err != nil {
		return nil, err
	}
	return map[string]any{"total": total, "enabled": enabled, "disabled": disabled, "today": today}, nil
}

func (s *Service) List(ctx context.Context, merchantID uint64, keyword, mode, status string, domainID uint64, page, size int) ([]View, int64, error) {
	rows, total, err := s.repo.List(ctx, mysql.CloakFilter{
		MerchantID: merchantID, Keyword: strings.TrimSpace(keyword), DomainID: domainID, Mode: mode, Status: status, Page: page, Size: size,
	})
	if err != nil {
		return nil, 0, err
	}
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	hits, err := s.repo.HitStats(ctx, ids, dayStart(m.Timezone, s.now()))
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, 0, len(rows))
	for _, row := range rows {
		view := toView(row.Cloak, row.Domain)
		if h, ok := hits[row.ID]; ok {
			view.ReviewHits, view.RealHits = h.ReviewHits, h.RealHits
			view.TodayReviewHits, view.TodayRealHits = h.TodayReview, h.TodayReal
		}
		out = append(out, view)
	}
	return out, total, nil
}

func (s *Service) Get(ctx context.Context, merchantID, id uint64) (*View, error) {
	row, err := s.repo.Find(ctx, merchantID, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "斗篷不存在")
		}
		return nil, err
	}
	view := toView(row.Cloak, row.Domain)
	return &view, nil
}

func (s *Service) DomainOptions(ctx context.Context, merchantID, keepID uint64, includeTaken bool) ([]mysql.DomainOption, error) {
	return s.repo.DomainOptions(ctx, merchantID, keepID, includeTaken)
}

func (s *Service) Create(ctx context.Context, merchantID uint64, in Input) (uint64, error) {
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return 0, err
	}
	n, err := s.repo.Count(ctx, merchantID)
	if err != nil {
		return 0, err
	}
	quota := m.CloakQuota
	if quota == 0 {
		quota = 20
	}
	if n >= int64(quota) {
		return 0, errf(429, "斗篷数量已达上限")
	}
	row, err := s.build(ctx, merchantID, 0, in, true)
	if err != nil {
		return 0, err
	}
	row.MerchantID = merchantID
	if err := s.repo.Create(ctx, row); err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *Service) Update(ctx context.Context, merchantID, id uint64, in Input) error {
	old, err := s.repo.Find(ctx, merchantID, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return errf(404, "斗篷不存在")
		}
		return err
	}
	in.DomainID = old.DomainID
	row, err := s.build(ctx, merchantID, id, in, false)
	if err != nil {
		return err
	}
	row.ID = id
	row.MerchantID = merchantID
	row.DomainID = old.DomainID
	return s.repo.Save(ctx, row)
}

func (s *Service) Copy(ctx context.Context, merchantID, id uint64) (uint64, error) {
	old, err := s.repo.Find(ctx, merchantID, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return 0, errf(404, "斗篷不存在")
		}
		return 0, err
	}
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return 0, err
	}
	n, err := s.repo.Count(ctx, merchantID)
	if err != nil {
		return 0, err
	}
	quota := m.CloakQuota
	if quota == 0 {
		quota = 20
	}
	if n >= int64(quota) {
		return 0, errf(429, "斗篷数量已达上限")
	}
	code, err := s.uniqueCode(ctx, 0)
	if err != nil {
		return 0, err
	}
	name := strings.TrimSpace(old.Name + " 副本")
	if len([]rune(name)) > 64 {
		name = string([]rune(name)[:64])
	}
	row := old.Cloak
	row.ID = 0
	row.Name = name
	row.Code = code
	row.CodeCustom = false
	row.Status = cloak.StatusDisabled
	row.CreatedAt, row.UpdatedAt = time.Time{}, time.Time{}
	if err := s.repo.Create(ctx, &row); err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *Service) ChangeStatus(ctx context.Context, merchantID, id uint64, status string) error {
	if status != cloak.StatusEnabled && status != cloak.StatusDisabled {
		return errf(400, "状态无效")
	}
	row, err := s.repo.Find(ctx, merchantID, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return errf(404, "斗篷不存在")
		}
		return err
	}
	if status == cloak.StatusEnabled {
		taken, err := s.repo.CodeTaken(ctx, row.Code, id)
		if err != nil {
			return err
		}
		if taken {
			return errf(409, "短链编码已被占用")
		}
	}
	row.Status = status
	return s.repo.Save(ctx, &row.Cloak)
}

func (s *Service) Delete(ctx context.Context, merchantID, id uint64) error {
	err := s.repo.Delete(ctx, merchantID, id)
	if errors.Is(err, mysql.ErrNotFound) {
		return errf(404, "斗篷不存在")
	}
	return err
}

func (s *Service) build(ctx context.Context, merchantID, id uint64, in Input, creating bool) (*cloak.Cloak, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len([]rune(name)) > 64 {
		return nil, errf(400, "名称须为 1-64 字")
	}
	dom, err := s.repo.DomainUse(ctx, in.DomainID)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(400, "请选择关联域名")
		}
		return nil, err
	}
	if dom.MerchantID != merchantID || dom.Status != site.DomainBound {
		return nil, errf(400, "请选择已校验通过的域名")
	}
	if dom.SiteType == site.TypeHosted || dom.SiteType == site.TypeHybrid {
		return nil, errf(409, "该域名已用于托管站点")
	}
	urls, err := cleanURLs(in.RealURLs)
	if err != nil {
		return nil, err
	}
	mode := in.ReviewMode
	if mode != cloak.ReviewLocal {
		mode = cloak.ReviewExternal
	}
	reviewURL := strings.TrimSpace(in.ReviewURL)
	if mode == cloak.ReviewExternal {
		if !validHTTP(reviewURL) {
			return nil, errf(400, "请填写 http 或 https 的审核链接")
		}
	} else if reviewURL != "" && !validHTTP(reviewURL) {
		return nil, errf(400, "审核链接须为 http 或 https")
	}
	tag := in.ReviewTag
	if mode == cloak.ReviewLocal && !oneOf(tag, "google", "facebook", "tiktok", "generic") {
		return nil, errf(400, "请选择快速审核页")
	}
	if mode != cloak.ReviewLocal {
		tag = ""
	}
	var code string
	custom := false
	if in.ShortLinkType == "custom" {
		code, err = parseCustomCode(in.ShortLinkURL, dom.Domain)
		if err != nil {
			return nil, err
		}
		custom = true
	} else if !creating && id > 0 {
		old, err := s.repo.Find(ctx, merchantID, id)
		if err != nil {
			return nil, err
		}
		if !old.CodeCustom {
			code = old.Code
		} else {
			code, err = s.uniqueCode(ctx, id)
			if err != nil {
				return nil, err
			}
		}
	} else {
		code, err = s.uniqueCode(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	taken, err := s.repo.CodeTaken(ctx, code, id)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, errf(400, "短链编码已被占用")
	}
	sources := normalizeSources(in.AllowedSources)
	runMode := in.Mode
	if !oneOf(runMode, cloak.ModeCloak, cloak.ModeGreen, cloak.ModeFullOpen, cloak.ModeReview) {
		return nil, errf(400, "请选择模式")
	}
	regions := upperCodes(in.TargetRegions)
	for _, c := range regions {
		if c == "CN" || c == "HK" || c == "MO" || c == "TW" {
			return nil, errf(400, "投放地区不包含中国及港澳台")
		}
	}
	spiders := filterSpiders(in.SpiderWhitelist)
	device := strings.TrimSpace(in.DeviceWhitelist)
	if device != "" && !oneOf(device, "mobile", "tablet", "pc", "mobile,tablet") {
		return nil, errf(400, "设备白名单无效")
	}
	page := in.IntermediatePage
	if !oneOf(page, cloak.PageNone, cloak.PageSystem, cloak.PageCustom) {
		page = cloak.PageNone
	}
	text := strings.TrimSpace(in.IntermediateText)
	if len([]rune(text)) > 500 {
		return nil, errf(400, "自定义中间页最多 500 字")
	}
	status := in.Status
	if status != cloak.StatusDisabled {
		status = cloak.StatusEnabled
	}
	remark := strings.TrimSpace(in.Remark)
	if len([]rune(remark)) > 200 {
		return nil, errf(400, "备注最多 200 字")
	}
	return &cloak.Cloak{
		DomainID: in.DomainID, Name: name, Status: status,
		RealURLs: mustJSON(urls), ReviewMode: mode, ReviewURL: reviewURL, ReviewTag: tag,
		Code: code, CodeCustom: custom, AllowedSources: mustJSON(sources), Mode: runMode,
		TargetRegions: mustJSON(regions), SpiderWhitelist: mustJSON(spiders), DeviceWhitelist: device,
		BlockEmptyLanguage: in.BlockEmptyLanguage, BlockPC: in.BlockPC, BlockProxy: in.BlockProxy,
		IntermediatePage: page, IntermediateText: text, Remark: remark,
	}, nil
}

func (s *Service) uniqueCode(ctx context.Context, except uint64) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < 5; i++ {
		buf := make([]byte, 6)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for j := range buf {
			buf[j] = alphabet[int(buf[j])%len(alphabet)]
		}
		code := string(buf)
		taken, err := s.repo.CodeTaken(ctx, code, except)
		if err != nil {
			return "", err
		}
		if !taken {
			return code, nil
		}
	}
	return "", errf(400, "短链生成失败，请重试")
}

func parseCustomCode(raw, domain string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), domain) {
		return "", errf(400, "自定义短链须为 https://"+domain+"/c/编码")
	}
	code := strings.TrimPrefix(u.Path, "/c/")
	code = strings.ToLower(code)
	if strings.Contains(code, "/") || !codeRE.MatchString(code) {
		return "", errf(400, "短链编码为 4-32 位小写字母和数字")
	}
	return code, nil
}

func cleanURLs(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if !validHTTP(item) || len(item) > 1000 {
			return nil, errf(400, "真实链接须为 http 或 https，且不超过 1000 字")
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, errf(400, "请至少填写一条真实链接")
	}
	return out, nil
}

func validHTTP(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func normalizeSources(in []string) []string {
	allowed := map[string]bool{"facebook": true, "google": true, "tiktok": true, "kwai": true, "all": true}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.ToLower(strings.TrimSpace(item))
		if allowed[item] {
			out = append(out, item)
		}
	}
	for _, item := range out {
		if item == "all" {
			return []string{"all"}
		}
	}
	if len(out) == 0 {
		return []string{"all"}
	}
	return out
}

func upperCodes(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.ToUpper(strings.TrimSpace(item))
		if len(item) < 2 || len(item) > 8 || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func filterSpiders(in []string) []string {
	allowed := map[string]bool{"googlebot": true, "bingbot": true, "facebookexternalhit": true, "tiktokbot": true}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.ToLower(strings.TrimSpace(item))
		if allowed[item] {
			out = append(out, item)
		}
	}
	return out
}

func oneOf(v string, options ...string) bool {
	for _, item := range options {
		if v == item {
			return true
		}
	}
	return false
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		return "[]"
	}
	return string(b)
}

func parseList(raw string) []string {
	var out []string
	if json.Unmarshal([]byte(raw), &out) != nil || out == nil {
		return []string{}
	}
	return out
}

func toView(row cloak.Cloak, domain string) View {
	urls := parseList(row.RealURLs)
	real := ""
	if len(urls) > 0 {
		real = urls[0]
	}
	regions := parseList(row.TargetRegions)
	regionText := "不限"
	if len(regions) > 0 {
		regionText = strings.Join(regions, "、")
	}
	sources := parseList(row.AllowedSources)
	sourceText := sourceLabels(sources)
	deviceText := "不限"
	switch row.DeviceWhitelist {
	case "mobile":
		deviceText = "仅移动端"
	case "tablet":
		deviceText = "仅平板"
	case "pc":
		deviceText = "仅 PC"
	case "mobile,tablet":
		deviceText = "移动端 + 平板"
	}
	return View{
		ID: row.ID, Name: row.Name, DomainID: row.DomainID, Domain: domain,
		RealURLs: urls, RealURL: real, ReviewMode: row.ReviewMode, ReviewURL: row.ReviewURL, ReviewTag: row.ReviewTag,
		ShortLink: "https://" + domain + "/c/" + row.Code, Code: row.Code, CodeCustom: row.CodeCustom,
		AllowedSources: sources, AllowedSourcesText: sourceText, Mode: row.Mode,
		TargetRegions: regions, TargetRegionsText: regionText,
		SpiderWhitelist: parseList(row.SpiderWhitelist),
		DeviceWhitelist: row.DeviceWhitelist, DeviceWhitelistText: deviceText,
		BlockEmptyLanguage: row.BlockEmptyLanguage, BlockPC: row.BlockPC, BlockProxy: row.BlockProxy,
		IntermediatePage: row.IntermediatePage, IntermediateText: row.IntermediateText,
		Status: row.Status, Remark: row.Remark,
		CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func sourceLabels(sources []string) string {
	names := map[string]string{"facebook": "Facebook", "google": "Google", "tiktok": "TikTok", "kwai": "Kwai", "all": "不限制"}
	out := make([]string, 0, len(sources))
	for _, item := range sources {
		if name, ok := names[item]; ok {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return "不限制"
	}
	return strings.Join(out, "、")
}

func dayStart(tz string, now time.Time) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Shanghai")
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}
