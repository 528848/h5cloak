// Package middleware 提供 gin 通用中间件：请求 ID、访问日志、恢复、指标。
package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"cloak/internal/pkg/requestid"
	"cloak/internal/transport/http/response"
)

// RequestID 为每个请求生成或沿用请求 ID，并写入响应头与 context。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestid.HeaderName)
		if id == "" || len(id) > 64 {
			id = requestid.New()
		}
		c.Set("request_id", id)
		c.Request = c.Request.WithContext(requestid.WithContext(c.Request.Context(), id))
		c.Header(requestid.HeaderName, id)
		c.Next()
	}
}

// Logger 记录访问日志。健康检查与指标路径不记录。
func Logger(log *zap.Logger) gin.HandlerFunc {
	skip := map[string]struct{}{"/health": {}, "/metrics": {}, "/_cloak/health": {}}
	return func(c *gin.Context) {
		if _, ok := skip[c.Request.URL.Path]; ok {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		fields := []zap.Field{
			zap.String("request_id", c.GetString("request_id")),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}
		log.Info("http", fields...)
	}
}

// Recovery 捕获 panic，返回若依格式 500，并记录堆栈。
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic",
					zap.Any("error", r),
					zap.String("request_id", c.GetString("request_id")),
					zap.String("path", c.Request.URL.Path),
					zap.Stack("stack"),
				)
				response.Fail(c, "系统内部错误")
			}
		}()
		c.Next()
	}
}

// Metrics 记录请求计数与耗时直方图。
type Metrics struct {
	requests *prometheus.CounterVec
	latency  *prometheus.HistogramVec
}

// NewMetrics 注册指标。namespace 为服务名，如 admin、gateway。
func NewMetrics(reg prometheus.Registerer, namespace string) *Metrics {
	m := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "http", Name: "requests_total",
			Help: "HTTP requests by route and status class.",
		}, []string{"route", "method", "status"}),
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "http", Name: "request_duration_seconds",
			Help:    "HTTP request latency.",
			Buckets: []float64{.002, .005, .01, .02, .05, .1, .2, .5, 1, 2, 5},
		}, []string{"route", "method"}),
	}
	reg.MustRegister(m.requests, m.latency)
	return m
}

// Handler 返回 gin 中间件。
func (m *Metrics) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status()/100) + "xx"
		m.requests.WithLabelValues(route, c.Request.Method, status).Inc()
		m.latency.WithLabelValues(route, c.Request.Method).Observe(time.Since(start).Seconds())
	}
}

// NoRoute 若依格式的 404。
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.Body{Code: http.StatusNotFound, Msg: "接口不存在"})
	}
}
