package gateway

import (
	"sort"
	"sync"
	"time"
)

// rolling 最近 60 秒的滚动统计：按秒分桶记录请求数与延迟样本，回源错误计数。
type rolling struct {
	mu      sync.Mutex
	buckets [60]bucket
	total   uint64
}

type bucket struct {
	sec       int64
	count     uint64
	originN   uint64
	originErr uint64
	lat       []float64 // 毫秒，最多 200 个样本
}

func newRolling() *rolling { return &rolling{} }

func (r *rolling) at(now time.Time) *bucket {
	sec := now.Unix()
	b := &r.buckets[sec%60]
	if b.sec != sec {
		*b = bucket{sec: sec}
	}
	return b
}

func (r *rolling) request(took time.Duration) {
	r.mu.Lock()
	b := r.at(time.Now())
	b.count++
	r.total++
	if len(b.lat) < 200 {
		b.lat = append(b.lat, float64(took.Microseconds())/1000)
	}
	r.mu.Unlock()
}

func (r *rolling) origin(failed bool) {
	r.mu.Lock()
	b := r.at(time.Now())
	b.originN++
	if failed {
		b.originErr++
	}
	r.mu.Unlock()
}

// snapshot 返回 qps（近 60 秒均值）、p99 毫秒、回源错误率、累计请求。
func (r *rolling) snapshot() (qps, p99, errRate float64, total uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().Unix()
	var count, on, oe uint64
	var lat []float64
	for i := range r.buckets {
		b := &r.buckets[i]
		if now-b.sec >= 60 {
			continue
		}
		count += b.count
		on += b.originN
		oe += b.originErr
		lat = append(lat, b.lat...)
	}
	qps = float64(count) / 60
	if len(lat) > 0 {
		sort.Float64s(lat)
		idx := int(float64(len(lat))*0.99) - 1
		if idx < 0 {
			idx = 0
		}
		p99 = lat[idx]
	}
	if on > 0 {
		errRate = float64(oe) / float64(on)
	}
	return qps, p99, errRate, r.total
}
