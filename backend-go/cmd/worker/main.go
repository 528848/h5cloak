// 异步任务：决策日志入库、归因写入、DNS 校验、健康检查、告警、Webhook。
// I0 只建立运行骨架与依赖连通性检查，具体消费者从 I3 起加入。
package main

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/adapter/redis"
	"cloak/internal/app/bootstrap"
	"cloak/internal/app/sitesvc"
	"cloak/internal/pkg/httpserver"
)

func main() {
	rt := bootstrap.Init("worker")
	defer rt.Close()

	rdb, err := redis.Open(rt.Cfg.Redis)
	if err != nil {
		rt.Log.Warn("redis unavailable, worker will retry in background", zap.Error(err))
	} else {
		defer rdb.Close()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if db, err := mysql.Open(rt.Cfg.MySQL, rt.Log); err != nil {
		rt.Log.Warn("mysql unavailable, origin probe disabled", zap.Error(err))
	} else {
		sites := sitesvc.New(mysql.NewSiteRepo(db), mysql.NewMerchantRepo(db), rt.Cfg.Gateway.EntryHost, rt.Cfg.App.UploadDir, rt.Cfg.App.PublicURL)
		go probeLoop(ctx, rt.Log, sites)
	}
	go heartbeat(ctx, rt.Log)

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

func heartbeat(ctx context.Context, log *zap.Logger) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			log.Debug("worker alive")
		}
	}
}
