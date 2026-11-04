package token

import (
	"errors"
	"strings"
	"testing"
	"time"

	"cloak/internal/engine/snapshot"
)

func site() *snapshot.Site {
	s := &snapshot.Site{SiteID: 5, Keys: []snapshot.TokenKey{{Version: 1, Key: "aabbcc", Status: "active"}}}
	_ = s.Compile()
	return s
}

func TestIssueVerify(t *testing.T) {
	s := site()
	now := time.Unix(1_762_000_000, 0)
	tok, err := Issue(s, Payload{VisitorID: "v1", AttributionID: "a1", IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(tok, "v1.") || len(tok) > 300 {
		t.Fatalf("format/length: %d %s", len(tok), tok)
	}
	p, err := Verify(s, tok, now)
	if err != nil || p.VisitorID != "v1" || p.KeyVersion != 1 || p.SiteID != 5 {
		t.Fatalf("verify: %v %+v", err, p)
	}
	if _, err := Verify(s, tok, now.Add(2*time.Hour)); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected expired, got %v", err)
	}
}

func TestVerifyErrors(t *testing.T) {
	s := site()
	now := time.Now()
	if _, err := Verify(s, "", now); !errors.Is(err, ErrMalformed) {
		t.Fatal("empty")
	}
	if _, err := Verify(s, "v2.a.b", now); !errors.Is(err, ErrMalformed) {
		t.Fatal("prefix")
	}
	if _, err := Verify(s, "v1.!!!.b", now); !errors.Is(err, ErrMalformed) {
		t.Fatal("base64")
	}
	if _, err := Verify(s, "v1.e30.b", now); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("unknown key version: %v", err)
	}
	tok, _ := Issue(s, Payload{VisitorID: "v", ExpiresAt: now.Add(time.Hour).Unix()})
	parts := strings.Split(tok, ".")
	if _, err := Verify(s, parts[0]+"."+parts[1]+".AAAA", now); !errors.Is(err, ErrSignature) {
		t.Fatalf("signature: %v", err)
	}
	empty := &snapshot.Site{SiteID: 1}
	_ = empty.Compile()
	if _, err := Issue(empty, Payload{}); !errors.Is(err, ErrNoKey) {
		t.Fatal("no key")
	}
}

func TestRotation(t *testing.T) {
	s := site()
	now := time.Now()
	tok, _ := Issue(s, Payload{VisitorID: "v", ExpiresAt: now.Add(time.Hour).Unix()})
	grace := now.Add(time.Hour)
	s.Keys = []snapshot.TokenKey{
		{Version: 1, Key: "aabbcc", Status: "grace", ExpiresAt: &grace},
		{Version: 2, Key: "ddeeff", Status: "active"},
	}
	_ = s.Compile()
	p, err := Verify(s, tok, now)
	if err != nil || !NeedsReissue(s, p) {
		t.Fatalf("grace verify/reissue: %v", err)
	}
	if _, err := Verify(s, tok, grace.Add(time.Second)); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("after grace: %v", err)
	}
	// 非 hex 密钥也可用
	s.Keys = []snapshot.TokenKey{{Version: 3, Key: "plain-text-key", Status: "active"}}
	_ = s.Compile()
	tok3, _ := Issue(s, Payload{VisitorID: "v", ExpiresAt: now.Add(time.Hour).Unix()})
	if _, err := Verify(s, tok3, now); err != nil {
		t.Fatal(err)
	}
}
