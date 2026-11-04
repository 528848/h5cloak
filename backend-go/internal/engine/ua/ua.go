// Package ua 内置浏览器与设备族识别。只做字符串特征匹配，不依赖外部库。
package ua

import (
	"regexp"
	"strings"
	"sync"
)

// Info 识别结果。
type Info struct {
	InApp        bool   `json:"detected"`
	App          string `json:"app,omitempty"`
	OS           string `json:"os"`
	OSMajor      int    `json:"osMajor"`
	DeviceFamily string `json:"deviceFamily"`
}

// Marker 内置浏览器特征。可被全局配置整体替换。
type Marker struct {
	Marker string `json:"marker"`
	App    string `json:"app"`
}

func builtinMarkers() []Marker {
	return []Marker{
		{"FBAN", "facebook"}, {"FBAV", "facebook"}, {"FB_IAB", "facebook"}, {"FBIOS", "facebook"},
		{"Instagram", "instagram"},
		{"musical_ly", "tiktok"}, {"BytedanceWebview", "tiktok"}, {"TikTok", "tiktok"}, {"Bytedance", "tiktok"},
		{"Snapchat", "snapchat"},
		{"Twitter", "twitter"}, {"TwitterAndroid", "twitter"},
		{"LinkedInApp", "linkedin"},
		{"Line/", "line"},
		{"MicroMessenger", "wechat"},
		{"Pinterest", "pinterest"},
		{"KAKAOTALK", "kakao"},
		{"Kwai", "kwai"}, {"KwaiPro", "kwai"}, {"Kuaishou", "kwai"},
		{"Telegram", "telegram"},
		{"GSA/", "google_app"},
	}
}

var (
	markerMu     sync.RWMutex
	inAppMarkers = builtinMarkers()
)

// BuiltinMarkers 返回内置特征的副本。
func BuiltinMarkers() []Marker { return builtinMarkers() }

// SetMarkers 用全局配置替换特征表。空列表恢复内置默认。
func SetMarkers(ms []Marker) {
	markerMu.Lock()
	defer markerMu.Unlock()
	next := make([]Marker, 0, len(ms))
	for _, m := range ms {
		m.Marker = strings.TrimSpace(m.Marker)
		m.App = strings.TrimSpace(m.App)
		if m.Marker != "" && m.App != "" {
			next = append(next, m)
		}
	}
	if len(next) == 0 {
		inAppMarkers = builtinMarkers()
		return
	}
	inAppMarkers = next
}

func markers() []Marker {
	markerMu.RLock()
	defer markerMu.RUnlock()
	out := make([]Marker, len(inAppMarkers))
	copy(out, inAppMarkers)
	return out
}

var (
	iosVer     = regexp.MustCompile(`OS (\d+)[_.]`)
	androidVer = regexp.MustCompile(`Android (\d+)`)
	androidDev = regexp.MustCompile(`Android [\d.]+; ([^;)]+)`)
)

// Parse 解析 UA。
func Parse(uaStr string) Info {
	info := Info{OS: "other"}
	if uaStr == "" {
		info.DeviceFamily = "other"
		return info
	}
	for _, m := range markers() {
		if strings.Contains(uaStr, m.Marker) {
			info.InApp = true
			info.App = m.App
			break
		}
	}
	// 通用 WebView 特征（Android）
	if !info.InApp && strings.Contains(uaStr, "; wv)") {
		info.InApp = true
		info.App = "webview"
	}
	switch {
	case strings.Contains(uaStr, "iPhone") || strings.Contains(uaStr, "iPad") || strings.Contains(uaStr, "iPod"):
		info.OS = "ios"
		if m := iosVer.FindStringSubmatch(uaStr); len(m) == 2 {
			info.OSMajor = atoi(m[1])
		}
		model := "iphone"
		if strings.Contains(uaStr, "iPad") {
			model = "ipad"
		}
		info.DeviceFamily = "ios" + itoa(info.OSMajor) + "-" + model
	case strings.Contains(uaStr, "Android"):
		info.OS = "android"
		if m := androidVer.FindStringSubmatch(uaStr); len(m) == 2 {
			info.OSMajor = atoi(m[1])
		}
		brand := "generic"
		if m := androidDev.FindStringSubmatch(uaStr); len(m) == 2 {
			brand = brandOf(m[1])
		}
		info.DeviceFamily = "android" + itoa(info.OSMajor) + "-" + brand
	case strings.Contains(uaStr, "Windows"):
		info.OS = "windows"
		info.DeviceFamily = "windows"
	case strings.Contains(uaStr, "Mac OS X"):
		info.OS = "macos"
		info.DeviceFamily = "macos"
	case strings.Contains(uaStr, "Linux"):
		info.OS = "linux"
		info.DeviceFamily = "linux"
	default:
		info.DeviceFamily = "other"
	}
	return info
}

func brandOf(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(m, "sm-"), strings.Contains(m, "samsung"):
		return "samsung"
	case strings.HasPrefix(m, "pixel"):
		return "pixel"
	case strings.Contains(m, "xiaomi"), strings.HasPrefix(m, "mi "), strings.HasPrefix(m, "redmi"), strings.HasPrefix(m, "m2"), strings.HasPrefix(m, "2201"), strings.HasPrefix(m, "2301"):
		return "xiaomi"
	case strings.Contains(m, "huawei"), strings.HasPrefix(m, "ana-"), strings.HasPrefix(m, "els-"):
		return "huawei"
	case strings.HasPrefix(m, "cph"), strings.Contains(m, "oppo"):
		return "oppo"
	case strings.HasPrefix(m, "v2"), strings.Contains(m, "vivo"):
		return "vivo"
	case strings.Contains(m, "oneplus"), strings.HasPrefix(m, "le2"), strings.HasPrefix(m, "ne2"):
		return "oneplus"
	case strings.Contains(m, "moto"):
		return "motorola"
	case strings.HasPrefix(m, "k"), strings.Contains(m, "lg"):
		return "lg"
	default:
		return "generic"
	}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
