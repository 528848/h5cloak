// 异步任务：决策日志入库、归因写入、源站健康检查、日志保留清理。
package main

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/adapter/redis"
	"cloak/internal/app/bootstrap"
	"cloak/internal/app/opssvc"
	"cloak/internal/app/sitesvc"
	"cloak/internal/pkg/httpserver"
	"cloak/internal/quota"
	"cloak/internal/worker/streams"
	"cloak/internal/worker/tasks"
)

func main() {
	rt := bootstrap.Init("worker")
	defer rt.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rdb, err := redis.Open(rt.Cfg.Redis)
	if err != nil {
		rt.Log.Warn("redis unavailable, stream consumers disabled", zap.Error(err))
		rdb = nil
	} else {
		defer rdb.Close()
	}

	db, err := mysql.Open(rt.Cfg.MySQL, rt.Log)
	if err != nil {
		rt.Log.Error("mysql unavailable", zap.Error(err))
		os.Exit(1)
	}
	siteRepo := mysql.NewSiteRepo(db)
	merchantRepo := mysql.NewMerchantRepo(db)
	logRepo := mysql.NewLogRepo(db)

	sites := sitesvc.New(siteRepo, merchantRepo, rt.Cfg.Gateway.EntryHost, rt.Cfg.App.UploadDir, rt.Cfg.App.PublicURL)
	go probeLoop(ctx, rt.Log, sites)
	platformRepo := mysql.NewPlatformRepo(db)
	retention := tasks.NewRetention(logRepo, merchantRepo, rt.Log)
	retention.SetCap(func(ctx context.Context) int {
		n, err := platformRepo.RetentionDays(ctx)
		if err != nil {
			return 0
		}
		return n
	})
	go retention.Run(ctx)

	// 聚合、告警、Webhook 投递、导出
	var qc *quota.Checker
	if rdb != nil {
		qc = quota.New(rdb)
	}
	opsSvc := opssvc.New(opssvc.Deps{
		Repo: mysql.NewOpsRepo(db), Logs: logRepo, Sites: siteRepo, Merchants: merchantRepo, Quota: qc,
		Notify: rt.Cfg.Notify, ExportDir: rt.Cfg.Worker.ExportDir, Log: rt.Log,
	})
	opsSvc.SetAttemptSource(func(ctx context.Context) int {
		n, err := platformRepo.WebhookAttempts(ctx)
		if err != nil {
			return 0
		}
		return n
	})
	tasks.NewOps(opsSvc, rt.Log).Run(ctx)

	if rdb != nil {
		consumer := streams.New(rdb, logRepo, rt.Log, rt.Cfg.App.NodeName)
		go consumer.Run(ctx)
		go func() {
			t := time.NewTicker(time.Minute)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					s := consumer.Stats()
					rt.Log.Info("consumer stats", zap.Int64("decisions", s.Decisions), zap.Int64("attributions", s.Attributions),
						zap.Int64("apiLogs", s.APILogs), zap.Int64("errors", s.Errors))
				}
			}
		}()
	}

	srv := httpserver.New(rt.Log, rt.Cfg.HTTP, nil, rt.Metrics)
	if err := srv.Run(ctx); err != nil {
		rt.Log.Error("exit with error", zap.Error(err))
		os.Exit(1)
	}
	rt.Log.Info("stopped")
}

func probeLoop(ctx context.Context, log *zap.Logger, sites *sitesvc.Service) {
	t := time.NewTicker(2 * time.Minute)
	defer t.Stop()
	sites.ProbeDue(ctx, 20)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			log.Debug("probe origins")
			sites.ProbeDue(ctx, 20)
		}
	}
}
