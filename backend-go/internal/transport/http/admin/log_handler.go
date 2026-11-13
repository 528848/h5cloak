package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"cloak/internal/app/logsvc"
	"cloak/internal/transport/http/response"
)

type logHandler struct{ svc *logsvc.Service }

func (h *logHandler) list(c *gin.Context) {
	in := logsvc.ListInput{
		SiteID: querySiteID(c), From: c.Query("from"), To: c.Query("to"), Path: c.Query("path"),
		TouchType: c.Query("touchType"), Content: c.Query("content"), Channel: c.Query("channel"), Code: c.Query("code"),
		Reason: c.Query("reason"), Country: c.Query("country"), InApp: c.Query("inApp"), VisitorID: c.Query("visitorId"),
		RequestID: c.Query("requestId"), ShowAll: c.Query("all") == "1",
		Page: atoi(c.Query("pageNum"), 1), Size: atoi(c.Query("pageSize"), 20),
	}
	rows, total, err := h.svc.List(c.Request.Context(), merchantScope(c), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *logHandler) detail(c *gin.Context) {
	row, err := h.svc.Detail(c.Request.Context(), merchantScope(c), c.Param("requestId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

// apiLogs 接口调用日志（api_logs），商户端与总后台共用。
func (h *logHandler) apiLogs(c *gin.Context) {
	rows, total, err := h.svc.ListAPI(c.Request.Context(), merchantScope(c), querySiteID(c), c.Query("endpoint"), atoi(c.Query("pageNum"), 1), atoi(c.Query("pageSize"), 20))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *logHandler) apiDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	row, err := h.svc.APIDetail(c.Request.Context(), merchantScope(c), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}

func (h *logHandler) stats(c *gin.Context) {
	row, err := h.svc.Stats(c.Request.Context(), merchantScope(c), querySiteID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, row, "查询成功")
}
