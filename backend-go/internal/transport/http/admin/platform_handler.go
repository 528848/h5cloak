package admin

import (
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"cloak/internal/adapter/redis"
	"cloak/internal/app/attrsvc"
	"cloak/internal/app/platformsvc"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

type platformHandler struct {
	svc  *platformsvc.Service
	attr *attrsvc.Service
	rdb  *goredis.Client
}

func (h *platformHandler) settings(c *gin.Context) {
	row, err := h.svc.GetSettings(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *platformHandler) saveSettings(c *gin.Context) {
	var in platformsvc.SettingsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数格式无效")
		return
	}
	if err := h.svc.SaveSettings(c.Request.Context(), in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已保存，网关将在数秒内生效")
}

func (h *platformHandler) nodes(c *gin.Context) {
	rows, err := redis.Nodes(c.Request.Context(), h.rdb)
	if err != nil {
		writeErr(c, err)
		return
	}
	if rows == nil {
		rows = []redis.NodeInfo{}
	}
	response.Success(c, rows, "查询成功")
}

func (h *platformHandler) templates(c *gin.Context) {
	rows, err := h.svc.ListTemplates(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *platformHandler) template(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	row, err := h.svc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *platformHandler) saveTemplate(c *gin.Context) {
	var in platformsvc.TemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数格式无效")
		return
	}
	var id uint64
	if c.Param("id") != "" {
		var ok bool
		id, ok = pathID(c)
		if !ok {
			return
		}
	}
	nid, err := h.svc.SaveTemplate(c.Request.Context(), id, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"templateId": nid}, "保存成功")
}

func (h *platformHandler) deleteTemplate(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteTemplate(c.Request.Context(), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "删除成功")
}

func (h *platformHandler) codes(c *gin.Context) {
	if h.attr == nil {
		response.Fail(c, "分销码服务不可用")
		return
	}
	rows, total, err := h.attr.ListCodes(c.Request.Context(), 0, attrsvc.CodeListInput{
		SiteID: querySiteID(c), Keyword: c.Query("keyword"), Status: c.Query("status"),
		Page: atoi(c.Query("pageNum"), 1), Size: atoi(c.Query("pageSize"), 20),
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *platformHandler) codeStatus(c *gin.Context) {
	var req struct {
		IDs    []uint64 `json:"ids"`
		Status string   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		response.Fail(c, "请选择分销码")
		return
	}
	n, err := h.attr.PlatformChangeCodesStatus(c.Request.Context(), req.IDs, req.Status)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"updated": n}, "状态已更新")
}

func (h *platformHandler) users(c *gin.Context) {
	rows, total, err := h.svc.ListUsers(c.Request.Context(), c.Query("keyword"), c.Query("status"), atoi(c.Query("pageNum"), 1), atoi(c.Query("pageSize"), 20))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *platformHandler) createUser(c *gin.Context) {
	var in platformsvc.UserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数格式无效")
		return
	}
	id, err := h.svc.CreateUser(c.Request.Context(), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"userId": id}, "创建成功")
}

func (h *platformHandler) updateUser(c *gin.Context) {
	var in platformsvc.UserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数格式无效")
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.UpdateUser(c.Request.Context(), actorID(c), id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "保存成功")
}

func (h *platformHandler) userStatus(c *gin.Context) {
	var req struct {
		UserID uint64 `json:"userId"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == 0 {
		response.Fail(c, "参数格式无效")
		return
	}
	if err := h.svc.ChangeUserStatus(c.Request.Context(), actorID(c), req.UserID, req.Status); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "状态已更新")
}

func (h *platformHandler) resetUserPwd(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数格式无效")
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.ResetUserPassword(c.Request.Context(), id, req.Password); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "密码已重置")
}

func (h *platformHandler) deleteUser(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteUser(c.Request.Context(), actorID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "删除成功")
}

func (h *platformHandler) roles(c *gin.Context) {
	rows, err := h.svc.ListRoles(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *platformHandler) createRole(c *gin.Context) {
	var in platformsvc.RoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数格式无效")
		return
	}
	id, err := h.svc.CreateRole(c.Request.Context(), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"roleId": id}, "创建成功")
}

func (h *platformHandler) updateRole(c *gin.Context) {
	var in platformsvc.RoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数格式无效")
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.UpdateRole(c.Request.Context(), id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "保存成功")
}

func (h *platformHandler) deleteRole(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteRole(c.Request.Context(), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "删除成功")
}

func (h *platformHandler) menus(c *gin.Context) {
	rows, err := h.svc.ListMenus(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *platformHandler) updateMenu(c *gin.Context) {
	var in platformsvc.MenuInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, "参数格式无效")
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.UpdateMenu(c.Request.Context(), id, in); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "保存成功")
}

func (h *platformHandler) impersonate(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	url, err := h.svc.Impersonate(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"url": url}, "已生成登录链接，60 秒内有效")
}

func (h *platformHandler) redeemTicket(c *gin.Context) {
	var req struct {
		Ticket string `json:"ticket"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 401, "登录票据无效或已过期")
		return
	}
	token, err := h.svc.RedeemTicket(c.Request.Context(), req.Ticket, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"token": token}, "登录成功")
}

func actorID(c *gin.Context) uint64 {
	if u := middleware.User(c); u != nil {
		return u.ID
	}
	return 0
}
