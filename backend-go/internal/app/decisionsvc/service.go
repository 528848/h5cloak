// Package decisionsvc 判定测试工具：用最新配置构建快照并执行一次 dry-run 判定，不写日志、不写 Redis。
package decisionsvc

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/app/snapshotsvc"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
)

type Service struct {
	builder *snapshotsvc.Builder
	rdb     *goredis.Client
}

func New(b *snapshotsvc.Builder, rdb *goredis.Client) *Service {
	return &Service{builder: b, rdb: rdb}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

// TestInput 判定测试输入。
type TestInput struct {
	SiteID      uint64
	URL         string
	CookieToken string
	IP          string
	UserAgent   string
	Referer     string
}

// Test 对站点执行一次 dry-run 判定。
func (s *Service) Test(ctx context.Context, merchantID uint64, in TestInput) (map[string]any, error) {
	if in.SiteID == 0 {
		return nil, errf(400, "请选择站点")
	}
	snap, err := s.builder.Build(ctx, in.SiteID)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "站点不存在")
		}
		return nil, err
	}
	if merchantID > 0 && snap.MerchantID != merchantID {
		return nil, errf(404, "站点不存在")
	}
	raw := strings.TrimSpace(in.URL)
	if raw == "" {
		return nil, errf(400, "请输入访问地址")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, errf(400, "访问地址无效")
	}
	ip, _ := netip.ParseAddr(strings.TrimSpace(in.IP))
	if !ip.IsValid() {
		ip = netip.MustParseAddr("203.0.113.10")
	}
	ua := in.UserAgent
	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"
	}
	start := time.Now()
	eng := engine.New(nil) // 测试不触碰 Redis
	d := eng.Decide(ctx, &engine.RequestContext{
		RequestID: "test", Entry: "test", Site: snap, Method: "GET", URL: u,
		CookieToken: strings.TrimSpace(in.CookieToken), IP: ip, UserAgent: ua, Referer: in.Referer,
		Now: start, DryRun: true,
	})
	out := map[string]any{
		"decision":  d,
		"latencyMs": time.Since(start).Milliseconds(),
		"hostKnown": snap.HasDomain(u.Host),
		"snapshot": map[string]any{
			"revision": snap.Revision, "builtAt": snap.BuiltAt.Format("2006-01-02 15:04:05"),
			"params": len(snap.Params), "codes": len(snap.Codes), "pathRules": len(snap.PathRules),
			"keyVersion": snap.KeyVersion, "serving": snap.Serving(),
		},
	}
	if d.VisitorToken != "" {
		out["cookie"] = map[string]any{
			"name": snap.Cookie.Name, "value": d.VisitorToken, "maxAge": d.CookieMaxAge(start),
			"domain": snap.Cookie.Domain, "path": "/", "secure": true, "httpOnly": true, "sameSite": "Lax",
		}
	}
	return out, nil
}

// Status 返回站点快照在 Redis 中的状态，供后台展示一致性。
func (s *Service) Status(ctx context.Context, merchantID, siteID uint64) (map[string]any, error) {
	snap, err := s.builder.Build(ctx, siteID)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "站点不存在")
		}
		return nil, err
	}
	if merchantID > 0 && snap.MerchantID != merchantID {
		return nil, errf(404, "站点不存在")
	}
	out := map[string]any{
		"siteId": siteID, "redis": s.rdb != nil, "published": false,
		"local": map[string]any{"params": len(snap.Params), "codes": len(snap.Codes), "pathRules": len(snap.PathRules), "keyVersion": snap.KeyVersion, "serving": snap.Serving()},
	}
	if s.rdb == nil {
		return out, nil
	}
	pub, err := snapshotsvc.Load(ctx, s.rdb, siteID)
	if err != nil {
		if err == goredis.Nil {
			return out, nil
		}
		out["error"] = err.Error()
		return out, nil
	}
	out["published"] = true
	out["publishedAt"] = pub.BuiltAt.Format("2006-01-02 15:04:05")
	out["revision"] = pub.Revision
	out["remote"] = map[string]any{"params": len(pub.Params), "codes": len(pub.Codes), "pathRules": len(pub.PathRules), "keyVersion": pub.KeyVersion, "serving": pub.Serving()}
	out["inSync"] = sameShape(snap, pub)
	return out, nil
}

func sameShape(a, b *snapshot.Site) bool {
	return len(a.Params) == len(b.Params) && len(a.Codes) == len(b.Codes) && len(a.PathRules) == len(b.PathRules) &&
		a.KeyVersion == b.KeyVersion && a.Status == b.Status && a.MerchantActive == b.MerchantActive &&
		len(a.Domains) == len(b.Domains) && len(a.Origins) == len(b.Origins) && a.Review.Source == b.Review.Source
}
