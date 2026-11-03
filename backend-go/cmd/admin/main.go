// 管理 API：总后台与商户端的后端。
package main

import (
	"context"
	"os"

	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/adapter/redis"
	"cloak/internal/app/attrsvc"
	"cloak/internal/app/authsvc"
	"cloak/internal/app/bootstrap"
	"cloak/internal/app/merchantsvc"
	"cloak/internal/app/sitesvc"
	"cloak/internal/app/usersvc"
	"cloak/internal/auth"
	"cloak/internal/pkg/httpserver"
	"cloak/internal/rbac"
	"cloak/internal/transport/http/admin"
	"cloak/internal/transport/http/middleware"
)

func main() {
	rt := bootstrap.Init("admin")
	defer rt.Close()

	db, err := mysql.Open(rt.Cfg.MySQL, rt.Log)
	if err != nil {
		rt.Log.Error("mysql", zap.Error(err))
		os.Exit(1)
	}
	rdb, err := redis.Open(rt.Cfg.Redis)
	if err != nil {
		// Redis 不可用时会话与锁定降级为不校验，服务仍可启动
		rt.Log.Warn("redis unavailable, sessions degrade to JWT-only", zap.Error(err))
		rdb = nil
	}

	repo := mysql.NewIdentityRepo(db)
	merchantRepo := mysql.NewMerchantRepo(db)
	rbacSvc := rbac.New(repo)
	tokens := auth.NewTokenManager(rt.Cfg.Auth.JWTSecret, rt.Cfg.Auth.TokenTTL, rdb)
	locker := auth.NewLoginLocker(rdb, rt.Cfg.Auth.LockAfter, rt.Cfg.Auth.LockDuration)
	authSvc := authsvc.New(repo, repo, rbacSvc, tokens, locker, rt.Log)
	merchantSvc := merchantsvc.New(merchantRepo, repo, repo, tokens)
	userSvc := usersvc.New(repo, repo, merchantRepo, tokens)
	siteRepo := mysql.NewSiteRepo(db)
	siteSvc := sitesvc.New(siteRepo, merchantRepo, rt.Cfg.Gateway.EntryHost, rt.Cfg.App.UploadDir, rt.Cfg.App.PublicURL)
	attrSvc := attrsvc.New(mysql.NewAttributionRepo(db), siteRepo, merchantRepo)

	router := admin.NewRouter(admin.Deps{
		Cfg: rt.Cfg, Log: rt.Log, Metrics: rt.Metrics,
		Auth:        authSvc,
		Merchants:   merchantSvc,
		Users:       userSvc,
		Sites:       siteSvc,
		Attribution: attrSvc,
		Repo:        merchantRepo,
		Authn:       middleware.NewAuthenticator(tokens, repo, rbacSvc),
		OpLog:       middleware.NewOperationLogger(repo, rt.Log),
		Upload: admin.UploadConfig{
			Dir:       rt.Cfg.App.UploadDir,
			PublicURL: rt.Cfg.App.PublicURL,
		},
	})

	srv := httpserver.New(rt.Log, rt.Cfg.HTTP, router, rt.Metrics)
	if err := srv.Run(context.Background()); err != nil {
		rt.Log.Error("exit with error", zap.Error(err))
		os.Exit(1)
	}
	rt.Log.Info("stopped")
}
