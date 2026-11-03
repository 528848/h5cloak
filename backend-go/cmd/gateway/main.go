// 判定网关：托管反向代理、开放判定接口、JS 上报。
package main

import (
	"context"
	"os"

	"go.uber.org/zap"

	"cloak/internal/app/bootstrap"
	"cloak/internal/pkg/httpserver"
	"cloak/internal/transport/http/gateway"
)

func main() {
	rt := bootstrap.Init("gateway")
	defer rt.Close()

	handler := gateway.New(gateway.Deps{Cfg: rt.Cfg, Log: rt.Log, Metrics: rt.Metrics})
	srv := httpserver.New(rt.Log, rt.Cfg.HTTP, handler, rt.Metrics)
	if err := srv.Run(context.Background()); err != nil {
		rt.Log.Error("exit with error", zap.Error(err))
		os.Exit(1)
	}
	rt.Log.Info("stopped")
}
