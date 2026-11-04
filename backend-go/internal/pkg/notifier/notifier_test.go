package notifier

import (
	"encoding/base64"
	"testing"
	"time"

	"cloak/internal/config"
)

func TestSignVerify(t *testing.T) {
	body := []byte(`{"event":"origin.unhealthy","data":{"siteId":1}}`)
	ts := time.Now().Unix()
	sig := Sign("whsec_test", ts, body)
	if !Verify("whsec_test", ts, body, sig, 5*time.Minute) {
		t.Fatal("verify should pass")
	}
	if Verify("whsec_other", ts, body, sig, 5*time.Minute) {
		t.Fatal("wrong secret should fail")
	}
	if Verify("whsec_test", ts, []byte(`{}`), sig, 5*time.Minute) {
		t.Fatal("tampered body should fail")
	}
	if Verify("whsec_test", ts-3600, body, Sign("whsec_test", ts-3600, body), 5*time.Minute) {
		t.Fatal("stale timestamp should fail")
	}
}

func TestValidateURL(t *testing.T) {
	w := NewWebhook(config.Webhook{})
	for _, bad := range []string{"http://example.com/h", "https://127.0.0.1/h", "https://10.0.0.1/h", "https://localhost/h", "ftp://x", ""} {
		if err := w.ValidateURL(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
	if err := w.ValidateURL("https://hooks.example.com/cloak"); err != nil {
		t.Errorf("public https rejected: %v", err)
	}
	dev := NewWebhook(config.Webhook{AllowHTTP: true})
	if err := dev.ValidateURL("http://127.0.0.1:9999/h"); err != nil {
		t.Errorf("allow_http should accept local: %v", err)
	}
}

func TestBackoffAndB64(t *testing.T) {
	if Backoff(1) != time.Minute || Backoff(5) != 6*time.Hour || Backoff(99) != 6*time.Hour {
		t.Fatal("backoff steps")
	}
	for _, s := range []string{"", "a", "ab", "abc", "告警：源站不可达"} {
		if b64(s) != base64.StdEncoding.EncodeToString([]byte(s)) {
			t.Fatalf("b64(%q)=%q", s, b64(s))
		}
	}
}
