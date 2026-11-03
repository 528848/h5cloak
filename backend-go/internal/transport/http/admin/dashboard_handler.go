package admin

import (
	"github.com/gin-gonic/gin"

	"cloak/internal/app/merchantsvc"
	"cloak/internal/domain/identity"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

// 看板占位：I11 实现完整指标前，先返回已有实体的计数，保证前端首页不报错。
type dashboardHandler struct {
	merchants *merchantsvc.Service
	repo      identity.MerchantRepository
}

// GET /admin/dashboard
func (h *dashboardHandler) admin(c *gin.Context) {
	stats, err := h.merchants.Stats(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{
		"todayVisits": 0, "todayWhiteHits": 0, "todayBlackHits": 0, "todayBlocks": 0, "todayApiCalls": 0,
		"merchantCount": stats["total"], "enabledMerchantCount": stats["enabled"],
		"domainCount": 0, "abnormalDomainCount": 0, "cloakCount": 0, "enabledCloakCount": 0,
	}, "查询成功")
}

// GET /merchant/dashboard
func (h *dashboardHandler) merchant(c *gin.Context) {
	u := middleware.User(c)
	m, err := h.repo.FindByID(c.Request.Context(), *u.MerchantID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{
		"todayVisits": 0, "todayWhiteHits": 0, "todayBlackHits": 0, "todayBlocks": 0, "todayApiCalls": 0,
		"domainCount": 0, "enabledDomainCount": 0, "cloakCount": 0, "enabledCloakCount": 0,
		"quota": m.QuotaOf(), "features": m.FeaturesOf(), "expireAt": m.ExpireAt,
	}, "查询成功")
}
