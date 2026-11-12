package pathrule

import (
	"strings"

	"cloak/internal/engine/snapshot"
)

// staticRule 平台静态资源规则。全站执行，不写入站点，商户端不展示。
type staticRule struct {
	pattern string
	remark  string
}

var platformStatic = []staticRule{
	{".css", "静态资源"},
	{".js", "静态资源"},
	{".png", "静态资源"},
	{".jpg", "静态资源"},
	{".jpeg", "静态资源"},
	{".gif", "静态资源"},
	{".webp", "静态资源"},
	{".svg", "静态资源"},
	{".ico", "静态资源"},
	{".woff", "静态资源"},
	{".woff2", "静态资源"},
	{".map", "静态资源"},
}

// StaticView 给总后台展示的静态资源规则。
func StaticView() []map[string]any {
	out := make([]map[string]any, 0, len(platformStatic))
	for i, it := range platformStatic {
		out = append(out, map[string]any{
			"pattern": it.pattern, "matchType": MatchSuffix, "action": "bypass",
			"sort": (i + 1) * 10, "remark": it.remark,
		})
	}
	return out
}

// IsPlatformStatic 是否为平台静态资源规则。这类规则不交给商户保存。
func IsPlatformStatic(pattern, matchType, action string) bool {
	if matchType != MatchSuffix || action != "bypass" {
		return false
	}
	p := strings.ToLower(strings.TrimSpace(pattern))
	for _, it := range platformStatic {
		if it.pattern == p {
			return true
		}
	}
	return false
}

// matchStatic 平台规则优先于站点规则，命中最长后缀。
func matchStatic(lower string) *snapshot.PathRule {
	var hit *staticRule
	for i := range platformStatic {
		p := platformStatic[i].pattern
		if strings.HasSuffix(lower, p) && (hit == nil || len(p) > len(hit.pattern)) {
			hit = &platformStatic[i]
		}
	}
	if hit == nil {
		return nil
	}
	rule := snapshot.PathRule{Pattern: hit.pattern, MatchType: MatchSuffix, Action: "bypass"}
	return &rule
}
