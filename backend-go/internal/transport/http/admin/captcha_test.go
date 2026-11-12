package admin

import (
	"context"
	"testing"
)

func TestCaptchaIssueAndVerify(t *testing.T) {
	s := newCaptchaStore(nil)
	uuid, img, err := s.Issue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if uuid == "" || img == "" {
		t.Fatal("empty captcha")
	}
	code, ok := s.take(context.Background(), uuid)
	if !ok || len(code) != 4 {
		t.Fatalf("stored code %q ok=%v", code, ok)
	}
	if _, ok := s.take(context.Background(), uuid); ok {
		t.Fatal("captcha should be single use")
	}

	if err := s.save(context.Background(), "fixed", "4821"); err != nil {
		t.Fatal(err)
	}
	if msg := s.Verify(context.Background(), "fixed", "4821"); msg != "" {
		t.Fatal(msg)
	}
	if msg := s.Verify(context.Background(), "fixed", "4821"); msg == "" {
		t.Fatal("captcha reused")
	}
	if s.Verify(context.Background(), "", "") == "" {
		t.Fatal("empty captcha accepted")
	}
}
