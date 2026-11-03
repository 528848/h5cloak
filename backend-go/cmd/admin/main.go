// 管理 API：总后台与商户端的后端。
package main

import (
	"context"
	"os"

	"go.uber.org/zap"

	"cloak/internal/app/bootstrap"
	"cloak/internal/pkg/httpserver"
	"cloak/internal/transport/http/admin"
)

func main() {
	rt := bootstrap.Init("admin")
	defer rt.Close()

	router := admin.NewRouter(admin.Deps{Cfg: rt.Cfg, Log: rt.Log, Metrics: rt.Metrics})
	srv := httpserver.New(rt.Log, rt.Cfg.HTTP, router, rt.Metrics)
	if err := srv.Run(context.Background()); err != nil {
		rt.Log.Error("exit with error", zap.Error(err))
		os.Exit(1)
	}
	rt.Log.Info("stopped")
}
