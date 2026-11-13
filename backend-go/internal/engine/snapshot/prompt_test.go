package snapshot

import "testing"

func TestPromptDefaults(t *testing.T) {
	p := (Handoff{BannerText: "旧文案"}).EffectivePrompt()
	if p.DelaySeconds != 10 || p.CooldownMinutes != 30 || p.MaxShows != 3 || p.DismissEnds {
		t.Fatalf("defaults: %+v", p)
	}
	if p.Mobile.Body != "旧文案" || p.Desktop.Body != "旧文案" {
		t.Fatalf("legacy body: %+v %+v", p.Mobile, p.Desktop)
	}
	if p.Mobile.Title != "在浏览器中打开" || p.Desktop.Title != "用系统浏览器打开" {
		t.Fatalf("titles: %q %q", p.Mobile.Title, p.Desktop.Title)
	}
}

func TestNormalizePromptKeepsZeroDelay(t *testing.T) {
	zero := 0
	cool := 45
	shows := 2
	p := NormalizePrompt(PromptInput{DelaySeconds: &zero, CooldownMinutes: &cool, MaxShows: &shows}, "")
	if p.DelaySeconds != 0 || p.CooldownMinutes != 45 || p.MaxShows != 2 {
		t.Fatalf("explicit: %+v", p)
	}
	if p.Mobile.Body == "" || p.Desktop.Stay != "稍后" {
		t.Fatalf("filled copy: %+v %+v", p.Mobile, p.Desktop)
	}
}

func TestNormalizePromptEmptyNumbers(t *testing.T) {
	p := NormalizePrompt(PromptInput{}, "")
	if p.DelaySeconds != 10 || p.CooldownMinutes != 30 || p.MaxShows != 3 {
		t.Fatalf("missing numbers: %+v", p)
	}
	high := 9999
	p = NormalizePrompt(PromptInput{DelaySeconds: &high, CooldownMinutes: &high, MaxShows: &high}, "")
	if p.DelaySeconds != 120 || p.CooldownMinutes != 1440 || p.MaxShows != 5 {
		t.Fatalf("clamped: %+v", p)
	}
}

func TestPromptCopyFallsAcrossSides(t *testing.T) {
	p := NormalizePrompt(PromptInput{
		Mobile:  PromptCopy{Title: "手机标题"},
		Desktop: PromptCopy{Body: "电脑说明"},
	}, "")
	if p.Desktop.Title != "手机标题" || p.Mobile.Body != "电脑说明" {
		t.Fatalf("cross fill: %+v %+v", p.Mobile, p.Desktop)
	}
}
