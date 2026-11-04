// 判定网关：托管反向代理、开放判定接口、JS 上报。
package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/geoip"
	"cloak/internal/adapter/mysql"
	"cloak/internal/adapter/redis"
	"cloak/internal/app/attrsvc"
	"cloak/internal/app/bootstrap"
	"cloak/internal/app/logsvc"
	"cloak/internal/app/snapshotsvc"
	"cloak/internal/engine"
	"cloak/internal/engine/snapshot"
	"cloak/internal/logpipe"
	"cloak/internal/pkg/httpserver"
	"cloak/internal/pkg/secretbox"
	"cloak/internal/pkg/tlsconf"
	"cloak/internal/pkg/version"
	"cloak/internal/proxy"
	"cloak/internal/quota"
	"cloak/internal/transport/http/gateway"
	"cloak/internal/transport/http/openapi"
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

	// 日志管道：Redis 不可用时写本地 WAL，恢复后回放
	pipe := logpipe.New(logpipe.Options{WALDir: rt.Cfg.Gateway.LogWALDir}, rdb, rt.Log)
	defer pipe.Close()
	deps.Pipe = pipe
	deps.Quota = quota.New(rdb)
	if geo, ok, err := geoip.Open(rt.Cfg.Gateway.GeoIPPath); err != nil {
		rt.Log.Warn("geoip open failed, country disabled", zap.Error(err))
	} else {
		deps.Geo = geo
		defer geo.Close()
		if ok {
			rt.Log.Info("geoip loaded", zap.String("path", rt.Cfg.Gateway.GeoIPPath))
		} else {
			rt.Log.Info("geoip database not found, country field disabled", zap.String("path", rt.Cfg.Gateway.GeoIPPath))
		}
	}

	// 开放接口：鉴权与判定只靠快照；分销码与日志接口需要 MySQL（可选）。
	// 处理器在网关 handler 建好后再装配（需要它的 ClientIP），这里先放一个转发壳。
	var openAPI *openapi.Handler
	if store != nil && deps.Engine != nil {
		deps.OpenAPI = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if openAPI == nil {
				http.Error(w, "open api not ready", http.StatusServiceUnavailable)
				return
			}
			openAPI.ServeHTTP(w, r)
		})
	}

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
	if deps.OpenAPI != nil {
		box, err := secretbox.New(rt.Cfg.Auth.SecretKey)
		if err != nil {
			rt.Log.Error("secretbox", zap.Error(err))
			os.Exit(1)
		}
		od := openapi.Deps{
			Log: rt.Log, Auth: openapi.NewAuthenticator(store, box, rdb), Limit: openapi.NewLimiter(rdb, 0, 0),
			Engine: deps.Engine, Pipe: pipe, Quota: deps.Quota, Geo: deps.Geo, Rdb: rdb, ClientIP: handler.ClientIP,
		}
		if db, err := mysql.Open(rt.Cfg.MySQL, rt.Log); err != nil {
			rt.Log.Warn("mysql unavailable, promo-codes and decision-logs open endpoints disabled", zap.Error(err))
		} else {
			siteRepo := mysql.NewSiteRepo(db)
			attrRepo := mysql.NewAttributionRepo(db)
			merchantRepo := mysql.NewMerchantRepo(db)
			publisher := snapshotsvc.NewPublisher(snapshotsvc.NewBuilder(siteRepo, attrRepo, merchantRepo), rdb, rt.Log)
			attr := attrsvc.New(attrRepo, siteRepo, merchantRepo)
			attr.OnChange(publisher.PublishAsync, nil)
			od.Attr = attr
			od.Logs = logsvc.New(mysql.NewLogRepo(db), siteRepo)
		}
		openAPI = openapi.New(od)
		rt.Log.Info("open api enabled", zap.Bool("mysql", od.Attr != nil))
	}
	// 节点心跳：总后台看板展示 QPS / P99 / 回源错误率 / 版本
	if rdb != nil {
		go func() {
			started := time.Now().Format("2006-01-02 15:04:05")
			tk := time.NewTicker(15 * time.Second)
			defer tk.Stop()
			beat := func() {
				st := handler.Stats()
				c, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				_ = redis.Heartbeat(c, rdb, redis.NodeInfo{
					Node: rt.Cfg.App.NodeName, Version: version.Version, Commit: version.Commit, StartedAt: started,
					Sites: toInt(st["sites"]), QPS: toF(st["qps"]), P99Ms: toF(st["p99Ms"]), OriginErrs: toF(st["originErrorRate"]), Requests: toU(st["requests"]),
				})
			}
			beat()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tk.C:
					beat()
				}
			}
		}()
	}
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

func toF(v any) float64 {
	f, _ := v.(float64)
	return f
}

func toU(v any) uint64 {
	u, _ := v.(uint64)
	return u
}

func toInt(v any) int {
	n, _ := v.(int)
	return n
}
