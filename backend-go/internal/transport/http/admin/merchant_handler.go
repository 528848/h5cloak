package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/merchantsvc"
	"cloak/internal/domain/identity"
	"cloak/internal/transport/http/response"
)

type merchantHandler struct{ svc *merchantsvc.Service }

// GET /admin/merchant/list
func (h *merchantHandler) list(c *gin.Context) {
	f := identity.MerchantFilter{
		Keyword: c.Query("keyword"),
		Status:  c.Query("status"),
		Page:    atoi(c.Query("pageNum"), 1),
		Size:    atoi(c.Query("pageSize"), 10),
	}
	rows, total, err := h.svc.List(c.Request.Context(), f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

// GET /admin/merchant/stats
func (h *merchantHandler) stats(c *gin.Context) {
	s, err := h.svc.Stats(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, s, "查询成功")
}

// GET /admin/merchant/:id
func (h *merchantHandler) get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	row, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

type merchantReq struct {
	Name          string             `json:"name"`
	ContactName   string             `json:"contactName"`
	ContactEmail  string             `json:"contactEmail"`
	ContactPhone  string             `json:"contactPhone"`
	ExpireAt      string             `json:"expireAt"` // "YYYY-MM-DD HH:mm:ss"，空字符串表示长期
	Timezone      string             `json:"timezone"`
	Remark        string             `json:"remark"`
	Quota         *identity.Quota    `json:"quota"`
	Features      *identity.Features `json:"features"`
	AdminUsername string             `json:"adminUsername"`
	AdminPassword string             `json:"adminPassword"`
	AdminNickname string             `json:"adminNickname"`
}

// POST /admin/merchant
func (h *merchantHandler) create(c *gin.Context) {
	var req merchantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	expire, err := parseTime(req.ExpireAt)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "到期时间格式错误")
		return
	}
	row, err := h.svc.Create(c.Request.Context(), merchantsvc.CreateInput{
		Name: req.Name, ContactName: req.ContactName, ContactEmail: req.ContactEmail, ContactPhone: req.ContactPhone,
		ExpireAt: expire, Timezone: req.Timezone, Remark: req.Remark, Quota: req.Quota, Features: req.Features,
		AdminUsername: req.AdminUsername, AdminPassword: req.AdminPassword, AdminNickname: req.AdminNickname,
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "商户开通成功")
}

// PUT /admin/merchant/:id
func (h *merchantHandler) update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req merchantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	expire, err := parseTime(req.ExpireAt)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "到期时间格式错误")
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, merchantsvc.UpdateInput{
		Name: req.Name, ContactName: req.ContactName, ContactEmail: req.ContactEmail, ContactPhone: req.ContactPhone,
		ExpireAt: expire, ClearExpire: req.ExpireAt == "", Timezone: req.Timezone, Remark: req.Remark,
	}); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "修改成功")
}

type statusReq struct {
	MerchantID uint64 `json:"merchantId"`
	Status     string `json:"status" binding:"required"`
}

// PUT /admin/merchant/changeStatus
func (h *merchantHandler) changeStatus(c *gin.Context) {
	var req statusReq
	if err := c.ShouldBindJSON(&req); err != nil || req.MerchantID == 0 {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.ChangeStatus(c.Request.Context(), req.MerchantID, req.Status); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "状态修改成功")
}

type quotaReq struct {
	Quota    identity.Quota    `json:"quota"`
	Features identity.Features `json:"features"`
}

// PUT /admin/merchant/:id/quota
func (h *merchantHandler) updateQuota(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req quotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.UpdateQuota(c.Request.Context(), id, req.Quota, req.Features); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "额度与功能已更新")
}

type resetPwdReq struct {
	Password string `json:"password" binding:"required"`
}

// PUT /admin/merchant/:id/resetPwd
func (h *merchantHandler) resetPwd(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req resetPwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请输入新密码")
		return
	}
	username, err := h.svc.ResetAdminPassword(c.Request.Context(), id, req.Password)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"adminUsername": username}, "管理员密码已重置")
}

func pathID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "ID 无效")
		return 0, false
	}
	return id, true
}

func atoi(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func parseTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02", time.RFC3339} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, errBadTime
}

var errBadTime = &parseErr{}

type parseErr struct{}

func (*parseErr) Error() string { return "bad time" }
