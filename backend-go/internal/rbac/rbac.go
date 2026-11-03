// Package rbac 计算用户的权限标识与动态菜单（若依格式）。
package rbac

import (
	"context"
	"sort"

	"cloak/internal/domain/identity"
)

// MenuSource 菜单来源，超级管理员需要全量菜单。
type MenuSource interface {
	ListByRoles(ctx context.Context, roleIDs []uint64, portal string) ([]identity.Menu, error)
	ListAllByPortal(ctx context.Context, portal string) ([]identity.Menu, error)
}

type Service struct{ menus MenuSource }

func New(menus MenuSource) *Service { return &Service{menus: menus} }

// RouterItem 若依 /getRouters 的菜单节点。
type RouterItem struct {
	Name      string       `json:"name"`
	Path      string       `json:"path"`
	Component string       `json:"component,omitempty"`
	Query     string       `json:"query,omitempty"`
	Hidden    bool         `json:"hidden"`
	Meta      RouterMeta   `json:"meta"`
	Children  []RouterItem `json:"children,omitempty"`
}

type RouterMeta struct {
	Title string `json:"title"`
	Icon  string `json:"icon,omitempty"`
}

// Permissions 返回权限标识列表。超级管理员返回 *:*:*。
func (s *Service) Permissions(ctx context.Context, u *identity.User) ([]string, error) {
	if u.HasRole(identity.RoleSuperAdmin) {
		return []string{identity.SuperPermission}, nil
	}
	menus, err := s.menusFor(ctx, u)
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	for _, m := range menus {
		if m.Perms != nil && *m.Perms != "" {
			set[*m.Perms] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// Routers 返回门户菜单树（不含按钮）。
func (s *Service) Routers(ctx context.Context, u *identity.User) ([]RouterItem, error) {
	menus, err := s.menusFor(ctx, u)
	if err != nil {
		return nil, err
	}
	byParent := map[uint64][]identity.Menu{}
	for _, m := range menus {
		if m.MenuType == "F" {
			continue
		}
		var pid uint64
		if m.ParentID != nil {
			pid = *m.ParentID
		}
		byParent[pid] = append(byParent[pid], m)
	}
	return build(byParent, 0), nil
}

// RoleKeys 返回启用角色 key 列表。
func RoleKeys(u *identity.User) []string {
	keys := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		if r.Status == identity.StatusEnabled {
			keys = append(keys, r.RoleKey)
		}
	}
	return keys
}

// HasPermission 判断权限集合是否满足所需标识。支持 *:*:* 与 module:*:* 形式。
func HasPermission(perms []string, required string) bool {
	if required == "" {
		return true
	}
	for _, p := range perms {
		if p == identity.SuperPermission || p == required {
			return true
		}
		if len(p) > 0 && p[len(p)-1] == '*' && len(required) >= len(p)-1 && required[:len(p)-1] == p[:len(p)-1] {
			return true
		}
	}
	return false
}

func (s *Service) menusFor(ctx context.Context, u *identity.User) ([]identity.Menu, error) {
	portal := u.Portal()
	if u.HasRole(identity.RoleSuperAdmin) {
		return s.menus.ListAllByPortal(ctx, portal)
	}
	ids := make([]uint64, 0, len(u.Roles))
	for _, r := range u.Roles {
		if r.Status == identity.StatusEnabled {
			ids = append(ids, r.ID)
		}
	}
	return s.menus.ListByRoles(ctx, ids, portal)
}

func build(byParent map[uint64][]identity.Menu, parent uint64) []RouterItem {
	items := byParent[parent]
	if len(items) == 0 {
		return nil
	}
	out := make([]RouterItem, 0, len(items))
	for _, m := range items {
		item := RouterItem{
			Name:      m.Name,
			Path:      identity.Deref(m.Path),
			Component: identity.Deref(m.Component),
			Query:     identity.Deref(m.Query),
			Hidden:    !m.Visible,
			Meta:      RouterMeta{Title: m.Name, Icon: identity.Deref(m.Icon)},
			Children:  build(byParent, m.ID),
		}
		out = append(out, item)
	}
	return out
}
