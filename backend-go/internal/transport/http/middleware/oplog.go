package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"cloak/internal/domain/identity"
)

// OperationLogger 记录登录用户的写操作（POST/PUT/PATCH/DELETE）。
// 模块与动作从路由推导：/api/merchant/site/123 → module=merchant.site action=PUT。
// 请求体保留前 2KB，密码类字段脱敏。写入异步，不影响响应。
type OperationLogger struct {
	logs identity.LogRepository
	log  *zap.Logger
}

func NewOperationLogger(logs identity.LogRepository, log *zap.Logger) *OperationLogger {
	return &OperationLogger{logs: logs, log: log}
}

const maxBodyCapture = 2048

var sensitiveKeys = []string{"password", "pwd", "secret", "token"}

type bodyWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	if w.buf.Len() < 512 {
		w.buf.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (o *OperationLogger) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		start := time.Now()
		var reqBody []byte
		if c.Request.Body != nil && !strings.HasPrefix(c.ContentType(), "multipart/") {
			reqBody, _ = io.ReadAll(io.LimitReader(c.Request.Body, maxBodyCapture+1))
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(reqBody), c.Request.Body))
		}
		bw := &bodyWriter{ResponseWriter: c.Writer}
		c.Writer = bw
		c.Next()

		user := User(c)
		if user == nil {
			return
		}
		module, action, objectID := deriveModule(c)
		entry := &identity.OperationLog{
			RequestID:      identity.Ptr(c.GetString("request_id")),
			ActorUserID:    &user.ID,
			ActorUsername:  &user.Username,
			ActorRole:      identity.Ptr(strings.Join(roleKeys(user), ",")),
			Portal:         identity.Ptr(user.Portal()),
			MerchantID:     user.MerchantID,
			Module:         module,
			Action:         action,
			Method:         c.Request.Method,
			Path:           c.Request.URL.Path,
			ObjectID:       identity.Ptr(objectID),
			RequestSummary: summarize(reqBody),
			StatusCode:     c.Writer.Status(),
			ResultCode:     resultCode(bw.buf.Bytes(), c.Writer.Status()),
			IP:             identity.Ptr(c.ClientIP()),
			LatencyMs:      uint32(time.Since(start).Milliseconds()),
			CreatedAt:      time.Now(),
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := o.logs.WriteOperation(ctx, entry); err != nil {
				o.log.Warn("write operation log failed", zap.Error(err))
			}
		}()
	}
}

func deriveModule(c *gin.Context) (module, action, objectID string) {
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/")
	parts := strings.Split(path, "/")
	var segs []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		if isID(p) {
			if objectID == "" {
				objectID = p
			}
			continue
		}
		segs = append(segs, p)
	}
	action = strings.ToLower(c.Request.Method)
	if len(segs) > 2 {
		action = segs[len(segs)-1]
		segs = segs[:len(segs)-1]
	}
	if len(segs) == 0 {
		return "unknown", action, objectID
	}
	return strings.Join(segs, "."), action, objectID
}

func isID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func summarize(body []byte) json.RawMessage {
	if len(body) == 0 {
		return nil
	}
	if len(body) > maxBodyCapture {
		body = body[:maxBodyCapture]
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		s, _ := json.Marshal(map[string]any{"raw": string(body)})
		return s
	}
	for k := range m {
		lk := strings.ToLower(k)
		for _, sk := range sensitiveKeys {
			if strings.Contains(lk, sk) {
				m[k] = "***"
				break
			}
		}
	}
	out, _ := json.Marshal(m)
	return out
}

func resultCode(body []byte, status int) int {
	var r struct {
		Code int `json:"code"`
	}
	if json.Unmarshal(body, &r) == nil && r.Code != 0 {
		return r.Code
	}
	return status
}

func roleKeys(u *identity.User) []string {
	out := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		out = append(out, r.RoleKey)
	}
	return out
}
