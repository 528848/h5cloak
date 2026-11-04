package tlsconf

import (
	"context"
	"crypto/tls"
	"testing"

	"go.uber.org/zap"

	"cloak/internal/config"
)

func TestOffAndUnknown(t *testing.T) {
	p, err := New(config.TLS{Mode: "off"}, nil, zap.NewNop())
	if err != nil || p != nil {
		t.Fatal("off should return nil")
	}
	if _, err := New(config.TLS{Mode: "weird"}, nil, zap.NewNop()); err == nil {
		t.Fatal("unknown mode")
	}
	if _, err := New(config.TLS{Mode: "acme"}, nil, zap.NewNop()); err == nil {
		t.Fatal("acme needs email")
	}
}

func TestSelfSigned(t *testing.T) {
	dir := t.TempDir()
	allowed := map[string]bool{"shop.test": true}
	allow := func(_ context.Context, h string) bool { return allowed[h] }
	p, err := New(config.TLS{Mode: "selfsigned", CertDir: dir}, allow, zap.NewNop())
	if err != nil || p == nil || p.Mode() != "selfsigned" {
		t.Fatal(err)
	}
	cfg := p.TLSConfig()
	cert, err := cfg.GetCertificate(&tls.ClientHelloInfo{ServerName: "shop.test"})
	if err != nil || cert == nil || len(cert.Certificate) != 2 {
		t.Fatalf("issue: %v", err)
	}
	// 缓存命中
	cert2, _ := cfg.GetCertificate(&tls.ClientHelloInfo{ServerName: "shop.test"})
	if cert2 != cert {
		t.Fatal("should cache")
	}
	if _, err := cfg.GetCertificate(&tls.ClientHelloInfo{ServerName: "evil.test"}); err == nil {
		t.Fatal("unbound host must be refused")
	}
	if _, err := cfg.GetCertificate(&tls.ClientHelloInfo{ServerName: ""}); err != nil {
		t.Fatal("localhost allowed without SNI")
	}
	// CA 复用：再建一个 Provider 读到同一 CA
	p2, err := New(config.TLS{Mode: "selfsigned", CertDir: dir}, allow, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	if string(p2.self.ca.Raw) != string(p.self.ca.Raw) {
		t.Fatal("CA should be persisted and reused")
	}
	if h := p.HTTPChallengeHandler(nil); h != nil {
		t.Fatal("selfsigned has no challenge handler")
	}
}
