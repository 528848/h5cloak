package admin

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/authsvc"
	"cloak/internal/domain/identity"
	"cloak/internal/pkg/requestid"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

type profileHandler struct {
	svc       *authsvc.Service
	uploadDir string
	publicURL string
}

// GET /system/user/profile
func (h *profileHandler) show(c *gin.Context) {
	u := middleware.User(c)
	roleNames := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roleNames = append(roleNames, r.Name)
	}
	roleGroup := strings.Join(roleNames, "、")
	if roleGroup == "" {
		roleGroup = "-"
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "msg": "查询成功",
		"data": gin.H{
			"userId":      u.ID,
			"userName":    u.Username,
			"nickName":    identity.Deref(u.Nickname),
			"phonenumber": identity.Deref(u.Phone),
			"email":       identity.Deref(u.Email),
			"sex":         identity.Deref(u.Sex),
			"avatar":      identity.Deref(u.Avatar),
			"merchantId":  u.MerchantID,
			"createTime":  u.CreatedAt.Format("2006-01-02 15:04:05"),
		},
		"roleGroup": roleGroup,
		"postGroup": "-",
	})
}

type profileReq struct {
	NickName    string `json:"nickName"`
	PhoneNumber string `json:"phonenumber"`
	Email       string `json:"email"`
	Sex         string `json:"sex"`
}

// PUT /system/user/profile
func (h *profileHandler) update(c *gin.Context) {
	var req profileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.UpdateProfile(c.Request.Context(), middleware.User(c), authsvc.ProfileInput{
		NickName: req.NickName, PhoneNumber: req.PhoneNumber, Email: req.Email, Sex: req.Sex,
	}); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "修改成功")
}

type pwdReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// PUT /system/user/profile/updatePwd
func (h *profileHandler) updatePassword(c *gin.Context) {
	var req pwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请输入旧密码和新密码")
		return
	}
	if err := h.svc.UpdatePassword(c.Request.Context(), middleware.User(c), req.OldPassword, req.NewPassword, middleware.Claims(c)); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "修改成功")
}

// POST /system/user/profile/avatar  表单字段 avatarfile
func (h *profileHandler) avatar(c *gin.Context) {
	file, err := c.FormFile("avatarfile")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请选择图片")
		return
	}
	if file.Size > 2<<20 {
		response.Error(c, http.StatusBadRequest, "图片不能超过 2MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		response.Error(c, http.StatusBadRequest, "仅支持 jpg/png/gif/webp")
		return
	}
	dir := filepath.Join(h.uploadDir, "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		response.Fail(c, "存储目录不可用")
		return
	}
	name := requestid.New() + ext
	if err := c.SaveUploadedFile(file, filepath.Join(dir, name)); err != nil {
		response.Fail(c, "保存失败")
		return
	}
	url := fmt.Sprintf("%s/uploads/avatars/%s", strings.TrimRight(h.publicURL, "/"), name)
	if err := h.svc.UpdateAvatar(c.Request.Context(), middleware.User(c), url); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "上传成功", "imgUrl": url})
}
