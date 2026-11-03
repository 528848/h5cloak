// Package gateway 是判定网关的 HTTP 入口。I0 只提供健康检查与占位响应，
// 托管反向代理与判定接口从 I5/I6 起接入。
package gateway

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/config"
	"cloak/internal/pkg/requestid"
)

// Deps 网关依赖。
type Deps struct {
	Cfg     *config.Config
	Log     *zap.Logger
	Metrics *prometheus.Registry
}

// Handler 网关主处理器。使用标准库而非 gin，为后续反向代理留出完全控制权。
type Handler struct {
	d        Deps
	requests *prometheus.CounterVec
}

// New 创建处理器并注册指标。
func New(d Deps) *Handler {
	h := &Handler{d: d}
	h.requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gateway", Subsystem: "http", Name: "requests_total",
		Help: "Gateway requests by kind.",
	}, []string{"kind"})
	d.Metrics.MustRegister(h.requests)
	return h
}

// ServeHTTP 分发请求。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get(requestid.HeaderName)
	if id == "" || len(id) > 64 {
		id = requestid.New()
	}
	w.Header().Set(requestid.HeaderName, id)
	r = r.WithContext(requestid.WithContext(r.Context(), id))

	switch r.URL.Path {
	case "/_cloak/health":
		h.requests.WithLabelValues("health").Inc()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}

	// 占位：未知域名或尚未接入的路径统一返回 404，不暴露平台特征。
	h.requests.WithLabelValues("unmatched").Inc()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Not Found"))
}
