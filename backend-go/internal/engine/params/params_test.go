package params

import (
	"net/url"
	"testing"
	"time"

	"cloak/internal/engine/snapshot"
)

func site() *snapshot.Site {
	s := &snapshot.Site{
		Params: []snapshot.Param{
			{Name: "fbclid", Type: TypeClick, Platform: "facebook", MinLen: 5, MaxLen: 50, Sufficient: true, Enabled: true},
			{Name: "utm_source", Type: TypeSource, MinLen: 1, MaxLen: 50, Sufficient: true, Enabled: true},
			{Name: "ref", Type: TypeDistributor, Aliases: []string{"r"}, MinLen: 2, MaxLen: 20, Pattern: "^[A-Z]+$", Sufficient: true, Enabled: true},
			{Name: "note", Type: TypeFree, MinLen: 1, MaxLen: 5, Sufficient: false, Enabled: true},
		},
		Rule:  snapshot.Rule{CombineRule: "require_any", ValidateCodes: true, ValidateClickFormat: true, ClickReuseLimit: 1},
		Codes: []snapshot.Code{{ID: 1, Code: "ALICE", Type: "distributor", Enabled: true}},
	}
	_ = s.Compile()
	return s
}

func q(s string) url.Values {
	v, _ := url.ParseQuery(s)
	return v
}

func TestExtractOrderAndDedup(t *testing.T) {
	ms := Extract(site(), q("utm_source=a&fbclid=abcdef&ref=&r=ALICE&unknown=1&note=hi"))
	if len(ms) != 4 {
		t.Fatalf("len %d: %+v", len(ms), ms)
	}
	// 码 > 点击 > 来源 > 自由
	if ms[0].Name != "ref" || ms[0].Raw != "r" || ms[1].Name != "fbclid" || ms[2].Name != "utm_source" || ms[3].Name != "note" {
		t.Fatalf("order: %+v", ms)
	}
}

func TestValidateNoCodeCheck(t *testing.T) {
	s := site()
	s.Rule.ValidateCodes = false
	_ = s.Compile()
	res := Validate(s, Extract(s, q("ref=ZZZ")), Env{Path: "/", Now: time.Now()})
	if !res.Attributed || res.Code == nil || res.Code.Code != "ZZZ" || res.Confidence != 3 {
		t.Fatalf("unregistered code accepted when check off: %+v", res)
	}
	res = Validate(s, Extract(s, q("ref=lower")), Env{Path: "/", Now: time.Now()})
	if res.Attributed || res.Errors[0].Reason != ErrFormatInvalid {
		t.Fatalf("format still enforced: %+v", res)
	}
}

func TestFreeAndSourceFormat(t *testing.T) {
	s := site()
	res := Validate(s, Extract(s, q("note=toolong")), Env{Path: "/", Now: time.Now()})
	if res.Attributed || res.Errors[0].Reason != ErrFormatInvalid {
		t.Fatalf("free param format: %+v", res)
	}
	res = Validate(s, Extract(s, q("note=ok")), Env{Path: "/", Now: time.Now()})
	if res.Attributed || !res.HasFree || len(res.Errors) != 0 {
		t.Fatalf("free param valid but insufficient: %+v", res)
	}
	long := make([]byte, 60)
	for i := range long {
		long[i] = 'a'
	}
	res = Validate(s, Extract(s, q("utm_source="+string(long))), Env{Path: "/", Now: time.Now()})
	if res.Attributed || res.Errors[0].Reason != ErrFormatInvalid {
		t.Fatalf("source format: %+v", res)
	}
}

func TestClickReuseCallback(t *testing.T) {
	s := site()
	calls := 0
	env := Env{Path: "/", Now: time.Now(), ClickReused: func(v string) bool { calls++; return true }}
	res := Validate(s, Extract(s, q("fbclid=abcdef")), env)
	if calls != 1 || !res.Attributed || res.Confidence != 1 {
		t.Fatalf("downgrade: calls=%d %+v", calls, res)
	}
	s.Rule.ClickReuseAction = "review"
	res = Validate(s, Extract(s, q("fbclid=abcdef")), env)
	if res.Attributed || res.Errors[0].Reason != ErrClickReused {
		t.Fatalf("review action: %+v", res)
	}
	// 有码时复用不降级
	s.Rule.ClickReuseAction = "downgrade"
	res = Validate(s, Extract(s, q("fbclid=abcdef&ref=ALICE")), env)
	if res.Confidence != 3 {
		t.Fatalf("code keeps high: %+v", res)
	}
}

func TestChannelFallbackToClickName(t *testing.T) {
	s := site()
	s.Params[0].Platform = ""
	_ = s.Compile()
	res := Validate(s, Extract(s, q("fbclid=abcdef")), Env{Path: "/", Now: time.Now()})
	if res.Channel != "click:fbclid" {
		t.Fatalf("channel: %s", res.Channel)
	}
}

func TestRefererIsAd(t *testing.T) {
	for ref, want := range map[string]bool{
		"https://l.facebook.com/l.php": true, "https://www.google.com/": true, "https://ads.tiktok.com/x": true,
		"https://example.com/": false, "not a url": false, "": false, "https://notfacebook.com": false,
	} {
		if RefererIsAd(ref) != want {
			t.Errorf("%s", ref)
		}
	}
}
