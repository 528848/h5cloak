// Package cloakdec 是斗篷短链的纯判定，不访问数据库。
package cloakdec

import (
	"net/url"
	"strings"
)

const (
	ActionReal     = "real"
	ActionReview   = "review"
	ActionNotFound = "not_found"
)

// Input 一次短链访问的判定输入。Country 为空表示地区库没有结果。
type Input struct {
	Mode               string
	Regions            []string
	Sources            []string
	Spiders            []string
	DeviceWhitelist    string
	BlockPC            bool
	BlockProxy         bool
	BlockEmptyLanguage bool
	Country            string
	UA                 string
	AcceptLanguage     string
	Via                string
	ForwardedFor       string
	Referer            string
}

// Result 判定结果。
type Result struct {
	Action  string
	HitRule string
	Device  string
}

// Decide 按产品文档第 7 节的顺序判定。调用方已确认斗篷存在且可用。
func Decide(in Input) Result {
	device := Device(in.UA)
	switch in.Mode {
	case "green":
		return Result{Action: ActionReview, HitRule: "mode:green", Device: device}
	case "review":
		return Result{Action: ActionReview, HitRule: "mode:review", Device: device}
	case "full_open":
		return Result{Action: ActionReal, HitRule: "mode:full_open", Device: device}
	}
	if in.BlockPC && device == "pc" {
		return Result{Action: ActionReview, HitRule: "rule:block_pc", Device: device}
	}
	if in.BlockProxy && looksLikeProxy(in.Via, in.ForwardedFor) {
		return Result{Action: ActionReview, HitRule: "rule:block_proxy", Device: device}
	}
	if in.BlockEmptyLanguage && strings.TrimSpace(in.AcceptLanguage) == "" {
		return Result{Action: ActionReview, HitRule: "rule:block_empty_language", Device: device}
	}
	if len(in.Regions) > 0 && in.Country != "" && !containsFold(in.Regions, in.Country) {
		return Result{Action: ActionReview, HitRule: "rule:region_mismatch", Device: device}
	}
	if !sourceAllowed(in.Referer, in.Sources) {
		return Result{Action: ActionReview, HitRule: "rule:source_mismatch", Device: device}
	}
	if spiderHit(in.UA, in.Spiders) {
		return Result{Action: ActionReview, HitRule: "rule:spider_whitelist", Device: device}
	}
	if in.DeviceWhitelist != "" && !deviceAllowed(device, in.DeviceWhitelist) {
		return Result{Action: ActionReview, HitRule: "rule:device_whitelist", Device: device}
	}
	return Result{Action: ActionReal, HitRule: "rule:pass", Device: device}
}

// Device 按 UA 分类。bot 不属于移动端、平板或 PC。
func Device(ua string) string {
	s := strings.ToLower(ua)
	switch {
	case strings.Contains(s, "bot") || strings.Contains(s, "spider") || strings.Contains(s, "crawler"):
		return "bot"
	case strings.Contains(s, "ipad") || strings.Contains(s, "tablet"):
		return "tablet"
	case strings.Contains(s, "mobile"):
		return "mobile"
	default:
		return "pc"
	}
}

func looksLikeProxy(via, forwarded string) bool {
	if strings.TrimSpace(via) != "" {
		return true
	}
	return strings.Contains(forwarded, ",")
}

func sourceAllowed(referer string, sources []string) bool {
	if len(sources) == 0 || containsFold(sources, "all") {
		return true
	}
	host := refererHost(referer)
	for _, src := range sources {
		if src != "" && strings.Contains(host, strings.ToLower(src)) {
			return true
		}
	}
	return false
}

func refererHost(referer string) string {
	u, err := url.Parse(strings.TrimSpace(referer))
	if err != nil || u.Host == "" {
		return strings.ToLower(referer)
	}
	return strings.ToLower(u.Hostname())
}

func spiderHit(ua string, spiders []string) bool {
	s := strings.ToLower(ua)
	for _, item := range spiders {
		needle := spiderNeedle(item)
		if needle != "" && strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func spiderNeedle(item string) string {
	switch strings.ToLower(strings.TrimSpace(item)) {
	case "googlebot":
		return "googlebot"
	case "bingbot":
		return "bingbot"
	case "facebookexternalhit", "facebook":
		return "facebookexternalhit"
	case "tiktokbot", "tiktok":
		return "tiktok"
	default:
		return strings.ToLower(strings.TrimSpace(item))
	}
}

func deviceAllowed(device, whitelist string) bool {
	for _, part := range strings.Split(whitelist, ",") {
		if strings.TrimSpace(part) == device {
			return true
		}
	}
	return false
}

func containsFold(list []string, want string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), want) {
			return true
		}
	}
	return false
}
