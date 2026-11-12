package proxy

import (
	"bytes"
	"net"
	"net/http"
	"net/url"
	"strings"

	"cloak/internal/engine/snapshot"
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
	if h := normalizeHostHeader(hostHeader); h != "" {
		out.Host = h
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

// normalizeHostHeader 回源 Host 只保留主机名。后台误填 https://iris6666.com/ 时也能用。
func normalizeHostHeader(raw string) string {
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
	return strings.TrimSuffix(stripPort(strings.TrimSpace(raw)), ".")
}

// tlsServerName 回源 Host 是另一个域名时，用它做 TLS 握手的服务器名。
// 这样源站地址可以是原服务器 IP，证书仍用客户主域名上已有的那张。
func tlsServerName(urlHost, hostHeader string) string {
	name := normalizeHostHeader(hostHeader)
	if name == "" || net.ParseIP(name) != nil {
		return ""
	}
	if strings.EqualFold(name, stripPort(urlHost)) {
		return ""
	}
	return name
}

// originHosts 源站在响应体里可能出现的主机名：源站 URL 的主机 + 覆盖的回源 Host。
func originHosts(target *url.URL, origin *snapshot.Origin) []string {
	hosts := []string{}
	if h := stripPort(target.Hostname()); h != "" {
		hosts = append(hosts, h)
	}
	if origin != nil {
		if h := normalizeHostHeader(origin.HostHeader); h != "" {
			hosts = append(hosts, h)
		}
	}
	return hosts
}

// needHostRewrite 代理审核 / 真实内容时，源站主机与访客域名不同就需要深度改写，
// 让页面内的链接、资源、表单都落到访客自己的域名上。
func needHostRewrite(kind string, target *url.URL, origin *snapshot.Origin, publicHost string) bool {
	if kind != "review" && kind != "real" {
		return false
	}
	pub := stripPort(publicHost)
	if pub == "" {
		return false
	}
	for _, h := range originHosts(target, origin) {
		if !strings.EqualFold(h, pub) {
			return true
		}
	}
	return false
}

// rewriteBodyHost 把响应体里的源站域名替换成访客域名。覆盖普通 URL、协议相对 URL、
// JSON / JS 里转义斜杠的写法、百分号编码的地址，以及正文里裸写的主机名（页脚、标题、
// JSON 字段等）。带协议前缀的形式会把 http 升级成访客协议；裸主机名两侧都要求边界，
// 避免误伤 cdn.toptime.cc 这类子域名或 toptime.ccx 这类更长的主机名。
func rewriteBodyHost(body []byte, hosts []string, publicHost, scheme string) []byte {
	pub := stripPort(publicHost)
	if pub == "" || len(body) == 0 {
		return body
	}
	if scheme == "" {
		scheme = "https"
	}
	for _, oh := range hosts {
		oh = stripPort(oh)
		if oh == "" || strings.EqualFold(oh, pub) {
			continue
		}
		// 带协议前缀：前缀锚定开头，只校验结尾边界；http 升级成访客协议
		body = replaceHostFold(body, "https://"+oh, scheme+"://"+pub)
		body = replaceHostFold(body, "http://"+oh, scheme+"://"+pub)
		body = replaceHostFold(body, "//"+oh, "//"+pub)
		body = replaceHostFold(body, `https:\/\/`+oh, scheme+`:\/\/`+pub)
		body = replaceHostFold(body, `http:\/\/`+oh, scheme+`:\/\/`+pub)
		body = replaceHostFold(body, `\/\/`+oh, `\/\/`+pub)
		// 百分号编码（出现在 ?url= 等查询参数里）
		body = replaceHostFold(body, "https%3A%2F%2F"+oh, "https%3A%2F%2F"+pub)
		body = replaceHostFold(body, "http%3A%2F%2F"+oh, "https%3A%2F%2F"+pub)
		body = replaceHostFold(body, "%2F%2F"+oh, "%2F%2F"+pub)
		// 正文里裸写的主机名：页脚版权、title 属性、JSON 字段等
		body = replaceBareHost(body, oh, pub)
	}
	return body
}

// replaceBareHost 替换正文里不带协议前缀的主机名。两侧都要求边界：
// 前面不能是字母/数字/点/连字符（挡住子域名和编码残留），后面不能是字母/数字/连字符
// （挡住更长的主机名，但允许点号作为句末或根标签）。
func replaceBareHost(body []byte, oh, pub string) []byte {
	fb := []byte(oh)
	if len(fb) == 0 || len(body) < len(fb) {
		return body
	}
	tb := []byte(pub)
	out := make([]byte, 0, len(body))
	i := 0
	for i < len(body) {
		j := indexFold(body[i:], fb)
		if j < 0 {
			out = append(out, body[i:]...)
			break
		}
		j += i
		end := j + len(fb)
		if !bareStartOK(body, j) || !bareEndOK(body, end) {
			out = append(out, body[i:end]...)
			i = end
			continue
		}
		out = append(out, body[i:j]...)
		out = append(out, tb...)
		i = end
	}
	return out
}

func bareStartOK(b []byte, i int) bool {
	if i == 0 {
		return true
	}
	c := b[i-1]
	return !isHostByte(c) && c != '.'
}

func bareEndOK(b []byte, i int) bool {
	if i >= len(b) {
		return true
	}
	return !isHostByte(b[i])
}

// isHostByte 主机名内部允许的字符（字母、数字、连字符）。
func isHostByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-'
}

// replaceHostFold 大小写不敏感地替换主机片段，且仅在主机名结尾是边界时替换。
func replaceHostFold(body []byte, from, to string) []byte {
	fb := []byte(from)
	if len(fb) == 0 || len(body) < len(fb) {
		return body
	}
	tb := []byte(to)
	out := make([]byte, 0, len(body))
	i := 0
	for i < len(body) {
		j := indexFold(body[i:], fb)
		if j < 0 {
			out = append(out, body[i:]...)
			break
		}
		j += i
		end := j + len(fb)
		if !hostBoundary(body, end) {
			out = append(out, body[i:end]...)
			i = end
			continue
		}
		out = append(out, body[i:j]...)
		out = append(out, tb...)
		i = end
	}
	return out
}

func indexFold(s, sep []byte) int {
	if len(sep) == 0 || len(s) < len(sep) {
		return -1
	}
	last := len(s) - len(sep)
	for i := 0; i <= last; i++ {
		if bytes.EqualFold(s[i:i+len(sep)], sep) {
			return i
		}
	}
	return -1
}

// hostBoundary 主机名后面应是路径、端口、引号或空白，不能是更长主机名的一部分。
func hostBoundary(b []byte, i int) bool {
	if i >= len(b) {
		return true
	}
	switch b[i] {
	case '/', '?', '#', ':', '"', '\'', '`', ' ', '\t', '\n', '\r', '<', '>', '(', ')', ';', ',', '&', '\\', ']', '%', '=':
		return true
	default:
		return false
	}
}

// rewritableText 深度代理需要改写正文的类型：页面、样式、脚本，以及接口里带绝对地址的 JSON / XML。
func rewritableText(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch ct {
	case "text/html", "application/xhtml+xml",
		"text/css",
		"text/javascript", "application/javascript", "application/x-javascript",
		"text/ecmascript", "application/ecmascript",
		"application/json", "text/json", "application/ld+json", "application/manifest+json",
		"text/xml", "application/xml", "application/rss+xml", "application/atom+xml",
		"image/svg+xml":
		return true
	default:
		return strings.HasSuffix(ct, "+json") || strings.HasSuffix(ct, "+xml")
	}
}

// isWebSocket 判断是否升级请求。
func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}
