package auth

import (
	"context"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// LoginLocker 连续失败锁定。Redis 不可用时降级为不锁定。
type LoginLocker struct {
	rdb       *goredis.Client
	threshold int
	duration  time.Duration
}

func NewLoginLocker(rdb *goredis.Client, threshold int, duration time.Duration) *LoginLocker {
	if threshold <= 0 {
		threshold = 5
	}
	if duration <= 0 {
		duration = 15 * time.Minute
	}
	return &LoginLocker{rdb: rdb, threshold: threshold, duration: duration}
}

// Locked 返回剩余锁定时长，0 表示未锁定。
func (l *LoginLocker) Locked(ctx context.Context, username string) time.Duration {
	if l.rdb == nil {
		return 0
	}
	ttl, err := l.rdb.TTL(ctx, lockKey(username)).Result()
	if err != nil || ttl <= 0 {
		return 0
	}
	return ttl
}

// Fail 记一次失败，达到阈值时锁定并返回 true。
func (l *LoginLocker) Fail(ctx context.Context, username string) (locked bool, remaining int) {
	if l.rdb == nil {
		return false, 0
	}
	n, err := l.rdb.Incr(ctx, failKey(username)).Result()
	if err != nil {
		return false, 0
	}
	l.rdb.Expire(ctx, failKey(username), l.duration)
	if int(n) >= l.threshold {
		l.rdb.Set(ctx, lockKey(username), 1, l.duration)
		l.rdb.Del(ctx, failKey(username))
		return true, 0
	}
	return false, l.threshold - int(n)
}

// Reset 登录成功后清零。
func (l *LoginLocker) Reset(ctx context.Context, username string) {
	if l.rdb == nil {
		return
	}
	l.rdb.Del(ctx, failKey(username), lockKey(username))
}

func failKey(u string) string { return "login:fail:" + strings.ToLower(u) }
func lockKey(u string) string { return "login:lock:" + strings.ToLower(u) }
