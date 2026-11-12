package proxy

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloak/internal/engine/snapshot"
)

func readFile(p string) ([]byte, error) { return os.ReadFile(p) }

// forwardWebSocket 处理 Upgrade 请求：与源站建连、转发握手、双向拷贝。
func (p *Proxy) forwardWebSocket(w http.ResponseWriter, r *http.Request, req Request, target *url.URL, origin *snapshot.Origin, kind string) forwardResult {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket not supported", http.StatusNotImplemented)
		return forwardResult{Result: Result{Kind: kind, Error: "no_hijack"}, headersSent: true}
	}
	addr := target.Host
	if _, _, err := net.SplitHostPort(addr); err != nil {
		if target.Scheme == "https" || target.Scheme == "wss" {
			addr += ":443"
		} else {
			addr += ":80"
		}
	}
	dialer := &net.Dialer{Timeout: p.cfg.DialTimeout}
	var backend net.Conn
	var err error
	if target.Scheme == "https" || target.Scheme == "wss" {
		name := target.Hostname()
		if sni := tlsServerName(name, origin.HostHeader); sni != "" {
			name = sni
		}
		backend, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: name, InsecureSkipVerify: p.cfg.InsecureOrigins, MinVersion: tls.VersionTLS12}) //nolint:gosec
	} else {
		backend, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return forwardResult{Result: Result{Kind: kind, Error: classify(err)}}
	}
	defer backend.Close()

	out := r.Clone(r.Context())
	out.Body = http.NoBody
	rewriteRequest(out, r, target, origin.HostHeader, req.ClientIP, req.Scheme, map[string]string{
		"X-Cloak-Content": req.Decision.Content, "X-Cloak-Request-Id": req.RequestID,
	})
	// 升级所需的逐跳头要保留
	out.Header.Set("Connection", "Upgrade")
	out.Header.Set("Upgrade", "websocket")
	if err := out.Write(backend); err != nil {
		return forwardResult{Result: Result{Kind: kind, Error: "origin_unreachable"}}
	}
	br := bufio.NewReader(backend)
	resp, err := http.ReadResponse(br, out)
	if err != nil {
		return forwardResult{Result: Result{Kind: kind, Error: "origin_unreachable"}}
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		// 源站拒绝升级：按普通响应转发
		defer resp.Body.Close()
		return p.writeResponse(w, r, req, resp, target, origin, kind, false, "")
	}

	client, clientBuf, err := hj.Hijack()
	if err != nil {
		return forwardResult{Result: Result{Kind: kind, Error: "hijack_failed"}, headersSent: true}
	}
	defer client.Close()
	_ = client.SetDeadline(time.Time{})
	_ = backend.SetDeadline(time.Time{})

	// 把 101 响应原样写回客户端
	if err := resp.Write(clientBuf); err != nil {
		return forwardResult{Result: Result{Kind: kind, OriginStatus: 101, Error: "write_upgrade"}, headersSent: true}
	}
	_ = clientBuf.Flush()

	errc := make(chan error, 2)
	go func() { _, e := io.Copy(backend, clientBuf); errc <- e }()
	go func() { _, e := io.Copy(client, br); errc <- e }()
	<-errc
	return forwardResult{Result: Result{Kind: kind, OriginStatus: 101}, headersSent: true}
}
