// Package admin 组装管理 API 的路由。I0 只有健康检查与版本信息，业务路由从 I1 起逐步加入。
package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/config"
	"cloak/internal/pkg/version"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

// Deps 路由依赖，后续迭代按需扩展。
type Deps struct {
	Cfg     *config.Config
	Log     *zap.Logger
	Metrics *prometheus.Registry
}

// NewRouter 创建 gin 引擎。
func NewRouter(d Deps) *gin.Engine {
	if d.Cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Recovery(d.Log),
		middleware.Logger(d.Log),
		middleware.NewMetrics(d.Metrics, "admin").Handler(),
	)
	r.NoRoute(middleware.NoRoute())
	r.NoMethod(middleware.NoRoute())

	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	api := r.Group("/api")
	api.GET("/version", func(c *gin.Context) {
		response.Success(c, gin.H{
			"service": "admin",
			"version": version.Version,
			"commit":  version.Commit,
			"built":   version.BuildTime,
		})
	})
	return r
}
