package telegramsvc

import (
	"context"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"cloak/internal/domain/ops"
	"cloak/internal/pkg/notifier"
)

// RunPolling 启动机器人长轮询管理器与日报定时。阻塞直至 ctx 结束。
func (s *Service) RunPolling(ctx context.Context) {
	if s.box == nil {
		s.log.Info("telegram self-bot disabled: no secret box")
		return
	}
	go s.dailyLoop(ctx)
	s.manageBots(ctx)
}

type botLoop struct {
	userID int64
	cancel context.CancelFunc
}

// manageBots 每 30s 刷新 active 机器人集合，按需起停长轮询。
func (s *Service) manageBots(ctx context.Context) {
	loops := map[uint64]*botLoop{}
	defer func() {
		for _, l := range loops {
			l.cancel()
		}
	}()
	tk := time.NewTicker(30 * time.Second)
	defer tk.Stop()
	s.refreshBots(ctx, loops)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			s.refreshBots(ctx, loops)
		}
	}
}

func (s *Service) refreshBots(ctx context.Context, loops map[uint64]*botLoop) {
	bots, err := s.repo.ActiveTelegramBots(ctx)
	if err != nil {
		s.log.Warn("list active telegram bots", zap.Error(err))
		return
	}
	seen := map[uint64]bool{}
	for _, b := range bots {
		seen[b.MerchantID] = true
		cur := loops[b.MerchantID]
		if cur != nil && cur.userID == b.BotUserID {
			continue // 已在跑且未换 Token
		}
		if cur != nil {
			cur.cancel() // Token 变了，重启
		}
		tok, err := s.box.Open(b.BotTokenEnc)
		if err != nil {
			s.log.Warn("decrypt bot token", zap.Uint64("merchant", b.MerchantID), zap.Error(err))
			continue
		}
		lctx, cancel := context.WithCancel(ctx)
		loops[b.MerchantID] = &botLoop{userID: b.BotUserID, cancel: cancel}
		go s.pollBot(lctx, b.MerchantID, string(tok), b.LastUpdateID)
	}
	// 停掉已不再 active 的
	for mid, l := range loops {
		if !seen[mid] {
			l.cancel()
			delete(loops, mid)
		}
	}
}

// pollBot 单个机器人的 getUpdates 循环。
func (s *Service) pollBot(ctx context.Context, merchantID uint64, token string, lastUpdateID int64) {
	tg := notifier.NewTelegramWithToken(s.apiBase(), token)
	offset := lastUpdateID
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		ups, err := tg.GetUpdates(ctx, offset+1, 50)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if strings.Contains(err.Error(), "401") {
				s.log.Warn("telegram bot token invalid, disabling", zap.Uint64("merchant", merchantID))
				_ = s.repo.SetTelegramBotStatus(context.Background(), merchantID, ops.TelegramBotDisabled)
				return
			}
			s.log.Warn("telegram getUpdates", zap.Uint64("merchant", merchantID), zap.Error(err))
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		for _, u := range ups {
			s.handleUpdate(ctx, merchantID, tg, u)
			if u.UpdateID > offset {
				offset = u.UpdateID
			}
		}
		if len(ups) > 0 {
			_ = s.repo.UpdateTelegramOffset(context.Background(), merchantID, offset)
		}
	}
}

// handleUpdate 处理一条更新：绑定或问答。
func (s *Service) handleUpdate(ctx context.Context, merchantID uint64, tg *notifier.Telegram, u notifier.TelegramUpdate) {
	defer func() { _ = recover() }()
	if u.Message == nil {
		return
	}
	text := strings.TrimSpace(u.Message.Text)
	if text == "" {
		return
	}
	chatID := strconv.FormatInt(u.Message.Chat.ID, 10)

	// 绑定指令
	if code, ok := bindArg(text); ok {
		s.doBind(ctx, merchantID, tg, u, chatID, code)
		return
	}

	// 未绑定会话：只回绑定指引
	chat, _ := s.repo.FindTelegramChat(ctx, merchantID, chatID)
	if chat == nil || chat.Status != ops.TelegramBotActive {
		_ = tg.Send(ctx, chatID, "这个会话还没绑定。请在商户后台「告警通知 → Telegram」点「绑定」，或发送 /bind 绑定码。")
		return
	}

	if !s.limiter.allow(chatID) {
		_ = tg.Send(ctx, chatID, "消息太频繁了，请稍后再问～")
		return
	}
	reply := s.Answer(ctx, merchantID, text)
	_ = tg.Send(ctx, chatID, reply)
}

// bindArg 从 /start <code>、/bind <code>、/start@bot <code>，以及深链偶发的 /start<code>（无空格、纯十六进制）取出绑定码。
func bindArg(text string) (string, bool) {
	t := strings.TrimSpace(text)
	for _, p := range []string{"/start@", "/bind@", "/start ", "/bind "} {
		if !strings.HasPrefix(t, p) {
			continue
		}
		rest := strings.TrimSpace(t[len(p):])
		if fields := strings.Fields(rest); len(fields) > 0 {
			return fields[len(fields)-1], true
		}
		return "", false
	}
	// 无空格深链：/startabc123def → 仅当后缀像绑定码（hex）才认，避免误伤 /startup
	for _, p := range []string{"/start", "/bind"} {
		if !strings.HasPrefix(t, p) || len(t) <= len(p) {
			continue
		}
		code := t[len(p):]
		if isBindCode(code) {
			return code, true
		}
	}
	return "", false
}

func isBindCode(s string) bool {
	if n := len(s); n < 8 || n > 32 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func (s *Service) doBind(ctx context.Context, merchantID uint64, tg *notifier.Telegram, u notifier.TelegramUpdate, chatID, code string) {
	if s.rdb == nil {
		_ = tg.Send(ctx, chatID, "绑定暂不可用，请联系平台。")
		return
	}
	val, err := s.rdb.GetDel(ctx, bindKey(code)).Result()
	if err != nil || val == "" {
		_ = tg.Send(ctx, chatID, "绑定码已失效，请回后台重新获取。")
		return
	}
	boundMerchant, _ := strconv.ParseUint(val, 10, 64)
	if boundMerchant != merchantID {
		_ = tg.Send(ctx, chatID, "绑定码与当前机器人不匹配。")
		return
	}
	existing, _ := s.repo.FindTelegramChat(ctx, merchantID, chatID)
	if existing == nil {
		if n, err := s.repo.CountTelegramChats(ctx, merchantID); err == nil && n >= maxBoundChats {
			_ = tg.Send(ctx, chatID, "已达到绑定上限（"+strconv.Itoa(maxBoundChats)+" 个），请先在后台解绑不用的会话。")
			return
		}
	}
	title := u.Message.Chat.Title
	if title == "" {
		title = strings.TrimSpace(u.Message.Chat.First + " " + u.Message.Chat.Username)
	}
	if title == "" {
		title = u.Message.From.First
	}
	chat := &ops.TelegramChat{
		MerchantID: merchantID, ChatID: chatID, ChatType: u.Message.Chat.Type,
		Title: trunc(title, 120), Status: ops.TelegramBotActive,
	}
	if err := s.repo.UpsertTelegramChat(ctx, chat); err != nil {
		_ = tg.Send(ctx, chatID, "绑定失败，请稍后再试。")
		return
	}
	_ = tg.Send(ctx, chatID, "✅ 绑定成功，这个会话会收到告警与日报。发「帮助」看用法。")
}

// dailyLoop 每分钟检查到点的日报。
func (s *Service) dailyLoop(ctx context.Context) {
	tk := time.NewTicker(time.Minute)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			s.SendDueDailyReports(ctx)
		}
	}
}

// SendDueDailyReports 发送到点的日报。
func (s *Service) SendDueDailyReports(ctx context.Context) {
	bots, err := s.repo.ActiveTelegramBots(ctx)
	if err != nil {
		return
	}
	for _, b := range bots {
		if !b.DailyReport {
			continue
		}
		m, err := s.merchants.FindByID(ctx, b.MerchantID)
		if err != nil || m == nil {
			continue
		}
		loc := loadLoc(m.Timezone)
		now := s.now().In(loc)
		if now.Hour() != b.DailyReportHour {
			continue
		}
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		if b.LastReportOn != nil && !b.LastReportOn.Before(today) {
			continue // 今天已发
		}
		tok, err := s.box.Open(b.BotTokenEnc)
		if err != nil {
			continue
		}
		chats, _ := s.repo.ActiveTelegramChats(ctx, b.MerchantID)
		if len(chats) == 0 {
			_ = s.repo.MarkDailyReportSent(ctx, b.MerchantID, today)
			continue
		}
		digest := s.BuildDigest(ctx, b.MerchantID, b.DailyReportSite)
		tg := notifier.NewTelegramWithToken(s.apiBase(), string(tok))
		for _, c := range chats {
			_ = tg.Send(ctx, c.ChatID, digest)
		}
		_ = s.repo.MarkDailyReportSent(ctx, b.MerchantID, today)
	}
}

func loadLoc(tz string) *time.Location {
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.Local
}
