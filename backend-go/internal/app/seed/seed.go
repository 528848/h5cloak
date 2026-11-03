// Package seed 写入初始角色、账号与菜单。幂等：按唯一键 upsert，可重复执行。
package seed

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cloak/internal/auth"
	"cloak/internal/domain/identity"
)

// Options 初始账号。
type Options struct {
	AdminUsername    string
	AdminPassword    string
	MerchantUsername string
	MerchantPassword string
}

// Run 执行种子。
func Run(ctx context.Context, db *gorm.DB, opt Options) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roles, err := seedRoles(tx)
		if err != nil {
			return err
		}
		menus, err := seedMenus(tx)
		if err != nil {
			return err
		}
		if err := bindRoleMenus(tx, roles, menus); err != nil {
			return err
		}
		if err := seedUsers(tx, roles, opt); err != nil {
			return err
		}
		return nil
	})
}

func seedRoles(tx *gorm.DB) (map[string]identity.Role, error) {
	defs := []identity.Role{
		{Name: "超级管理员", RoleKey: identity.RoleSuperAdmin, Scope: identity.ScopePlatform, Sort: 1},
		{Name: "运营管理员", RoleKey: identity.RoleOpsAdmin, Scope: identity.ScopePlatform, Sort: 2},
		{Name: "商户管理员", RoleKey: identity.RoleMerchantAdmin, Scope: identity.ScopeMerchant, Sort: 10},
		{Name: "商户操作员", RoleKey: identity.RoleMerchantOperator, Scope: identity.ScopeMerchant, Sort: 11},
		{Name: "商户只读", RoleKey: identity.RoleMerchantViewer, Scope: identity.ScopeMerchant, Sort: 12},
	}
	out := map[string]identity.Role{}
	for _, r := range defs {
		r.Status = identity.StatusEnabled
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "role_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "scope", "sort", "status"}),
		}).Create(&r).Error; err != nil {
			return nil, err
		}
		var saved identity.Role
		if err := tx.Where("role_key = ?", r.RoleKey).First(&saved).Error; err != nil {
			return nil, err
		}
		out[saved.RoleKey] = saved
	}
	return out, nil
}

// menuDef 菜单定义。Key 用于父子引用与幂等。
type menuDef struct {
	Key       string
	Parent    string
	Portal    string
	Name      string
	Path      string
	Component string
	Type      string // M C F
	Perms     string
	Icon      string
	Sort      uint32
	Visible   bool
	Roles     []string // 除超级管理员外，哪些角色可见
}

// 菜单只包含当前前端已存在的页面；后续迭代随页面一起追加。
func menuDefs() []menuDef {
	admin := identity.PortalAdmin
	merchant := identity.PortalMerchant
	allMerchant := []string{identity.RoleMerchantAdmin, identity.RoleMerchantOperator, identity.RoleMerchantViewer}
	writers := []string{identity.RoleMerchantAdmin, identity.RoleMerchantOperator}
	ops := []string{identity.RoleOpsAdmin}

	return []menuDef{
		// 总后台
		{Key: "a.dashboard", Portal: admin, Name: "平台首页", Path: "/admin/dashboard", Component: "admin/dashboard", Type: "C", Perms: "admin:dashboard:view", Icon: "dashboard", Sort: 10, Visible: true, Roles: ops},
		{Key: "a.merchant", Portal: admin, Name: "商户账号", Path: "/admin/merchant", Component: "admin/merchant", Type: "C", Perms: "merchant:account:list", Icon: "peoples", Sort: 20, Visible: true, Roles: ops},
		{Key: "a.merchant.add", Parent: "a.merchant", Portal: admin, Name: "商户新增", Type: "F", Perms: "merchant:account:add", Roles: ops},
		{Key: "a.merchant.edit", Parent: "a.merchant", Portal: admin, Name: "商户编辑", Type: "F", Perms: "merchant:account:edit", Roles: ops},
		{Key: "a.merchant.status", Parent: "a.merchant", Portal: admin, Name: "商户状态", Type: "F", Perms: "merchant:account:status", Roles: ops},
		{Key: "a.merchant.resetPwd", Parent: "a.merchant", Portal: admin, Name: "重置管理员密码", Type: "F", Perms: "merchant:account:resetPwd", Roles: ops},
		{Key: "a.domain", Portal: admin, Name: "全部域名", Path: "/admin/domain", Component: "admin/domain", Type: "C", Perms: "site:domain:list", Icon: "tree", Sort: 30, Visible: true, Roles: ops},
		{Key: "a.cloak", Portal: admin, Name: "全部斗篷", Path: "/admin/cloak", Component: "admin/cloak", Type: "C", Perms: "cloak:strategy:list", Icon: "system", Sort: 40, Visible: true, Roles: ops},
		{Key: "a.cloakLog", Portal: admin, Name: "斗篷日志", Path: "/admin/cloak-log", Component: "admin/cloak-log", Type: "C", Perms: "cloak:log:list", Icon: "log", Sort: 50, Visible: true, Roles: ops},
		{Key: "a.apiLog", Portal: admin, Name: "API 调用日志", Path: "/admin/api-log", Component: "admin/api-log", Type: "C", Perms: "cloak:apiLog:list", Icon: "log", Sort: 60, Visible: true, Roles: ops},

		// 商户端
		{Key: "m.dashboard", Portal: merchant, Name: "商户首页", Path: "/merchant/dashboard", Component: "merchant/dashboard", Type: "C", Perms: "merchant:dashboard:view", Icon: "dashboard", Sort: 110, Visible: true, Roles: allMerchant},
		{Key: "m.domain", Portal: merchant, Name: "域名列表", Path: "/merchant/domain", Component: "merchant/domain", Type: "C", Perms: "site:domain:list", Icon: "tree", Sort: 120, Visible: true, Roles: allMerchant},
		{Key: "m.domain.add", Parent: "m.domain", Portal: merchant, Name: "域名新增", Type: "F", Perms: "site:domain:add", Roles: writers},
		{Key: "m.domain.verify", Parent: "m.domain", Portal: merchant, Name: "域名校验", Type: "F", Perms: "site:domain:verify", Roles: writers},
		{Key: "m.cloak", Portal: merchant, Name: "斗篷列表", Path: "/merchant/cloak", Component: "merchant/cloak", Type: "C", Perms: "cloak:strategy:list", Icon: "system", Sort: 130, Visible: true, Roles: allMerchant},
		{Key: "m.cloak.add", Parent: "m.cloak", Portal: merchant, Name: "斗篷新增", Type: "F", Perms: "cloak:strategy:add", Roles: writers},
		{Key: "m.cloak.edit", Parent: "m.cloak", Portal: merchant, Name: "斗篷编辑", Type: "F", Perms: "cloak:strategy:edit", Roles: writers},
		{Key: "m.cloak.copy", Parent: "m.cloak", Portal: merchant, Name: "斗篷复制", Type: "F", Perms: "cloak:strategy:copy", Roles: writers},
		{Key: "m.cloak.remove", Parent: "m.cloak", Portal: merchant, Name: "斗篷删除", Type: "F", Perms: "cloak:strategy:remove", Roles: writers},
		{Key: "m.cloak.status", Parent: "m.cloak", Portal: merchant, Name: "斗篷状态", Type: "F", Perms: "cloak:strategy:status", Roles: writers},
		{Key: "m.cloakLog", Portal: merchant, Name: "访问日志", Path: "/merchant/cloak-log", Component: "merchant/cloak-log", Type: "C", Perms: "cloak:log:list", Icon: "log", Sort: 140, Visible: true, Roles: allMerchant},
		{Key: "m.accessList", Portal: merchant, Name: "黑白名单", Path: "/merchant/access-list", Component: "merchant/access-list", Type: "C", Perms: "cloak:accessList:list", Icon: "peoples", Sort: 150, Visible: true, Roles: allMerchant},
		{Key: "m.accessList.add", Parent: "m.accessList", Portal: merchant, Name: "名单新增", Type: "F", Perms: "cloak:accessList:add", Roles: writers},
		{Key: "m.accessList.edit", Parent: "m.accessList", Portal: merchant, Name: "名单编辑", Type: "F", Perms: "cloak:accessList:edit", Roles: writers},
		{Key: "m.accessList.remove", Parent: "m.accessList", Portal: merchant, Name: "名单删除", Type: "F", Perms: "cloak:accessList:remove", Roles: writers},
		{Key: "m.apiKey", Portal: merchant, Name: "API Key", Path: "/merchant/api-key", Component: "merchant/api-key", Type: "C", Perms: "cloak:apiKey:list", Icon: "link", Sort: 160, Visible: true, Roles: allMerchant},
		{Key: "m.apiKey.add", Parent: "m.apiKey", Portal: merchant, Name: "Key新增", Type: "F", Perms: "cloak:apiKey:add", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.apiKey.reset", Parent: "m.apiKey", Portal: merchant, Name: "Key重置", Type: "F", Perms: "cloak:apiKey:reset", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.apiKey.remove", Parent: "m.apiKey", Portal: merchant, Name: "Key删除", Type: "F", Perms: "cloak:apiKey:remove", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.apiLog", Portal: merchant, Name: "API 调用日志", Path: "/merchant/api-log", Component: "merchant/api-log", Type: "C", Perms: "cloak:apiLog:list", Icon: "log", Sort: 170, Visible: true, Roles: allMerchant},
	}
}

func seedMenus(tx *gorm.DB) (map[string]identity.Menu, error) {
	defs := menuDefs()
	saved := map[string]identity.Menu{}
	for _, d := range defs {
		m := identity.Menu{
			Portal:    d.Portal,
			Name:      d.Name,
			Path:      identity.Ptr(d.Path),
			Component: identity.Ptr(d.Component),
			MenuType:  d.Type,
			Visible:   d.Visible,
			Status:    identity.StatusEnabled,
			Perms:     identity.Ptr(d.Perms),
			Icon:      identity.Ptr(d.Icon),
			Sort:      d.Sort,
		}
		if d.Parent != "" {
			p, ok := saved[d.Parent]
			if !ok {
				return nil, fmt.Errorf("menu %s references unknown parent %s", d.Key, d.Parent)
			}
			m.ParentID = &p.ID
		}
		// 幂等键：按钮用 perms，页面用 path
		var existing identity.Menu
		q := tx.Where("portal = ?", d.Portal)
		if d.Type == "F" {
			q = q.Where("perms = ? AND menu_type = 'F'", d.Perms)
		} else {
			q = q.Where("path = ?", d.Path)
		}
		err := q.First(&existing).Error
		switch {
		case err == nil:
			m.ID = existing.ID
			if err := tx.Model(&existing).Updates(map[string]any{
				"parent_id": m.ParentID, "name": m.Name, "component": m.Component, "menu_type": m.MenuType,
				"visible": m.Visible, "status": m.Status, "perms": m.Perms, "icon": m.Icon, "sort": m.Sort,
			}).Error; err != nil {
				return nil, err
			}
		case err == gorm.ErrRecordNotFound:
			if err := tx.Create(&m).Error; err != nil {
				return nil, err
			}
		default:
			return nil, err
		}
		saved[d.Key] = m
	}
	return saved, nil
}

func bindRoleMenus(tx *gorm.DB, roles map[string]identity.Role, menus map[string]identity.Menu) error {
	for _, d := range menuDefs() {
		m := menus[d.Key]
		for _, rk := range d.Roles {
			r, ok := roles[rk]
			if !ok {
				continue
			}
			if err := tx.Exec("INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (?, ?)", r.ID, m.ID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedUsers(tx *gorm.DB, roles map[string]identity.Role, opt Options) error {
	adminHash, err := auth.HashPassword(opt.AdminPassword)
	if err != nil {
		return err
	}
	if err := upsertUser(tx, &identity.User{
		Username: opt.AdminUsername, Nickname: identity.Ptr("超级管理员"), Password: adminHash, Status: identity.StatusEnabled,
	}, roles[identity.RoleSuperAdmin]); err != nil {
		return err
	}

	var merchant identity.Merchant
	err = tx.Where("name = ?", "演示商户").First(&merchant).Error
	if err == gorm.ErrRecordNotFound {
		merchant = identity.Merchant{
			Name: "演示商户", Status: identity.StatusEnabled, Timezone: "Asia/Shanghai",
			DecisionQuota: 1_000_000, SiteQuota: 5, DomainQuota: 20, CodeQuota: 1000, UserQuota: 10,
			LogRetentionDays: 90, QuotaPolicy: "alert",
			Features: []byte(`{"hosted":true,"server_api":true,"js_sdk":true,"conversion":true,"webhook":true,"enhancements":false}`),
		}
		if err := tx.Create(&merchant).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	merchantHash, err := auth.HashPassword(opt.MerchantPassword)
	if err != nil {
		return err
	}
	return upsertUser(tx, &identity.User{
		MerchantID: &merchant.ID, Username: opt.MerchantUsername, Nickname: identity.Ptr("演示商户管理员"),
		Password: merchantHash, Status: identity.StatusEnabled,
	}, roles[identity.RoleMerchantAdmin])
}

func upsertUser(tx *gorm.DB, u *identity.User, role identity.Role) error {
	var existing identity.User
	err := tx.Where("username = ?", u.Username).First(&existing).Error
	switch {
	case err == nil:
		// 已存在的账号不重置密码，避免覆盖线上修改
		u.ID = existing.ID
	case err == gorm.ErrRecordNotFound:
		if err := tx.Create(u).Error; err != nil {
			return err
		}
	default:
		return err
	}
	return tx.Exec("INSERT IGNORE INTO user_roles (user_id, role_id) VALUES (?, ?)", u.ID, role.ID).Error
}
