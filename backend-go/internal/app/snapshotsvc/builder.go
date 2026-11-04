// Package snapshotsvc 构建站点快照并发布到 Redis，供网关热加载。
package snapshotsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/domain/attribution"
	"cloak/internal/domain/identity"
	domainsite "cloak/internal/domain/site"
	"cloak/internal/engine/snapshot"
)

// Builder 从数据库组装快照。
type Builder struct {
	sites     *mysql.SiteRepo
	attrs     *mysql.AttributionRepo
	merchants identity.MerchantRepository
}

func NewBuilder(sites *mysql.SiteRepo, attrs *mysql.AttributionRepo, merchants identity.MerchantRepository) *Builder {
	return &Builder{sites: sites, attrs: attrs, merchants: merchants}
}

// Build 组装一个站点的快照。站点不存在返回 mysql.ErrNotFound。
func (b *Builder) Build(ctx context.Context, siteID uint64) (*snapshot.Site, error) {
	site, err := b.sites.FindSite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	m, err := b.merchants.FindByID(ctx, site.MerchantID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := &snapshot.Site{
		Format: snapshot.Version, Revision: now.UnixNano(), BuiltAt: now,
		SiteID: site.ID, MerchantID: site.MerchantID, Name: site.Name, Status: site.Status, SiteType: site.SiteType,
		MerchantActive: m.Active(now),
		Quota:          snapshot.Quota{Decisions: m.DecisionQuota, Policy: m.QuotaPolicy, Retention: int(m.LogRetentionDays)},
		Cookie:         snapshot.Cookie{Name: site.CookieName, Domain: deref(site.CookieDomain)},
		Attribution: snapshot.Attribution{
			TTLDays: site.AttributionTTLDays, RenewMode: site.RenewMode, NoAttributionAction: site.NoAttributionAction,
			FailMode: site.FailMode, MinConfidence: site.MinConfidence,
		},
		Handoff: snapshot.Handoff{
			Banner: site.HandoffBanner, BannerText: site.HandoffBannerText, LinkRewrite: site.HandoffLinkRewrite,
			TokenTTLHours: site.HandoffTokenTTLHours, TokenMaxUses: site.HandoffTokenMaxUses,
			MatchEnabled: site.HandoffMatchEnabled, MatchMinutes: site.HandoffMatchMinutes, MatchSystemOnly: site.HandoffMatchSysOnly,
		},
		KeyVersion:     site.TokenKeyVersion,
		PublishableKey: site.PublishableKey, SecretPrefix: site.SecretPrefix, SecretHash: site.SecretHash, SecretEnc: site.SecretEnc,
	}
	_ = json.Unmarshal(site.TokenKeys, &out.Keys)
	_ = json.Unmarshal(site.SecretIPAllowlist, &out.SecretAllow)
	if out.SecretAllow == nil {
		out.SecretAllow = []string{}
	}

	domains, _, err := b.sites.ListDomains(ctx, 0, siteID, "", "", 1, 200)
	if err != nil {
		return nil, err
	}
	for _, d := range domains {
		out.Domains = append(out.Domains, snapshot.Domain{Domain: strings.ToLower(d.Domain), DomainType: d.DomainType, Status: d.Status})
	}

	origins, err := b.sites.ListOrigins(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, o := range origins {
		out.Origins = append(out.Origins, snapshot.Origin{
			Kind: o.Kind, URL: o.URL, HostHeader: deref(o.HostHeader), TimeoutMs: o.TimeoutMs,
			Healthy: o.HealthStatus != domainsite.HealthUnhealthy,
		})
	}

	if review, err := b.sites.FindReview(ctx, siteID); err == nil {
		out.Review.Source = review.Source
		switch review.Source {
		case domainsite.ReviewTemplate:
			if review.TemplateID != nil {
				if t, err := b.sites.FindTemplate(ctx, *review.TemplateID); err == nil {
					out.Review.TemplateHTML = t.HTML
				}
			}
			vars := map[string]string{}
			_ = json.Unmarshal(review.TemplateVars, &vars)
			out.Review.TemplateVars = vars
		case domainsite.ReviewStatic:
			if review.StaticPackageID != nil {
				if p, err := b.sites.FindPackage(ctx, *review.StaticPackageID); err == nil {
					out.Review.StaticDir = p.ObjectKey
					out.Review.StaticEntry = p.Entry
				}
			}
		}
	} else if !errors.Is(err, mysql.ErrNotFound) {
		return nil, err
	}

	rules, err := b.sites.ListPathRules(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		out.PathRules = append(out.PathRules, snapshot.PathRule{Pattern: r.Pattern, MatchType: r.MatchType, Action: r.Action, Sort: r.Sort})
	}

	params, err := b.attrs.ListParams(ctx, siteID)
	if err != nil {
		return nil, err
	}
	out.Params = mergeParams(params)

	rule, err := b.attrs.FindRule(ctx, siteID)
	if err != nil {
		return nil, err
	}
	var sources []string
	_ = json.Unmarshal(rule.AllowedSources, &sources)
	out.Rule = snapshot.Rule{
		CombineRule: rule.CombineRule, ValidateCodes: rule.ValidateCodes, ValidateClickFormat: rule.ValidateClickFormat,
		AllowedSources: sources, ClickReuseLimit: int(rule.ClickReuseLimit), ClickReuseWindowH: int(rule.ClickReuseWindowHours),
		ClickReuseAction: rule.ClickReuseAction, CodeRevokeMode: rule.CodeRevokeMode,
	}

	codes, err := b.attrs.AllCodes(ctx, mysql.CodeFilter{SiteID: siteID})
	if err != nil {
		return nil, err
	}
	for _, c := range codes {
		var paths []string
		_ = json.Unmarshal(c.AllowedPaths, &paths)
		out.Codes = append(out.Codes, snapshot.Code{
			ID: c.ID, Code: c.Code, Type: c.CodeType, Name: deref(c.Name), Channel: deref(c.Channel),
			Enabled: c.Status == attribution.StatusEnabled, StartsAt: c.StartsAt, ExpiresAt: c.ExpiresAt, AllowedPaths: paths,
		})
	}

	if err := out.Compile(); err != nil {
		return nil, fmt.Errorf("compile snapshot: %w", err)
	}
	return out, nil
}

// mergeParams 站点行覆盖同名内置行。
func mergeParams(rows []attribution.Param) []snapshot.Param {
	byName := map[string]*attribution.Param{}
	order := []string{}
	for i := range rows {
		p := &rows[i]
		key := strings.ToLower(p.Name)
		if existing, ok := byName[key]; ok {
			if p.SiteID > existing.SiteID {
				byName[key] = p
			}
			continue
		}
		byName[key] = p
		order = append(order, key)
	}
	out := make([]snapshot.Param, 0, len(order))
	for _, key := range order {
		p := byName[key]
		var aliases []string
		_ = json.Unmarshal(p.Aliases, &aliases)
		out = append(out, snapshot.Param{
			Name: strings.ToLower(p.Name), Type: p.ParamType, Platform: deref(p.Platform), Aliases: aliases,
			MinLen: int(p.MinLen), MaxLen: int(p.MaxLen), Pattern: deref(p.Pattern),
			Sufficient: p.Sufficient, Enabled: p.Status == attribution.StatusEnabled,
		})
	}
	return out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
