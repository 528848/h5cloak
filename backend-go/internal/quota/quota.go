// Package quota 网关侧判定计量与额度策略。
// 每次判定 INCR Redis usage:{merchant}:{yyyymm}；是否超额用本地缓存（5 秒刷新）判断，不增加请求延迟。
package quota

import (
	"context"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// 策略
const (
	PolicyAlert     = "alert"      // 只告警，照常放行
	PolicyReviewAll = "review_all" // 全部返回审核内容
	PolicyDeny      = "deny"       // 拒绝服务
)

// Checker 计量器。rdb 为 nil 时不计量、不限额。
type Checker struct {
	rdb   *goredis.Client
	mu    sync.Mutex
	cache map[uint64]cached
	now   func() time.Time
}

type cached struct {
	value uint64
	at    time.Time
}

func New(rdb *goredis.Client) *Checker {
	return &Checker{rdb: rdb, cache: map[uint64]cached{}, now: time.Now}
}

// Key 当月计数 key。
func Key(merchantID uint64, t time.Time) string {
	return fmt.Sprintf("usage:%d:%s", merchantID, t.Format("200601"))
}

// Count 异步 +1，返回后不等待结果。
func (c *Checker) Count(merchantID uint64) {
	if c == nil || c.rdb == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		key := Key(merchantID, c.now())
		pipe := c.rdb.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, 40*24*time.Hour)
		_, _ = pipe.Exec(ctx)
	}()
}

// Usage 当月已用（带 5 秒本地缓存）。
func (c *Checker) Usage(ctx context.Context, merchantID uint64) uint64 {
	if c == nil || c.rdb == nil {
		return 0
	}
	c.mu.Lock()
	v, ok := c.cache[merchantID]
	c.mu.Unlock()
	now := c.now()
	if ok && now.Sub(v.at) < 5*time.Second {
		return v.value
	}
	n, err := c.rdb.Get(ctx, Key(merchantID, now)).Uint64()
	if err != nil {
		if ok {
			return v.value
		}
		return 0
	}
	c.mu.Lock()
	c.cache[merchantID] = cached{value: n, at: now}
	c.mu.Unlock()
	return n
}

// Over 是否已超额。limit 为 0 不限。
func (c *Checker) Over(ctx context.Context, merchantID, limit uint64) bool {
	if limit == 0 {
		return false
	}
	return c.Usage(ctx, merchantID) >= limit
}
