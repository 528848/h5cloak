// Package sitesvc 站点、域名、源站与审核内容。
package sitesvc

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/auth"
	"cloak/internal/domain/identity"
	domainsite "cloak/internal/domain/site"
	"cloak/internal/pkg/probe"
)

type Service struct {
	repo      *mysql.SiteRepo
	merchants identity.MerchantRepository
	entryHost string
	uploadDir string
	publicURL string
}

func New(repo *mysql.SiteRepo, merchants identity.MerchantRepository, entryHost, uploadDir, publicURL string) *Service {
	if entryHost == "" {
		entryHost = "entry.cloak.local"
	}
	return &Service{repo: repo, merchants: merchants, entryHost: strings.ToLower(entryHost), uploadDir: uploadDir, publicURL: strings.TrimRight(publicURL, "/")}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

type SiteInput struct {
	Name                 string
	SiteType             string
	Remark               string
	CookieName           string
	CookieDomain         string
	AttributionTTLDays   int
	RenewMode            string
	NoAttributionAction  string
	FailMode             string
	MinConfidence        string
	HandoffBanner        bool
	HandoffBannerText    string
	HandoffLinkRewrite   bool
	HandoffTokenTTLHours int
	HandoffTokenMaxUses  int
	HandoffMatchEnabled  bool
	HandoffMatchMinutes  int
	SecretIPAllowlist    []string
}

func (s *Service) Create(ctx context.Context, merchantID uint64, in SiteInput) (map[string]any, string, error) {
	in = normalize(in)
	if err := validateSite(in); err != nil {
		return nil, "", err
	}
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return nil, "", err
	}
	n, err := s.repo.CountSites(ctx, merchantID)
	if err != nil {
		return nil, "", err
	}
	if n >= int64(m.SiteQuota) {
		return nil, "", errf(400, "站点数量已达上限，请联系平台调整额度")
	}
	pk, sk, prefix, hash, keys, err := newSecrets()
	if err != nil {
		return nil, "", err
	}
	allow, _ := json.Marshal(in.SecretIPAllowlist)
	site := &domainsite.Site{
		MerchantID: merchantID, Name: in.Name, SiteType: in.SiteType, Status: domainsite.StatusEnabled,
		Remark: strPtr(in.Remark), PublishableKey: pk, SecretPrefix: prefix, SecretHash: hash,
		SecretIPAllowlist: allow, CookieName: in.CookieName, CookieDomain: strPtr(in.CookieDomain),
		AttributionTTLDays: in.AttributionTTLDays, RenewMode: in.RenewMode,
		NoAttributionAction: in.NoAttributionAction, FailMode: in.FailMode, MinConfidence: in.MinConfidence,
		HandoffBanner: in.HandoffBanner, HandoffBannerText: in.HandoffBannerText, HandoffLinkRewrite: in.HandoffLinkRewrite,
		HandoffTokenTTLHours: in.HandoffTokenTTLHours, HandoffTokenMaxUses: in.HandoffTokenMaxUses,
		HandoffMatchEnabled: in.HandoffMatchEnabled, HandoffMatchMinutes: in.HandoffMatchMinutes,
		TokenKeyVersion: 1, TokenKeys: keys,
	}
	review := &domainsite.ReviewContent{Source: domainsite.ReviewTemplate, TemplateVars: []byte(`{"siteName":"","slogan":""}`)}
	if err := s.repo.CreateSite(ctx, site, domainsite.DefaultPathRules(), review); err != nil {
		return nil, "", err
	}
	row, err := s.detail(ctx, site)
	if err != nil {
		return nil, "", err
	}
	row["serverSecret"] = sk
	return row, sk, nil
}

func (s *Service) Update(ctx context.Context, merchantID, id uint64, in SiteInput) error {
	site, err := s.owned(ctx, merchantID, id)
	if err != nil {
		return err
	}
	in = normalize(in)
	if err := validateSite(in); err != nil {
		return err
	}
	allow, _ := json.Marshal(in.SecretIPAllowlist)
	site.Name, site.SiteType, site.Remark = in.Name, in.SiteType, strPtr(in.Remark)
	site.CookieName, site.CookieDomain = in.CookieName, strPtr(in.CookieDomain)
	site.AttributionTTLDays, site.RenewMode = in.AttributionTTLDays, in.RenewMode
	site.NoAttributionAction, site.FailMode, site.MinConfidence = in.NoAttributionAction, in.FailMode, in.MinConfidence
	site.HandoffBanner, site.HandoffBannerText = in.HandoffBanner, in.HandoffBannerText
	site.HandoffLinkRewrite = in.HandoffLinkRewrite
	site.HandoffTokenTTLHours, site.HandoffTokenMaxUses = in.HandoffTokenTTLHours, in.HandoffTokenMaxUses
	site.HandoffMatchEnabled, site.HandoffMatchMinutes = in.HandoffMatchEnabled, in.HandoffMatchMinutes
	site.SecretIPAllowlist = allow
	return s.repo.UpdateSite(ctx, site,
		"Name", "SiteType", "Remark", "CookieName", "CookieDomain", "AttributionTTLDays", "RenewMode",
		"NoAttributionAction", "FailMode", "MinConfidence", "HandoffBanner", "HandoffBannerText", "HandoffLinkRewrite",
		"HandoffTokenTTLHours", "HandoffTokenMaxUses", "HandoffMatchEnabled", "HandoffMatchMinutes", "SecretIPAllowlist")
}

func (s *Service) ChangeStatus(ctx context.Context, merchantID, id uint64, status string, platform bool) error {
	if status != domainsite.StatusEnabled && status != domainsite.StatusDisabled {
		return errf(400, "状态无效")
	}
	var site *domainsite.Site
	var err error
	if platform {
		site, err = s.repo.FindSite(ctx, id)
		if errors.Is(err, mysql.ErrNotFound) {
			return errf(404, "站点不存在")
		}
	} else {
		site, err = s.owned(ctx, merchantID, id)
	}
	if err != nil {
		return err
	}
	site.Status = status
	return s.repo.UpdateSite(ctx, site, "Status")
}

func (s *Service) Delete(ctx context.Context, merchantID, id uint64) error {
	if _, err := s.owned(ctx, merchantID, id); err != nil {
		return err
	}
	return s.repo.DeleteSite(ctx, id)
}

func (s *Service) Get(ctx context.Context, merchantID, id uint64) (map[string]any, error) {
	site, err := s.owned(ctx, merchantID, id)
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, site)
}

func (s *Service) List(ctx context.Context, merchantID uint64, keyword, status string, page, size int) ([]map[string]any, int64, error) {
	rows, total, err := s.repo.ListSites(ctx, merchantID, keyword, status, page, size)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(rows))
	mids := make([]uint64, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
		mids = append(mids, rows[i].MerchantID)
	}
	counters, _ := s.repo.SiteCounters(ctx, ids)
	names := map[uint64]string{}
	if merchantID == 0 {
		names, _ = s.repo.MerchantNames(ctx, mids)
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		row := brief(&rows[i])
		c := counters[rows[i].ID]
		if c == nil {
			c = &mysql.SiteCounters{}
		}
		row["domainCount"], row["boundDomainCount"] = c.Domains, c.BoundDomains
		row["originCount"], row["unhealthyOriginCount"] = c.Origins, c.Unhealthy
		if merchantID == 0 {
			row["merchantName"] = names[rows[i].MerchantID]
		}
		out = append(out, row)
	}
	return out, total, nil
}

// Stats 站点状态统计（商户端传商户 ID，总后台传 0）。
func (s *Service) Stats(ctx context.Context, merchantID uint64) (map[string]any, error) {
	m, err := s.repo.SiteStats(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	d, err := s.repo.DomainStats(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	var siteTotal, domainTotal int64
	for _, n := range m {
		siteTotal += n
	}
	for _, n := range d {
		domainTotal += n
	}
	return map[string]any{
		"total": siteTotal, "enabled": m[domainsite.StatusEnabled], "disabled": m[domainsite.StatusDisabled],
		"domainTotal": domainTotal, "domainBound": d[domainsite.DomainBound], "domainPending": d[domainsite.DomainPending],
		"domainAbnormal": d[domainsite.DomainAbnormal], "domainDisabled": d[domainsite.DomainDisabled],
	}, nil
}

// Options 商户站点下拉。
func (s *Service) Options(ctx context.Context, merchantID uint64) ([]map[string]any, error) {
	rows, err := s.repo.SiteOptions(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{"siteId": r.ID, "name": r.Name, "status": r.Status, "siteType": r.SiteType})
	}
	return out, nil
}

// ResetSecret 重置服务端密钥，只在返回值中出现一次。
func (s *Service) ResetSecret(ctx context.Context, merchantID, id uint64, allow []string) (string, error) {
	site, err := s.owned(ctx, merchantID, id)
	if err != nil {
		return "", err
	}
	_, sk, prefix, hash, _, err := newSecrets()
	if err != nil {
		return "", err
	}
	raw, _ := json.Marshal(allow)
	site.SecretPrefix, site.SecretHash, site.SecretIPAllowlist = prefix, hash, raw
	if err := s.repo.UpdateSite(ctx, site, "SecretPrefix", "SecretHash", "SecretIPAllowlist"); err != nil {
		return "", err
	}
	return sk, nil
}

type OriginInput struct {
	Kind       string
	URL        string
	HostHeader string
	TimeoutMs  int
}

func (s *Service) SaveOrigin(ctx context.Context, merchantID, siteID uint64, in OriginInput) error {
	if _, err := s.owned(ctx, merchantID, siteID); err != nil {
		return err
	}
	if in.Kind != domainsite.OriginReal && in.Kind != domainsite.OriginReview {
		return errf(400, "源站类型无效")
	}
	in.URL = strings.TrimSpace(in.URL)
	if !strings.HasPrefix(in.URL, "http://") && !strings.HasPrefix(in.URL, "https://") {
		return errf(400, "源站地址须以 http:// 或 https:// 开头")
	}
	if in.TimeoutMs <= 0 {
		in.TimeoutMs = 15000
	}
	if in.TimeoutMs > 60000 {
		in.TimeoutMs = 60000
	}
	return s.repo.UpsertOrigin(ctx, &domainsite.Origin{
		SiteID: siteID, Kind: in.Kind, URL: in.URL, HostHeader: strPtr(in.HostHeader), TimeoutMs: in.TimeoutMs,
		HealthStatus: domainsite.HealthUnknown,
	})
}

func (s *Service) ProbeOrigin(ctx context.Context, merchantID, siteID uint64, kind string) (map[string]any, error) {
	if _, err := s.owned(ctx, merchantID, siteID); err != nil {
		return nil, err
	}
	return s.probeKind(ctx, siteID, kind)
}

func (s *Service) probeKind(ctx context.Context, siteID uint64, kind string) (map[string]any, error) {
	origins, err := s.repo.ListOrigins(ctx, siteID)
	if err != nil {
		return nil, err
	}
	var target *domainsite.Origin
	for i := range origins {
		if origins[i].Kind == kind {
			target = &origins[i]
			break
		}
	}
	if target == nil {
		return nil, errf(404, "尚未配置该源站")
	}
	ok, msg := probe.Origin(ctx, target.URL, time.Duration(target.TimeoutMs)*time.Millisecond)
	status := domainsite.HealthUnhealthy
	if ok {
		status = domainsite.HealthHealthy
	}
	now := time.Now()
	if err := s.repo.UpdateOriginHealth(ctx, target.ID, status, msg, now); err != nil {
		return nil, err
	}
	return map[string]any{"kind": kind, "healthStatus": status, "healthMessage": msg, "lastHealthAt": now.Format("2006-01-02 15:04:05")}, nil
}

// ProbeDue 供 worker 周期调用，每次最多检查 limitN 个源站。
func (s *Service) ProbeDue(ctx context.Context, limitN int) {
	rows, err := s.repo.ListOriginsForProbe(ctx, limitN)
	if err != nil {
		return
	}
	for i := range rows {
		ok, msg := probe.Origin(ctx, rows[i].URL, 8*time.Second)
		status := domainsite.HealthUnhealthy
		if ok {
			status = domainsite.HealthHealthy
		}
		_ = s.repo.UpdateOriginHealth(ctx, rows[i].ID, status, msg, time.Now())
	}
}

type ReviewInput struct {
	Source       string
	TemplateID   uint64
	TemplateVars map[string]string
	PackageID    uint64
}

func (s *Service) SaveReview(ctx context.Context, merchantID, siteID uint64, in ReviewInput) error {
	if _, err := s.owned(ctx, merchantID, siteID); err != nil {
		return err
	}
	row := &domainsite.ReviewContent{SiteID: siteID, Source: in.Source}
	switch in.Source {
	case domainsite.ReviewOrigin:
	case domainsite.ReviewTemplate:
		if in.TemplateID == 0 {
			return errf(400, "请选择模板")
		}
		if _, err := s.repo.FindTemplate(ctx, in.TemplateID); err != nil {
			return errf(400, "模板不存在")
		}
		id := in.TemplateID
		row.TemplateID = &id
		if in.TemplateVars == nil {
			in.TemplateVars = map[string]string{}
		}
		row.TemplateVars, _ = json.Marshal(in.TemplateVars)
	case domainsite.ReviewStatic:
		if in.PackageID == 0 {
			return errf(400, "请先上传静态包")
		}
		p, err := s.repo.FindPackage(ctx, in.PackageID)
		if err != nil || p.SiteID != siteID {
			return errf(400, "静态包不存在")
		}
		row.StaticPackageID = &p.ID
	default:
		return errf(400, "审核内容来源无效")
	}
	return s.repo.SaveReview(ctx, row)
}

func (s *Service) Templates(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.repo.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, t := range rows {
		out = append(out, map[string]any{"templateId": t.ID, "code": t.Code, "name": t.Name, "description": deref(t.Description)})
	}
	return out, nil
}

func (s *Service) PreviewTemplate(ctx context.Context, id uint64, vars map[string]string) (string, error) {
	t, err := s.repo.FindTemplate(ctx, id)
	if err != nil {
		return "", errf(404, "模板不存在")
	}
	tpl, err := template.New("review").Parse(t.HTML)
	if err != nil {
		return "", err
	}
	if vars == nil {
		vars = map[string]string{}
	}
	var b strings.Builder
	if err := tpl.Execute(&b, vars); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (s *Service) UploadPackage(ctx context.Context, merchantID, siteID uint64, r io.ReaderAt, size int64) (map[string]any, error) {
	if _, err := s.owned(ctx, merchantID, siteID); err != nil {
		return nil, err
	}
	if size <= 0 || size > 20<<20 {
		return nil, errf(400, "静态包须小于 20MB")
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, errf(400, "请上传 zip 压缩包")
	}
	ver, err := s.repo.NextPackageVersion(ctx, siteID)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(s.uploadDir, "sites", fmt.Sprint(siteID), "static", fmt.Sprint(ver))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	entry := ""
	for _, f := range zr.File {
		name := filepath.Clean(f.Name)
		if strings.Contains(name, "..") || filepath.IsAbs(name) {
			return nil, errf(400, "压缩包包含非法路径")
		}
		target := filepath.Join(dir, name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return nil, err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, 20<<20))
		out.Close()
		rc.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		base := filepath.ToSlash(name)
		if base == "index.html" || strings.HasSuffix(base, "/index.html") {
			if entry == "" || len(base) < len(entry) {
				entry = base
			}
		}
	}
	if entry == "" {
		entry = "index.html"
	}
	pkg := &domainsite.StaticPackage{SiteID: siteID, Version: ver, ObjectKey: dir, Entry: entry, Status: "ready"}
	if err := s.repo.CreatePackage(ctx, pkg); err != nil {
		return nil, err
	}
	_ = s.repo.SaveReview(ctx, &domainsite.ReviewContent{SiteID: siteID, Source: domainsite.ReviewStatic, StaticPackageID: &pkg.ID})
	return map[string]any{
		"packageId": pkg.ID, "version": ver, "entry": entry,
		"previewUrl": fmt.Sprintf("%s/uploads/sites/%d/static/%d/%s", s.publicURL, siteID, ver, entry),
	}, nil
}

func (s *Service) ReplaceRules(ctx context.Context, merchantID, siteID uint64, rules []domainsite.PathRule) error {
	if _, err := s.owned(ctx, merchantID, siteID); err != nil {
		return err
	}
	for _, rule := range rules {
		switch rule.MatchType {
		case domainsite.MatchPrefix, domainsite.MatchExact, domainsite.MatchSuffix:
		default:
			return errf(400, "匹配方式无效")
		}
		switch rule.Action {
		case domainsite.PathBypass, domainsite.PathAlwaysReal, domainsite.PathAlwaysReview, domainsite.PathBlock:
		default:
			return errf(400, "路径动作无效")
		}
		if strings.TrimSpace(rule.Pattern) == "" {
			return errf(400, "路径不能为空")
		}
	}
	return s.repo.ReplacePathRules(ctx, siteID, rules)
}

type DomainInput struct {
	Domain     string
	DomainType string
	VerifyType string
	Remark     string
}

func (s *Service) AddDomain(ctx context.Context, merchantID, siteID uint64, in DomainInput) (map[string]any, error) {
	if _, err := s.owned(ctx, merchantID, siteID); err != nil {
		return nil, err
	}
	host := normalizeDomain(in.Domain)
	if host == "" || strings.Contains(host, "/") || net.ParseIP(host) != nil {
		return nil, errf(400, "请输入域名，不要带协议和路径")
	}
	if exists, err := s.repo.DomainExists(ctx, host); err != nil {
		return nil, err
	} else if exists {
		return nil, errf(400, "该域名已被绑定")
	}
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.CountDomains(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if n >= int64(m.DomainQuota) {
		return nil, errf(400, "域名数量已达上限，请联系平台调整额度")
	}
	kind := in.DomainType
	if kind != domainsite.DomainOwn {
		kind = domainsite.DomainHosted
	}
	verifyType := strings.ToUpper(in.VerifyType)
	if verifyType != "TXT" {
		verifyType = "CNAME"
	}
	value := s.entryHost
	if verifyType == "TXT" {
		buf := make([]byte, 8)
		_, _ = rand.Read(buf)
		value = "cloak-verify=" + hex.EncodeToString(buf)
	}
	d := &domainsite.Domain{
		MerchantID: merchantID, SiteID: siteID, Domain: host, DomainType: kind,
		VerifyType: verifyType, VerifyValue: value, Status: domainsite.DomainPending, CertStatus: "none",
		HealthStatus: domainsite.HealthUnknown, Remark: strPtr(in.Remark),
	}
	if err := s.repo.CreateDomain(ctx, d); err != nil {
		return nil, err
	}
	return domainRow(d, ""), nil
}

func (s *Service) VerifyDomain(ctx context.Context, merchantID, id uint64, platform bool) (map[string]any, error) {
	d, err := s.repo.FindDomain(ctx, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "域名不存在")
		}
		return nil, err
	}
	if !platform && d.MerchantID != merchantID {
		return nil, errf(404, "域名不存在")
	}
	if d.DomainType == domainsite.DomainOwn {
		now := time.Now()
		d.Status = domainsite.DomainBound
		d.VerifyMessage = strPtr("自有域名只校验归属，不要求解析到网关")
		d.LastCheckedAt = &now
		if err := s.repo.UpdateDomain(ctx, d, "Status", "VerifyMessage", "LastCheckedAt"); err != nil {
			return nil, err
		}
		return domainRow(d, ""), nil
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	now := time.Now()
	d.LastCheckedAt = &now
	var ok bool
	var msg string
	if d.VerifyType == "TXT" {
		name := "_cloak." + d.Domain
		recs, err := probe.TXT(ctx, name)
		if err != nil {
			msg = "TXT 查询失败：" + err.Error()
		} else if probe.ContainsTXT(recs, d.VerifyValue) {
			ok = true
			msg = "TXT 校验通过"
		} else {
			msg = "未找到记录 " + d.VerifyValue
		}
	} else {
		cname, err := probe.CNAME(ctx, d.Domain)
		if err != nil {
			msg = "CNAME 查询失败：" + err.Error()
		} else if probe.MatchCNAME(cname, d.VerifyValue) {
			ok = true
			msg = "CNAME 指向 " + cname
		} else if cname == "" || cname == d.Domain {
			if ips, ierr := net.DefaultResolver.LookupHost(ctx, d.Domain); ierr == nil && len(ips) > 0 {
				msg = "该域名当前解析到 " + strings.Join(ips, ", ") + "，未配置指向 " + d.VerifyValue + " 的 CNAME"
			} else {
				msg = "未配置 CNAME，且域名无法解析"
			}
		} else {
			msg = "当前 CNAME 指向 " + cname + "，期望 " + d.VerifyValue
		}
	}
	if ok {
		if d.Status != domainsite.DomainDisabled {
			d.Status = domainsite.DomainBound
		}
		d.HealthStatus = domainsite.HealthHealthy
	} else if d.Status == domainsite.DomainBound {
		d.Status = domainsite.DomainAbnormal
		d.HealthStatus = domainsite.HealthUnhealthy
	}
	d.VerifyMessage = strPtr(msg)
	if err := s.repo.UpdateDomain(ctx, d, "Status", "VerifyMessage", "LastCheckedAt", "HealthStatus"); err != nil {
		return nil, err
	}
	return domainRow(d, msg), nil
}

func (s *Service) DeleteDomain(ctx context.Context, merchantID, id uint64) error {
	d, err := s.repo.FindDomain(ctx, id)
	if err != nil || d.MerchantID != merchantID {
		return errf(404, "域名不存在")
	}
	return s.repo.DeleteDomain(ctx, id)
}

func (s *Service) ListDomains(ctx context.Context, merchantID, siteID uint64, keyword, status string, page, size int) ([]map[string]any, int64, error) {
	rows, total, err := s.repo.ListDomains(ctx, merchantID, siteID, keyword, status, page, size)
	if err != nil {
		return nil, 0, err
	}
	sids := make([]uint64, 0, len(rows))
	mids := make([]uint64, 0, len(rows))
	for i := range rows {
		sids = append(sids, rows[i].SiteID)
		mids = append(mids, rows[i].MerchantID)
	}
	siteNames, _ := s.repo.SiteNames(ctx, sids)
	merchantNames := map[uint64]string{}
	if merchantID == 0 {
		merchantNames, _ = s.repo.MerchantNames(ctx, mids)
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		row := domainRow(&rows[i], deref(rows[i].VerifyMessage))
		row["siteName"] = siteNames[rows[i].SiteID]
		if merchantID == 0 {
			row["merchantName"] = merchantNames[rows[i].MerchantID]
		}
		out = append(out, row)
	}
	return out, total, nil
}

func (s *Service) EntryGuide() map[string]any {
	return map[string]any{"entryHost": s.entryHost, "txtPrefix": "_cloak"}
}

func (s *Service) owned(ctx context.Context, merchantID, id uint64) (*domainsite.Site, error) {
	site, err := s.repo.FindSite(ctx, id)
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

func (s *Service) detail(ctx context.Context, site *domainsite.Site) (map[string]any, error) {
	row := brief(site)
	origins, _ := s.repo.ListOrigins(ctx, site.ID)
	os := []map[string]any{}
	for _, o := range origins {
		os = append(os, map[string]any{
			"kind": o.Kind, "url": o.URL, "hostHeader": deref(o.HostHeader), "timeoutMs": o.TimeoutMs,
			"healthStatus": o.HealthStatus, "healthMessage": deref(o.HealthMessage), "lastHealthAt": fmtTime(o.LastHealthAt),
		})
	}
	row["origins"] = os
	if review, err := s.repo.FindReview(ctx, site.ID); err == nil {
		vars := map[string]string{}
		_ = json.Unmarshal(review.TemplateVars, &vars)
		row["review"] = map[string]any{"source": review.Source, "templateId": review.TemplateID, "templateVars": vars, "staticPackageId": review.StaticPackageID}
	}
	rules, _ := s.repo.ListPathRules(ctx, site.ID)
	rs := []map[string]any{}
	for _, rule := range rules {
		rs = append(rs, map[string]any{"pattern": rule.Pattern, "matchType": rule.MatchType, "action": rule.Action, "sort": rule.Sort, "remark": deref(rule.Remark)})
	}
	row["pathRules"] = rs
	return row, nil
}

func brief(site *domainsite.Site) map[string]any {
	var allow []string
	_ = json.Unmarshal(site.SecretIPAllowlist, &allow)
	return map[string]any{
		"siteId": site.ID, "merchantId": site.MerchantID, "name": site.Name, "siteType": site.SiteType, "status": site.Status,
		"remark": deref(site.Remark), "publishableKey": site.PublishableKey, "secretPrefix": site.SecretPrefix,
		"secretIpAllowlist": allow, "cookieName": site.CookieName, "cookieDomain": deref(site.CookieDomain),
		"attributionTtlDays": site.AttributionTTLDays, "renewMode": site.RenewMode,
		"noAttributionAction": site.NoAttributionAction, "failMode": site.FailMode, "minConfidence": site.MinConfidence,
		"handoffBanner": site.HandoffBanner, "handoffBannerText": site.HandoffBannerText, "handoffLinkRewrite": site.HandoffLinkRewrite,
		"handoffTokenTtlHours": site.HandoffTokenTTLHours, "handoffTokenMaxUses": site.HandoffTokenMaxUses,
		"handoffMatchEnabled": site.HandoffMatchEnabled, "handoffMatchMinutes": site.HandoffMatchMinutes,
		"createdAt": site.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func domainRow(d *domainsite.Domain, msg string) map[string]any {
	if msg == "" {
		msg = deref(d.VerifyMessage)
	}
	record := d.Domain
	if d.VerifyType == "TXT" {
		record = "_cloak." + d.Domain
	}
	return map[string]any{
		"domainId": d.ID, "siteId": d.SiteID, "merchantId": d.MerchantID, "domain": d.Domain, "domainType": d.DomainType,
		"verifyType": d.VerifyType, "verifyValue": d.VerifyValue, "verifyRecord": record, "status": d.Status,
		"verifyMessage": msg, "lastCheckedAt": fmtTime(d.LastCheckedAt), "certStatus": d.CertStatus,
		"healthStatus": d.HealthStatus, "remark": deref(d.Remark), "createdAt": d.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func newSecrets() (pk, sk, prefix, hash string, keys []byte, err error) {
	pk = "pk_" + randHex(16)
	sk = "sk_" + randHex(24)
	prefix = sk[:12]
	hash, err = auth.HashPassword(sk)
	if err != nil {
		return
	}
	key := randHex(32)
	keys, _ = json.Marshal([]map[string]any{{"version": 1, "key": key, "status": "active"}})
	return
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func normalize(in SiteInput) SiteInput {
	in.Name = strings.TrimSpace(in.Name)
	if in.SiteType != domainsite.TypeHosted && in.SiteType != domainsite.TypeServer {
		in.SiteType = domainsite.TypeHybrid
	}
	if in.CookieName == "" {
		in.CookieName = "_ck_vid"
	}
	if in.AttributionTTLDays <= 0 {
		in.AttributionTTLDays = 30
	}
	if in.RenewMode != domainsite.RenewFixed {
		in.RenewMode = domainsite.RenewSliding
	}
	switch in.NoAttributionAction {
	case domainsite.ActionBlock, domainsite.ActionNotFound:
	default:
		in.NoAttributionAction = domainsite.ActionReview
	}
	switch in.FailMode {
	case domainsite.ActionReal, domainsite.ActionMaintain:
	default:
		in.FailMode = domainsite.ActionReview
	}
	switch in.MinConfidence {
	case "low", "high":
	default:
		in.MinConfidence = "medium"
	}
	if in.HandoffBannerText == "" {
		in.HandoffBannerText = "在浏览器中打开，继续访问"
	}
	if in.HandoffTokenTTLHours <= 0 {
		in.HandoffTokenTTLHours = 24
	}
	if in.HandoffTokenMaxUses <= 0 {
		in.HandoffTokenMaxUses = 3
	}
	if in.HandoffMatchMinutes < 5 {
		in.HandoffMatchMinutes = 15
	}
	if in.HandoffMatchMinutes > 60 {
		in.HandoffMatchMinutes = 60
	}
	return in
}

func validateSite(in SiteInput) error {
	if in.Name == "" || len([]rune(in.Name)) > 64 {
		return errf(400, "站点名称须为 1-64 字")
	}
	if in.AttributionTTLDays > 365 {
		return errf(400, "归因窗口最长 365 天")
	}
	for _, ip := range in.SecretIPAllowlist {
		if _, _, err := net.ParseCIDR(ip); err != nil && net.ParseIP(ip) == nil {
			return errf(400, "IP 白名单格式无效："+ip)
		}
	}
	return nil
}

func normalizeDomain(d string) string {
	d = strings.TrimSpace(strings.ToLower(d))
	d = strings.TrimPrefix(d, "https://")
	d = strings.TrimPrefix(d, "http://")
	d = strings.TrimSuffix(d, "/")
	return d
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
	return t.Format("2006-01-02 15:04:05")
}
