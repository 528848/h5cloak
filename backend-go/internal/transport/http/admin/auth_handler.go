package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/authsvc"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

type authHandler struct {
	svc      *authsvc.Service
	captchas *captchaStore
}

type loginReq struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Portal     string `json:"portal"`
	Code       string `json:"code"`
	UUID       string `json:"uuid"`
	RememberMe bool   `json:"rememberMe"`
}

// POST /login
func (h *authHandler) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请输入账号和密码")
		return
	}
	if req.Portal == "" {
		req.Portal = "admin"
	}
	if msg := h.captchas.Verify(c.Request.Context(), req.UUID, req.Code); msg != "" {
		response.Error(c, http.StatusBadRequest, msg)
		return
	}
	token, err := h.svc.Login(c.Request.Context(), authsvc.LoginInput{
		Username: req.Username, Password: req.Password, Portal: req.Portal,
		IP: c.ClientIP(), UA: c.Request.UserAgent(),
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"token": token}, "登录成功")
}

// GET /captchaImage  登录图形验证码。总后台与商户端共用。
func (h *authHandler) captcha(c *gin.Context) {
	uuid, img, err := h.captchas.Issue(c.Request.Context())
	if err != nil {
		response.Fail(c, "验证码生成失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "msg": "操作成功",
		"captchaEnabled": true, "uuid": uuid, "img": img,
	})
}

// POST /logout
func (h *authHandler) logout(c *gin.Context) {
	if cl := middleware.Claims(c); cl != nil {
		h.svc.Logout(c.Request.Context(), cl, c.ClientIP(), c.Request.UserAgent())
	}
	response.Success(c, nil, "退出成功")
}

// GET /getInfo
func (h *authHandler) getInfo(c *gin.Context) {
	info, err := h.svc.GetInfo(c.Request.Context(), middleware.User(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "msg": "查询成功",
		"user": info.User, "roles": info.Roles, "permissions": info.Permissions,
	})
}

// GET /getRouters
func (h *authHandler) getRouters(c *gin.Context) {
	routers, err := h.svc.Routers(c.Request.Context(), middleware.User(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	if routers == nil {
		routers = []rbacItem{}
	}
	response.Success(c, routers, "查询成功")
}

func writeErr(c *gin.Context, err error) {
	if e, ok := authsvc.AsError(err); ok {
		response.Error(c, e.Code, e.Msg)
		return
	}
	response.Fail(c, "系统繁忙，请稍后重试")
}
