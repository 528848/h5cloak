// Package tlsconf 为网关提供 HTTPS 证书：本地自签或 ACME 按需签发（certmagic）。
// 两种模式都只对快照中已绑定的托管域名放行，防止被滥用签发。
package tlsconf

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	"go.uber.org/zap"

	"cloak/internal/config"
)

// HostChecker 判断域名是否允许签发/服务证书。
type HostChecker func(ctx context.Context, host string) bool

// Provider 产出 tls.Config，并暴露 ACME HTTP-01 挑战的 handler。
type Provider struct {
	cfg   config.TLS
	log   *zap.Logger
	allow HostChecker
	magic *certmagic.Config
	self  *selfSigned
}

// New 创建 Provider。mode=off 返回 nil。
func New(cfg config.TLS, allow HostChecker, log *zap.Logger) (*Provider, error) {
	switch strings.ToLower(cfg.Mode) {
	case "", "off":
		return nil, nil
	case "selfsigned":
		ss, err := newSelfSigned(filepath.Join(cfg.CertDir, "selfsigned"))
		if err != nil {
			return nil, err
		}
		return &Provider{cfg: cfg, log: log, allow: allow, self: ss}, nil
	case "acme":
		if cfg.Email == "" {
			return nil, errors.New("tls.email is required for acme mode")
		}
		p := &Provider{cfg: cfg, log: log, allow: allow}
		p.magic = p.newMagic()
		return p, nil
	default:
		return nil, errors.New("unknown tls.mode: " + cfg.Mode)
	}
}

func (p *Provider) newMagic() *certmagic.Config {
	certmagic.Default.Storage = &certmagic.FileStorage{Path: filepath.Join(p.cfg.CertDir, "acme")}
	certmagic.Default.OnDemand = &certmagic.OnDemandConfig{
		DecisionFunc: func(ctx context.Context, name string) error {
			if p.allow != nil && !p.allow(ctx, name) {
				return errors.New("host not bound to any site: " + name)
			}
			return nil
		},
	}
	certmagic.DefaultACME.Email = p.cfg.Email
	certmagic.DefaultACME.Agreed = true
	if p.cfg.Staging {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	} else {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	}
	// 只用 HTTP-01 与 TLS-ALPN-01，DNS 挑战不在本阶段范围
	certmagic.DefaultACME.DisableHTTPChallenge = false
	certmagic.DefaultACME.DisableTLSALPNChallenge = false
	return certmagic.NewDefault()
}

// TLSConfig 返回服务端 tls.Config。
func (p *Provider) TLSConfig() *tls.Config {
	if p.magic != nil {
		c := p.magic.TLSConfig()
		c.MinVersion = tls.VersionTLS12
		c.NextProtos = append([]string{"h2", "http/1.1"}, c.NextProtos...)
		return c
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			host := strings.ToLower(hello.ServerName)
			if host == "" {
				host = "localhost"
			}
			if p.allow != nil && host != "localhost" && !p.allow(hello.Context(), host) {
				return nil, errors.New("host not bound: " + host)
			}
			return p.self.get(host)
		},
	}
}

// HTTPChallengeHandler 包装 HTTP 处理器，拦截 ACME HTTP-01 挑战路径。非 ACME 模式原样返回。
func (p *Provider) HTTPChallengeHandler(next httpHandler) httpHandler {
	if p.magic == nil {
		return next
	}
	return p.magic.Issuers[0].(*certmagic.ACMEIssuer).HTTPChallengeHandler(next)
}

// Mode 当前模式。
func (p *Provider) Mode() string { return strings.ToLower(p.cfg.Mode) }

// ---- 自签 ----

type selfSigned struct {
	dir   string
	mu    sync.Mutex
	cache map[string]*tls.Certificate
	ca    *x509.Certificate
	caKey *ecdsa.PrivateKey
}

func newSelfSigned(dir string) (*selfSigned, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &selfSigned{dir: dir, cache: map[string]*tls.Certificate{}}
	if err := s.loadOrCreateCA(); err != nil {
		return nil, err
	}
	return s, nil
}

// loadOrCreateCA 本地 CA，首次生成后写入 cert_dir/selfsigned/ca.{crt,key}，可导入系统信任用于浏览器验证。
func (s *selfSigned) loadOrCreateCA() error {
	certPath := filepath.Join(s.dir, "ca.crt")
	keyPath := filepath.Join(s.dir, "ca.key")
	if cb, err := os.ReadFile(certPath); err == nil {
		if kb, err := os.ReadFile(keyPath); err == nil {
			cblock, _ := pem.Decode(cb)
			kblock, _ := pem.Decode(kb)
			if cblock != nil && kblock != nil {
				ca, err1 := x509.ParseCertificate(cblock.Bytes)
				key, err2 := x509.ParseECPrivateKey(kblock.Bytes)
				if err1 == nil && err2 == nil {
					s.ca, s.caKey = ca, key
					return nil
				}
			}
		}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	tpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "Cloak Local CA", Organization: []string{"Cloak Dev"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(5, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	ca, _ := x509.ParseCertificate(der)
	kder, _ := x509.MarshalECPrivateKey(key)
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder}), 0o600); err != nil {
		return err
	}
	s.ca, s.caKey = ca, key
	return nil
}

func (s *selfSigned) get(host string) (*tls.Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.cache[host]; ok {
		return c, nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{host},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, s.ca, &key.PublicKey, s.caKey)
	if err != nil {
		return nil, err
	}
	cert := &tls.Certificate{Certificate: [][]byte{der, s.ca.Raw}, PrivateKey: key}
	s.cache[host] = cert
	return cert, nil
}
