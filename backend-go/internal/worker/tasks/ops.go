package tasks

import (
	"context"
	"time"

	"go.uber.org/zap"

	"cloak/internal/app/opssvc"
)

// Ops 聚合 / 告警评估 / Webhook 投递 / 导出执行。
type Ops struct {
	svc *opssvc.Service
	log *zap.Logger
}

func NewOps(svc *opssvc.Service, log *zap.Logger) *Ops { return &Ops{svc: svc, log: log} }

// Run 启动三条循环：聚合（每分钟，回溯 3 小时）、告警（每分钟）、投递与导出（每 10 秒）。
func (t *Ops) Run(ctx context.Context) {
	go t.loop(ctx, time.Minute, 5*time.Second, "aggregate", func(ctx context.Context) {
		if err := t.svc.Aggregate(ctx, 3); err != nil {
			t.log.Warn("aggregate", zap.Error(err))
		}
	})
	go t.loop(ctx, time.Minute, 20*time.Second, "alerts", func(ctx context.Context) {
		fired, resolved := t.svc.Evaluate(ctx)
		if fired > 0 || resolved > 0 {
			t.log.Info("alerts evaluated", zap.Int("fired", fired), zap.Int("resolved", resolved))
		}
	})
	go t.loop(ctx, 10*time.Second, 10*time.Second, "deliver", func(ctx context.Context) {
		if n := t.svc.ProcessDue(ctx, 50); n > 0 {
			t.log.Debug("webhooks delivered", zap.Int("n", n))
		}
		if n := t.svc.RunPendingExports(ctx, 3); n > 0 {
			t.log.Info("exports done", zap.Int("n", n))
		}
	})
	go t.loop(ctx, 6*time.Hour, 2*time.Minute, "purge", func(ctx context.Context) {
		t.svc.PurgeExpiredExports(ctx)
	})
	// 启动时先补算最近 48 小时，保证重启后看板连续
	go func() {
		c, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		if err := t.svc.Aggregate(c, 48); err != nil {
			t.log.Warn("initial aggregate", zap.Error(err))
		}
	}()
}

func (t *Ops) loop(ctx context.Context, every, initial time.Duration, name string, fn func(context.Context)) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(initial):
	}
	run := func() {
		c, cancel := context.WithTimeout(ctx, every)
		defer cancel()
		fn(c)
	}
	run()
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			run()
		}
	}
}
