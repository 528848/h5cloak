package probe

import "testing"

func TestMatchCNAME(t *testing.T) {
	if !MatchCNAME("entry.cloak.local.", "entry.cloak.local") {
		t.Fatal("exact")
	}
	if !MatchCNAME("shop.entry.cloak.local", "entry.cloak.local") {
		t.Fatal("suffix")
	}
	if MatchCNAME("other.example.com", "entry.cloak.local") {
		t.Fatal("mismatch")
	}
}

func TestContainsTXT(t *testing.T) {
	if !ContainsTXT([]string{"v=spf1", "cloak-verify=abc"}, "cloak-verify=abc") {
		t.Fatal("expected hit")
	}
	if ContainsTXT([]string{"cloak-verify=abc"}, "cloak-verify=xyz") {
		t.Fatal("expected miss")
	}
}
