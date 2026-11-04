package globalcfg

import "testing"

func TestApplyEmergencyAndEntry(t *testing.T) {
	c := Defaults()
	if next, reason := Apply(c, "real", "hosted"); next != "real" || reason != "" {
		t.Fatalf("default should keep real, got %s %s", next, reason)
	}
	c.EmergencyReview = true
	if next, reason := Apply(c, "real", "api_v2"); next != "review" || reason != "emergency_review" {
		t.Fatalf("emergency: %s %s", next, reason)
	}
	for _, content := range []string{"review", "block", "bypass", "not_found", "maintenance"} {
		if next, reason := Apply(c, content, "hosted"); next != content || reason != "" {
			t.Fatalf("emergency must leave %s, got %s %s", content, next, reason)
		}
	}
	c.EmergencyReview = false
	c.Entries["js"] = false
	if next, reason := Apply(c, "real", "js"); next != "review" || reason != "entry_off" {
		t.Fatalf("entry off: %s %s", next, reason)
	}
	if next, _ := Apply(c, "bypass", "js"); next != "bypass" {
		t.Fatalf("bypass stays, got %s", next)
	}
	if next, _ := Apply(c, "real", "hosted"); next != "real" {
		t.Fatalf("other entry stays real, got %s", next)
	}
	if next, reason := Apply(nil, "real", "hosted"); next != "real" || reason != "" {
		t.Fatalf("nil config: %s %s", next, reason)
	}
}

func TestOverlayClearsToken(t *testing.T) {
	c := Defaults()
	c.EmergencyReview = true
	content, token, trace := Overlay(c, "real", "vt", "hosted", nil)
	if content != "review" || token != "" || len(trace) != 1 || trace[0].Result != "emergency_review" {
		t.Fatalf("overlay %+v token %q", trace, token)
	}
}
