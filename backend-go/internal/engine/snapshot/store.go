package snapshot

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Loader 从外部（Redis）加载快照。
type Loader interface {
	LoadSite(ctx context.Context, siteID uint64) (*Site, error)
	ResolveHost(ctx context.Context, host string) (uint64, error)
	ResolvePrefix(ctx context.Context, prefix string) (uint64, error)
	ResolvePublishableKey(ctx context.Context, pk string) (uint64, error)
}

// Store 网关内存快照仓库：按站点 ID 常驻，Host → 站点 ID 缓存，支持增量替换。
type Store struct {
	loader   Loader
	sites    sync.Map // uint64 -> *Site
	hosts    sync.Map // string -> uint64
	prefixes sync.Map // secretPrefix -> uint64
	loads    atomic.Int64
	misses   atomic.Int64
	// 未命中的 host 短期负缓存，避免被不存在的域名打穿
	negative sync.Map // string -> time.Time
}

// NewStore 创建仓库。loader 可为 nil（只用 Put 注入，单测用）。
func NewStore(loader Loader) *Store { return &Store{loader: loader} }

// Put 直接放入快照（测试或启动预热）。
func (s *Store) Put(site *Site) {
	_ = site.Compile()
	s.sites.Store(site.SiteID, site)
	for _, d := range site.Domains {
		if d.Status == "bound" {
			s.hosts.Store(d.Domain, site.SiteID)
		}
	}
}

// Get 按站点 ID 取快照，未命中时懒加载。
func (s *Store) Get(ctx context.Context, siteID uint64) (*Site, bool) {
	if v, ok := s.sites.Load(siteID); ok {
		return v.(*Site), true
	}
	if s.loader == nil {
		return nil, false
	}
	site, err := s.loader.LoadSite(ctx, siteID)
	if err != nil || site == nil {
		s.misses.Add(1)
		return nil, false
	}
	s.loads.Add(1)
	s.sites.Store(siteID, site)
	return site, true
}

// ByHost 按访问域名取快照。
func (s *Store) ByHost(ctx context.Context, host string) (*Site, bool) {
	host = normalizeHost(host)
	if host == "" {
		return nil, false
	}
	if v, ok := s.hosts.Load(host); ok {
		return s.Get(ctx, v.(uint64))
	}
	if t, ok := s.negative.Load(host); ok && time.Since(t.(time.Time)) < 30*time.Second {
		return nil, false
	}
	if s.loader == nil {
		return nil, false
	}
	id, err := s.loader.ResolveHost(ctx, host)
	if err != nil || id == 0 {
		s.negative.Store(host, time.Now())
		return nil, false
	}
	s.hosts.Store(host, id)
	return s.Get(ctx, id)
}

// ByPrefix 按服务端密钥前缀取快照（开放接口鉴权）。
func (s *Store) ByPrefix(ctx context.Context, prefix string) (*Site, bool) {
	if prefix == "" {
		return nil, false
	}
	if v, ok := s.prefixes.Load(prefix); ok {
		if site, ok := s.Get(ctx, v.(uint64)); ok && site.SecretPrefix == prefix {
			return site, true
		}
		s.prefixes.Delete(prefix)
	}
	if s.loader == nil {
		// 无 loader：遍历已加载站点
		var found *Site
		s.sites.Range(func(_, v any) bool {
			if v.(*Site).SecretPrefix == prefix {
				found = v.(*Site)
				return false
			}
			return true
		})
		return found, found != nil
	}
	id, err := s.loader.ResolvePrefix(ctx, prefix)
	if err != nil || id == 0 {
		return nil, false
	}
	site, ok := s.Get(ctx, id)
	if ok && site.SecretPrefix == prefix {
		s.prefixes.Store(prefix, id)
		return site, true
	}
	return nil, false
}

// ByPublishableKey 按公钥取快照（JS SDK）。
func (s *Store) ByPublishableKey(ctx context.Context, pk string) (*Site, bool) {
	if pk == "" {
		return nil, false
	}
	var found *Site
	s.sites.Range(func(_, v any) bool {
		if v.(*Site).PublishableKey == pk {
			found = v.(*Site)
			return false
		}
		return true
	})
	if found != nil {
		return found, true
	}
	if s.loader == nil {
		return nil, false
	}
	id, err := s.loader.ResolvePublishableKey(ctx, pk)
	if err != nil || id == 0 {
		return nil, false
	}
	site, ok := s.Get(ctx, id)
	if ok && site.PublishableKey == pk {
		return site, true
	}
	return nil, false
}

// Refresh 重新加载一个站点（收到变更通知时），返回是否成功。
func (s *Store) Refresh(ctx context.Context, siteID uint64) bool {
	if s.loader == nil {
		return false
	}
	site, err := s.loader.LoadSite(ctx, siteID)
	if err != nil || site == nil {
		return false
	}
	// 只接受更新的 revision，避免乱序消息回退
	if old, ok := s.sites.Load(siteID); ok && old.(*Site).Revision > site.Revision {
		return true
	}
	s.replace(site)
	return true
}

// Remove 删除站点快照与其域名索引。
func (s *Store) Remove(siteID uint64) {
	if v, ok := s.sites.LoadAndDelete(siteID); ok {
		for _, d := range v.(*Site).Domains {
			s.hosts.Delete(d.Domain)
		}
		s.prefixes.Delete(v.(*Site).SecretPrefix)
	}
}

func (s *Store) replace(site *Site) {
	if v, ok := s.sites.Load(site.SiteID); ok {
		for _, d := range v.(*Site).Domains {
			s.hosts.Delete(d.Domain)
		}
		s.prefixes.Delete(v.(*Site).SecretPrefix)
	}
	s.sites.Store(site.SiteID, site)
	for _, d := range site.Domains {
		if d.Status == "bound" {
			s.hosts.Store(d.Domain, site.SiteID)
			s.negative.Delete(d.Domain)
		}
	}
}

// Stats 仓库状态（版本上报用）。
type Stats struct {
	Sites  int              `json:"sites"`
	Hosts  int              `json:"hosts"`
	Loads  int64            `json:"loads"`
	Misses int64            `json:"misses"`
	Revs   map[uint64]int64 `json:"revisions"`
}

func (s *Store) Stats() Stats {
	st := Stats{Revs: map[uint64]int64{}, Loads: s.loads.Load(), Misses: s.misses.Load()}
	s.sites.Range(func(k, v any) bool {
		st.Sites++
		st.Revs[k.(uint64)] = v.(*Site).Revision
		return true
	})
	s.hosts.Range(func(_, _ any) bool { st.Hosts++; return true })
	return st
}

func normalizeHost(h string) string {
	for i := 0; i < len(h); i++ {
		if h[i] == ':' {
			h = h[:i]
			break
		}
	}
	b := []byte(h)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
