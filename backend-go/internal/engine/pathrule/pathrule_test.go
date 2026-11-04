package pathrule

import (
	"testing"

	"cloak/internal/engine/snapshot"
)

func TestMatch(t *testing.T) {
	s := &snapshot.Site{PathRules: []snapshot.PathRule{
		{Pattern: ".JS", MatchType: MatchSuffix, Action: "bypass", Sort: 1},
		{Pattern: "/exact", MatchType: MatchExact, Action: "block", Sort: 2},
		{Pattern: "/pre", MatchType: MatchPrefix, Action: "always_real", Sort: 3},
	}}
	_ = s.Compile()
	if r := Match(s, "/a/b.js"); r == nil || r.Action != "bypass" {
		t.Fatal("suffix case-insensitive")
	}
	if r := Match(s, "/exact"); r == nil || r.Action != "block" {
		t.Fatal("exact")
	}
	if r := Match(s, "/exact/x"); r != nil {
		t.Fatal("exact must not match sub path")
	}
	if r := Match(s, "/prefix/zzz"); r == nil || r.Action != "always_real" {
		t.Fatal("prefix")
	}
	if r := Match(s, "/none"); r != nil {
		t.Fatal("no match")
	}
	if r := Match(s, ""); r != nil {
		t.Fatal("empty path treated as /")
	}
}

func TestAllowed(t *testing.T) {
	if !Allowed("/x", nil) || !Allowed("/x", []string{}) {
		t.Fatal("empty allows all")
	}
	if !Allowed("/landing/a", []string{"/other", "/landing"}) || Allowed("/home", []string{"/landing"}) || Allowed("/x", []string{""}) {
		t.Fatal("prefix list")
	}
}
