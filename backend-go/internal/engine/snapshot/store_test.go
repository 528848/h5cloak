package snapshot

import (
	"context"
	"errors"
	"testing"
)

type fakeLoader struct {
	sites map[uint64]*Site
	hosts map[string]uint64
	calls int
}

func (f *fakeLoader) LoadSite(_ context.Context, id uint64) (*Site, error) {
	f.calls++
	if s, ok := f.sites[id]; ok {
		return s, nil
	}
	return nil, errors.New("nf")
}

func (f *fakeLoader) ResolvePrefix(_ context.Context, p string) (uint64, error) {
	for id, s := range f.sites {
		if s.SecretPrefix == p {
			return id, nil
		}
	}
	return 0, errors.New("nf")
}

func (f *fakeLoader) ResolveHost(_ context.Context, h string) (uint64, error) {
	if id, ok := f.hosts[h]; ok {
		return id, nil
	}
	return 0, errors.New("nf")
}

func TestStoreLazyLoadAndRefresh(t *testing.T) {
	ctx := context.Background()
	v1 := &Site{SiteID: 1, Revision: 1, Domains: []Domain{{Domain: "a.com", Status: "bound"}}}
	fl := &fakeLoader{sites: map[uint64]*Site{1: v1}, hosts: map[string]uint64{"a.com": 1}}
	st := NewStore(fl)

	s, ok := st.ByHost(ctx, "A.com:443")
	if !ok || s.SiteID != 1 || fl.calls != 1 {
		t.Fatalf("lazy load: %v %d", ok, fl.calls)
	}
	if _, ok := st.Get(ctx, 1); !ok || fl.calls != 1 {
		t.Fatal("second get must hit cache")
	}
	if _, ok := st.ByHost(ctx, "nope.com"); ok {
		t.Fatal("unknown host")
	}
	// 负缓存：第二次不再调用 loader
	_, _ = st.ByHost(ctx, "nope.com")

	// 刷新到更新版本，域名变更
	v2 := &Site{SiteID: 1, Revision: 2, Domains: []Domain{{Domain: "b.com", Status: "bound"}}}
	fl.sites[1] = v2
	if !st.Refresh(ctx, 1) {
		t.Fatal("refresh")
	}
	if _, ok := st.hosts.Load("a.com"); ok {
		t.Fatal("old host index should be gone")
	}
	if s, ok := st.Get(ctx, 1); !ok || s.Revision != 2 {
		t.Fatal("new revision")
	}
	// 旧 revision 不回退
	fl.sites[1] = &Site{SiteID: 1, Revision: 1}
	st.Refresh(ctx, 1)
	if s, _ := st.Get(ctx, 1); s.Revision != 2 {
		t.Fatal("must not downgrade")
	}
	// 删除
	st.Remove(1)
	if _, ok := st.sites.Load(uint64(1)); ok {
		t.Fatal("removed")
	}
	stats := st.Stats()
	if stats.Sites != 0 || stats.Loads == 0 {
		t.Fatalf("stats %+v", stats)
	}
	// 无 loader
	empty := NewStore(nil)
	empty.Put(&Site{SiteID: 9, Domains: []Domain{{Domain: "x.com", Status: "bound"}}})
	if _, ok := empty.ByHost(ctx, "x.com"); !ok {
		t.Fatal("put + byhost")
	}
	if _, ok := empty.Get(ctx, 8); ok || empty.Refresh(ctx, 8) {
		t.Fatal("no loader")
	}
	if _, ok := empty.ByHost(ctx, ""); ok {
		t.Fatal("empty host")
	}
}
