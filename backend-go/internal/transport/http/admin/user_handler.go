package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/usersvc"
	"cloak/internal/domain/identity"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

// 商户端子账号管理。
type userHandler struct{ svc *usersvc.Service }

// GET /merchant/role/list
func (h *userHandler) roles(c *gin.Context) {
	rows, err := h.svc.Roles(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

// GET /merchant/user/list
func (h *userHandler) list(c *gin.Context) {
	u := middleware.User(c)
	rows, total, err := h.svc.List(c.Request.Context(), *u.MerchantID, identity.UserFilter{
		Keyword: c.Query("keyword"), Status: c.Query("status"), RoleKey: c.Query("roleKey"),
		Page: atoi(c.Query("pageNum"), 1), Size: atoi(c.Query("pageSize"), 10),
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

type userReq struct {
	Username string `json:"userName"`
	Password string `json:"password"`
	Nickname string `json:"nickName"`
	Email    string `json:"email"`
	Phone    string `json:"phonenumber"`
	RoleKey  string `json:"roleKey"`
	Remark   string `json:"remark"`
}

func (r userReq) input() usersvc.Input {
	return usersvc.Input{Username: r.Username, Password: r.Password, Nickname: r.Nickname, Email: r.Email, Phone: r.Phone, RoleKey: r.RoleKey, Remark: r.Remark}
}

// POST /merchant/user
func (h *userHandler) create(c *gin.Context) {
	var req userReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.Create(c.Request.Context(), middleware.User(c), req.input())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "账号创建成功")
}

// PUT /merchant/user/:id
func (h *userHandler) update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req userReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.Update(c.Request.Context(), middleware.User(c), id, req.input()); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "修改成功")
}

type userStatusReq struct {
	UserID uint64 `json:"userId"`
	Status string `json:"status" binding:"required"`
}

// PUT /merchant/user/changeStatus
func (h *userHandler) changeStatus(c *gin.Context) {
	var req userStatusReq
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == 0 {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.ChangeStatus(c.Request.Context(), middleware.User(c), req.UserID, req.Status); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "状态修改成功")
}

// PUT /merchant/user/:id/resetPwd
func (h *userHandler) resetPwd(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req resetPwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请输入新密码")
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), middleware.User(c), id, req.Password); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "密码已重置")
}

// DELETE /merchant/user/:id
func (h *userHandler) remove(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), middleware.User(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "删除成功")
}
