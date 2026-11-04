package proxy

import (
	"net/http"
	"net/url"
	"strings"
)

// hopHeaders 逐跳头，转发时删除。
var hopHeaders = []string{
	"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// stripResponseHeaders 不向访客暴露的源站头。
var stripResponseHeaders = []string{"Server", "X-Powered-By", "Via", "X-AspNet-Version", "X-AspNetMvc-Version"}

// rewriteRequest 把访客请求改写为发往源站的请求。
func rewriteRequest(out *http.Request, in *http.Request, origin *url.URL, hostHeader string, clientIP, scheme string, meta map[string]string) {
	out.URL.Scheme = origin.Scheme
	out.URL.Host = origin.Host
	// 源站 URL 可带路径前缀
	if p := strings.TrimSuffix(origin.Path, "/"); p != "" {
		out.URL.Path = p + in.URL.Path
		if in.URL.RawPath != "" {
			out.URL.RawPath = p + in.URL.RawPath
		}
	}
	out.URL.RawQuery = in.URL.RawQuery
	out.Host = origin.Host
	if hostHeader != "" {
		out.Host = hostHeader
	}
	out.RequestURI = ""

	// 删逐跳头与客户端伪造的 X-Cloak-*
	for _, h := range hopHeaders {
		out.Header.Del(h)
	}
	for k := range out.Header {
		if strings.HasPrefix(strings.ToLower(k), "x-cloak-") {
			out.Header.Del(k)
		}
	}
	// 转发头
	if clientIP != "" {
		if prior := out.Header.Get("X-Forwarded-For"); prior != "" {
			out.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			out.Header.Set("X-Forwarded-For", clientIP)
		}
		out.Header.Set("X-Real-IP", clientIP)
	}
	out.Header.Set("X-Forwarded-Proto", scheme)
	out.Header.Set("X-Forwarded-Host", in.Host)
	for k, v := range meta {
		out.Header.Set(k, v)
	}
}

// rewriteResponseHeaders 改写源站响应头：删敏感头、Location 与 Set-Cookie Domain 换回商户域名。
func rewriteResponseHeaders(h http.Header, origin *url.URL, originHost, publicHost, scheme string) {
	for _, k := range stripResponseHeaders {
		h.Del(k)
	}
	for _, k := range hopHeaders {
		if k == "Transfer-Encoding" || k == "Upgrade" || k == "Connection" {
			continue // 由 Go 的 http 层处理
		}
		h.Del(k)
	}
	if loc := h.Get("Location"); loc != "" {
		h.Set("Location", rewriteLocation(loc, origin, originHost, publicHost, scheme))
	}
	if cookies := h.Values("Set-Cookie"); len(cookies) > 0 {
		h.Del("Set-Cookie")
		for _, c := range cookies {
			h.Add("Set-Cookie", rewriteCookieDomain(c, origin.Hostname(), originHost, publicHost))
		}
	}
}

// rewriteLocation Location 指向源站地址时换成访客看到的地址。
func rewriteLocation(loc string, origin *url.URL, originHost, publicHost, scheme string) string {
	u, err := url.Parse(loc)
	if err != nil || u.Host == "" {
		return loc
	}
	if hostEq(u.Host, origin.Host) || hostEq(u.Host, originHost) {
		u.Scheme = scheme
		u.Host = publicHost
		if p := strings.TrimSuffix(origin.Path, "/"); p != "" && strings.HasPrefix(u.Path, p) {
			u.Path = strings.TrimPrefix(u.Path, p)
			if u.Path == "" {
				u.Path = "/"
			}
		}
		return u.String()
	}
	return loc
}

// rewriteCookieDomain 源站 Set-Cookie 带 Domain=源站域名 时去掉 Domain（落到访客当前域名）。
func rewriteCookieDomain(c, originHostname, originHost, publicHost string) string {
	parts := strings.Split(c, ";")
	out := parts[:0]
	for _, p := range parts {
		kv := strings.TrimSpace(p)
		if strings.HasPrefix(strings.ToLower(kv), "domain=") {
			d := strings.TrimPrefix(strings.TrimPrefix(kv[7:], "."), "")
			pub := stripPort(publicHost)
			if hostEq(d, originHostname) || hostEq(d, stripPort(originHost)) || (!hostEq(d, pub) && !strings.HasSuffix(pub, "."+d)) {
				continue // 丢掉源站域名的 Domain
			}
		}
		out = append(out, p)
	}
	return strings.Join(out, ";")
}

func hostEq(a, b string) bool {
	return a != "" && b != "" && strings.EqualFold(stripPort(a), stripPort(b))
}

func stripPort(h string) string {
	if i := strings.LastIndexByte(h, ':'); i >= 0 && !strings.Contains(h[i:], "]") {
		return h[:i]
	}
	return h
}

// isWebSocket 判断是否升级请求。
func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}
