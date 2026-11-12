package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/cloaksvc"
	"cloak/internal/transport/http/response"
)

type cloakHandler struct{ svc *cloaksvc.Service }

func (h *cloakHandler) stats(c *gin.Context) {
	row, err := h.svc.Stats(c.Request.Context(), merchantID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *cloakHandler) list(c *gin.Context) {
	domainID, _ := strconv.ParseUint(c.Query("domainId"), 10, 64)
	rows, total, err := h.svc.List(c.Request.Context(), merchantID(c), c.Query("keyword"), c.Query("mode"), c.Query("status"), domainID, atoi(c.Query("pageNum"), 1), atoi(c.Query("pageSize"), 20))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *cloakHandler) get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	row, err := h.svc.Get(c.Request.Context(), merchantID(c), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *cloakHandler) domainOptions(c *gin.Context) {
	keep, _ := strconv.ParseUint(c.Query("keepId"), 10, 64)
	rows, err := h.svc.DomainOptions(c.Request.Context(), merchantID(c), keep, c.Query("scope") == "all")
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *cloakHandler) create(c *gin.Context) {
	var in cloaksvc.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, 400, "请填写斗篷信息")
		return
	}
	id, err := h.svc.Create(c.Request.Context(), merchantID(c), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, ginID(id), "创建成功")
}

func (h *cloakHandler) update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in cloaksvc.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, 400, "请填写斗篷信息")
		return
	}
	if err := h.svc.Update(c.Request.Context(), merchantID(c), id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "保存成功")
}

func (h *cloakHandler) copy(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	newID, err := h.svc.Copy(c.Request.Context(), merchantID(c), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, ginID(newID), "已复制")
}

func (h *cloakHandler) changeStatus(c *gin.Context) {
	var req struct {
		ID     uint64 `json:"id"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		response.Error(c, 400, "请选择斗篷")
		return
	}
	if err := h.svc.ChangeStatus(c.Request.Context(), merchantID(c), req.ID, req.Status); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已更新")
}

func (h *cloakHandler) remove(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), merchantID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已删除")
}

func ginID(id uint64) map[string]any { return map[string]any{"id": id} }
