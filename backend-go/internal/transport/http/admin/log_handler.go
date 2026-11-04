package admin

import (
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
		RequestID: c.Query("requestId"), Page: atoi(c.Query("pageNum"), 1), Size: atoi(c.Query("pageSize"), 20),
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

func (h *logHandler) stats(c *gin.Context) {
	row, err := h.svc.Stats(c.Request.Context(), merchantScope(c), querySiteID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	if mid := merchantScope(c); mid > 0 {
		if n, err := h.svc.Usage(c.Request.Context(), mid); err == nil {
			row["monthUsage"] = n
		}
	}
	response.Success(c, row, "查询成功")
}
