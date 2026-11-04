// Package attrsvc 归因参数字典、站点校验规则、分销码与链接生成。
package attrsvc

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/domain/attribution"
	"cloak/internal/domain/identity"
	domainsite "cloak/internal/domain/site"
)

type Service struct {
	repo      *mysql.AttributionRepo
	sites     *mysql.SiteRepo
	merchants identity.MerchantRepository
	changed   func(siteID uint64) // 站点级变更
	changeAll func()              // 内置字典变更，影响全部站点
}

func New(repo *mysql.AttributionRepo, sites *mysql.SiteRepo, merchants identity.MerchantRepository) *Service {
	return &Service{repo: repo, sites: sites, merchants: merchants}
}

// OnChange 注册配置变更回调。
func (s *Service) OnChange(site func(siteID uint64), all func()) {
	s.changed, s.changeAll = site, all
}

func (s *Service) notify(siteID uint64) {
	if s.changed != nil && siteID > 0 {
		s.changed(siteID)
	}
}

func (s *Service) notifyAll() {
	if s.changeAll != nil {
		s.changeAll()
	}
}

// notifyCodes 按码 ID 找到涉及的站点并通知。
func (s *Service) notifyCodes(ctx context.Context, ids []uint64) {
	if s.changed == nil {
		return
	}
	seen := map[uint64]bool{}
	for _, id := range ids {
		c, err := s.repo.FindCode(ctx, id)
		if err != nil || seen[c.SiteID] {
			continue
		}
		seen[c.SiteID] = true
		s.changed(c.SiteID)
	}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

const timeLayout = "2006-01-02 15:04:05"

var nameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)
var codeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{2,64}$`)

// ---- 参数字典 ----

type ParamInput struct {
	Name       string
	ParamType  string
	Platform   string
	Aliases    []string
	MinLen     uint32
	MaxLen     uint32
	Pattern    string
	Sufficient bool
	Status     string
	Sort       uint32
	Remark     string
}

// Params 返回站点生效的参数字典：内置行 + 站点行，站点行覆盖同名内置行。
func (s *Service) Params(ctx context.Context, merchantID, siteID uint64) ([]map[string]any, error) {
	if _, err := s.ownedSite(ctx, merchantID, siteID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListParams(ctx, siteID)
	if err != nil {
		return nil, err
	}
	builtin := map[string]*attribution.Param{}
	site := map[string]*attribution.Param{}
	for i := range rows {
		p := &rows[i]
		if p.SiteID == 0 {
			builtin[strings.ToLower(p.Name)] = p
		} else {
			site[strings.ToLower(p.Name)] = p
		}
	}
	out := make([]map[string]any, 0, len(rows))
	seen := map[string]bool{}
	for _, p := range rows {
		key := strings.ToLower(p.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		var row map[string]any
		switch {
		case site[key] != nil && builtin[key] != nil:
			row = paramRow(site[key])
			row["source"] = "override"
			row["builtinId"] = builtin[key].ID
		case site[key] != nil:
			row = paramRow(site[key])
			row["source"] = "site"
		default:
			row = paramRow(builtin[key])
			row["source"] = "builtin"
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i]["sort"].(uint32) < out[j]["sort"].(uint32)
	})
	return out, nil
}

// SaveParam 新建或覆盖站点参数。同名内置行存在时生成覆盖行。
func (s *Service) SaveParam(ctx context.Context, merchantID, siteID uint64, in ParamInput) (map[string]any, error) {
	if _, err := s.ownedSite(ctx, merchantID, siteID); err != nil {
		return nil, err
	}
	p, err := s.buildParam(ctx, siteID, in)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.FindParamByName(ctx, siteID, p.Name)
	switch {
	case err == nil:
		p.ID = existing.ID
		p.CreatedAt = existing.CreatedAt
		if err := s.repo.UpdateParam(ctx, p, "ParamType", "Platform", "Aliases", "MinLen", "MaxLen", "Pattern", "Sufficient", "Status", "Sort", "Remark"); err != nil {
			return nil, err
		}
	case errors.Is(err, mysql.ErrNotFound):
		if err := s.repo.CreateParam(ctx, p); err != nil {
			return nil, err
		}
	default:
		return nil, err
	}
	s.notify(siteID)
	return paramRow(p), nil
}

// DeleteParam 删除站点参数；若为覆盖行则恢复内置。
func (s *Service) DeleteParam(ctx context.Context, merchantID, siteID, id uint64) error {
	if _, err := s.ownedSite(ctx, merchantID, siteID); err != nil {
		return err
	}
	p, err := s.repo.FindParam(ctx, id)
	if err != nil || p.SiteID != siteID {
		return errf(404, "参数不存在")
	}
	if err := s.repo.DeleteParam(ctx, id); err != nil {
		return err
	}
	s.notify(siteID)
	return nil
}

// Builtins 平台内置参数（总后台）。
func (s *Service) Builtins(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.repo.ListParams(ctx, 0)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		out = append(out, paramRow(&rows[i]))
	}
	return out, nil
}

func (s *Service) CreateBuiltin(ctx context.Context, in ParamInput) (map[string]any, error) {
	p, err := s.buildParam(ctx, 0, in)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.FindParamByName(ctx, 0, p.Name); err == nil {
		return nil, errf(400, "参数名已存在")
	}
	if err := s.repo.CreateParam(ctx, p); err != nil {
		return nil, err
	}
	s.notifyAll()
	return paramRow(p), nil
}

func (s *Service) UpdateBuiltin(ctx context.Context, id uint64, in ParamInput) error {
	existing, err := s.repo.FindParam(ctx, id)
	if err != nil || existing.SiteID != 0 {
		return errf(404, "参数不存在")
	}
	in.Name = existing.Name
	p, err := s.buildParam(ctx, 0, in)
	if err != nil {
		return err
	}
	p.ID = existing.ID
	if err := s.repo.UpdateParam(ctx, p, "ParamType", "Platform", "Aliases", "MinLen", "MaxLen", "Pattern", "Sufficient", "Status", "Sort", "Remark"); err != nil {
		return err
	}
	s.notifyAll()
	return nil
}

func (s *Service) DeleteBuiltin(ctx context.Context, id uint64) error {
	existing, err := s.repo.FindParam(ctx, id)
	if err != nil || existing.SiteID != 0 {
		return errf(404, "参数不存在")
	}
	if err := s.repo.DeleteParam(ctx, id); err != nil {
		return err
	}
	s.notifyAll()
	return nil
}

func (s *Service) buildParam(ctx context.Context, siteID uint64, in ParamInput) (*attribution.Param, error) {
	in.Name = strings.ToLower(strings.TrimSpace(in.Name))
	if !nameRe.MatchString(in.Name) {
		return nil, errf(400, "参数名须为 1-64 位字母、数字、_ . -")
	}
	valid := false
	for _, t := range attribution.ValidTypes {
		if in.ParamType == t {
			valid = true
		}
	}
	if !valid {
		return nil, errf(400, "参数类型无效")
	}
	if in.MinLen == 0 {
		in.MinLen = 1
	}
	if in.MaxLen == 0 {
		in.MaxLen = 512
	}
	if in.MinLen > in.MaxLen || in.MaxLen > 2048 {
		return nil, errf(400, "长度区间无效")
	}
	in.Pattern = strings.TrimSpace(in.Pattern)
	if in.Pattern != "" {
		if _, err := regexp.Compile(in.Pattern); err != nil {
			return nil, errf(400, "正则无效："+err.Error())
		}
	}
	if in.Status != attribution.StatusDisabled {
		in.Status = attribution.StatusEnabled
	}
	aliases := make([]string, 0, len(in.Aliases))
	seen := map[string]bool{in.Name: true}
	for _, a := range in.Aliases {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" || seen[a] {
			continue
		}
		if !nameRe.MatchString(a) {
			return nil, errf(400, "别名格式无效："+a)
		}
		seen[a] = true
		aliases = append(aliases, a)
	}
	// 别名不能与同作用域其它参数的名字或别名冲突
	rows, err := s.repo.ListParams(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, other := range rows {
		if strings.EqualFold(other.Name, in.Name) {
			continue
		}
		names := append(parseStrings(other.Aliases), strings.ToLower(other.Name))
		for _, n := range names {
			if seen[n] {
				return nil, errf(400, fmt.Sprintf("「%s」已被参数 %s 使用", n, other.Name))
			}
		}
	}
	raw, _ := json.Marshal(aliases)
	return &attribution.Param{
		SiteID: siteID, Name: in.Name, ParamType: in.ParamType, Platform: strPtr(in.Platform),
		Aliases: raw, MinLen: in.MinLen, MaxLen: in.MaxLen, Pattern: strPtr(in.Pattern),
		Sufficient: in.Sufficient, Status: in.Status, Sort: in.Sort, Remark: strPtr(in.Remark),
	}, nil
}

// ---- 站点规则 ----

type RuleInput struct {
	CombineRule           string
	ValidateCodes         bool
	ValidateClickFormat   bool
	AllowedSources        []string
	ClickReuseLimit       uint32
	ClickReuseWindowHours uint32
	ClickReuseAction      string
	CodeRevokeMode        string
}

func (s *Service) Rule(ctx context.Context, merchantID, siteID uint64) (map[string]any, error) {
	if _, err := s.ownedSite(ctx, merchantID, siteID); err != nil {
		return nil, err
	}
	r, err := s.repo.FindRule(ctx, siteID)
	if err != nil {
		return nil, err
	}
	return ruleRow(r), nil
}

func (s *Service) SaveRule(ctx context.Context, merchantID, siteID uint64, in RuleInput) error {
	if _, err := s.ownedSite(ctx, merchantID, siteID); err != nil {
		return err
	}
	if in.CombineRule != attribution.CombineClickOrCode {
		in.CombineRule = attribution.CombineAny
	}
	if in.ClickReuseAction != attribution.ReuseReview {
		in.ClickReuseAction = attribution.ReuseDowngrade
	}
	if in.CodeRevokeMode != attribution.RevokeImmediate {
		in.CodeRevokeMode = attribution.RevokeWindow
	}
	if in.ClickReuseWindowHours == 0 {
		in.ClickReuseWindowHours = 24
	}
	if in.ClickReuseWindowHours > 720 {
		return errf(400, "复用时间窗最长 720 小时")
	}
	sources := make([]string, 0, len(in.AllowedSources))
	seen := map[string]bool{}
	for _, src := range in.AllowedSources {
		src = strings.ToLower(strings.TrimSpace(src))
		if src == "" || seen[src] {
			continue
		}
		seen[src] = true
		sources = append(sources, src)
	}
	raw, _ := json.Marshal(sources)
	if err := s.repo.SaveRule(ctx, &attribution.Rule{
		SiteID: siteID, CombineRule: in.CombineRule, ValidateCodes: in.ValidateCodes,
		ValidateClickFormat: in.ValidateClickFormat, AllowedSources: raw,
		ClickReuseLimit: in.ClickReuseLimit, ClickReuseWindowHours: in.ClickReuseWindowHours,
		ClickReuseAction: in.ClickReuseAction, CodeRevokeMode: in.CodeRevokeMode,
	}); err != nil {
		return err
	}
	s.notify(siteID)
	return nil
}

// ---- 分销码 ----

type CodeInput struct {
	SiteID       uint64
	Code         string
	CodeType     string
	Name         string
	Channel      string
	Status       string
	StartsAt     string
	ExpiresAt    string
	AllowedPaths []string
	Remark       string
}

type CodeListInput struct {
	SiteID   uint64
	Keyword  string
	CodeType string
	Status   string
	Channel  string
	Page     int
	Size     int
}

func (s *Service) ListCodes(ctx context.Context, merchantID uint64, in CodeListInput) ([]map[string]any, int64, error) {
	rows, total, err := s.repo.ListCodes(ctx, mysql.CodeFilter{
		MerchantID: merchantID, SiteID: in.SiteID, Keyword: in.Keyword, CodeType: in.CodeType,
		Status: in.Status, Channel: in.Channel, Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return nil, 0, err
	}
	names := s.siteNames(ctx, rows)
	merchantNames := map[uint64]string{}
	if merchantID == 0 && s.sites != nil {
		mids := make([]uint64, 0, len(rows))
		for i := range rows {
			mids = append(mids, rows[i].MerchantID)
		}
		merchantNames, _ = s.sites.MerchantNames(ctx, mids)
	}
	now := time.Now()
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		row := codeRow(&rows[i], now)
		row["siteName"] = names[rows[i].SiteID]
		row["merchantId"] = rows[i].MerchantID
		row["merchantName"] = merchantNames[rows[i].MerchantID]
		out = append(out, row)
	}
	return out, total, nil
}

func (s *Service) CodeStats(ctx context.Context, merchantID, siteID uint64) (map[string]any, error) {
	st, err := s.repo.CodeStats(ctx, merchantID, siteID)
	if err != nil {
		return nil, err
	}
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	total, _ := s.repo.CountCodes(ctx, merchantID)
	return map[string]any{
		"total": st.Total, "enabled": st.Enabled, "disabled": st.Disabled, "expired": st.Expired,
		"visits": st.Visits, "quota": m.CodeQuota, "used": total,
	}, nil
}

func (s *Service) Channels(ctx context.Context, merchantID uint64) ([]string, error) {
	return s.repo.Channels(ctx, merchantID)
}

func (s *Service) CreateCode(ctx context.Context, merchantID, userID uint64, in CodeInput) (map[string]any, error) {
	if _, err := s.ownedSite(ctx, merchantID, in.SiteID); err != nil {
		return nil, err
	}
	c, err := s.buildCode(merchantID, userID, in)
	if err != nil {
		return nil, err
	}
	if err := s.checkQuota(ctx, merchantID, 1); err != nil {
		return nil, err
	}
	if _, err := s.repo.FindCodeByValue(ctx, in.SiteID, c.Code); err == nil {
		return nil, errf(400, "该码已登记")
	}
	if err := s.repo.CreateCode(ctx, c); err != nil {
		return nil, err
	}
	s.notify(in.SiteID)
	return codeRow(c, time.Now()), nil
}

func (s *Service) UpdateCode(ctx context.Context, merchantID, id uint64, in CodeInput) error {
	c, err := s.ownedCode(ctx, merchantID, id)
	if err != nil {
		return err
	}
	in.SiteID = c.SiteID
	in.Code = c.Code
	next, err := s.buildCode(merchantID, 0, in)
	if err != nil {
		return err
	}
	next.ID = c.ID
	if err := s.repo.UpdateCode(ctx, next, "CodeType", "Name", "Channel", "Status", "StartsAt", "ExpiresAt", "AllowedPaths", "Remark"); err != nil {
		return err
	}
	s.notify(c.SiteID)
	return nil
}

// PlatformChangeCodesStatus 总后台强制停用或恢复，不校验商户归属。
func (s *Service) PlatformChangeCodesStatus(ctx context.Context, ids []uint64, status string) (int64, error) {
	if status != attribution.StatusEnabled && status != attribution.StatusDisabled {
		return 0, errf(400, "状态无效")
	}
	n, err := s.repo.UpdateCodesStatusPlatform(ctx, ids, status)
	if err == nil {
		s.notifyCodes(ctx, ids)
	}
	return n, err
}

func (s *Service) ChangeCodesStatus(ctx context.Context, merchantID uint64, ids []uint64, status string) (int64, error) {
	if status != attribution.StatusEnabled && status != attribution.StatusDisabled {
		return 0, errf(400, "状态无效")
	}
	n, err := s.repo.UpdateCodesStatus(ctx, merchantID, ids, status)
	if err == nil {
		s.notifyCodes(ctx, ids)
	}
	return n, err
}

func (s *Service) DeleteCodes(ctx context.Context, merchantID uint64, ids []uint64) (int64, error) {
	// 删除前先找出站点，删除后再通知
	seen := map[uint64]bool{}
	for _, id := range ids {
		if c, err := s.repo.FindCode(ctx, id); err == nil && c.MerchantID == merchantID {
			seen[c.SiteID] = true
		}
	}
	n, err := s.repo.DeleteCodes(ctx, merchantID, ids)
	if err == nil {
		for sid := range seen {
			s.notify(sid)
		}
	}
	return n, err
}

type GenerateInput struct {
	SiteID    uint64
	CodeType  string
	Prefix    string
	Count     int
	Length    int
	Channel   string
	ExpiresAt string
	Remark    string
}

// GenerateCodes 批量生成：前缀 + 随机，跳过与现有冲突的码。
func (s *Service) GenerateCodes(ctx context.Context, merchantID, userID uint64, in GenerateInput) (map[string]any, error) {
	if _, err := s.ownedSite(ctx, merchantID, in.SiteID); err != nil {
		return nil, err
	}
	if in.Count <= 0 || in.Count > 10000 {
		return nil, errf(400, "单次生成 1-10000 个")
	}
	if in.Length <= 0 {
		in.Length = 8
	}
	if in.Length < 4 || in.Length > 32 {
		return nil, errf(400, "随机部分长度 4-32")
	}
	in.Prefix = strings.TrimSpace(in.Prefix)
	if in.Prefix != "" && !codeRe.MatchString(in.Prefix+"aa") {
		return nil, errf(400, "前缀只能是字母、数字、_ -")
	}
	if err := s.checkQuota(ctx, merchantID, in.Count); err != nil {
		return nil, err
	}
	exp, err := parseTime(in.ExpiresAt)
	if err != nil {
		return nil, errf(400, "有效期格式无效")
	}
	codeType := in.CodeType
	if codeType != attribution.CodeCampaign {
		codeType = attribution.CodeDistributor
	}
	start := time.Now()
	candidates := make([]string, 0, in.Count*2)
	seen := map[string]bool{}
	for len(candidates) < in.Count*2 {
		c := in.Prefix + randomCode(in.Length)
		if !seen[c] {
			seen[c] = true
			candidates = append(candidates, c)
		}
	}
	existing, err := s.repo.ExistingCodes(ctx, in.SiteID, candidates)
	if err != nil {
		return nil, err
	}
	var uid *uint64
	if userID > 0 {
		uid = &userID
	}
	rows := make([]attribution.Code, 0, in.Count)
	allow, _ := json.Marshal([]string{})
	for _, c := range candidates {
		if existing[c] {
			continue
		}
		rows = append(rows, attribution.Code{
			MerchantID: merchantID, SiteID: in.SiteID, Code: c, CodeType: codeType,
			Channel: strPtr(in.Channel), Status: attribution.StatusEnabled, ExpiresAt: exp,
			AllowedPaths: allow, Remark: strPtr(in.Remark), CreatedBy: uid,
		})
		if len(rows) == in.Count {
			break
		}
	}
	n, err := s.repo.CreateCodes(ctx, rows)
	if err != nil {
		return nil, err
	}
	s.notify(in.SiteID)
	sample := make([]string, 0, 5)
	for i := 0; i < len(rows) && i < 5; i++ {
		sample = append(sample, rows[i].Code)
	}
	return map[string]any{"created": n, "sample": sample, "elapsedMs": time.Since(start).Milliseconds()}, nil
}

type ImportInput struct {
	SiteID   uint64
	CodeType string
	Content  string // 每行：code[,name[,channel[,expires_at]]]
	Channel  string
}

// ImportCodes 文本导入，返回成功数与逐行冲突。
func (s *Service) ImportCodes(ctx context.Context, merchantID, userID uint64, in ImportInput) (map[string]any, error) {
	if _, err := s.ownedSite(ctx, merchantID, in.SiteID); err != nil {
		return nil, err
	}
	codeType := in.CodeType
	if codeType != attribution.CodeCampaign {
		codeType = attribution.CodeDistributor
	}
	lines := strings.Split(strings.ReplaceAll(in.Content, "\r\n", "\n"), "\n")
	type conflict struct {
		Line   int    `json:"line"`
		Code   string `json:"code"`
		Reason string `json:"reason"`
	}
	var conflicts []conflict
	rows := make([]attribution.Code, 0, len(lines))
	seen := map[string]bool{}
	codes := make([]string, 0, len(lines))
	var uid *uint64
	if userID > 0 {
		uid = &userID
	}
	allow, _ := json.Marshal([]string{})
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i == 0 && strings.HasPrefix(strings.ToLower(line), "code") {
			continue // 表头
		}
		parts := strings.Split(line, ",")
		for j := range parts {
			parts[j] = strings.TrimSpace(parts[j])
		}
		code := parts[0]
		if !codeRe.MatchString(code) {
			conflicts = append(conflicts, conflict{Line: i + 1, Code: code, Reason: "格式无效，须为 2-64 位字母数字 _ -"})
			continue
		}
		if seen[code] {
			conflicts = append(conflicts, conflict{Line: i + 1, Code: code, Reason: "文件内重复"})
			continue
		}
		seen[code] = true
		row := attribution.Code{
			MerchantID: merchantID, SiteID: in.SiteID, Code: code, CodeType: codeType,
			Status: attribution.StatusEnabled, AllowedPaths: allow, CreatedBy: uid, Channel: strPtr(in.Channel),
		}
		if len(parts) > 1 {
			row.Name = strPtr(parts[1])
		}
		if len(parts) > 2 && parts[2] != "" {
			row.Channel = strPtr(parts[2])
		}
		if len(parts) > 3 && parts[3] != "" {
			exp, err := parseTime(parts[3])
			if err != nil {
				conflicts = append(conflicts, conflict{Line: i + 1, Code: code, Reason: "有效期格式无效"})
				continue
			}
			row.ExpiresAt = exp
		}
		rows = append(rows, row)
		codes = append(codes, code)
	}
	if len(rows) > 10000 {
		return nil, errf(400, "单次导入最多 10000 行")
	}
	if err := s.checkQuota(ctx, merchantID, len(rows)); err != nil {
		return nil, err
	}
	existing, err := s.repo.ExistingCodes(ctx, in.SiteID, codes)
	if err != nil {
		return nil, err
	}
	fresh := rows[:0]
	for _, r := range rows {
		if existing[r.Code] {
			conflicts = append(conflicts, conflict{Code: r.Code, Reason: "已登记"})
			continue
		}
		fresh = append(fresh, r)
	}
	n, err := s.repo.CreateCodes(ctx, fresh)
	if err != nil {
		return nil, err
	}
	s.notify(in.SiteID)
	if conflicts == nil {
		conflicts = []conflict{}
	}
	return map[string]any{"created": n, "skipped": len(conflicts), "conflicts": conflicts}, nil
}

// ExportCodes 导出 CSV。
func (s *Service) ExportCodes(ctx context.Context, merchantID uint64, in CodeListInput) ([]byte, error) {
	rows, err := s.repo.AllCodes(ctx, mysql.CodeFilter{
		MerchantID: merchantID, SiteID: in.SiteID, Keyword: in.Keyword, CodeType: in.CodeType,
		Status: in.Status, Channel: in.Channel,
	})
	if err != nil {
		return nil, err
	}
	names := s.siteNames(ctx, rows)
	var b strings.Builder
	b.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"code", "type", "name", "channel", "status", "starts_at", "expires_at", "site", "visits", "first_pass", "returns", "conversions", "last_visit_at", "created_at"})
	now := time.Now()
	for i := range rows {
		c := &rows[i]
		_ = w.Write([]string{
			c.Code, c.CodeType, deref(c.Name), deref(c.Channel), c.EffectiveStatus(now),
			fmtTime(c.StartsAt), fmtTime(c.ExpiresAt), names[c.SiteID],
			fmt.Sprint(c.VisitCount), fmt.Sprint(c.FirstPassCount), fmt.Sprint(c.ReturnCount), fmt.Sprint(c.ConversionCount),
			fmtTime(c.LastVisitAt), c.CreatedAt.Format(timeLayout),
		})
	}
	w.Flush()
	return []byte(b.String()), nil
}

// ---- 链接生成 ----

type LinkInput struct {
	SiteID  uint64
	BaseURL string
	Path    string
	Code    string
	Params  map[string]string
}

// LinkOptions 链接生成器需要的站点域名与参数名。
func (s *Service) LinkOptions(ctx context.Context, merchantID, siteID uint64) (map[string]any, error) {
	if _, err := s.ownedSite(ctx, merchantID, siteID); err != nil {
		return nil, err
	}
	domains, _, err := s.sites.ListDomains(ctx, merchantID, siteID, "", "", 1, 100)
	if err != nil {
		return nil, err
	}
	hosts := make([]map[string]any, 0, len(domains))
	for _, d := range domains {
		hosts = append(hosts, map[string]any{"domain": d.Domain, "status": d.Status})
	}
	params, err := s.Params(ctx, merchantID, siteID)
	if err != nil {
		return nil, err
	}
	codeParam, campaignParam := "", ""
	sources := []map[string]any{}
	for _, p := range params {
		if p["status"] != attribution.StatusEnabled {
			continue
		}
		switch p["paramType"] {
		case attribution.TypeDistributor:
			if codeParam == "" {
				codeParam = p["name"].(string)
			}
		case attribution.TypeCampaign:
			if campaignParam == "" {
				campaignParam = p["name"].(string)
			}
		case attribution.TypeSource:
			sources = append(sources, map[string]any{"name": p["name"]})
		}
	}
	return map[string]any{
		"domains": hosts, "codeParam": codeParam, "campaignParam": campaignParam, "sourceParams": sources,
	}, nil
}

// BuildLink 拼接推广链接，并校验分销码已登记且可用。
func (s *Service) BuildLink(ctx context.Context, merchantID uint64, in LinkInput) (map[string]any, error) {
	if _, err := s.ownedSite(ctx, merchantID, in.SiteID); err != nil {
		return nil, err
	}
	base := strings.TrimSpace(in.BaseURL)
	if base == "" {
		return nil, errf(400, "请选择域名或填写地址")
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return nil, errf(400, "地址无效")
	}
	if p := strings.TrimSpace(in.Path); p != "" {
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		u.Path = p
	}
	if u.Path == "" {
		u.Path = "/"
	}
	warnings := []string{}
	q := u.Query()
	opts, err := s.LinkOptions(ctx, merchantID, in.SiteID)
	if err != nil {
		return nil, err
	}
	if code := strings.TrimSpace(in.Code); code != "" {
		name, _ := opts["codeParam"].(string)
		if name == "" {
			return nil, errf(400, "站点未启用分销码参数")
		}
		c, err := s.repo.FindCodeByValue(ctx, in.SiteID, code)
		if err != nil {
			return nil, errf(400, "分销码未登记："+code)
		}
		if c.CodeType == attribution.CodeCampaign {
			if cp, _ := opts["campaignParam"].(string); cp != "" {
				name = cp
			}
		}
		if st := c.EffectiveStatus(time.Now()); st != attribution.StatusEnabled {
			warnings = append(warnings, "该码当前状态为 "+st+"，访客访问时不会放行")
		}
		q.Set(name, c.Code)
	}
	keys := make([]string, 0, len(in.Params))
	for k := range in.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		k2 := strings.TrimSpace(k)
		v := strings.TrimSpace(in.Params[k])
		if k2 == "" || v == "" {
			continue
		}
		q.Set(k2, v)
	}
	u.RawQuery = q.Encode()
	return map[string]any{"url": u.String(), "warnings": warnings}, nil
}

// ---- helpers ----

func (s *Service) ownedSite(ctx context.Context, merchantID, siteID uint64) (*domainsite.Site, error) {
	if siteID == 0 {
		return nil, errf(400, "请选择站点")
	}
	site, err := s.sites.FindSite(ctx, siteID)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "站点不存在")
		}
		return nil, err
	}
	if merchantID > 0 && site.MerchantID != merchantID {
		return nil, errf(404, "站点不存在")
	}
	return site, nil
}

func (s *Service) ownedCode(ctx context.Context, merchantID, id uint64) (*attribution.Code, error) {
	c, err := s.repo.FindCode(ctx, id)
	if err != nil || c.MerchantID != merchantID {
		return nil, errf(404, "分销码不存在")
	}
	return c, nil
}

func (s *Service) checkQuota(ctx context.Context, merchantID uint64, add int) error {
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return err
	}
	n, err := s.repo.CountCodes(ctx, merchantID)
	if err != nil {
		return err
	}
	if n+int64(add) > int64(m.CodeQuota) {
		return errf(400, fmt.Sprintf("分销码额度不足：已用 %d，上限 %d，本次需要 %d", n, m.CodeQuota, add))
	}
	return nil
}

func (s *Service) buildCode(merchantID, userID uint64, in CodeInput) (*attribution.Code, error) {
	in.Code = strings.TrimSpace(in.Code)
	if !codeRe.MatchString(in.Code) {
		return nil, errf(400, "码值须为 2-64 位字母、数字、_ -")
	}
	if in.CodeType != attribution.CodeCampaign {
		in.CodeType = attribution.CodeDistributor
	}
	if in.Status != attribution.StatusDisabled {
		in.Status = attribution.StatusEnabled
	}
	starts, err := parseTime(in.StartsAt)
	if err != nil {
		return nil, errf(400, "生效时间格式无效")
	}
	exp, err := parseTime(in.ExpiresAt)
	if err != nil {
		return nil, errf(400, "有效期格式无效")
	}
	if starts != nil && exp != nil && !exp.After(*starts) {
		return nil, errf(400, "有效期须晚于生效时间")
	}
	paths := make([]string, 0, len(in.AllowedPaths))
	for _, p := range in.AllowedPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") {
			return nil, errf(400, "允许路径须以 / 开头："+p)
		}
		paths = append(paths, p)
	}
	raw, _ := json.Marshal(paths)
	var uid *uint64
	if userID > 0 {
		uid = &userID
	}
	return &attribution.Code{
		MerchantID: merchantID, SiteID: in.SiteID, Code: in.Code, CodeType: in.CodeType,
		Name: strPtr(in.Name), Channel: strPtr(in.Channel), Status: in.Status,
		StartsAt: starts, ExpiresAt: exp, AllowedPaths: raw, Remark: strPtr(in.Remark), CreatedBy: uid,
	}, nil
}

func (s *Service) siteNames(ctx context.Context, rows []attribution.Code) map[uint64]string {
	ids := make([]uint64, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].SiteID)
	}
	names, _ := s.sites.SiteNames(ctx, ids)
	return names
}

func paramRow(p *attribution.Param) map[string]any {
	return map[string]any{
		"paramId": p.ID, "siteId": p.SiteID, "name": p.Name, "paramType": p.ParamType, "platform": deref(p.Platform),
		"aliases": parseStrings(p.Aliases), "minLen": p.MinLen, "maxLen": p.MaxLen, "pattern": deref(p.Pattern),
		"sufficient": p.Sufficient, "status": p.Status, "sort": p.Sort, "remark": deref(p.Remark),
		"updatedAt": p.UpdatedAt.Format(timeLayout),
	}
}

func ruleRow(r *attribution.Rule) map[string]any {
	return map[string]any{
		"siteId": r.SiteID, "combineRule": r.CombineRule, "validateCodes": r.ValidateCodes,
		"validateClickFormat": r.ValidateClickFormat, "allowedSources": parseStrings(r.AllowedSources),
		"clickReuseLimit": r.ClickReuseLimit, "clickReuseWindowHours": r.ClickReuseWindowHours,
		"clickReuseAction": r.ClickReuseAction, "codeRevokeMode": r.CodeRevokeMode,
	}
}

func codeRow(c *attribution.Code, now time.Time) map[string]any {
	return map[string]any{
		"codeId": c.ID, "siteId": c.SiteID, "code": c.Code, "codeType": c.CodeType, "name": deref(c.Name),
		"channel": deref(c.Channel), "status": c.Status, "effectiveStatus": c.EffectiveStatus(now),
		"startsAt": fmtTime(c.StartsAt), "expiresAt": fmtTime(c.ExpiresAt), "allowedPaths": parseStrings(c.AllowedPaths),
		"remark": deref(c.Remark), "visitCount": c.VisitCount, "firstPassCount": c.FirstPassCount,
		"returnCount": c.ReturnCount, "conversionCount": c.ConversionCount, "lastVisitAt": fmtTime(c.LastVisitAt),
		"createdAt": c.CreatedAt.Format(timeLayout),
	}
}

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomCode(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(codeAlphabet)))
	for i := range b {
		v, _ := rand.Int(rand.Reader, max)
		b[i] = codeAlphabet[v.Int64()]
	}
	return string(b)
}

func parseStrings(raw []byte) []string {
	out := []string{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func parseTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{timeLayout, "2006-01-02 15:04", "2006-01-02", time.RFC3339} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("bad time")
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(timeLayout)
}
