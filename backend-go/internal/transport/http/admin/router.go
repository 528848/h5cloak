// Package admin 组装管理 API 的路由（若依协议），供总后台与商户端两套前端使用。
package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cloak/internal/app/attrsvc"
	"cloak/internal/app/authsvc"
	"cloak/internal/app/decisionsvc"
	"cloak/internal/app/logsvc"
	"cloak/internal/app/merchantsvc"
	"cloak/internal/app/opssvc"
	"cloak/internal/app/platformsvc"
	"cloak/internal/app/sitesvc"
	"cloak/internal/app/snapshotsvc"
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

	Auth        *authsvc.Service
	Merchants   *merchantsvc.Service
	Users       *usersvc.Service
	Sites       *sitesvc.Service
	Attribution *attrsvc.Service
	Decision    *decisionsvc.Service
	Publisher   *snapshotsvc.Publisher
	Logs        *logsvc.Service
	Ops         *opssvc.Service
	Platform    *platformsvc.Service
	Rdb         *goredis.Client
	Repo        identity.MerchantRepository
	Authn       *middleware.Authenticator
	OpLog       *middleware.OperationLogger
	Upload      UploadConfig
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
	if d.Platform != nil {
		ticket := &platformHandler{svc: d.Platform, attr: d.Attribution, rdb: d.Rdb}
		api.POST("/login/ticket", ticket.redeemTicket)
	}

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

	if d.Ops != nil {
		oh := &opsHandler{svc: d.Ops, rdb: d.Rdb, secret: d.Cfg.Auth.JWTSecret}
		adminG.GET("/dashboard", middleware.RequirePerm("admin:dashboard:view"), oh.adminDashboard)
		merchantG.GET("/dashboard", middleware.RequirePerm("merchant:dashboard:view"), oh.merchantDashboard)
		api.GET("/export/download/:token", oh.download)

		merchantG.GET("/report/channel", middleware.RequirePerm("report:view"), oh.reportChannel)
		merchantG.GET("/report/code", middleware.RequirePerm("report:view"), oh.reportCode)
		merchantG.GET("/report/path", middleware.RequirePerm("report:view"), oh.reportPath)
		merchantG.GET("/report/conversion", middleware.RequirePerm("report:view"), oh.reportConversion)
		merchantG.GET("/report/reconcile", middleware.RequirePerm("report:view"), oh.reconcile)
		merchantG.POST("/export", middleware.RequirePerm("report:export"), oh.createExport)
		merchantG.GET("/export/list", middleware.RequirePerm("report:view"), oh.exports)
		merchantG.GET("/export/:id/link", middleware.RequirePerm("report:view"), oh.exportLink)

		merchantG.GET("/alert/rules", middleware.RequirePerm("alert:view"), oh.rules)
		merchantG.POST("/alert/rules", middleware.RequirePerm("alert:edit"), oh.saveRule)
		merchantG.DELETE("/alert/rules/:id", middleware.RequirePerm("alert:edit"), oh.deleteRule)
		merchantG.GET("/alert/list", middleware.RequirePerm("alert:view"), oh.alerts)
		merchantG.GET("/alert/stats", middleware.RequirePerm("alert:view"), oh.alertStats)
		merchantG.POST("/alert/:id/resolve", middleware.RequirePerm("alert:edit"), oh.resolveAlert)
		merchantG.GET("/notify/settings", middleware.RequirePerm("alert:view"), oh.notifySettings)
		merchantG.PUT("/notify/settings", middleware.RequirePerm("alert:edit"), oh.saveNotifySettings)
		merchantG.POST("/notify/test/:channel", middleware.RequirePerm("alert:edit"), oh.testChannel)
		merchantG.GET("/notification/list", oh.notifications)
		merchantG.GET("/notification/unread", oh.unread)
		merchantG.POST("/notification/read", oh.markRead)

		merchantG.GET("/webhook/list", middleware.RequirePerm("webhook:view"), oh.endpoints)
		merchantG.POST("/webhook", middleware.RequirePerm("webhook:edit"), oh.createEndpoint)
		merchantG.PUT("/webhook/:id", middleware.RequirePerm("webhook:edit"), oh.updateEndpoint)
		merchantG.POST("/webhook/:id/rotate", middleware.RequirePerm("webhook:edit"), oh.rotateEndpoint)
		merchantG.POST("/webhook/:id/test", middleware.RequirePerm("webhook:edit"), oh.testEndpoint)
		merchantG.DELETE("/webhook/:id", middleware.RequirePerm("webhook:edit"), oh.deleteEndpoint)
		merchantG.GET("/webhook/deliveries", middleware.RequirePerm("webhook:view"), oh.deliveries)
		merchantG.GET("/webhook/deliveries/:id", middleware.RequirePerm("webhook:view"), oh.delivery)
		merchantG.POST("/webhook/deliveries/:id/redeliver", middleware.RequirePerm("webhook:edit"), oh.redeliver)

		adminG.GET("/alert/list", middleware.RequirePerm("admin:dashboard:view"), oh.alerts)
		adminG.GET("/alert/stats", middleware.RequirePerm("admin:dashboard:view"), oh.alertStats)
		adminG.POST("/alert/:id/resolve", middleware.RequirePerm("admin:dashboard:view"), oh.resolveAlert)
		adminG.GET("/notification/list", oh.notifications)
		adminG.GET("/notification/unread", oh.unread)
		adminG.POST("/notification/read", oh.markRead)
	} else {
		dh := &dashboardHandler{merchants: d.Merchants, repo: d.Repo}
		adminG.GET("/dashboard", middleware.RequirePerm("admin:dashboard:view"), dh.admin)
		merchantG.GET("/dashboard", middleware.RequirePerm("merchant:dashboard:view"), dh.merchant)
	}

	mh := &merchantHandler{svc: d.Merchants}
	adminG.GET("/merchant/list", middleware.RequirePerm("merchant:account:list"), mh.list)
	adminG.GET("/merchant/stats", middleware.RequirePerm("merchant:account:list"), mh.stats)
	adminG.POST("/merchant", middleware.RequirePerm("merchant:account:add"), mh.create)
	adminG.PUT("/merchant/changeStatus", middleware.RequirePerm("merchant:account:status"), mh.changeStatus)
	adminG.GET("/merchant/:id", middleware.RequirePerm("merchant:account:list"), mh.get)
	adminG.PUT("/merchant/:id", middleware.RequirePerm("merchant:account:edit"), mh.update)
	adminG.PUT("/merchant/:id/quota", middleware.RequirePerm("merchant:account:quota"), mh.updateQuota)
	adminG.PUT("/merchant/:id/resetPwd", middleware.RequirePerm("merchant:account:resetPwd"), mh.resetPwd)
	if d.Platform != nil {
		ph := &platformHandler{svc: d.Platform, attr: d.Attribution, rdb: d.Rdb}
		adminG.POST("/merchant/:id/impersonate", middleware.RequirePerm("merchant:account:impersonate"), ph.impersonate)
		adminG.GET("/system/settings", middleware.RequirePerm("system:config:view"), ph.settings)
		adminG.PUT("/system/settings", middleware.RequirePerm("system:config:edit"), ph.saveSettings)
		adminG.GET("/system/nodes", middleware.RequirePerm("system:config:view"), ph.nodes)
		adminG.GET("/template/list", middleware.RequirePerm("system:template:list"), ph.templates)
		adminG.GET("/template/:id", middleware.RequirePerm("system:template:list"), ph.template)
		adminG.POST("/template", middleware.RequirePerm("system:template:edit"), ph.saveTemplate)
		adminG.PUT("/template/:id", middleware.RequirePerm("system:template:edit"), ph.saveTemplate)
		adminG.DELETE("/template/:id", middleware.RequirePerm("system:template:edit"), ph.deleteTemplate)
		adminG.GET("/code/list", middleware.RequirePerm("platform:code:list"), ph.codes)
		adminG.PUT("/code/changeStatus", middleware.RequirePerm("platform:code:status"), ph.codeStatus)
		adminG.GET("/account/list", middleware.RequirePerm("system:user:list"), ph.users)
		adminG.POST("/account", middleware.RequirePerm("system:user:edit"), ph.createUser)
		adminG.PUT("/account/changeStatus", middleware.RequirePerm("system:user:edit"), ph.userStatus)
		adminG.PUT("/account/:id", middleware.RequirePerm("system:user:edit"), ph.updateUser)
		adminG.PUT("/account/:id/resetPwd", middleware.RequirePerm("system:user:edit"), ph.resetUserPwd)
		adminG.DELETE("/account/:id", middleware.RequirePerm("system:user:edit"), ph.deleteUser)
		adminG.GET("/role/list", middleware.RequirePerm("system:role:list"), ph.roles)
		adminG.POST("/role", middleware.RequirePerm("system:role:edit"), ph.createRole)
		adminG.PUT("/role/:id", middleware.RequirePerm("system:role:edit"), ph.updateRole)
		adminG.DELETE("/role/:id", middleware.RequirePerm("system:role:edit"), ph.deleteRole)
		adminG.GET("/menu/list", middleware.RequirePerm("system:menu:list"), ph.menus)
		adminG.PUT("/menu/:id", middleware.RequirePerm("system:menu:edit"), ph.updateMenu)
	}

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

	if d.Attribution != nil {
		xh := &attributionHandler{svc: d.Attribution}
		merchantG.GET("/attribution/params", middleware.RequirePerm("attribution:rule:list"), xh.params)
		merchantG.POST("/attribution/params", middleware.RequirePerm("attribution:rule:edit"), xh.saveParam)
		merchantG.DELETE("/attribution/params/:id", middleware.RequirePerm("attribution:rule:edit"), xh.deleteParam)
		merchantG.GET("/attribution/rule", middleware.RequirePerm("attribution:rule:list"), xh.rule)
		merchantG.PUT("/attribution/rule", middleware.RequirePerm("attribution:rule:edit"), xh.saveRule)
		merchantG.GET("/code/list", middleware.RequirePerm("attribution:code:list"), xh.listCodes)
		merchantG.GET("/code/stats", middleware.RequirePerm("attribution:code:list"), xh.codeStats)
		merchantG.GET("/code/channels", middleware.RequirePerm("attribution:code:list"), xh.channels)
		merchantG.GET("/code/export", middleware.RequirePerm("attribution:code:export"), xh.exportCodes)
		merchantG.POST("/code", middleware.RequirePerm("attribution:code:add"), xh.createCode)
		merchantG.POST("/code/generate", middleware.RequirePerm("attribution:code:add"), xh.generateCodes)
		merchantG.POST("/code/import", middleware.RequirePerm("attribution:code:add"), xh.importCodes)
		merchantG.PUT("/code/changeStatus", middleware.RequirePerm("attribution:code:status"), xh.changeCodesStatus)
		merchantG.POST("/code/remove", middleware.RequirePerm("attribution:code:remove"), xh.deleteCodes)
		merchantG.PUT("/code/:id", middleware.RequirePerm("attribution:code:edit"), xh.updateCode)
		merchantG.GET("/link/options", middleware.RequirePerm("attribution:link:build"), xh.linkOptions)
		merchantG.POST("/link/build", middleware.RequirePerm("attribution:link:build"), xh.buildLink)

		adminG.GET("/param/list", middleware.RequirePerm("attribution:param:list"), xh.builtins)
		adminG.POST("/param", middleware.RequirePerm("attribution:param:edit"), xh.createBuiltin)
		adminG.PUT("/param/:id", middleware.RequirePerm("attribution:param:edit"), xh.updateBuiltin)
		adminG.DELETE("/param/:id", middleware.RequirePerm("attribution:param:edit"), xh.deleteBuiltin)
	}

	if d.Decision != nil && d.Publisher != nil && d.Sites != nil {
		dh := &decisionHandler{svc: d.Decision, sites: d.Sites, pub: d.Publisher}
		merchantG.POST("/decision/test", middleware.RequirePerm("attribution:test:run"), dh.test)
		merchantG.GET("/snapshot/status", middleware.RequirePerm("site:site:list"), dh.status)
		merchantG.POST("/snapshot/:id/publish", middleware.RequirePerm("site:site:edit"), dh.publish)
		merchantG.POST("/site/:id/rotate-key", middleware.RequirePerm("site:site:secret"), dh.rotateKey)

		adminG.GET("/snapshot/status", middleware.RequirePerm("site:site:list"), dh.status)
		adminG.POST("/snapshot/:id/publish", middleware.RequirePerm("site:site:status"), dh.publish)
		adminG.POST("/snapshot/publish-all", middleware.RequirePerm("site:site:status"), dh.publishAll)
	}

	if d.Logs != nil {
		lh := &logHandler{svc: d.Logs}
		merchantG.GET("/decision-log/list", middleware.RequirePerm("log:decision:list"), lh.list)
		merchantG.GET("/decision-log/stats", middleware.RequirePerm("log:decision:list"), lh.stats)
		merchantG.GET("/decision-log/:requestId", middleware.RequirePerm("log:decision:list"), lh.detail)
		adminG.GET("/decision-log/list", middleware.RequirePerm("log:decision:list"), lh.list)
		adminG.GET("/decision-log/stats", middleware.RequirePerm("log:decision:list"), lh.stats)
		adminG.GET("/decision-log/:requestId", middleware.RequirePerm("log:decision:list"), lh.detail)
		merchantG.GET("/api-log/list", middleware.RequirePerm("cloak:apiLog:list"), lh.apiLogs)
		adminG.GET("/api-log/list", middleware.RequirePerm("cloak:apiLog:list"), lh.apiLogs)
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
