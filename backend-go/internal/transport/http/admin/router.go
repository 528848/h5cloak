// Package admin 组装管理 API 的路由（若依协议），供总后台与商户端两套前端使用。
package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/app/authsvc"
	"cloak/internal/app/merchantsvc"
	"cloak/internal/app/sitesvc"
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
	Sites     *sitesvc.Service
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

	if d.Sites != nil {
		sh := &siteHandler{svc: d.Sites}
		merchantG.GET("/site/list", middleware.RequirePerm("site:site:list"), sh.list)
		merchantG.POST("/site", middleware.RequirePerm("site:site:add"), sh.create)
		merchantG.GET("/site/stats", middleware.RequirePerm("site:site:list"), sh.stats)
		merchantG.GET("/site/options", middleware.RequirePerm("site:site:list"), sh.options)
		merchantG.GET("/site/guide", middleware.RequirePerm("site:site:list"), sh.guide)
		merchantG.GET("/site/templates", middleware.RequirePerm("site:site:list"), sh.templates)
		merchantG.POST("/site/templates/:id/preview", middleware.RequirePerm("site:site:list"), sh.previewTemplate)
		merchantG.GET("/site/:id", middleware.RequirePerm("site:site:list"), sh.get)
		merchantG.PUT("/site/:id", middleware.RequirePerm("site:site:edit"), sh.update)
		merchantG.DELETE("/site/:id", middleware.RequirePerm("site:site:remove"), sh.remove)
		merchantG.PUT("/site/changeStatus", middleware.RequirePerm("site:site:status"), sh.changeStatus)
		merchantG.POST("/site/:id/secret", middleware.RequirePerm("site:site:secret"), sh.resetSecret)
		merchantG.PUT("/site/:id/origin", middleware.RequirePerm("site:site:edit"), sh.saveOrigin)
		merchantG.POST("/site/:id/origin/probe", middleware.RequirePerm("site:site:edit"), sh.probeOrigin)
		merchantG.PUT("/site/:id/review", middleware.RequirePerm("site:site:edit"), sh.saveReview)
		merchantG.POST("/site/:id/package", middleware.RequirePerm("site:site:edit"), sh.uploadPackage)
		merchantG.PUT("/site/:id/path-rules", middleware.RequirePerm("site:site:edit"), sh.saveRules)
		merchantG.GET("/domain/list", middleware.RequirePerm("site:domain:list"), sh.listDomains)
		merchantG.POST("/site/:id/domain", middleware.RequirePerm("site:domain:add"), sh.addDomain)
		merchantG.POST("/domain/:id/verify", middleware.RequirePerm("site:domain:verify"), sh.verifyDomain)
		merchantG.DELETE("/domain/:id", middleware.RequirePerm("site:domain:remove"), sh.removeDomain)

		adminG.GET("/site/list", middleware.RequirePerm("site:site:list"), sh.list)
		adminG.GET("/site/stats", middleware.RequirePerm("site:site:list"), sh.stats)
		adminG.PUT("/site/changeStatus", middleware.RequirePerm("site:site:status"), sh.changeStatus)
		adminG.GET("/domain/list", middleware.RequirePerm("site:domain:list"), sh.listDomains)
		adminG.POST("/domain/:id/verify", middleware.RequirePerm("site:domain:verify"), sh.verifyDomain)
	}

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
