// Package cloak 是斗篷（Cloak）服务端 SDK 的 Go 参考实现。
//
// 用法：在客户自己的服务器上，于网站入口调用判定接口，决定「放行」还是「审核」；
// 审核时在本机对审核站做深度代理（抓取 + 域名改写 + 输出），访客流量不经过斗篷网关。
//
//	c := cloak.New(cloak.Config{
//	    APIBase:      "https://gw.example.com",
//	    SecretKey:    "sk_xxx",
//	    ReviewOrigin: "https://review.com",
//	})
//	http.Handle("/", c.Handle(myRealHandler))
package cloak

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Config SDK 配置。
type Config struct {
	APIBase      string        // 斗篷网关地址，如 https://gw.example.com
	SecretKey    string        // 服务端密钥 sk_...，只放服务器
	ReviewOrigin string        // 审核站地址，如 https://review.com
	VisitorHost  string        // 客户对外域名；留空则用请求 Host
	ReviewHost   string        // 回源 Host（源站用 IP + 自定义 Host 时填），留空用 ReviewOrigin 的主机
	CookieName   string        // 访客 Cookie 名，默认 _ck_vid
	Timeout      time.Duration // 判定与回源超时，默认 3s
	FailOpen     bool          // true=判定失败放行真实内容（危险）；默认 false=按审核处理
	HTTPClient   *http.Client  // 自定义客户端，留空用内置
	SkipPaths    []string      // 跳过判定的路径前缀；默认见 defaultSkipPaths
	SkipSuffixes []string      // 跳过判定的后缀（静态资源）；默认见 defaultSkipSuffixes
	ClientIPHeaders []string   // 真实 IP 头，默认 CF-Connecting-IP / X-Real-IP / X-Forwarded-For
	UserAgent    string        // 回源时带的标识，默认 cloak-sdk-go
}

// Client 判定 + 深度代理客户端。
type Client struct {
	cfg Config
	hc  *http.Client
}

const decisionPath = "/api/open/v2/decision"

var (
	defaultSkipPaths    = []string{"/wp-admin", "/wp-login", "/wp-json", "/wp-cron", "/xmlrpc.php", "/_cloak", "/favicon.ico", "/robots.txt", "/sitemap"}
	defaultSkipSuffixes = []string{".css", ".js", ".mjs", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".map", ".mp4", ".webm", ".pdf", ".zip"}
	defaultIPHeaders    = []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"}
)

// New 创建客户端。
func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3 * time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "cloak-sdk-go/1.0"
	}
	if cfg.SkipPaths == nil {
		cfg.SkipPaths = defaultSkipPaths
	}
	if cfg.SkipSuffixes == nil {
		cfg.SkipSuffixes = defaultSkipSuffixes
	}
	if cfg.ClientIPHeaders == nil {
		cfg.ClientIPHeaders = defaultIPHeaders
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "_ck_vid"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{cfg: cfg, hc: hc}
}

// Cookie 判定返回的访客 Cookie。
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	MaxAge   int    `json:"maxAge"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"httpOnly"`
	SameSite string `json:"sameSite"`
}

// RenderHint 审核内容提示。type: html / url / static / none。
type RenderHint struct {
	Type string `json:"type"`
	HTML string `json:"html"`
	URL  string `json:"url"`
}

// Decision 判定结果。
type Decision struct {
	Content       string          `json:"content"` // real / bypass / review / block / not_found
	Attributed    bool            `json:"attributed"`
	Confidence    string          `json:"confidence"`
	Channel       string          `json:"channel"`
	VisitorToken  string          `json:"visitorToken"`
	Cookie        *Cookie         `json:"cookie"`
	HandoffURL    string          `json:"handoffUrl"`
	RenderHint    *RenderHint     `json:"renderHint"`
	HostKnown     bool            `json:"hostKnown"`
	raw           json.RawMessage
}

// Pass 是否放行真实内容。
func (d *Decision) Pass() bool { return d != nil && (d.Content == "real" || d.Content == "bypass") }

type decisionReq struct {
	URL            string `json:"url"`
	Method         string `json:"method"`
	VisitorToken   string `json:"visitorToken"`
	IP             string `json:"ip"`
	UserAgent      string `json:"userAgent"`
	Referer        string `json:"referer"`
	AcceptLanguage string `json:"acceptLanguage"`
}

type apiEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// Decide 调用判定接口。网络/解析失败返回 error（调用方按 FailOpen 决定如何兜底）。
func (c *Client) Decide(ctx context.Context, r *http.Request) (*Decision, error) {
	reqBody := decisionReq{
		URL:            c.visitorURL(r),
		Method:         r.Method,
		VisitorToken:   c.visitorToken(r),
		IP:             c.clientIP(r),
		UserAgent:      r.UserAgent(),
		Referer:        r.Referer(),
		AcceptLanguage: r.Header.Get("Accept-Language"),
	}
	body, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.cfg.APIBase, "/")+decisionPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range c.signHeaders(http.MethodPost, decisionPath, body) {
		httpReq.Header.Set(k, v)
	}
	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloak decision http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var env apiEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	var d Decision
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return nil, err
	}
	d.raw = env.Data
	if d.Content == "" {
		return nil, errors.New("cloak decision: empty content")
	}
	return &d, nil
}

// signHeaders v2 签名头：HMAC-SHA256(secret, ts\nnonce\nMETHOD\npath\nsha256hex(body))。
func (c *Client) signHeaders(method, path string, body []byte) map[string]string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randNonce()
	prefix := c.cfg.SecretKey
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	sum := sha256.Sum256(body)
	msg := ts + "\n" + nonce + "\n" + strings.ToUpper(method) + "\n" + path + "\n" + hex.EncodeToString(sum[:])
	mac := hmac.New(sha256.New, []byte(c.cfg.SecretKey))
	mac.Write([]byte(msg))
	return map[string]string{
		"X-Cloak-Key":       prefix,
		"X-Cloak-Timestamp": ts,
		"X-Cloak-Nonce":     nonce,
		"X-Cloak-Signature": hex.EncodeToString(mac.Sum(nil)),
	}
}

func randNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (c *Client) visitorHost(r *http.Request) string {
	if c.cfg.VisitorHost != "" {
		return c.cfg.VisitorHost
	}
	return r.Host
}

func (c *Client) visitorURL(r *http.Request) string {
	scheme := requestScheme(r)
	return scheme + "://" + c.visitorHost(r) + r.URL.RequestURI()
}

func (c *Client) visitorToken(r *http.Request) string {
	if ck, err := r.Cookie(c.cfg.CookieName); err == nil && ck.Value != "" {
		return ck.Value
	}
	return ""
}

func (c *Client) clientIP(r *http.Request) string {
	for _, h := range c.cfg.ClientIPHeaders {
		if v := r.Header.Get(h); v != "" {
			return strings.TrimSpace(strings.Split(v, ",")[0])
		}
	}
	if i := strings.LastIndexByte(r.RemoteAddr, ':'); i >= 0 {
		return r.RemoteAddr[:i]
	}
	return r.RemoteAddr
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		return strings.TrimSpace(strings.Split(p, ",")[0])
	}
	return "http"
}

func (c *Client) shouldSkip(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return true
	}
	p := strings.ToLower(r.URL.Path)
	for _, sp := range c.cfg.SkipPaths {
		if strings.HasPrefix(p, sp) {
			return true
		}
	}
	for _, sx := range c.cfg.SkipSuffixes {
		if strings.HasSuffix(p, sx) {
			return true
		}
	}
	return false
}
