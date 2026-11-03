// Package admin 组装管理 API 的路由（若依协议），供总后台与商户端两套前端使用。
package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/app/authsvc"
	"cloak/internal/app/merchantsvc"
	"cloak/internal/app/usersvc"
	"cloak/internal/config"
	"cloak/internal/domain/identity"
	"cloak/internal/pkg/version"
	"cloak/internal/rbac"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

type rbacItem = rbac.RouterItem

// Deps 路由依赖。
type Deps struct {
	Cfg     *config.Config
	Log     *zap.Logger
	Metrics *prometheus.Registry

	Auth      *authsvc.Service
	Merchants *merchantsvc.Service
	Users     *usersvc.Service
	Repo      identity.MerchantRepository
	Authn     *middleware.Authenticator
	OpLog     *middleware.OperationLogger
	Upload    UploadConfig
}

// UploadConfig 本地上传目录与对外地址。
type UploadConfig struct {
	Dir       string
	PublicURL string
}

// NewRouter 创建 gin 引擎。Auth 为空时只注册系统端点（I0 兼容与单测）。
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
		middleware.CORS(),
	)
	r.NoRoute(middleware.NoRoute())
	r.NoMethod(middleware.NoRoute())

	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	api := r.Group("/api")
	api.GET("/version", func(c *gin.Context) {
		response.Success(c, gin.H{
			"service": "admin", "version": version.Version, "commit": version.Commit, "built": version.BuildTime,
		})
	})

	if d.Auth == nil {
		return r
	}

	if d.Upload.Dir != "" {
		r.Static("/uploads", d.Upload.Dir)
	}

	ah := &authHandler{svc: d.Auth}
	ph := &profileHandler{svc: d.Auth, uploadDir: d.Upload.Dir, publicURL: d.Upload.PublicURL}

	api.POST("/login", ah.login)
	api.GET("/captchaImage", ah.captcha)

	authed := api.Group("", d.Authn.Required(), d.OpLog.Handler())
	authed.POST("/logout", ah.logout)
	authed.GET("/getInfo", ah.getInfo)
	authed.GET("/getRouters", ah.getRouters)

	system := authed.Group("/system")
	system.GET("/user/profile", ph.show)
	system.PUT("/user/profile", ph.update)
	system.PUT("/user/profile/updatePwd", ph.updatePassword)
	system.POST("/user/profile/avatar", ph.avatar)
	system.GET("/dict/data/type/:type", dictByType)

	adminG := authed.Group("/admin", middleware.RequirePortal(identity.PortalAdmin))
	merchantG := authed.Group("/merchant", middleware.RequirePortal(identity.PortalMerchant))

	dh := &dashboardHandler{merchants: d.Merchants, repo: d.Repo}
	adminG.GET("/dashboard", middleware.RequirePerm("admin:dashboard:view"), dh.admin)
	merchantG.GET("/dashboard", middleware.RequirePerm("merchant:dashboard:view"), dh.merchant)

	mh := &merchantHandler{svc: d.Merchants}
	adminG.GET("/merchant/list", middleware.RequirePerm("merchant:account:list"), mh.list)
	adminG.GET("/merchant/stats", middleware.RequirePerm("merchant:account:list"), mh.stats)
	adminG.POST("/merchant", middleware.RequirePerm("merchant:account:add"), mh.create)
	adminG.PUT("/merchant/changeStatus", middleware.RequirePerm("merchant:account:status"), mh.changeStatus)
	adminG.GET("/merchant/:id", middleware.RequirePerm("merchant:account:list"), mh.get)
	adminG.PUT("/merchant/:id", middleware.RequirePerm("merchant:account:edit"), mh.update)
	adminG.PUT("/merchant/:id/quota", middleware.RequirePerm("merchant:account:quota"), mh.updateQuota)
	adminG.PUT("/merchant/:id/resetPwd", middleware.RequirePerm("merchant:account:resetPwd"), mh.resetPwd)

	uh := &userHandler{svc: d.Users}
	merchantG.GET("/role/list", middleware.RequirePerm("merchant:user:list"), uh.roles)
	merchantG.GET("/user/list", middleware.RequirePerm("merchant:user:list"), uh.list)
	merchantG.POST("/user", middleware.RequirePerm("merchant:user:add"), uh.create)
	merchantG.PUT("/user/changeStatus", middleware.RequirePerm("merchant:user:status"), uh.changeStatus)
	merchantG.PUT("/user/:id", middleware.RequirePerm("merchant:user:edit"), uh.update)
	merchantG.PUT("/user/:id/resetPwd", middleware.RequirePerm("merchant:user:resetPwd"), uh.resetPwd)
	merchantG.DELETE("/user/:id", middleware.RequirePerm("merchant:user:remove"), uh.remove)

	return r
}
