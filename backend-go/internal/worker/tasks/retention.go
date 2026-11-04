// Package tasks 周期任务：日志保留清理等。
package tasks

import (
	"context"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/domain/identity"
)

// Retention 按商户 log_retention_days 删除过期决策日志。
type Retention struct {
	logs      *mysql.LogRepo
	merchants identity.MerchantRepository
	log       *zap.Logger
}

func NewRetention(logs *mysql.LogRepo, merchants identity.MerchantRepository, log *zap.Logger) *Retention {
	return &Retention{logs: logs, merchants: merchants, log: log}
}

// Run 每 6 小时执行一次，启动后 1 分钟先跑一次。
func (t *Retention) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Minute):
	}
	t.once(ctx)
	tk := time.NewTicker(6 * time.Hour)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			t.once(ctx)
		}
	}
}

func (t *Retention) once(ctx context.Context) {
	rows, _, err := t.merchants.List(ctx, identity.MerchantFilter{Page: 1, Size: 500})
	if err != nil {
		t.log.Warn("retention: list merchants", zap.Error(err))
		return
	}
	for _, m := range rows {
		days := int(m.LogRetentionDays)
		if days <= 0 {
			days = 90
		}
		before := time.Now().AddDate(0, 0, -days)
		n, err := t.logs.Purge(ctx, m.ID, before, 5000)
		if err != nil {
			t.log.Warn("retention purge", zap.Uint64("merchant", m.ID), zap.Error(err))
			continue
		}
		if n > 0 {
			t.log.Info("retention purged", zap.Uint64("merchant", m.ID), zap.Int64("rows", n), zap.Int("days", days))
		}
	}
}
