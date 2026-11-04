package admin

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"cloak/internal/adapter/redis"
	"cloak/internal/app/opssvc"
	"cloak/internal/domain/ops"
	"cloak/internal/transport/http/middleware"
	"cloak/internal/transport/http/response"
)

// opsHandler 看板、报表、导出、告警、通知、Webhook。
type opsHandler struct {
	svc    *opssvc.Service
	rdb    *goredis.Client
	secret string // 下载链接签名
}

func (h *opsHandler) rng(c *gin.Context) opssvc.Range {
	return opssvc.ParseRange(c.Query("range"), c.Query("from"), c.Query("to"), time.Now())
}

func (h *opsHandler) reportInput(c *gin.Context) opssvc.ReportInput {
	return opssvc.ReportInput{SiteID: querySiteID(c), Range: h.rng(c), CodeType: c.Query("codeType"),
		Page: atoi(c.Query("pageNum"), 1), Size: atoi(c.Query("pageSize"), 20)}
}

// ---- 看板 ----

// GET /merchant/dashboard
func (h *opsHandler) merchantDashboard(c *gin.Context) {
	out, err := h.svc.Dashboard(c.Request.Context(), merchantID(c), querySiteID(c), h.rng(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "查询成功")
}

// GET /admin/dashboard
func (h *opsHandler) adminDashboard(c *gin.Context) {
	var nodes []map[string]any
	if list, err := redis.Nodes(c.Request.Context(), h.rdb); err == nil {
		for _, n := range list {
			nodes = append(nodes, map[string]any{"node": n.Node, "version": n.Version, "commit": n.Commit, "startedAt": n.StartedAt, "seenAt": n.SeenAt,
				"sites": n.Sites, "qps": n.QPS, "p99Ms": n.P99Ms, "originErrorRate": n.OriginErrs, "requests": n.Requests})
		}
	}
	if nodes == nil {
		nodes = []map[string]any{}
	}
	out, err := h.svc.PlatformDashboard(c.Request.Context(), h.rng(c), nodes)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "查询成功")
}

// GET /merchant/report/reconcile
func (h *opsHandler) reconcile(c *gin.Context) {
	r := h.rng(c)
	out, err := h.svc.Reconcile(c.Request.Context(), merchantScope(c), querySiteID(c), r.From, r.To)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "查询成功")
}

// ---- 报表 ----

func (h *opsHandler) reportChannel(c *gin.Context) {
	rows, err := h.svc.ChannelReport(c.Request.Context(), merchantScope(c), h.reportInput(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, rows, "查询成功")
}

func (h *opsHandler) reportCode(c *gin.Context) {
	rows, total, err := h.svc.CodeReport(c.Request.Context(), merchantScope(c), h.reportInput(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *opsHandler) reportPath(c *gin.Context) {
	rows, total, err := h.svc.PathReport(c.Request.Context(), merchantScope(c), h.reportInput(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *opsHandler) reportConversion(c *gin.Context) {
	rows, total, err := h.svc.ConversionReport(c.Request.Context(), merchantScope(c), h.reportInput(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

// ---- 导出 ----

// POST /merchant/export  { kind, siteId, range, from, to, ...filters }
func (h *opsHandler) createExport(c *gin.Context) {
	var in struct {
		Kind string `json:"kind"`
		opssvc.ExportParams
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	u := middleware.User(c)
	out, err := h.svc.CreateExport(c.Request.Context(), merchantID(c), u.ID, in.Kind, in.ExportParams)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "已加入导出队列")
}

func (h *opsHandler) exports(c *gin.Context) {
	rows, total, err := h.svc.Exports(c.Request.Context(), merchantID(c), atoi(c.Query("pageNum"), 1), atoi(c.Query("pageSize"), 20))
	if err != nil {
		writeErr(c, err)
		return
	}
	// 附带签名下载链接
	for _, r := range rows {
		if r["status"] == ops.ExportDone {
			id, _ := r["id"].(uint64)
			r["downloadToken"] = h.svc.DownloadToken(id, time.Now().Add(24*time.Hour), h.secret)
		}
	}
	response.Page(c, rows, total)
}

// GET /merchant/export/:id/link
func (h *opsHandler) exportLink(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if _, _, err := h.svc.ExportFile(c.Request.Context(), merchantID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"token": h.svc.DownloadToken(id, time.Now().Add(24*time.Hour), h.secret)}, "查询成功")
}

// GET /export/download/:token （无需登录，签名校验）
func (h *opsHandler) download(c *gin.Context) {
	id, ok := opssvc.VerifyDownloadToken(c.Param("token"), h.secret, time.Now())
	if !ok {
		c.String(http.StatusForbidden, "链接无效或已过期")
		return
	}
	e, path, err := h.svc.ExportFile(c.Request.Context(), 0, id)
	if err != nil {
		c.String(http.StatusNotFound, "文件不存在或已过期")
		return
	}
	name := e.Kind + "-" + strconv.FormatUint(e.ID, 10) + ".csv"
	c.Header("Content-Disposition", "attachment; filename=\""+name+"\"")
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.File(filepath.Clean(path))
}

// ---- 告警 ----

func (h *opsHandler) rules(c *gin.Context) {
	rules, defaults, err := h.svc.Rules(c.Request.Context(), merchantID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"rules": rules, "defaults": defaults, "types": ops.AlertTypes}, "查询成功")
}

func (h *opsHandler) saveRule(c *gin.Context) {
	var in struct {
		SiteID          uint64        `json:"siteId"`
		Type            string        `json:"type"`
		Enabled         *bool         `json:"enabled"`
		Severity        string        `json:"severity"`
		Threshold       ops.Threshold `json:"threshold"`
		WindowMinutes   int           `json:"windowMinutes"`
		CooldownMinutes int           `json:"cooldownMinutes"`
		Channels        []string      `json:"channels"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	out, err := h.svc.SaveRule(c.Request.Context(), merchantID(c), opssvc.RuleInput{SiteID: in.SiteID, Type: in.Type, Enabled: in.Enabled, Severity: in.Severity,
		Threshold: in.Threshold, WindowMinutes: in.WindowMinutes, CooldownMinutes: in.CooldownMinutes, Channels: in.Channels})
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "保存成功")
}

func (h *opsHandler) deleteRule(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteRule(c.Request.Context(), merchantID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已恢复默认")
}

func (h *opsHandler) alerts(c *gin.Context) {
	rows, total, err := h.svc.Alerts(c.Request.Context(), merchantScope(c), opssvc.AlertListInput{SiteID: querySiteID(c), Type: c.Query("type"),
		Status: c.Query("status"), Severity: c.Query("severity"), Page: atoi(c.Query("pageNum"), 1), Size: atoi(c.Query("pageSize"), 20)})
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *opsHandler) alertStats(c *gin.Context) {
	out, err := h.svc.AlertCounts(c.Request.Context(), merchantScope(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "查询成功")
}

func (h *opsHandler) resolveAlert(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.ResolveAlert(c.Request.Context(), merchantScope(c), id, middleware.User(c).ID); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已关闭")
}

// ---- 通知 ----

func (h *opsHandler) notifications(c *gin.Context) {
	rows, total, err := h.svc.Notifications(c.Request.Context(), merchantScope(c), c.Query("unread") == "1", atoi(c.Query("pageNum"), 1), atoi(c.Query("pageSize"), 20))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *opsHandler) unread(c *gin.Context) {
	n, err := h.svc.UnreadCount(c.Request.Context(), merchantScope(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"unread": n}, "查询成功")
}

func (h *opsHandler) markRead(c *gin.Context) {
	var in struct {
		IDs []uint64 `json:"ids"`
	}
	_ = c.ShouldBindJSON(&in)
	if err := h.svc.MarkRead(c.Request.Context(), merchantScope(c), in.IDs); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已读")
}

func (h *opsHandler) notifySettings(c *gin.Context) {
	out, err := h.svc.NotifySettings(c.Request.Context(), merchantID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "查询成功")
}

func (h *opsHandler) saveNotifySettings(c *gin.Context) {
	var in struct {
		Emails         []string `json:"emails"`
		TelegramChatID string   `json:"telegramChatId"`
		Channels       []string `json:"channels"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.svc.SaveNotifySettings(c.Request.Context(), merchantID(c), in.Emails, in.TelegramChatID, in.Channels); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "保存成功")
}

func (h *opsHandler) testChannel(c *gin.Context) {
	if err := h.svc.TestChannel(c.Request.Context(), merchantID(c), strings.ToLower(c.Param("channel"))); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已发送")
}

// ---- Webhook ----

func (h *opsHandler) endpoints(c *gin.Context) {
	rows, err := h.svc.Endpoints(c.Request.Context(), merchantID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows, "events": opssvc.WebhookEvents}, "查询成功")
}

type endpointBody struct {
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
}

func (h *opsHandler) createEndpoint(c *gin.Context) {
	var in endpointBody
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	view, secret, err := h.svc.SaveEndpoint(c.Request.Context(), merchantID(c), 0, opssvc.EndpointInput{Name: in.Name, URL: in.URL, Events: in.Events, Enabled: in.Enabled})
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"endpoint": view, "secret": secret}, "创建成功，密钥只显示一次")
}

func (h *opsHandler) updateEndpoint(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in endpointBody
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	view, _, err := h.svc.SaveEndpoint(c.Request.Context(), merchantID(c), id, opssvc.EndpointInput{Name: in.Name, URL: in.URL, Events: in.Events, Enabled: in.Enabled})
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, view, "保存成功")
}

func (h *opsHandler) rotateEndpoint(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	secret, err := h.svc.RotateEndpointSecret(c.Request.Context(), merchantID(c), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, gin.H{"secret": secret}, "密钥已重置，只显示一次")
}

func (h *opsHandler) deleteEndpoint(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteEndpoint(c.Request.Context(), merchantID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, nil, "已删除")
}

func (h *opsHandler) testEndpoint(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.SendTest(c.Request.Context(), merchantID(c), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "已发送")
}

func (h *opsHandler) deliveries(c *gin.Context) {
	eid, _ := strconv.ParseUint(c.Query("endpointId"), 10, 64)
	rows, total, err := h.svc.Deliveries(c.Request.Context(), merchantID(c), eid, c.Query("status"), atoi(c.Query("pageNum"), 1), atoi(c.Query("pageSize"), 20))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Page(c, rows, total)
}

func (h *opsHandler) delivery(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Delivery(c.Request.Context(), merchantID(c), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "查询成功")
}

func (h *opsHandler) redeliver(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Redeliver(c.Request.Context(), merchantID(c), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out, "已重发")
}
