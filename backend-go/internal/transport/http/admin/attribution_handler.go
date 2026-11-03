package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/attrsvc"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

type attributionHandler struct{ svc *attrsvc.Service }

func merchantID(c *gin.Context) uint64 { return *middleware.User(c).MerchantID }

func querySiteID(c *gin.Context) uint64 {
	id, _ := strconv.ParseUint(c.Query("siteId"), 10, 64)
	return id
}

// ---- 商户端：参数字典 ----

func (h *attributionHandler) params(c *gin.Context) {
	rows, err := h.svc.Params(c.Request.Context(), merchantID(c), querySiteID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *attributionHandler) saveParam(c *gin.Context) {
	var in struct {
		SiteID uint64 `json:"siteId"`
		attrsvc.ParamInput
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.SaveParam(c.Request.Context(), merchantID(c), in.SiteID, in.ParamInput)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "参数已保存")
}

func (h *attributionHandler) deleteParam(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteParam(c.Request.Context(), merchantID(c), querySiteID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已删除")
}

// ---- 商户端：规则 ----

func (h *attributionHandler) rule(c *gin.Context) {
	row, err := h.svc.Rule(c.Request.Context(), merchantID(c), querySiteID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *attributionHandler) saveRule(c *gin.Context) {
	var in struct {
		SiteID uint64 `json:"siteId"`
		attrsvc.RuleInput
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.SaveRule(c.Request.Context(), merchantID(c), in.SiteID, in.RuleInput); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "规则已保存")
}

// ---- 商户端：分销码 ----

func codeListInput(c *gin.Context) attrsvc.CodeListInput {
	return attrsvc.CodeListInput{
		SiteID: querySiteID(c), Keyword: c.Query("keyword"), CodeType: c.Query("codeType"),
		Status: c.Query("status"), Channel: c.Query("channel"),
		Page: atoi(c.Query("pageNum"), 1), Size: atoi(c.Query("pageSize"), 10),
	}
}

func (h *attributionHandler) listCodes(c *gin.Context) {
	rows, total, err := h.svc.ListCodes(c.Request.Context(), merchantID(c), codeListInput(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *attributionHandler) codeStats(c *gin.Context) {
	row, err := h.svc.CodeStats(c.Request.Context(), merchantID(c), querySiteID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *attributionHandler) channels(c *gin.Context) {
	rows, err := h.svc.Channels(c.Request.Context(), merchantID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *attributionHandler) createCode(c *gin.Context) {
	var in attrsvc.CodeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.CreateCode(c.Request.Context(), merchantID(c), middleware.User(c).ID, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "登记成功")
}

func (h *attributionHandler) updateCode(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in attrsvc.CodeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.UpdateCode(c.Request.Context(), merchantID(c), id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "修改成功")
}

func (h *attributionHandler) changeCodesStatus(c *gin.Context) {
	var req struct {
		CodeIDs []uint64 `json:"codeIds"`
		Status  string   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.CodeIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	n, err := h.svc.ChangeCodesStatus(c.Request.Context(), merchantID(c), req.CodeIDs, req.Status)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"affected": n}, fmt.Sprintf("已更新 %d 个码", n))
}

func (h *attributionHandler) deleteCodes(c *gin.Context) {
	var req struct {
		CodeIDs []uint64 `json:"codeIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.CodeIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	n, err := h.svc.DeleteCodes(c.Request.Context(), merchantID(c), req.CodeIDs)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"affected": n}, fmt.Sprintf("已删除 %d 个码", n))
}

func (h *attributionHandler) generateCodes(c *gin.Context) {
	var in attrsvc.GenerateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.GenerateCodes(c.Request.Context(), merchantID(c), middleware.User(c).ID, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, fmt.Sprintf("已生成 %v 个码", row["created"]))
}

func (h *attributionHandler) importCodes(c *gin.Context) {
	var in attrsvc.ImportInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.ImportCodes(c.Request.Context(), merchantID(c), middleware.User(c).ID, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, fmt.Sprintf("导入 %v 个，跳过 %v 个", row["created"], row["skipped"]))
}

func (h *attributionHandler) exportCodes(c *gin.Context) {
	data, err := h.svc.ExportCodes(c.Request.Context(), merchantID(c), codeListInput(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	name := "promo-codes-" + time.Now().Format("20060102-150405") + ".csv"
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

// ---- 商户端：链接生成 ----

func (h *attributionHandler) linkOptions(c *gin.Context) {
	row, err := h.svc.LinkOptions(c.Request.Context(), merchantID(c), querySiteID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *attributionHandler) buildLink(c *gin.Context) {
	var in attrsvc.LinkInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.BuildLink(c.Request.Context(), merchantID(c), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "生成成功")
}

// ---- 总后台：内置字典 ----

func (h *attributionHandler) builtins(c *gin.Context) {
	rows, err := h.svc.Builtins(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *attributionHandler) createBuiltin(c *gin.Context) {
	var in attrsvc.ParamInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.CreateBuiltin(c.Request.Context(), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "新增成功")
}

func (h *attributionHandler) updateBuiltin(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in attrsvc.ParamInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.UpdateBuiltin(c.Request.Context(), id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "修改成功")
}

func (h *attributionHandler) deleteBuiltin(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteBuiltin(c.Request.Context(), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "删除成功")
}
