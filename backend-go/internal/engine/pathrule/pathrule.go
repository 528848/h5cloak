// Package pathrule 路径规则匹配。
package pathrule

import (
	"strings"

	"cloak/internal/engine/snapshot"
)

const (
	MatchPrefix = "prefix"
	MatchExact  = "exact"
	MatchSuffix = "suffix"
)

// Match 返回第一条命中的规则（按 sort 升序），未命中返回 nil。
// 后缀规则忽略大小写（静态资源扩展名），前缀与精确按原样比较。
func Match(site *snapshot.Site, path string) *snapshot.PathRule {
	if path == "" {
		path = "/"
	}
	rules := site.SortedRules()
	lower := strings.ToLower(path)
	for i := range rules {
		r := &rules[i]
		switch r.MatchType {
		case MatchExact:
			if path == r.Pattern {
				return r
			}
		case MatchSuffix:
			if strings.HasSuffix(lower, strings.ToLower(r.Pattern)) {
				return r
			}
		default:
			if strings.HasPrefix(path, r.Pattern) {
				return r
			}
		}
	}
	return nil
}

// Allowed 判断路径是否在允许前缀列表内；列表为空表示任意路径。
func Allowed(path string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
