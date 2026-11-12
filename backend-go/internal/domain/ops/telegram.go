package ops

import "time"

// TelegramBot 商户专属 Telegram 机器人。一个商户一条。Token 加密存储。
type TelegramBot struct {
	MerchantID      uint64     `json:"merchantId" gorm:"primaryKey;column:merchant_id"`
	BotTokenEnc     []byte     `json:"-" gorm:"column:bot_token_enc"`
	BotUserID       int64      `json:"botUserId" gorm:"column:bot_user_id"`
	BotUsername     string     `json:"botUsername" gorm:"column:bot_username"`
	Status          string     `json:"status" gorm:"column:status"`
	LastUpdateID    int64      `json:"-" gorm:"column:last_update_id"`
	DailyReport     bool       `json:"dailyReport" gorm:"column:daily_report"`
	DailyReportHour int        `json:"dailyReportHour" gorm:"column:daily_report_hour"`
	DailyReportSite uint64     `json:"dailyReportSite" gorm:"column:daily_report_site"`
	LastReportOn    *time.Time `json:"-" gorm:"column:last_report_on"`
	CreatedAt       time.Time  `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt       time.Time  `json:"updatedAt" gorm:"column:updated_at"`
}

func (TelegramBot) TableName() string { return "telegram_bots" }

// TelegramBot 状态。
const (
	TelegramBotActive   = "active"
	TelegramBotDisabled = "disabled"
)

// TelegramChat 已绑定的接收会话。
type TelegramChat struct {
	ID         uint64    `json:"id" gorm:"primaryKey"`
	MerchantID uint64    `json:"merchantId" gorm:"column:merchant_id"`
	ChatID     string    `json:"chatId" gorm:"column:chat_id"`
	ChatType   string    `json:"chatType" gorm:"column:chat_type"`
	Title      string    `json:"title" gorm:"column:title"`
	Status     string    `json:"status" gorm:"column:status"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (TelegramChat) TableName() string { return "telegram_chats" }
