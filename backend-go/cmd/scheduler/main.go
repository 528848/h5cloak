// 定时任务：巡检、聚合、清理、到期处理。单实例运行，依赖 Redis 锁保证只有一个活跃节点。
// I0 只建立运行骨架，任务从 I3 起加入。
package main

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/redis"
	"cloak/internal/app/bootstrap"
	"cloak/internal/pkg/httpserver"
)

func main() {
	rt := bootstrap.Init("scheduler")
	defer rt.Close()

	rdb, err := redis.Open(rt.Cfg.Redis)
	if err != nil {
		rt.Log.Warn("redis unavailable, scheduler will stay idle", zap.Error(err))
	} else {
		defer rdb.Close()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go tick(ctx, rt.Log)

	srv := httpserver.New(rt.Log, rt.Cfg.HTTP, nil, rt.Metrics)
	if err := srv.Run(ctx); err != nil {
		rt.Log.Error("exit with error", zap.Error(err))
		os.Exit(1)
	}
	rt.Log.Info("stopped")
}

func tick(ctx context.Context, log *zap.Logger) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			log.Debug("scheduler tick")
		}
	}
}
