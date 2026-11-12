// Package notifier 邮件、Telegram、Webhook 的发送实现。
package notifier

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cloak/internal/config"
)

// ErrNotConfigured 渠道未配置。
var ErrNotConfigured = errors.New("not_configured")

// Mailer SMTP 发信。
type Mailer struct{ cfg config.SMTP }

func NewMailer(cfg config.SMTP) *Mailer { return &Mailer{cfg: cfg} }

func (m *Mailer) Configured() bool { return m.cfg.Host != "" && m.cfg.From != "" }

// Send 发送纯文本 + HTML 邮件。
func (m *Mailer) Send(ctx context.Context, to []string, subject, text, html string) error {
	if !m.Configured() {
		return ErrNotConfigured
	}
	if len(to) == 0 {
		return errors.New("no_recipients")
	}
	boundary := "cloak-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: =?UTF-8?B?%s?=\r\nMIME-Version: 1.0\r\n", m.cfg.From, strings.Join(to, ", "), b64(subject))
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)
	fmt.Fprintf(&b, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", boundary, text)
	if html != "" {
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n", boundary, html)
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)

	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	var c *smtp.Client
	if m.cfg.Port == 465 {
		tc := tls.Client(conn, &tls.Config{ServerName: m.cfg.Host})
		c, err = smtp.NewClient(tc, m.cfg.Host)
	} else {
		c, err = smtp.NewClient(conn, m.cfg.Host)
	}
	if err != nil {
		return err
	}
	defer c.Close()
	if m.cfg.StartTLS && m.cfg.Port != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil {
				return err
			}
		}
	}
	if m.cfg.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)); err != nil {
			return err
		}
	}
	if err := c.Mail(fromAddr(m.cfg.From)); err != nil {
		return err
	}
	for _, t := range to {
		if err := c.Rcpt(t); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(b.Bytes()); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// Telegram Bot 发消息。apiBase + token 决定用哪个机器人。
type Telegram struct {
	apiBase string
	token   string
	client  *http.Client
}

const defaultTelegramAPI = "https://api.telegram.org"

// NewTelegram 用全局配置（平台共用 Bot）构造。
func NewTelegram(cfg config.Telegram) *Telegram {
	return NewTelegramWithToken(cfg.APIBase, cfg.BotToken)
}

// NewTelegramWithToken 用指定 Token 构造（每商户自有 Bot）。
func NewTelegramWithToken(apiBase, token string) *Telegram {
	if apiBase == "" {
		apiBase = defaultTelegramAPI
	}
	return &Telegram{apiBase: strings.TrimRight(apiBase, "/"), token: token, client: &http.Client{Timeout: 60 * time.Second}}
}

func (t *Telegram) Configured() bool { return t.token != "" }

func (t *Telegram) method(name string) string {
	return t.apiBase + "/bot" + t.token + "/" + name
}

// Send 发送 HTML 格式消息。
func (t *Telegram) Send(ctx context.Context, chatID, html string) error {
	if !t.Configured() {
		return ErrNotConfigured
	}
	if chatID == "" {
		return errors.New("no_chat_id")
	}
	body, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": html, "parse_mode": "HTML", "disable_web_page_preview": true})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, t.method("sendMessage"), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("telegram %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// TelegramUser getMe 返回的机器人账号。
type TelegramUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"first_name"`
}

// GetMe 校验 Token，返回机器人账号信息。401 等非 200 返回错误。
func (t *Telegram) GetMe(ctx context.Context) (*TelegramUser, error) {
	if !t.Configured() {
		return nil, ErrNotConfigured
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, t.method("getMe"), nil)
	res, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("telegram %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		OK     bool         `json:"ok"`
		Result TelegramUser `json:"result"`
	}
	if err := json.Unmarshal(b, &out); err != nil || !out.OK {
		return nil, errors.New("telegram getMe failed")
	}
	return &out.Result, nil
}

// TelegramUpdate 入站更新（只关心文字消息）。
type TelegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		MessageID int64  `json:"message_id"`
		Text      string `json:"text"`
		Chat      struct {
			ID       int64  `json:"id"`
			Type     string `json:"type"`
			Title    string `json:"title"`
			Username string `json:"username"`
			First    string `json:"first_name"`
		} `json:"chat"`
		From struct {
			Username string `json:"username"`
			First    string `json:"first_name"`
		} `json:"from"`
	} `json:"message"`
}

// GetUpdates 长轮询拉取更新。offset 为已处理的最大 update_id + 1；timeout 为长轮询秒数。
func (t *Telegram) GetUpdates(ctx context.Context, offset int64, timeoutSec int) ([]TelegramUpdate, error) {
	if !t.Configured() {
		return nil, ErrNotConfigured
	}
	q := url.Values{}
	if offset > 0 {
		q.Set("offset", strconv.FormatInt(offset, 10))
	}
	q.Set("timeout", strconv.Itoa(timeoutSec))
	q.Set("allowed_updates", `["message"]`)
	u := t.method("getUpdates") + "?" + q.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	res, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("telegram %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		OK     bool             `json:"ok"`
		Result []TelegramUpdate `json:"result"`
	}
	if err := json.Unmarshal(b, &out); err != nil || !out.OK {
		return nil, errors.New("telegram getUpdates failed")
	}
	return out.Result, nil
}

// ---- Webhook ----

// Webhook 投递器。
type Webhook struct {
	cfg    config.Webhook
	client *http.Client
}

func NewWebhook(cfg config.Webhook) *Webhook {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Webhook{cfg: cfg, client: &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// Sign 签名：hex(HMAC-SHA256(secret, ts + "." + body))。
func Sign(secret string, ts int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify 校验签名（供文档示例与测试）。
func Verify(secret string, ts int64, body []byte, sig string, skew time.Duration) bool {
	if skew > 0 && time.Since(time.Unix(ts, 0)).Abs() > skew {
		return false
	}
	return hmac.Equal([]byte(Sign(secret, ts, body)), []byte(strings.ToLower(sig)))
}

// ValidateURL 目标地址校验：https（可配置允许 http）、不得指向内网/回环/链路本地。
func (w *Webhook) ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return errors.New("地址无效")
	}
	if u.Scheme != "https" && !(w.cfg.AllowHTTP && u.Scheme == "http") {
		return errors.New("只允许 https 地址")
	}
	host := u.Hostname()
	if w.cfg.AllowHTTP {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return errors.New("不允许内网地址")
		}
	} else if host == "localhost" {
		return errors.New("不允许内网地址")
	}
	return nil
}

// Result 一次投递结果。
type Result struct {
	Status int
	Body   string
	Err    string
	Took   time.Duration
}

// Deliver 发送事件。头：X-Cloak-Event、X-Cloak-Event-Id、X-Cloak-Timestamp、X-Cloak-Signature、X-Cloak-Delivery。
func (w *Webhook) Deliver(ctx context.Context, target, secret, event, eventID string, deliveryID uint64, payload []byte) Result {
	start := time.Now()
	ts := time.Now().Unix()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return Result{Err: err.Error(), Took: time.Since(start)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Cloak-Webhook/2.0")
	req.Header.Set("X-Cloak-Event", event)
	req.Header.Set("X-Cloak-Event-Id", eventID)
	req.Header.Set("X-Cloak-Delivery", strconv.FormatUint(deliveryID, 10))
	req.Header.Set("X-Cloak-Timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("X-Cloak-Signature", "v1="+Sign(secret, ts, payload))
	res, err := w.client.Do(req)
	if err != nil {
		return Result{Err: trunc(err.Error(), 500), Took: time.Since(start)}
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1000))
	out := Result{Status: res.StatusCode, Body: trunc(string(b), 1000), Took: time.Since(start)}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		out.Err = "http " + strconv.Itoa(res.StatusCode)
	}
	return out
}

// Backoff 第 n 次失败后的等待：1m, 5m, 30m, 2h, 6h。
func Backoff(attempt int) time.Duration {
	steps := []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour, 6 * time.Hour}
	if attempt <= 0 {
		return steps[0]
	}
	if attempt > len(steps) {
		return steps[len(steps)-1]
	}
	return steps[attempt-1]
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func fromAddr(s string) string {
	if i := strings.Index(s, "<"); i >= 0 {
		if j := strings.Index(s[i:], ">"); j > 0 {
			return s[i+1 : i+j]
		}
	}
	return s
}

func b64(s string) string {
	const tbl = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	src := []byte(s)
	var out strings.Builder
	for i := 0; i < len(src); i += 3 {
		var n uint32
		rem := len(src) - i
		for j := 0; j < 3; j++ {
			n <<= 8
			if j < rem {
				n |= uint32(src[i+j])
			}
		}
		for j := 0; j < 4; j++ {
			if j <= rem {
				out.WriteByte(tbl[(n>>(18-6*j))&0x3f])
			} else {
				out.WriteByte('=')
			}
		}
	}
	return out.String()
}
