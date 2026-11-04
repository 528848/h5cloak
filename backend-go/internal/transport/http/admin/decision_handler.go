package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/decisionsvc"
	"cloak/internal/app/sitesvc"
	"cloak/internal/app/snapshotsvc"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

type decisionHandler struct {
	svc   *decisionsvc.Service
	sites *sitesvc.Service
	pub   *snapshotsvc.Publisher
}

func (h *decisionHandler) test(c *gin.Context) {
	var in decisionsvc.TestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.Test(c.Request.Context(), merchantScope(c), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "判定完成")
}

func (h *decisionHandler) status(c *gin.Context) {
	row, err := h.svc.Status(c.Request.Context(), merchantScope(c), querySiteID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *decisionHandler) publish(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	// 商户只能发布自己的站点
	if mid := merchantScope(c); mid > 0 {
		if _, err := h.sites.Get(c.Request.Context(), mid, id); err != nil {
			writeErr(c, err)
			return
		}
	}
	if err := h.pub.Publish(c.Request.Context(), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "快照已发布")
}

func (h *decisionHandler) publishAll(c *gin.Context) {
	n, err := h.pub.PublishAll(c.Request.Context())
	if err != nil {
		response.Fail(c, "部分站点发布失败："+err.Error())
		return
	}
	response.Success(c, gin.H{"published": n}, "已发布全部站点快照")
}

func (h *decisionHandler) rotateKey(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req struct {
		GraceHours *int `json:"graceHours"`
	}
	_ = c.ShouldBindJSON(&req)
	grace := 72
	if req.GraceHours != nil {
		grace = *req.GraceHours
	}
	row, err := h.sites.RotateTokenKey(c.Request.Context(), *middleware.User(c).MerchantID, id, grace)
	if err != nil {
		writeErr(c, err)
		return
	}
	msg := "密钥已轮换，旧令牌在宽限期内仍有效"
	if grace == 0 {
		msg = "密钥已轮换，全部访客需重新归因"
	}
	response.Success(c, row, msg)
}
