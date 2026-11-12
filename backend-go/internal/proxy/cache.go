package proxy

import (
	"strconv"
	"sync"
	"time"
)

// reviewEntry 一条缓存的审核页（已改写、未压缩）。
type reviewEntry struct {
	status      int
	contentType string
	body        []byte
	expiresAt   time.Time
}

// reviewCache 进程内审核页缓存。审核内容对所有无归因访客相同，可跨访客复用。
// 真实页、带 Set-Cookie 的响应不进缓存。每个网关节点各自缓存，TTL 到期自动失效。
type reviewCache struct {
	mu       sync.Mutex
	m        map[string]*reviewEntry
	ttl      time.Duration
	maxBytes int64
	curBytes int64
}

func newReviewCache(ttl time.Duration, maxBytes int64) *reviewCache {
	if ttl <= 0 {
		return nil
	}
	if maxBytes <= 0 {
		maxBytes = 128 << 20
	}
	return &reviewCache{m: make(map[string]*reviewEntry), ttl: ttl, maxBytes: maxBytes}
}

func (c *reviewCache) get(key string) *reviewEntry {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.m[key]
	if e == nil {
		return nil
	}
	if time.Now().After(e.expiresAt) {
		c.curBytes -= int64(len(e.body))
		delete(c.m, key)
		return nil
	}
	return e
}

func (c *reviewCache) set(key string, status int, contentType string, body []byte) {
	if c == nil || int64(len(body)) > c.maxBytes {
		return
	}
	cp := make([]byte, len(body))
	copy(cp, body)
	e := &reviewEntry{status: status, contentType: contentType, body: cp, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Lock()
	defer c.mu.Unlock()
	if old := c.m[key]; old != nil {
		c.curBytes -= int64(len(old.body))
	}
	// 容量不足时先清过期项，再按需丢弃其它项腾空间
	if c.curBytes+int64(len(cp)) > c.maxBytes {
		c.evictLocked(int64(len(cp)))
	}
	c.m[key] = e
	c.curBytes += int64(len(cp))
}

func (c *reviewCache) evictLocked(need int64) {
	now := time.Now()
	for k, e := range c.m {
		if now.After(e.expiresAt) {
			c.curBytes -= int64(len(e.body))
			delete(c.m, k)
		}
	}
	for k, e := range c.m {
		if c.curBytes+need <= c.maxBytes {
			break
		}
		c.curBytes -= int64(len(e.body))
		delete(c.m, k)
	}
}

// reviewCacheKey 审核页缓存键：站点 + 快照版本 + 访客域名 + 协议 + 方法 + 路径查询。
// 带上快照版本，商户改配置后旧缓存自动失效。
func reviewCacheKey(siteID uint64, revision int64, host, scheme, method, pathQuery string) string {
	return strconv.FormatUint(siteID, 10) + "|" + strconv.FormatInt(revision, 10) + "|" +
		scheme + "|" + host + "|" + method + "|" + pathQuery
}
