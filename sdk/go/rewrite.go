package cloak

import (
	"bytes"
	"strings"
)

// RewriteBodyHost 把响应体里的审核站域名替换成客户域名。
//
// 这是跨语言 SDK 的「深度代理」核心，与网关 internal/proxy/rewrite.go 的 rewriteBodyHost
// 行为一致，并以 sdk/testdata/rewrite-golden.json 做跨语言对拍基准。覆盖：
//   - 带协议前缀 https:// http:// 与协议相对 //
//   - JS/JSON 里转义斜杠 https:\/\/ \/\/
//   - 查询参数里的百分号编码 https%3A%2F%2F %2F%2F
//   - 正文里裸写的主机名（页脚、title、JSON 字段等）
//
// 带协议/编码前缀的形态把 http 升级为访客协议；裸主机名两侧都要求边界，避免误伤
// cdn.review.com 这类子域名或 review.comx 这类更长的主机名。
func RewriteBodyHost(body []byte, hosts []string, publicHost, scheme string) []byte {
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
		body = replaceHostFold(body, "https://"+oh, scheme+"://"+pub)
		body = replaceHostFold(body, "http://"+oh, scheme+"://"+pub)
		body = replaceHostFold(body, "//"+oh, "//"+pub)
		body = replaceHostFold(body, `https:\/\/`+oh, scheme+`:\/\/`+pub)
		body = replaceHostFold(body, `http:\/\/`+oh, scheme+`:\/\/`+pub)
		body = replaceHostFold(body, `\/\/`+oh, `\/\/`+pub)
		body = replaceHostFold(body, "https%3A%2F%2F"+oh, "https%3A%2F%2F"+pub)
		body = replaceHostFold(body, "http%3A%2F%2F"+oh, "https%3A%2F%2F"+pub)
		body = replaceHostFold(body, "%2F%2F"+oh, "%2F%2F"+pub)
		body = replaceBareHost(body, oh, pub)
	}
	return body
}

// replaceBareHost 替换正文里不带协议前缀的主机名。两侧都要求边界：
// 前面不能是字母/数字/点/连字符，后面不能是字母/数字/连字符（允许点号作句末/根标签）。
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

func stripPort(h string) string {
	if i := strings.LastIndexByte(h, ':'); i >= 0 && !strings.Contains(h[i:], "]") {
		return h[:i]
	}
	return h
}

// RewritableContentType 深度代理需要改写正文的类型：页面、样式、脚本，及带绝对地址的 JSON/XML。
func RewritableContentType(contentType string) bool {
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

// LooksLikeHTML 上游没给 Content-Type 时，按正文嗅探是否文本/HTML，决定要不要改写。
func LooksLikeHTML(body []byte) bool {
	probe := body
	if len(probe) > 1024 {
		probe = probe[:1024]
	}
	if len(probe) == 0 || bytes.IndexByte(probe, 0) >= 0 {
		return false
	}
	head := bytes.TrimLeft(probe, " \t\r\n")
	if len(head) == 0 {
		return false
	}
	lower := bytes.ToLower(head)
	return bytes.HasPrefix(lower, []byte("<!doctype")) ||
		bytes.HasPrefix(lower, []byte("<html")) ||
		bytes.HasPrefix(lower, []byte("<?xml")) ||
		head[0] == '<'
}
