// Package proxy 托管反向代理：按判定结果回源真实站/审核站，或直接响应模板与静态包。
package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloak/internal/config"
)

// transports 每个源站一个 Transport，按 (超时) 复用。
type transports struct {
	mu  sync.Mutex
	cfg config.Proxy
	m   map[time.Duration]*http.Transport
}

func newTransports(cfg config.Proxy) *transports {
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = 90 * time.Second
	}
	if cfg.MaxIdlePerHost <= 0 {
		cfg.MaxIdlePerHost = 64
	}
	return &transports{cfg: cfg, m: map[time.Duration]*http.Transport{}}
}

// roundTrip 回源。serverName 非空时用它校验源站证书，连接仍然打到请求 URL 的地址。
func (t *transports) roundTrip(req *http.Request, timeout time.Duration, serverName string) (*http.Response, error) {
	tr := t.get(timeout)
	if serverName != "" && !strings.EqualFold(serverName, req.URL.Hostname()) {
		tr = tr.Clone()
		base := tr.TLSClientConfig
		if base == nil {
			base = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		cfg := base.Clone()
		cfg.ServerName = serverName
		tr.TLSClientConfig = cfg
	}
	return tr.RoundTrip(req)
}

// get 返回响应头超时为 timeout 的 Transport。
func (t *transports) get(timeout time.Duration) *http.Transport {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if tr, ok := t.m[timeout]; ok {
		return tr
	}
	tr := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   t.cfg.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false, // 对源站不开 HTTP/2，行为更可预期
		MaxIdleConns:          t.cfg.MaxIdlePerHost * 4,
		MaxIdleConnsPerHost:   t.cfg.MaxIdlePerHost,
		IdleConnTimeout:       t.cfg.IdleConnTimeout,
		ResponseHeaderTimeout: timeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: t.cfg.InsecureOrigins, MinVersion: tls.VersionTLS12}, //nolint:gosec
		// 源站响应体不解压，由浏览器处理；需要改写 HTML 时请求 identity
		DisableCompression: true,
	}
	t.m[timeout] = tr
	return tr
}

// closeIdle 释放空闲连接（停机时）。
func (t *transports) closeIdle() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, tr := range t.m {
		tr.CloseIdleConnections()
	}
}
