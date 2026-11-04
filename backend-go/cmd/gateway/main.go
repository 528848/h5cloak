// 判定网关：托管反向代理、开放判定接口、JS 上报。
package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"

	"cloak/internal/adapter/redis"
	"cloak/internal/app/bootstrap"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
	"cloak/internal/pkg/httpserver"
	"cloak/internal/pkg/tlsconf"
	"cloak/internal/proxy"
	"cloak/internal/transport/http/gateway"
)

func main() {
	rt := bootstrap.Init("gateway")
	defer rt.Close()

	deps := gateway.Deps{Cfg: rt.Cfg, Log: rt.Log, Metrics: rt.Metrics}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var store *snapshot.Store
	rdb, err := redis.Open(rt.Cfg.Redis)
	if err != nil {
		// 没有 Redis 时网关只能提供健康检查；快照无法加载。
		rt.Log.Error("redis unavailable, gateway cannot load snapshots", zap.Error(err))
	} else {
		loader := redis.NewSnapshotLoader(rdb)
		store = snapshot.NewStore(loader)
		deps.Store = store
		deps.Engine = engine.New(redis.NewEngineStore(rdb))
		go loader.Subscribe(ctx, store, rt.Log)
		rt.Log.Info("snapshot store ready, subscribed to cfg:changed")
	}

	px := proxy.New(rt.Cfg.Gateway.Proxy, rt.Log)
	defer px.Close()
	deps.Proxy = px

	// TLS：只对已绑定域名签发
	allow := func(ctx context.Context, host string) bool {
		if store == nil {
			return false
		}
		_, ok := store.ByHost(ctx, host)
		return ok
	}
	tlsProv, err := tlsconf.New(rt.Cfg.Gateway.TLS, allow, rt.Log)
	if err != nil {
		rt.Log.Error("tls init", zap.Error(err))
		os.Exit(1)
	}
	deps.RedirectHTTPS = tlsProv != nil && rt.Cfg.Gateway.TLS.RedirectTo

	handler := gateway.New(deps)
	var plain http.Handler = handler
	if tlsProv != nil {
		// 明文入口先交给 ACME 挑战处理器（非 ACME 模式原样返回）
		plain = tlsProv.HTTPChallengeHandler(handler)
		rt.Log.Info("tls enabled", zap.String("mode", tlsProv.Mode()), zap.String("addr", rt.Cfg.Gateway.TLS.Addr),
			zap.Bool("redirect", deps.RedirectHTTPS))
	}

	srv := httpserver.New(rt.Log, rt.Cfg.HTTP, plain, rt.Metrics)
	if tlsProv != nil {
		srv.WithTLS(rt.Cfg.Gateway.TLS.Addr, tlsProv.TLSConfig(), handler)
	}
	if err := srv.Run(ctx); err != nil {
		if strings.Contains(err.Error(), "permission") {
			rt.Log.Error("bind failed; port 443 needs CAP_NET_BIND_SERVICE or run via systemd", zap.Error(err))
		} else {
			rt.Log.Error("exit with error", zap.Error(err))
		}
		os.Exit(1)
	}
	rt.Log.Info("stopped")
}
