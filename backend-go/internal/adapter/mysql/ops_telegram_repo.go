package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cloak/internal/domain/ops"
)

// ---- Telegram 机器人 ----

// GetTelegramBot 取商户机器人；不存在返回 nil, nil。
func (r *OpsRepo) GetTelegramBot(ctx context.Context, merchantID uint64) (*ops.TelegramBot, error) {
	var row ops.TelegramBot
	err := r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// FindTelegramBotByUserID 按 Telegram bot id 查占用方；不存在返回 nil, nil。
func (r *OpsRepo) FindTelegramBotByUserID(ctx context.Context, botUserID int64) (*ops.TelegramBot, error) {
	if botUserID == 0 {
		return nil, nil
	}
	var row ops.TelegramBot
	err := r.db.WithContext(ctx).Where("bot_user_id = ?", botUserID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// CountTelegramChats 商户已绑定会话数（含停用）。
func (r *OpsRepo) CountTelegramChats(ctx context.Context, merchantID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&ops.TelegramChat{}).Where("merchant_id = ?", merchantID).Count(&n).Error
	return n, err
}

// SaveTelegramBot 新建或覆盖机器人配置（Token、用户名、状态等）。
func (r *OpsRepo) SaveTelegramBot(ctx context.Context, b *ops.TelegramBot) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "merchant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"bot_token_enc", "bot_user_id", "bot_username", "status", "updated_at",
		}),
	}).Create(b).Error
}

// DeleteTelegramBot 移除机器人；绑定会话保留。
func (r *OpsRepo) DeleteTelegramBot(ctx context.Context, merchantID uint64) error {
	return r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).Delete(&ops.TelegramBot{}).Error
}

// ActiveTelegramBots 全部启用中的机器人，供 worker 长轮询管理。
func (r *OpsRepo) ActiveTelegramBots(ctx context.Context) ([]ops.TelegramBot, error) {
	var rows []ops.TelegramBot
	return rows, r.db.WithContext(ctx).Where("status = ?", ops.TelegramBotActive).Find(&rows).Error
}

// UpdateTelegramOffset 推进 getUpdates 的 offset。
func (r *OpsRepo) UpdateTelegramOffset(ctx context.Context, merchantID uint64, lastUpdateID int64) error {
	return r.db.WithContext(ctx).Model(&ops.TelegramBot{}).
		Where("merchant_id = ? AND last_update_id < ?", merchantID, lastUpdateID).
		Update("last_update_id", lastUpdateID).Error
}

// SetTelegramBotStatus 改状态（如 Token 失效置 disabled）。
func (r *OpsRepo) SetTelegramBotStatus(ctx context.Context, merchantID uint64, status string) error {
	return r.db.WithContext(ctx).Model(&ops.TelegramBot{}).
		Where("merchant_id = ?", merchantID).Update("status", status).Error
}

// SaveDailyReport 更新日报设置。
func (r *OpsRepo) SaveDailyReport(ctx context.Context, merchantID uint64, enabled bool, hour int, siteID uint64) error {
	return r.db.WithContext(ctx).Model(&ops.TelegramBot{}).
		Where("merchant_id = ?", merchantID).
		Updates(map[string]any{"daily_report": enabled, "daily_report_hour": hour, "daily_report_site": siteID}).Error
}

// MarkDailyReportSent 记下今天已发日报，防重复。
func (r *OpsRepo) MarkDailyReportSent(ctx context.Context, merchantID uint64, day time.Time) error {
	return r.db.WithContext(ctx).Model(&ops.TelegramBot{}).
		Where("merchant_id = ?", merchantID).Update("last_report_on", day).Error
}

// ---- Telegram 会话 ----

// ListTelegramChats 商户已绑定的会话。
func (r *OpsRepo) ListTelegramChats(ctx context.Context, merchantID uint64) ([]ops.TelegramChat, error) {
	var rows []ops.TelegramChat
	return rows, r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).Order("id").Find(&rows).Error
}

// ActiveTelegramChats 商户启用中的会话，用于发送。
func (r *OpsRepo) ActiveTelegramChats(ctx context.Context, merchantID uint64) ([]ops.TelegramChat, error) {
	var rows []ops.TelegramChat
	return rows, r.db.WithContext(ctx).
		Where("merchant_id = ? AND status = ?", merchantID, ops.TelegramBotActive).Order("id").Find(&rows).Error
}

// FindTelegramChat 按 chat_id 找会话。
func (r *OpsRepo) FindTelegramChat(ctx context.Context, merchantID uint64, chatID string) (*ops.TelegramChat, error) {
	var row ops.TelegramChat
	err := r.db.WithContext(ctx).Where("merchant_id = ? AND chat_id = ?", merchantID, chatID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// UpsertTelegramChat 绑定或更新会话（标题/类型变化时刷新，并重新置为 active）。
func (r *OpsRepo) UpsertTelegramChat(ctx context.Context, c *ops.TelegramChat) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "merchant_id"}, {Name: "chat_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"chat_type", "title", "status"}),
	}).Create(c).Error
}

// SetTelegramChatStatus 停用 / 启用某会话。
func (r *OpsRepo) SetTelegramChatStatus(ctx context.Context, merchantID, id uint64, status string) error {
	return r.db.WithContext(ctx).Model(&ops.TelegramChat{}).
		Where("merchant_id = ? AND id = ?", merchantID, id).Update("status", status).Error
}

// DeleteTelegramChat 解绑某会话。
func (r *OpsRepo) DeleteTelegramChat(ctx context.Context, merchantID, id uint64) error {
	return r.db.WithContext(ctx).Where("merchant_id = ? AND id = ?", merchantID, id).Delete(&ops.TelegramChat{}).Error
}
