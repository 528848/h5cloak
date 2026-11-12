package cloakdec

import "testing"

func TestDecideModes(t *testing.T) {
	cases := []struct {
		mode, action, rule string
	}{
		{"green", ActionReview, "mode:green"},
		{"review", ActionReview, "mode:review"},
		{"full_open", ActionReal, "mode:full_open"},
		{"cloak", ActionReal, "rule:pass"},
	}
	for _, c := range cases {
		got := Decide(Input{Mode: c.mode, UA: "Mozilla/5.0 (Windows NT 10.0)"})
		if got.Action != c.action || got.HitRule != c.rule {
			t.Fatalf("%s: got %s %s", c.mode, got.Action, got.HitRule)
		}
	}
}

func TestDecideCloakRules(t *testing.T) {
	base := Input{Mode: "cloak", UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile"}
	if got := Decide(base); got.Action != ActionReal || got.Device != "mobile" {
		t.Fatalf("mobile pass: %+v", got)
	}
	pc := base
	pc.UA = "Mozilla/5.0 (Windows NT 10.0; Win64)"
	pc.BlockPC = true
	if got := Decide(pc); got.HitRule != "rule:block_pc" {
		t.Fatalf("pc: %+v", got)
	}
	proxy := base
	proxy.BlockProxy = true
	proxy.ForwardedFor = "1.1.1.1, 2.2.2.2"
	if got := Decide(proxy); got.HitRule != "rule:block_proxy" {
		t.Fatalf("proxy: %+v", got)
	}
	lang := base
	lang.BlockEmptyLanguage = true
	if got := Decide(lang); got.HitRule != "rule:block_empty_language" {
		t.Fatalf("lang: %+v", got)
	}
	region := base
	region.Regions = []string{"US"}
	region.Country = "JP"
	if got := Decide(region); got.HitRule != "rule:region_mismatch" {
		t.Fatalf("region: %+v", got)
	}
	unknown := region
	unknown.Country = ""
	if got := Decide(unknown); got.Action != ActionReal {
		t.Fatalf("unknown country should pass: %+v", got)
	}
	source := base
	source.Sources = []string{"facebook"}
	source.Referer = "https://news.google.com/x"
	if got := Decide(source); got.HitRule != "rule:source_mismatch" {
		t.Fatalf("source: %+v", got)
	}
	source.Referer = "https://l.facebook.com/l.php"
	if got := Decide(source); got.Action != ActionReal {
		t.Fatalf("facebook referer: %+v", got)
	}
	spider := base
	spider.UA = "Mozilla/5.0 (compatible; Googlebot/2.1)"
	spider.Spiders = []string{"googlebot"}
	if got := Decide(spider); got.HitRule != "rule:spider_whitelist" || got.Device != "bot" {
		t.Fatalf("spider: %+v", got)
	}
	dev := base
	dev.DeviceWhitelist = "pc"
	if got := Decide(dev); got.HitRule != "rule:device_whitelist" {
		t.Fatalf("device: %+v", got)
	}
	open := pc
	open.Mode = "full_open"
	if got := Decide(open); got.Action != ActionReal || got.HitRule != "mode:full_open" {
		t.Fatalf("full_open ignores pc block: %+v", got)
	}
}

func TestDevice(t *testing.T) {
	if Device("Mozilla/5.0 (iPad)") != "tablet" {
		t.Fatal("ipad")
	}
	if Device("curl/8.0") != "pc" {
		t.Fatal("pc")
	}
}
