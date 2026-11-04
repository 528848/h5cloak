package ua

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		ua      string
		inApp   bool
		app, os string
		family  string
	}{
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 [FBAN/FBIOS;FBAV/430.0.0]", true, "facebook", "ios", "ios17-iphone"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 Instagram 290.0.0.0", true, "instagram", "ios", "ios16-iphone"},
		{"Mozilla/5.0 (Linux; Android 14; SM-S918B Build/UP1A; wv) AppleWebKit/537.36 Chrome/120 Mobile Safari/537.36 musical_ly_2023", true, "tiktok", "android", "android14-samsung"},
		{"Mozilla/5.0 (Linux; Android 13; Pixel 7; wv) AppleWebKit/537.36 Chrome/118 Mobile Safari/537.36", true, "webview", "android", "android13-pixel"},
		{"Mozilla/5.0 (Linux; Android 12; 2201116SG) AppleWebKit/537.36 Chrome/118 Mobile Safari/537.36", false, "", "android", "android12-xiaomi"},
		{"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1", false, "", "ios", "ios17-ipad"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120 Safari/537.36", false, "", "windows", "windows"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/17.0 Safari/605.1.15", false, "", "macos", "macos"},
		{"Mozilla/5.0 (X11; Linux x86_64) Gecko Firefox/120", false, "", "linux", "linux"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Snapchat/12.80.0.40", true, "snapchat", "ios", "ios17-iphone"},
		{"Mozilla/5.0 (Linux; Android 13; SM-A536B Build/TP1A) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/119 Mobile Safari/537.36 Line/13.18.1", true, "line", "android", "android13-samsung"},
		{"Mozilla/5.0 (Linux; Android 14; Pixel 8 Build/UD1A) AppleWebKit/537.36 Chrome/120 Mobile Safari/537.36 TwitterAndroid", true, "twitter", "android", "android14-pixel"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 [LinkedInApp]/9.29", true, "linkedin", "ios", "ios16-iphone"},
		{"Mozilla/5.0 (Linux; Android 12; 2201116SG Build/SKQ1) AppleWebKit/537.36 Chrome/116 Mobile Safari/537.36 MicroMessenger/8.0.42", true, "wechat", "android", "android12-xiaomi"},
		{"Mozilla/5.0 (Linux; Android 13; CPH2451) AppleWebKit/537.36 Chrome/118 Mobile Safari/537.36 Kwai/10.2.30", true, "kwai", "android", "android13-oppo"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 GSA/300.0.0", true, "google_app", "ios", "ios17-iphone"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0 Mobile/15E148 Safari/604.1", false, "", "ios", "ios17-iphone"},
		{"Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Mobile Safari/537.36", false, "", "android", "android14-samsung"},
		{"curl/8.0", false, "", "other", "other"},
		{"", false, "", "other", "other"},
	}
	for _, c := range cases {
		got := Parse(c.ua)
		if got.InApp != c.inApp || got.App != c.app || got.OS != c.os || got.DeviceFamily != c.family {
			t.Errorf("%q: got %+v", c.ua, got)
		}
	}
}

func TestBrandOf(t *testing.T) {
	for in, want := range map[string]string{
		"SM-G998B": "samsung", "Pixel 8": "pixel", "Redmi Note 12": "xiaomi", "ANA-NX9": "huawei",
		"CPH2451": "oppo", "V2250": "vivo", "LE2123": "oneplus", "moto g84": "motorola", "K41S": "lg", "Unknown": "generic",
	} {
		if got := brandOf(in); got != want {
			t.Errorf("%s: %s want %s", in, got, want)
		}
	}
}
