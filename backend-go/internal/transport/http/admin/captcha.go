package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const captchaTTL = 2 * time.Minute

// 去掉 0/1/I/O，避免和字母混淆。与 PHP 版验证码字符集一致。
const captchaPool = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

var captchaColors = []string{"#2563eb", "#16a34a", "#dc2626", "#7c3aed", "#0f766e", "#ea580c"}

type captchaStore struct {
	rdb *goredis.Client
	mu  sync.Mutex
	mem map[string]captchaEntry
}

type captchaEntry struct {
	code string
	exp  time.Time
}

func newCaptchaStore(rdb *goredis.Client) *captchaStore {
	return &captchaStore{rdb: rdb, mem: map[string]captchaEntry{}}
}

// Issue 生成验证码，返回 uuid 与 SVG 的 base64（不含 data: 前缀）。
func (s *captchaStore) Issue(ctx context.Context) (uuid, img string, err error) {
	code := randomCode(4)
	id := randomHex(16)
	if err = s.save(ctx, id, code); err != nil {
		return "", "", err
	}
	return id, base64.StdEncoding.EncodeToString([]byte(renderCaptcha(code))), nil
}

// Verify 校验并消费。错误时返回给登录接口展示的文案。
func (s *captchaStore) Verify(ctx context.Context, uuid, code string) string {
	uuid = strings.TrimSpace(uuid)
	code = strings.TrimSpace(code)
	if uuid == "" || code == "" {
		return "请输入验证码"
	}
	got, ok := s.take(ctx, uuid)
	if !ok || !strings.EqualFold(got, code) {
		return "验证码错误"
	}
	return ""
}

func (s *captchaStore) save(ctx context.Context, uuid, code string) error {
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, "cloak:captcha:"+uuid, code, captchaTTL).Err(); err == nil {
			return nil
		}
	}
	s.mu.Lock()
	s.mem[uuid] = captchaEntry{code: code, exp: time.Now().Add(captchaTTL)}
	s.mu.Unlock()
	return nil
}

func (s *captchaStore) take(ctx context.Context, uuid string) (string, bool) {
	if s.rdb != nil {
		key := "cloak:captcha:" + uuid
		code, err := s.rdb.Get(ctx, key).Result()
		if err == nil && code != "" {
			_ = s.rdb.Del(ctx, key).Err()
			return code, true
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.mem {
		if now.After(v.exp) {
			delete(s.mem, k)
		}
	}
	v, ok := s.mem[uuid]
	if !ok || now.After(v.exp) {
		return "", false
	}
	delete(s.mem, uuid)
	return v.code, true
}

// renderCaptcha 画法和 PHP CaptchaCenter 一致：浅底、彩色干扰线、加粗倾斜字符。
func renderCaptcha(code string) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="40" viewBox="0 0 120 40">`)
	b.WriteString(`<rect width="120" height="40" rx="6" fill="#f7f9fc"/>`)
	for i := 0; i < 7; i++ {
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1" opacity=".35"/>`,
			randInt(121), randInt(41), randInt(121), randInt(41), captchaColor())
	}
	for i := 0; i < len(code); i++ {
		x := 16 + i*25
		y := 27 + randInt(6)
		rot := randInt(29) - 14
		fmt.Fprintf(&b, `<text x="%d" y="%d" fill="%s" font-size="24" font-family="Arial,Verdana,sans-serif" font-weight="700" transform="rotate(%d %d 24)">%c</text>`,
			x, y, captchaColor(), rot, x, code[i])
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func captchaColor() string {
	return captchaColors[randInt(len(captchaColors))]
}

func randomCode(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = captchaPool[randInt(len(captchaPool))]
	}
	return string(b)
}

func randomHex(n int) string {
	raw := make([]byte, n)
	_, _ = rand.Read(raw)
	const hexd = "0123456789abcdef"
	out := make([]byte, n*2)
	for i, v := range raw {
		out[i*2] = hexd[v>>4]
		out[i*2+1] = hexd[v&0x0f]
	}
	return string(out)
}

func randInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}
