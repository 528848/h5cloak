package telegramsvc

import "testing"

func TestParseText(t *testing.T) {
	s := &Service{}
	sites := map[string]uint64{"站点a": 7}
	cases := []struct {
		text string
		kind string
		rng  string
		site uint64
	}{
		{"今天多少访问", "visits", "today", 0},
		{"昨天转化多少", "conversion", "yesterday", 0},
		{"近7天怎么样", "visits", "7d", 0},
		{"有没有告警", "alerts", "today", 0},
		{"/alerts", "alerts", "today", 0},
		{"日报", "report", "today", 0},
		{"帮助", "help", "today", 0},
		{"/today", "visits", "today", 0},
		{"今天 站点A 多少访问", "visits", "today", 7},
		{"随便说点啥", "unknown", "today", 0},
	}
	for _, c := range cases {
		in := s.parseText(c.text, sites)
		if in.kind != c.kind || in.rangeKey != c.rng || in.siteID != c.site {
			t.Fatalf("%q => kind=%s range=%s site=%d, want kind=%s range=%s site=%d",
				c.text, in.kind, in.rangeKey, in.siteID, c.kind, c.rng, c.site)
		}
	}
}

func TestBindArg(t *testing.T) {
	cases := map[string]struct {
		code string
		ok   bool
	}{
		"/start abc123":         {"abc123", true},
		"/bind deadbeef":        {"deadbeef", true},
		"/start@shop_bot code1": {"code1", true},
		"/startabcdef12":        {"abcdef12", true},
		"/startup":              {"", false},
		"hello":                 {"", false},
		"/start":                {"", false},
	}
	for text, want := range cases {
		code, ok := bindArg(text)
		if ok != want.ok || code != want.code {
			t.Fatalf("%q => code=%q ok=%v, want code=%q ok=%v", text, code, ok, want.code, want.ok)
		}
	}
}

func TestNormalize(t *testing.T) {
	if got := normalize("　今天　ＡＢＣ "); got != "今天abc" {
		t.Fatalf("normalize = %q", got)
	}
}
