package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/telegramsvc"
	"cloak/internal/transport/http/response"
)

// telegramHandler 商户专属 Telegram 机器人配置。
type telegramHandler struct {
	svc *telegramsvc.Service
}

func (h *telegramHandler) config(c *gin.Context) {
	out, err := h.svc.Config(c.Request.Context(), merchantID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "查询成功")
}

func (h *telegramHandler) saveBot(c *gin.Context) {
	var in struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	username, err := h.svc.SaveBot(c.Request.Context(), merchantID(c), in.Token)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"botUsername": username}, "已保存")
}

func (h *telegramHandler) deleteBot(c *gin.Context) {
	if err := h.svc.DeleteBot(c.Request.Context(), merchantID(c)); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已移除")
}

func (h *telegramHandler) bindCode(c *gin.Context) {
	out, err := h.svc.NewBindCode(c.Request.Context(), merchantID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "已生成")
}

func (h *telegramHandler) chats(c *gin.Context) {
	out, err := h.svc.Chats(c.Request.Context(), merchantID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "查询成功")
}

func (h *telegramHandler) toggleChat(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.ToggleChat(c.Request.Context(), merchantID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已更新")
}

func (h *telegramHandler) deleteChat(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.DeleteChat(c.Request.Context(), merchantID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已解绑")
}

func (h *telegramHandler) saveDaily(c *gin.Context) {
	var in struct {
		Enabled bool   `json:"enabled"`
		Hour    int    `json:"hour"`
		SiteID  uint64 `json:"siteId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.SaveDaily(c.Request.Context(), merchantID(c), in.Enabled, in.Hour, in.SiteID); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已保存")
}

func (h *telegramHandler) test(c *gin.Context) {
	if err := h.svc.Test(c.Request.Context(), merchantID(c)); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已发送")
}
