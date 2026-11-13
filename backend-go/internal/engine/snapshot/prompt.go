package snapshot

import "strings"

// PromptCopy 一端的引导文案。
type PromptCopy struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Open  string `json:"open"`
	Stay  string `json:"stay"`
}

// Prompt 引导弹框。未配置的站点用预设：10 秒、30 分钟、3 次。
type Prompt struct {
	DelaySeconds    int        `json:"delaySeconds"`
	CooldownMinutes int        `json:"cooldownMinutes"`
	MaxShows        int        `json:"maxShows"`
	DismissEnds     bool       `json:"dismissEnds"`
	Mobile          PromptCopy `json:"mobile"`
	Desktop         PromptCopy `json:"desktop"`
}

// PromptInput 保存请求。数字用指针区分「没传」和「传了 0」。
type PromptInput struct {
	DelaySeconds    *int       `json:"delaySeconds"`
	CooldownMinutes *int       `json:"cooldownMinutes"`
	MaxShows        *int       `json:"maxShows"`
	DismissEnds     *bool      `json:"dismissEnds"`
	Mobile          PromptCopy `json:"mobile"`
	Desktop         PromptCopy `json:"desktop"`
}

// EffectivePrompt 快照里没写过这三项时返回预设，并把旧的单句文案当作两端说明。
func (h Handoff) EffectivePrompt() Prompt {
	return resolvePrompt(h.Prompt, h.BannerText, promptConfigured(h.Prompt))
}

// NormalizePrompt 把后台提交收成可入库的值。没传的数字用预设，延迟 0 会保留。
func NormalizePrompt(in PromptInput, legacy string) Prompt {
	p := Prompt{Mobile: in.Mobile, Desktop: in.Desktop}
	if in.DelaySeconds == nil {
		p.DelaySeconds = 10
	} else {
		p.DelaySeconds = clampInt(*in.DelaySeconds, 0, 120)
	}
	if in.CooldownMinutes == nil {
		p.CooldownMinutes = 30
	} else {
		p.CooldownMinutes = clampInt(*in.CooldownMinutes, 5, 1440)
	}
	if in.MaxShows == nil {
		p.MaxShows = 3
	} else {
		p.MaxShows = clampInt(*in.MaxShows, 1, 5)
	}
	if in.DismissEnds != nil {
		p.DismissEnds = *in.DismissEnds
	}
	return resolvePrompt(p, legacy, true)
}

func promptConfigured(p Prompt) bool {
	return p.DelaySeconds != 0 || p.CooldownMinutes != 0 || p.MaxShows != 0 || p.DismissEnds ||
		p.Mobile != (PromptCopy{}) || p.Desktop != (PromptCopy{})
}

func resolvePrompt(p Prompt, legacy string, configured bool) Prompt {
	if !configured {
		p = Prompt{DelaySeconds: 10, CooldownMinutes: 30, MaxShows: 3}
	} else {
		if p.DelaySeconds < 0 {
			p.DelaySeconds = 0
		}
		if p.DelaySeconds > 120 {
			p.DelaySeconds = 120
		}
		if p.CooldownMinutes < 5 {
			p.CooldownMinutes = 30
		}
		if p.CooldownMinutes > 1440 {
			p.CooldownMinutes = 1440
		}
		if p.MaxShows < 1 {
			p.MaxShows = 3
		}
		if p.MaxShows > 5 {
			p.MaxShows = 5
		}
	}
	mobile := defaultMobile()
	desktop := defaultDesktop()
	legacy = strings.TrimSpace(legacy)
	if legacy != "" && strings.TrimSpace(p.Mobile.Body) == "" && strings.TrimSpace(p.Desktop.Body) == "" {
		mobile.Body = legacy
		desktop.Body = legacy
	}
	rawMobile, rawDesktop := p.Mobile, p.Desktop
	p.Mobile = fillCopy(rawMobile, rawDesktop, mobile)
	p.Desktop = fillCopy(rawDesktop, rawMobile, desktop)
	return p
}

func defaultMobile() PromptCopy {
	return PromptCopy{
		Title: "在浏览器中打开",
		Body:  "用系统浏览器打开，页面和登录会更完整。",
		Open:  "打开",
		Stay:  "继续浏览",
	}
}

func defaultDesktop() PromptCopy {
	return PromptCopy{
		Title: "用系统浏览器打开",
		Body:  "当前是应用内置浏览器。换到系统浏览器后可以继续这次访问。",
		Open:  "打开",
		Stay:  "稍后",
	}
}

func fillCopy(primary, other, def PromptCopy) PromptCopy {
	return PromptCopy{
		Title: clip(firstText(primary.Title, other.Title, def.Title), 40),
		Body:  clip(firstText(primary.Body, other.Body, def.Body), 120),
		Open:  clip(firstText(primary.Open, other.Open, def.Open), 12),
		Stay:  clip(firstText(primary.Stay, other.Stay, def.Stay), 12),
	}
}

func firstText(vals ...string) string {
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
