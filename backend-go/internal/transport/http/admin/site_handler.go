package admin

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/sitesvc"
	"cloak/internal/domain/site"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

type siteHandler struct{ svc *sitesvc.Service }

func (h *siteHandler) list(c *gin.Context) {
	mid := merchantScope(c)
	// 总后台可按所属商户筛选；商户端只能看自己的站点
	if mid == 0 {
		if q := c.Query("merchantId"); q != "" {
			if id, err := strconv.ParseUint(q, 10, 64); err == nil {
				mid = id
			}
		}
	}
	rows, total, err := h.svc.List(c.Request.Context(), mid, c.Query("keyword"), c.Query("status"), c.Query("siteType"), atoi(c.Query("pageNum"), 1), atoi(c.Query("pageSize"), 10))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *siteHandler) stats(c *gin.Context) {
	row, err := h.svc.Stats(c.Request.Context(), merchantScope(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *siteHandler) options(c *gin.Context) {
	rows, err := h.svc.Options(c.Request.Context(), *middleware.User(c).MerchantID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *siteHandler) get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	row, err := h.svc.Get(c.Request.Context(), merchantScope(c), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *siteHandler) create(c *gin.Context) {
	var in sitesvc.SiteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, _, err := h.svc.Create(c.Request.Context(), *middleware.User(c).MerchantID, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "站点创建成功，请立即保存服务端密钥")
}

func (h *siteHandler) update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in sitesvc.SiteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.Update(c.Request.Context(), *middleware.User(c).MerchantID, id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "修改成功")
}

func (h *siteHandler) changeStatus(c *gin.Context) {
	var req struct {
		SiteID uint64 `json:"siteId"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SiteID == 0 {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	platform := middleware.User(c).MerchantID == nil
	var mid uint64
	if !platform {
		mid = *middleware.User(c).MerchantID
	}
	if err := h.svc.ChangeStatus(c.Request.Context(), mid, req.SiteID, req.Status, platform); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "状态修改成功")
}

func (h *siteHandler) remove(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), *middleware.User(c).MerchantID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "删除成功")
}

func (h *siteHandler) resetSecret(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req struct {
		SecretIPAllowlist []string `json:"secretIpAllowlist"`
	}
	_ = c.ShouldBindJSON(&req)
	secret, err := h.svc.ResetSecret(c.Request.Context(), *middleware.User(c).MerchantID, id, req.SecretIPAllowlist)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"serverSecret": secret}, "密钥已重置，请立即保存")
}

func (h *siteHandler) saveOrigin(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in sitesvc.OriginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.SaveOrigin(c.Request.Context(), *middleware.User(c).MerchantID, id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "源站已保存")
}

func (h *siteHandler) probeOrigin(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	row, err := h.svc.ProbeOrigin(c.Request.Context(), *middleware.User(c).MerchantID, id, c.Query("kind"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "检测完成")
}

func (h *siteHandler) saveReview(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in sitesvc.ReviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.SaveReview(c.Request.Context(), *middleware.User(c).MerchantID, id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "审核内容已保存")
}

func (h *siteHandler) templates(c *gin.Context) {
	rows, err := h.svc.Templates(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *siteHandler) previewTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var vars map[string]string
	_ = c.ShouldBindJSON(&vars)
	html, err := h.svc.PreviewTemplate(c.Request.Context(), id, vars)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func (h *siteHandler) uploadPackage(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请选择 zip 文件")
		return
	}
	f, err := file.Open()
	if err != nil {
		response.Fail(c, "读取文件失败")
		return
	}
	defer f.Close()
	ra, ok := f.(io.ReaderAt)
	if !ok {
		response.Fail(c, "无法读取压缩包")
		return
	}
	row, err := h.svc.UploadPackage(c.Request.Context(), *middleware.User(c).MerchantID, id, ra, file.Size)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "静态包已上传")
}

func (h *siteHandler) saveRules(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var rules []site.PathRule
	if err := c.ShouldBindJSON(&rules); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.ReplaceRules(c.Request.Context(), *middleware.User(c).MerchantID, id, rules); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "路径规则已保存")
}

func (h *siteHandler) guide(c *gin.Context) {
	response.Success(c, h.svc.EntryGuide(c.Request.Context()), "查询成功")
}

func (h *siteHandler) addDomain(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in sitesvc.DomainInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.AddDomain(c.Request.Context(), *middleware.User(c).MerchantID, id, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "域名已添加，请按指引完成解析")
}

func (h *siteHandler) listDomains(c *gin.Context) {
	var mid uint64
	if u := middleware.User(c); u.MerchantID != nil {
		mid = *u.MerchantID
	} else if q := c.Query("merchantId"); q != "" {
		if id, err := strconv.ParseUint(q, 10, 64); err == nil {
			mid = id
		}
	}
	siteID, _ := strconv.ParseUint(c.Query("siteId"), 10, 64)
	rows, total, err := h.svc.ListDomains(c.Request.Context(), mid, siteID, c.Query("keyword"), c.Query("status"), atoi(c.Query("pageNum"), 1), atoi(c.Query("pageSize"), 10))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *siteHandler) verifyDomain(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	platform := middleware.User(c).MerchantID == nil
	var mid uint64
	if !platform {
		mid = *middleware.User(c).MerchantID
	}
	row, err := h.svc.VerifyDomain(c.Request.Context(), mid, id, platform)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "校验完成")
}

func (h *siteHandler) removeDomain(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteDomain(c.Request.Context(), *middleware.User(c).MerchantID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "删除成功")
}

func merchantScope(c *gin.Context) uint64 {
	if u := middleware.User(c); u != nil && u.MerchantID != nil {
		return *u.MerchantID
	}
	return 0
}
