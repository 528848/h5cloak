package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
)

var (
	bodyClose   = []byte("</body>")
	hrefAttrRe  = regexp.MustCompile(`(?i)(<a\b[^>]*?\bhref\s*=\s*)(["'])([^"']*)(["'])`)
	formActRe   = regexp.MustCompile(`(?i)(<form\b[^>]*?\baction\s*=\s*)(["'])([^"']*)(["'])`)
	bannerShown = "_ck_hb"
)

// handoffURL 当前 URL 追加 _ck。
func handoffURL(current *url.URL, token string) string {
	u := *current
	q := u.Query()
	q.Set(engine.HandoffParam, token)
	u.RawQuery = q.Encode()
	return u.String()
}

// openInBrowserHref 按 OS 生成唤起系统浏览器的链接；无法识别时用普通 https 链接。
func openInBrowserHref(target string, osName string) string {
	switch osName {
	case "ios":
		// x-safari-https:// 在 iOS 17+ 的多数内置浏览器可唤起 Safari
		if strings.HasPrefix(target, "https://") {
			return "x-safari-" + target
		}
	case "android":
		u, err := url.Parse(target)
		if err == nil && u.Host != "" && (u.Scheme == "https" || u.Scheme == "http") {
			// BROWSABLE 交给系统解析默认浏览器，不写死 Chrome。
			return "intent://" + u.Host + u.RequestURI() + "#Intent;scheme=" + u.Scheme +
				";action=android.intent.action.VIEW;category=android.intent.category.BROWSABLE;S.browser_fallback_url=" +
				url.QueryEscape(target) + ";end"
		}
	}
	return target
}

// BannerDismissedCookie 访客关闭引导条后写的会话 cookie。
var BannerDismissedCookie = bannerShown

// injectHTML 在 </body> 前插入引导配置和 JS SDK，并可对同域链接追加 _ck。
// 弹框由 sdk.js 按延迟绘制。showBanner=false 时只做链接延续。
func injectHTML(body []byte, site *snapshot.Site, current *url.URL, token, osName string, rewriteLinks, showBanner bool) []byte {
	if rewriteLinks && token != "" {
		body = appendTokenToLinks(body, current, token)
	}
	if showBanner && site != nil && site.Handoff.Banner {
		if snippet := handoffScript(site, current, token, osName); len(snippet) > 0 {
			if i := bytes.LastIndex(bytes.ToLower(body), bodyClose); i >= 0 {
				out := make([]byte, 0, len(body)+len(snippet))
				out = append(out, body[:i]...)
				out = append(out, snippet...)
				out = append(out, body[i:]...)
				return out
			}
			return append(body, snippet...)
		}
	}
	return body
}

// handoffScript 把已经填好默认值的引导配置交给页面上的 JS SDK。
func handoffScript(site *snapshot.Site, current *url.URL, token, osName string) []byte {
	if current == nil {
		return nil
	}
	target := current.String()
	if token != "" {
		target = handoffURL(current, token)
	}
	p := site.Handoff.EffectivePrompt()
	raw, err := json.Marshal(map[string]any{
		"url": target, "os": osName, "enabled": true,
		"delaySeconds": p.DelaySeconds, "cooldownMinutes": p.CooldownMinutes,
		"maxShows": p.MaxShows, "dismissEnds": p.DismissEnds,
		"mobile": p.Mobile, "desktop": p.Desktop,
	})
	if err != nil {
		return nil
	}
	return append(append([]byte(`<script>window.__CLOAK_HANDOFF__=`), raw...), []byte(`;</script><script src="/_cloak/sdk.js" data-banner="preset" async></script>`)...)
}

// appendTokenToLinks 对 <a href> 与 <form action> 的同域链接追加 _ck。
func appendTokenToLinks(body []byte, current *url.URL, token string) []byte {
	fix := func(m []byte, re *regexp.Regexp) []byte {
		sub := re.FindSubmatch(m)
		if len(sub) != 5 {
			return m
		}
		raw := string(sub[3])
		if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "javascript:") || strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "tel:") {
			return m
		}
		u, err := url.Parse(html.UnescapeString(raw))
		if err != nil {
			return m
		}
		if u.Host != "" && !strings.EqualFold(stripPort(u.Host), stripPort(current.Host)) {
			return m // 外域不处理
		}
		q := u.Query()
		if q.Get(engine.HandoffParam) != "" {
			return m
		}
		q.Set(engine.HandoffParam, token)
		u.RawQuery = q.Encode()
		return append(append(append(append([]byte{}, sub[1]...), sub[2]...), html.EscapeString(u.String())...), sub[4]...)
	}
	body = hrefAttrRe.ReplaceAllFunc(body, func(m []byte) []byte { return fix(m, hrefAttrRe) })
	body = formActRe.ReplaceAllFunc(body, func(m []byte) []byte { return fix(m, formActRe) })
	return body
}

// bannerDismissed 用户关闭过引导条（会话 cookie）。
func bannerDismissed(r *http.Request) bool {
	c, err := r.Cookie(bannerShown)
	return err == nil && c.Value != ""
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip")
}

func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// isHTML 判断响应是否 HTML 文档。
func isHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "text/html") || strings.HasPrefix(ct, "application/xhtml+xml")
}
