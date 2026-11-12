// Package probe 做真实 DNS 校验和源站可达性检查。
package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CNAME 查询域名的 CNAME 链，返回最终别名（小写、无尾点）。
func CNAME(ctx context.Context, domain string) (string, error) {
	r := &net.Resolver{}
	name, err := r.LookupCNAME(ctx, domain)
	if err != nil {
		return "", err
	}
	return normalizeHost(name), nil
}

// TXT 查询域名的 TXT 记录。
func TXT(ctx context.Context, domain string) ([]string, error) {
	r := &net.Resolver{}
	return r.LookupTXT(ctx, domain)
}

// MatchCNAME 判断 CNAME 是否指向期望主机（允许多级别名以期望主机结尾）。
func MatchCNAME(actual, expect string) bool {
	actual, expect = normalizeHost(actual), normalizeHost(expect)
	if actual == "" || expect == "" {
		return false
	}
	return actual == expect || strings.HasSuffix(actual, "."+expect)
}

// ContainsTXT 是否包含期望值。
func ContainsTXT(records []string, expect string) bool {
	expect = strings.TrimSpace(expect)
	for _, rec := range records {
		if strings.TrimSpace(rec) == expect {
			return true
		}
	}
	return false
}

// NormalizeHost 从回源 Host 里取出主机名。允许误填成 https://host/。
func NormalizeHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			raw = u.Host
		}
	}
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		raw = raw[:i]
	}
	if h, _, err := net.SplitHostPort(raw); err == nil {
		raw = h
	}
	return strings.TrimSuffix(strings.TrimSpace(raw), ".")
}

// Origin 对源站发 HEAD，失败再 GET，2xx/3xx 视为健康。
// hostHeader 非空且是域名时，同时作为 HTTP Host 和 TLS 服务器名，以便沿用该名字上已有的证书。
func Origin(ctx context.Context, rawURL, hostHeader string, timeout time.Duration) (bool, string) {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	host := NormalizeHost(hostHeader)
	client := &http.Client{Timeout: timeout, Transport: RoundTripper(rawURL, host), CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return false, err.Error()
		}
		if host != "" {
			req.Host = host
		}
		req.Header.Set("User-Agent", "CloakHealth/2")
		resp, err := client.Do(req)
		if err != nil {
			if method == http.MethodGet {
				return false, err.Error()
			}
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true, fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		if method == http.MethodGet {
			return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}
	return false, "无响应"
}

// RoundTripper 回源 Host 与 URL 主机不同时，用回源 Host 校验证书。
func RoundTripper(rawURL, host string) http.RoundTripper {
	if host == "" || net.ParseIP(host) != nil {
		return http.DefaultTransport
	}
	u, err := url.Parse(rawURL)
	if err != nil || strings.EqualFold(u.Hostname(), host) {
		return http.DefaultTransport
	}
	return &http.Transport{
		TLSClientConfig: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}, //nolint:gosec
	}
}

func normalizeHost(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	return strings.TrimSuffix(h, ".")
}
