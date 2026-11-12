package gateway

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"cloak/internal/adapter/mysql"
	"cloak/internal/cloakdec"
	"cloak/internal/domain/cloak"
	"cloak/internal/domain/identity"
)

// CloakCatalog 每隔几秒从数据库刷新启用中的斗篷。
type CloakCatalog struct {
	repo *mysql.CloakRepo
	log  *zap.Logger
	mu   sync.RWMutex
	rows map[string]mysql.GatewayCloak
}

func NewCloakCatalog(db *gorm.DB, log *zap.Logger) *CloakCatalog {
	return &CloakCatalog{repo: mysql.NewCloakRepo(db), log: log, rows: map[string]mysql.GatewayCloak{}}
}

func (c *CloakCatalog) Run(ctx context.Context) {
	c.reload(ctx)
	tk := time.NewTicker(5 * time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			c.reload(ctx)
		}
	}
}

func (c *CloakCatalog) reload(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := c.repo.ListForGateway(cctx)
	if err != nil {
		c.log.Warn("cloak catalog reload", zap.Error(err))
		return
	}
	next := make(map[string]mysql.GatewayCloak, len(rows))
	for _, row := range rows {
		next[strings.ToLower(row.Code)] = row
	}
	c.mu.Lock()
	c.rows = next
	c.mu.Unlock()
}

func (c *CloakCatalog) Get(code string) (mysql.GatewayCloak, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	row, ok := c.rows[strings.ToLower(code)]
	return row, ok
}

func (h *Handler) SetCloaks(c *CloakCatalog) { h.cloaks = c }

func (h *Handler) serveCloak(w http.ResponseWriter, r *http.Request, code string) {
	start := time.Now()
	h.requests.WithLabelValues("cloak").Inc()
	if h.cloaks == nil {
		h.writeCloakLog(r, code, nil, nil, cloakdec.ActionNotFound, "cloak:missing", start)
		cloakNotFound(w)
		return
	}
	row, ok := h.cloaks.Get(code)
	host := hostOnly(r.Host)
	if !ok || !strings.EqualFold(row.Domain, host) {
		h.writeCloakLog(r, code, nil, nil, cloakdec.ActionNotFound, "cloak:missing", start)
		cloakNotFound(w)
		return
	}
	if !cloakUsable(row, time.Now()) {
		id := row.ID
		mid := row.MerchantID
		h.writeCloakLog(r, code, &mid, &id, cloakdec.ActionNotFound, "cloak:disabled", start)
		cloakNotFound(w)
		return
	}
	country := ""
	if h.d.Geo != nil {
		country = h.d.Geo.Country(h.clientIP(r))
	}
	decision := cloakdec.Decide(cloakdec.Input{
		Mode: row.Mode, Regions: splitJSON(row.TargetRegions), Sources: splitJSON(row.AllowedSources),
		Spiders: splitJSON(row.SpiderWhitelist), DeviceWhitelist: row.DeviceWhitelist,
		BlockPC: row.BlockPC, BlockProxy: row.BlockProxy, BlockEmptyLanguage: row.BlockEmptyLanguage,
		Country: country, UA: r.UserAgent(), AcceptLanguage: r.Header.Get("Accept-Language"),
		Via: r.Header.Get("Via"), ForwardedFor: r.Header.Get("X-Forwarded-For"), Referer: r.Referer(),
	})
	id := row.ID
	mid := row.MerchantID
	h.writeCloakLog(r, code, &mid, &id, decision.Action, decision.HitRule, start)
	h.writeCloakResult(w, r, row, decision)
}

func cloakUsable(row mysql.GatewayCloak, now time.Time) bool {
	if row.Status != cloak.StatusEnabled || row.DomainStatus != "bound" || row.MerchantStatus != identity.StatusEnabled {
		return false
	}
	if row.ExpireAt != nil && row.ExpireAt.Before(now) {
		return false
	}
	return true
}

var cloakCodeRE = regexp.MustCompile(`^[a-z0-9]{4,32}$`)

func parseCloakCode(path string) (string, bool) {
	if !strings.HasPrefix(path, "/c/") {
		return "", false
	}
	code := strings.ToLower(strings.TrimPrefix(path, "/c/"))
	if strings.Contains(code, "/") || !cloakCodeRE.MatchString(code) {
		return "", false
	}
	return code, true
}

func (h *Handler) writeCloakResult(w http.ResponseWriter, r *http.Request, row mysql.GatewayCloak, d cloakdec.Result) {
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Cache-Control", "no-store")
	if d.Action == cloakdec.ActionReview && row.ReviewMode == cloak.ReviewLocal {
		writeHTML(w, localReviewHTML, localReviewData(row.ReviewTag))
		return
	}
	target := ""
	if d.Action == cloakdec.ActionReal {
		urls := splitJSON(row.RealURLs)
		if len(urls) > 0 {
			target = urls[0]
		}
	} else if d.Action == cloakdec.ActionReview {
		target = row.ReviewURL
	}
	if target == "" {
		cloakNotFound(w)
		return
	}
	if row.IntermediatePage == cloak.PageNone {
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	text := "继续访问"
	if row.IntermediatePage == cloak.PageCustom && strings.TrimSpace(row.IntermediateText) != "" {
		text = row.IntermediateText
	}
	writeHTML(w, intermediateHTML, map[string]string{"Text": text, "URL": target})
}

func cloakNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Not Found"))
}

func (h *Handler) writeCloakLog(r *http.Request, code string, merchantID, cloakID *uint64, action, rule string, start time.Time) {
	if h.cloaks == nil {
		return
	}
	country := ""
	if h.d.Geo != nil && merchantID != nil {
		country = h.d.Geo.Country(h.clientIP(r))
	}
	row := &cloak.AccessLog{
		MerchantID: merchantID, CloakID: cloakID, Code: code, Country: country,
		Device: cloakdec.Device(r.UserAgent()), HitRule: rule, FinalAction: action,
		LatencyMs: uint32(time.Since(start).Milliseconds()),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.cloaks.repo.InsertLog(ctx, row); err != nil && h.d.Log != nil {
		h.d.Log.Warn("cloak access log", zap.Error(err))
	}
}

func hostOnly(host string) string {
	if i := strings.LastIndex(host, ":"); i > 0 && !strings.Contains(host, "]") {
		return host[:i]
	}
	return host
}

func splitJSON(raw string) []string {
	out := []string{}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		return []string{}
	}
	return out
}

func localReviewData(tag string) map[string]string {
	pages := map[string][2]string{
		"google":   {"Google", "此页面用于确认访问来源。"},
		"facebook": {"Facebook", "此页面用于确认广告访问。"},
		"tiktok":   {"TikTok", "此页面用于确认短视频访问。"},
		"generic":  {"安全页", "当前访问展示审核内容。"},
	}
	item, ok := pages[tag]
	if !ok {
		item = pages["generic"]
	}
	return map[string]string{"Title": item[0], "Text": item[1]}
}

var localReviewHTML = template.Must(template.New("review").Parse(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="robots" content="noindex"><title>{{.Title}}</title></head><body style="font-family:sans-serif;padding:48px"><h1>{{.Title}}</h1><p>{{.Text}}</p></body></html>`))

var intermediateHTML = template.Must(template.New("mid").Parse(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="robots" content="noindex"><title>继续访问</title></head><body style="font-family:sans-serif;padding:48px"><p>{{.Text}}</p><p><a href="{{.URL}}">继续访问</a></p></body></html>`))

func writeHTML(w http.ResponseWriter, tpl *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = tpl.Execute(w, data)
}
