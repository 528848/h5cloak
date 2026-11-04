// 判定网关：托管反向代理、开放判定接口、JS 上报。
package main

import (
	"context"
	"os"

	"go.uber.org/zap"

	"cloak/internal/adapter/redis"
	"cloak/internal/app/bootstrap"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
	"cloak/internal/pkg/httpserver"
	"cloak/internal/transport/http/gateway"
)

func main() {
	rt := bootstrap.Init("gateway")
	defer rt.Close()

	deps := gateway.Deps{Cfg: rt.Cfg, Log: rt.Log, Metrics: rt.Metrics}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rdb, err := redis.Open(rt.Cfg.Redis)
	if err != nil {
		// 没有 Redis 时网关只能提供健康检查；快照无法加载。
		rt.Log.Error("redis unavailable, gateway cannot load snapshots", zap.Error(err))
	} else {
		loader := redis.NewSnapshotLoader(rdb)
		store := snapshot.NewStore(loader)
		deps.Store = store
		deps.Engine = engine.New(redis.NewEngineStore(rdb))
		go loader.Subscribe(ctx, store, rt.Log)
		rt.Log.Info("snapshot store ready, subscribed to cfg:changed")
	}

	handler := gateway.New(deps)
	srv := httpserver.New(rt.Log, rt.Cfg.HTTP, handler, rt.Metrics)
	if err := srv.Run(ctx); err != nil {
		rt.Log.Error("exit with error", zap.Error(err))
		os.Exit(1)
	}
	rt.Log.Info("stopped")
}
