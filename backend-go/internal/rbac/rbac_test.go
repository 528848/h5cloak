package rbac

import (
	"context"
	"testing"

	"cloak/internal/domain/identity"
)

type fakeMenus struct{ all []identity.Menu }

func (f fakeMenus) ListByRoles(_ context.Context, _ []uint64, portal string) ([]identity.Menu, error) {
	return f.filter(portal), nil
}
func (f fakeMenus) ListAllByPortal(_ context.Context, portal string) ([]identity.Menu, error) {
	return f.filter(portal), nil
}
func (f fakeMenus) filter(portal string) []identity.Menu {
	var out []identity.Menu
	for _, m := range f.all {
		if m.Portal == portal {
			out = append(out, m)
		}
	}
	return out
}

func u64(v uint64) *uint64 { return &v }

func TestHasPermission(t *testing.T) {
	cases := []struct {
		perms    []string
		required string
		want     bool
	}{
		{[]string{"*:*:*"}, "site:domain:add", true},
		{[]string{"site:domain:list"}, "site:domain:add", false},
		{[]string{"site:domain:add"}, "site:domain:add", true},
		{[]string{"site:*"}, "site:domain:add", true},
		{[]string{"merchant:*"}, "site:domain:add", false},
		{nil, "", true},
	}
	for _, c := range cases {
		if got := HasPermission(c.perms, c.required); got != c.want {
			t.Errorf("HasPermission(%v,%q)=%v want %v", c.perms, c.required, got, c.want)
		}
	}
}

func TestRoutersBuildsTreeAndSkipsButtons(t *testing.T) {
	menus := []identity.Menu{
		{ID: 1, Portal: "merchant", Name: "站点", Path: identity.Ptr("/merchant/site"), MenuType: "M", Visible: true, Status: "enabled"},
		{ID: 2, ParentID: u64(1), Portal: "merchant", Name: "站点列表", Path: identity.Ptr("list"), Component: identity.Ptr("merchant/site"), MenuType: "C", Visible: true, Status: "enabled"},
		{ID: 3, ParentID: u64(2), Portal: "merchant", Name: "新增", MenuType: "F", Perms: identity.Ptr("site:site:add"), Status: "enabled"},
		{ID: 4, Portal: "admin", Name: "平台首页", Path: identity.Ptr("/admin/dashboard"), MenuType: "C", Visible: true, Status: "enabled"},
	}
	svc := New(fakeMenus{all: menus})
	user := &identity.User{MerchantID: u64(9), Roles: []identity.Role{{ID: 1, RoleKey: "merchant_admin", Status: "enabled"}}}

	routers, err := svc.Routers(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	if len(routers) != 1 || routers[0].Path != "/merchant/site" {
		t.Fatalf("unexpected routers %+v", routers)
	}
	if len(routers[0].Children) != 1 || len(routers[0].Children[0].Children) != 0 {
		t.Fatalf("buttons must not appear in tree: %+v", routers[0].Children)
	}

	perms, _ := svc.Permissions(context.Background(), user)
	if len(perms) != 1 || perms[0] != "site:site:add" {
		t.Fatalf("perms = %v", perms)
	}
}

func TestSuperAdminPermissions(t *testing.T) {
	svc := New(fakeMenus{})
	user := &identity.User{Roles: []identity.Role{{ID: 1, RoleKey: identity.RoleSuperAdmin, Status: "enabled"}}}
	perms, _ := svc.Permissions(context.Background(), user)
	if len(perms) != 1 || perms[0] != identity.SuperPermission {
		t.Fatalf("perms = %v", perms)
	}
}
