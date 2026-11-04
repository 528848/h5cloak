package snapshot

import (
	"encoding/json"
	"testing"
)

func TestCompileAndLookups(t *testing.T) {
	s := &Site{
		SiteID: 1, Status: "enabled", MerchantActive: true,
		Domains:     []Domain{{Domain: "Shop.Example.com"}},
		Origins:     []Origin{{Kind: "real", URL: "https://o"}},
		PathRules:   []PathRule{{Pattern: "/b", Sort: 20}, {Pattern: "/a", Sort: 10}},
		Keys:        []TokenKey{{Version: 1, Key: "k1", Status: "grace"}, {Version: 2, Key: "k2", Status: "active"}},
		Params:      []Param{{Name: "Ref", Aliases: []string{"R", ""}, Pattern: "^[a-z]+$"}},
		Rule:        Rule{AllowedSources: []string{"FaceBook"}},
		Codes:       []Code{{Code: "AbC"}},
		Attribution: Attribution{MinConfidence: "high"},
	}
	if err := s.Compile(); err != nil {
		t.Fatal(err)
	}
	if s.LookupParam("REF") == nil || s.LookupParam("r") == nil || s.LookupParam("x") != nil {
		t.Fatal("param lookup")
	}
	if s.LookupParam("ref").Re == nil {
		t.Fatal("regex not compiled")
	}
	if s.LookupCode("AbC") == nil || s.LookupCode("abc") == nil || s.LookupCode("zzz") != nil {
		t.Fatal("code lookup")
	}
	if !s.SourceAllowed("facebook") || s.SourceAllowed("tiktok") {
		t.Fatal("source allowed")
	}
	if s.ActiveKey().Version != 2 || s.KeyByVersion(1) == nil || s.KeyByVersion(9) != nil {
		t.Fatal("keys")
	}
	if r := s.SortedRules(); r[0].Pattern != "/a" || r[1].Pattern != "/b" {
		t.Fatal("rules sort")
	}
	if s.Origin("real") == nil || s.Origin("review") != nil {
		t.Fatal("origin")
	}
	if !s.HasDomain("shop.example.com:443") || s.HasDomain("other.com") {
		t.Fatal("domain")
	}
	if !s.Serving() || s.MinConfidenceLevel() != 3 {
		t.Fatal("serving/min confidence")
	}
	s.MerchantActive = false
	if s.Serving() {
		t.Fatal("merchant inactive should not serve")
	}
}

func TestCompileErrorsAndDefaults(t *testing.T) {
	s := &Site{Params: []Param{{Name: "bad", Pattern: "["}}}
	if err := s.Compile(); err == nil {
		t.Fatal("expected regex error")
	}
	empty := &Site{Rule: Rule{}}
	if !empty.SourceAllowed("anything") {
		t.Fatal("empty allow list should allow all")
	}
	if empty.ActiveKey() != nil {
		t.Fatal("no keys")
	}
	noActive := &Site{Keys: []TokenKey{{Version: 1, Key: "a", Status: "grace"}}}
	if noActive.ActiveKey() == nil || noActive.ActiveKey().Version != 1 {
		t.Fatal("fallback to last key")
	}
	for in, want := range map[string]int{"high": 3, "medium": 2, "low": 1, "": 2, "weird": 2} {
		if ConfidenceLevel(in) != want {
			t.Errorf("level %s", in)
		}
	}
	for in, want := range map[int]string{3: "high", 4: "high", 2: "medium", 1: "low", 0: "low"} {
		if ConfidenceName(in) != want {
			t.Errorf("name %d", in)
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	s := &Site{SiteID: 9, Params: []Param{{Name: "ref", Pattern: "^x$"}}, Codes: []Code{{Code: "A"}}}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back Site
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.SiteID != 9 || back.LookupParam("ref") == nil || back.LookupCode("A") == nil {
		t.Fatal("round trip lost data")
	}
}
