package engine

import (
	"context"
	"sync"
	"time"
)

// MemStore 内存实现的 Store，供单测与无 Redis 的本地开发使用。
type MemStore struct {
	mu         sync.Mutex
	now        func() time.Time
	clicks     map[string]map[string]time.Time // key -> ipPrefix -> seenAt
	handoffs   map[string]*memHandoff
	candidates map[string][]memCandidate
}

type memHandoff struct {
	ref           HandoffRef
	used, maxUses int
	expiresAt     time.Time
}

type memCandidate struct {
	ref HandoffRef
	at  time.Time
}

// NewMemStore 创建内存 Store。
func NewMemStore() *MemStore {
	return &MemStore{
		now:        time.Now,
		clicks:     map[string]map[string]time.Time{},
		handoffs:   map[string]*memHandoff{},
		candidates: map[string][]memCandidate{},
	}
}

// SetClock 单测注入时钟。
func (m *MemStore) SetClock(f func() time.Time) { m.now = f }

func (m *MemStore) ClickSeen(_ context.Context, siteID uint64, value, ipPrefix string, window time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := itoa64(siteID) + ":" + value
	set := m.clicks[key]
	if set == nil {
		set = map[string]time.Time{}
		m.clicks[key] = set
	}
	now := m.now()
	for p, t := range set {
		if now.Sub(t) > window {
			delete(set, p)
		}
	}
	set[ipPrefix] = now
	return len(set), nil
}

func (m *MemStore) HandoffIssue(_ context.Context, siteID uint64, tokenHash, attributionID, visitorID, fromRequestID string, maxUses int, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handoffs[itoa64(siteID)+":"+tokenHash] = &memHandoff{
		ref: HandoffRef{AttributionID: attributionID, VisitorID: visitorID, FromRequestID: fromRequestID}, maxUses: maxUses, expiresAt: m.now().Add(ttl),
	}
	return nil
}

func (m *MemStore) HandoffConsume(_ context.Context, siteID uint64, tokenHash string) (HandoffRef, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := m.handoffs[itoa64(siteID)+":"+tokenHash]
	if h == nil || m.now().After(h.expiresAt) || h.used >= h.maxUses {
		return HandoffRef{}, false, nil
	}
	h.used++
	return h.ref, true, nil
}

func (m *MemStore) HandoffCandidateAdd(_ context.Context, siteID uint64, ipHash, deviceFamily string, ref HandoffRef, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := itoa64(siteID) + ":" + ipHash + ":" + deviceFamily
	m.candidates[key] = append(m.candidates[key], memCandidate{ref: ref, at: m.now()})
	return nil
}

func (m *MemStore) HandoffCandidateTake(_ context.Context, siteID uint64, ipHash, deviceFamily string, window time.Duration) (HandoffRef, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := itoa64(siteID) + ":" + ipHash + ":" + deviceFamily
	list := m.candidates[key]
	now := m.now()
	for i := len(list) - 1; i >= 0; i-- {
		c := list[i]
		if now.Sub(c.at) <= window {
			m.candidates[key] = append(list[:i], list[i+1:]...)
			return c.ref, true, nil
		}
	}
	return HandoffRef{}, false, nil
}

func itoa64(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
