// Package requestid 生成与传递请求 ID。
package requestid

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

type ctxKey struct{}

// HeaderName 请求与响应中携带请求 ID 的头。
const HeaderName = "X-Request-Id"

// New 生成一个按时间排序的 ULID 请求 ID。
func New() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// WithContext 把请求 ID 放入 context。
func WithContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext 读取请求 ID，没有时返回空串。
func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}
