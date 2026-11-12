// Package telegramsvc 商户专属 Telegram 机器人：配置、绑定、自然语言问答、每日日报。
// 入站用长轮询（worker 内运行），出站用商户自己的 Bot Token。
package telegramsvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/app/opssvc"
	"cloak/internal/config"
	"cloak/internal/domain/ops"
	"cloak/internal/pkg/notifier"
	"cloak/internal/pkg/secretbox"
)

// Service Telegram 机器人服务。
type Service struct {
	repo      *mysql.OpsRepo
	ops       *opssvc.Service
	merchants *mysql.MerchantRepo
	sites     *mysql.SiteRepo
	box       *secretbox.Box
	rdb       *goredis.Client
	cfg       config.Notify
	log       *zap.Logger
	now       func() time.Time

	limiter *rateLimiter
}

// Deps 依赖。
type Deps struct {
	Repo      *mysql.OpsRepo
	Ops       *opssvc.Service
	Merchants *mysql.MerchantRepo
	Sites     *mysql.SiteRepo
	Box       *secretbox.Box
	Rdb       *goredis.Client
	Notify    config.Notify
	Log       *zap.Logger
}

// New 创建服务。
func New(d Deps) *Service {
	if d.Log == nil {
		d.Log = zap.NewNop()
	}
	return &Service{
		repo: d.Repo, ops: d.Ops, merchants: d.Merchants, sites: d.Sites,
		box: d.Box, rdb: d.Rdb, cfg: d.Notify, log: d.Log, now: time.Now,
		limiter: newRateLimiter(20, time.Minute),
	}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

// Enabled 是否具备启用自有机器人的条件（配置了加密主密钥）。
func (s *Service) Enabled() bool { return s.box != nil }

func (s *Service) apiBase() string { return s.cfg.Telegram.APIBase }

// ---- 配置视图 ----

// ConfigView 配置页数据。
type ConfigView struct {
	HasBot         bool       `json:"hasBot"`
	BotUsername    string     `json:"botUsername"`
	TokenSuffix    string     `json:"tokenSuffix"`
	Status         string     `json:"status"`
	DailyReport    bool       `json:"dailyReport"`
	DailyHour      int        `json:"dailyHour"`
	DailySite      uint64     `json:"dailySite"`
	Chats          []ChatView `json:"chats"`
	PlatformBotSet bool       `json:"platformBotSet"` // 平台兜底 Bot 是否可用
}

// ChatView 绑定会话。
type ChatView struct {
	ID       uint64 `json:"id"`
	ChatID   string `json:"chatId"`
	ChatType string `json:"chatType"`
	Title    string `json:"title"`
	Status   string `json:"status"`
}

// Config 返回商户的机器人与绑定配置。
func (s *Service) Config(ctx context.Context, merchantID uint64) (ConfigView, error) {
	out := ConfigView{Chats: []ChatView{}, DailyHour: 9, PlatformBotSet: s.cfg.Telegram.BotToken != ""}
	bot, err := s.repo.GetTelegramBot(ctx, merchantID)
	if err != nil {
		return out, err
	}
	if bot != nil {
		out.HasBot = true
		out.BotUsername = bot.BotUsername
		out.Status = bot.Status
		out.DailyReport = bot.DailyReport
		out.DailyHour = bot.DailyReportHour
		out.DailySite = bot.DailyReportSite
		if tok, e := s.box.Open(bot.BotTokenEnc); e == nil {
			out.TokenSuffix = lastN(string(tok), 4)
		}
	}
	chats, _ := s.repo.ListTelegramChats(ctx, merchantID)
	for _, c := range chats {
		out.Chats = append(out.Chats, ChatView{ID: c.ID, ChatID: c.ChatID, ChatType: c.ChatType, Title: c.Title, Status: c.Status})
	}
	return out, nil
}

// SaveBot 校验并保存 Token。返回机器人用户名。
func (s *Service) SaveBot(ctx context.Context, merchantID uint64, token string) (string, error) {
	if s.box == nil {
		return "", errf(400, "服务端未配置加密主密钥，无法保存机器人 Token")
	}
	token = strings.TrimSpace(token)
	if token == "" || !strings.Contains(token, ":") {
		return "", errf(400, "Token 格式不正确，应形如 123456789:AA...")
	}
	me, err := notifier.NewTelegramWithToken(s.apiBase(), token).GetMe(ctx)
	if err != nil {
		return "", errf(400, "Token 无效或无法连接 Telegram："+trunc(err.Error(), 100))
	}
	if other, e := s.repo.FindTelegramBotByUserID(ctx, me.ID); e != nil {
		return "", e
	} else if other != nil && other.MerchantID != merchantID {
		return "", errf(400, "该机器人已被其他商户使用，请换一个 Token")
	}
	enc, err := s.box.Seal([]byte(token))
	if err != nil {
		return "", errf(500, "加密失败")
	}
	b := &ops.TelegramBot{
		MerchantID: merchantID, BotTokenEnc: enc, BotUserID: me.ID, BotUsername: me.Username,
		Status: ops.TelegramBotActive, UpdatedAt: s.now(),
	}
	if err := s.repo.SaveTelegramBot(ctx, b); err != nil {
		if isDuplicate(err) {
			return "", errf(400, "该机器人已被其他商户使用，请换一个 Token")
		}
		return "", err
	}
	return me.Username, nil
}

// DeleteBot 移除机器人。
func (s *Service) DeleteBot(ctx context.Context, merchantID uint64) error {
	return s.repo.DeleteTelegramBot(ctx, merchantID)
}

// BindCodeResult 绑定码。
type BindCodeResult struct {
	Code         string `json:"code"`
	DeepLink     string `json:"deepLink"`
	GroupCommand string `json:"groupCommand"`
	ExpiresInSec int    `json:"expiresInSec"`
}

const bindTTL = 10 * time.Minute

// NewBindCode 生成一次性绑定码。
func (s *Service) NewBindCode(ctx context.Context, merchantID uint64) (BindCodeResult, error) {
	var out BindCodeResult
	if s.rdb == nil {
		return out, errf(400, "绑定功能需要 Redis，请联系平台开启")
	}
	bot, err := s.repo.GetTelegramBot(ctx, merchantID)
	if err != nil {
		return out, err
	}
	if bot == nil || bot.BotUsername == "" {
		return out, errf(400, "请先保存机器人 Token")
	}
	code := randCode()
	if err := s.rdb.Set(ctx, bindKey(code), strconv.FormatUint(merchantID, 10), bindTTL).Err(); err != nil {
		return out, errf(500, "生成绑定码失败")
	}
	out = BindCodeResult{
		Code:         code,
		DeepLink:     "https://t.me/" + bot.BotUsername + "?start=" + code,
		GroupCommand: "/bind " + code,
		ExpiresInSec: int(bindTTL / time.Second),
	}
	return out, nil
}

// Chats 绑定会话列表。
func (s *Service) Chats(ctx context.Context, merchantID uint64) ([]ChatView, error) {
	rows, err := s.repo.ListTelegramChats(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	out := make([]ChatView, 0, len(rows))
	for _, c := range rows {
		out = append(out, ChatView{ID: c.ID, ChatID: c.ChatID, ChatType: c.ChatType, Title: c.Title, Status: c.Status})
	}
	return out, nil
}

// ToggleChat 停用 / 启用会话。
func (s *Service) ToggleChat(ctx context.Context, merchantID, id uint64) error {
	rows, err := s.repo.ListTelegramChats(ctx, merchantID)
	if err != nil {
		return err
	}
	for _, c := range rows {
		if c.ID == id {
			status := ops.TelegramBotActive
			if c.Status == ops.TelegramBotActive {
				status = ops.TelegramBotDisabled
			}
			return s.repo.SetTelegramChatStatus(ctx, merchantID, id, status)
		}
	}
	return errf(404, "会话不存在")
}

// DeleteChat 解绑会话。
func (s *Service) DeleteChat(ctx context.Context, merchantID, id uint64) error {
	return s.repo.DeleteTelegramChat(ctx, merchantID, id)
}

// SaveDaily 保存日报设置。
func (s *Service) SaveDaily(ctx context.Context, merchantID uint64, enabled bool, hour int, siteID uint64) error {
	if hour < 0 || hour > 23 {
		return errf(400, "时间应在 0-23 之间")
	}
	bot, err := s.repo.GetTelegramBot(ctx, merchantID)
	if err != nil {
		return err
	}
	if bot == nil {
		return errf(400, "请先配置机器人")
	}
	return s.repo.SaveDailyReport(ctx, merchantID, enabled, hour, siteID)
}

// Test 向商户所有启用会话发测试消息。
func (s *Service) Test(ctx context.Context, merchantID uint64) error {
	bot, token, err := s.loadBot(ctx, merchantID)
	if err != nil {
		return err
	}
	if bot == nil {
		return errf(400, "请先配置机器人 Token")
	}
	chats, _ := s.repo.ActiveTelegramChats(ctx, merchantID)
	if len(chats) == 0 {
		return errf(400, "请先绑定接收会话")
	}
	tg := notifier.NewTelegramWithToken(s.apiBase(), token)
	msg := "✅ 这是一条来自斗篷机器人的测试消息，发送时间 " + s.now().Format("2006-01-02 15:04:05")
	var firstErr error
	for _, c := range chats {
		if e := tg.Send(ctx, c.ChatID, msg); e != nil && firstErr == nil {
			firstErr = e
		}
	}
	return firstErr
}

// loadBot 取机器人与解密后的 Token。
func (s *Service) loadBot(ctx context.Context, merchantID uint64) (*ops.TelegramBot, string, error) {
	bot, err := s.repo.GetTelegramBot(ctx, merchantID)
	if err != nil || bot == nil {
		return nil, "", err
	}
	if s.box == nil {
		return bot, "", errors.New("no secret box")
	}
	tok, err := s.box.Open(bot.BotTokenEnc)
	if err != nil {
		return bot, "", err
	}
	return bot, string(tok), nil
}

func bindKey(code string) string { return "tg:bind:" + code }

func randCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}

// maxBoundChats 单商户最多绑定会话数。
const maxBoundChats = 10

// rateLimiter 按 key 的滑动计数限流（进程内）。
type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}, max: max, window: window}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := now.Add(-l.window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}
