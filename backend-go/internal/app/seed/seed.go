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
		if err := retireMenus(tx); err != nil {
			return err
		}
		menus, err := seedMenus(tx)
		if err != nil {
			return err
		}
		if err := backfillAlertChildren(tx, menus); err != nil {
			return err
		}
		if err := ensureDevCenterParent(tx, menus); err != nil {
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
		{Key: "a.merchant.quota", Parent: "a.merchant", Portal: admin, Name: "额度与功能", Type: "F", Perms: "merchant:account:quota", Roles: ops},
		{Key: "a.site", Portal: admin, Name: "全部站点", Path: "/admin/site", Component: "admin/site", Type: "C", Perms: "site:site:list", Icon: "system", Sort: 25, Visible: true, Roles: ops},
		{Key: "a.site.status", Parent: "a.site", Portal: admin, Name: "站点状态", Type: "F", Perms: "site:site:status", Roles: ops},
		{Key: "a.domain", Portal: admin, Name: "全部域名", Path: "/admin/domain", Component: "admin/domain", Type: "C", Perms: "site:domain:list", Icon: "tree", Sort: 30, Visible: true, Roles: ops},
		{Key: "a.domain.verify", Parent: "a.domain", Portal: admin, Name: "域名校验", Type: "F", Perms: "site:domain:verify", Roles: ops},
		{Key: "a.param", Portal: admin, Name: "参数字典", Path: "/admin/param", Component: "admin/param", Type: "C", Perms: "attribution:param:list", Icon: "dict", Sort: 35, Visible: true, Roles: ops},
		{Key: "a.param.edit", Parent: "a.param", Portal: admin, Name: "字典维护", Type: "F", Perms: "attribution:param:edit", Roles: ops},
		{Key: "a.decisionLog", Portal: admin, Name: "决策日志", Path: "/admin/decision-log", Component: "admin/decision-log", Type: "C", Perms: "log:decision:list", Icon: "log", Sort: 45, Visible: true, Roles: ops},
		{Key: "a.apiLog", Portal: admin, Name: "API 调用日志", Path: "/admin/api-log", Component: "admin/api-log", Type: "C", Perms: "cloak:apiLog:list", Icon: "log", Sort: 60, Visible: true, Roles: ops},
		{Key: "a.merchant.impersonate", Parent: "a.merchant", Portal: admin, Name: "模拟登录", Type: "F", Perms: "merchant:account:impersonate", Roles: ops},
		{Key: "a.code", Portal: admin, Name: "全平台归因码", Path: "/admin/code", Component: "admin/code", Type: "C", Perms: "platform:code:list", Icon: "dict", Sort: 32, Visible: true, Roles: ops},
		{Key: "a.code.status", Parent: "a.code", Portal: admin, Name: "归因码状态", Type: "F", Perms: "platform:code:status", Roles: ops},
		{Key: "a.system", Portal: admin, Name: "系统配置", Path: "/admin/system", Component: "admin/system", Type: "C", Perms: "system:config:view", Icon: "system", Sort: 70, Visible: true, Roles: ops},
		{Key: "a.system.edit", Parent: "a.system", Portal: admin, Name: "修改系统配置", Type: "F", Perms: "system:config:edit"},
		{Key: "a.template", Portal: admin, Name: "默认模板", Path: "/admin/template", Component: "admin/template", Type: "C", Perms: "system:template:list", Icon: "dict", Sort: 72, Visible: true, Roles: ops},
		{Key: "a.template.edit", Parent: "a.template", Portal: admin, Name: "编辑模板", Type: "F", Perms: "system:template:edit"},
		{Key: "a.access", Portal: admin, Name: "权限管理", Path: "/admin/access", Component: "Layout", Type: "M", Perms: "system:access:list", Icon: "peoples", Sort: 80, Visible: true},
		{Key: "a.user", Parent: "a.access", Portal: admin, Name: "账号管理", Path: "/admin/user", Component: "admin/user", Type: "C", Perms: "system:user:list", Icon: "user", Sort: 1, Visible: true},
		{Key: "a.user.edit", Parent: "a.user", Portal: admin, Name: "账号维护", Type: "F", Perms: "system:user:edit"},
		{Key: "a.role", Parent: "a.access", Portal: admin, Name: "角色管理", Path: "/admin/role", Component: "admin/role", Type: "C", Perms: "system:role:list", Icon: "peoples", Sort: 2, Visible: true},
		{Key: "a.role.edit", Parent: "a.role", Portal: admin, Name: "角色维护", Type: "F", Perms: "system:role:edit"},
		{Key: "a.menu", Parent: "a.access", Portal: admin, Name: "菜单管理", Path: "/admin/menu", Component: "admin/menu", Type: "C", Perms: "system:menu:list", Icon: "tree", Sort: 3, Visible: true},
		{Key: "a.menu.edit", Parent: "a.menu", Portal: admin, Name: "菜单维护", Type: "F", Perms: "system:menu:edit"},

		// 商户端
		{Key: "m.dashboard", Portal: merchant, Name: "数据看板", Path: "/merchant/dashboard", Component: "merchant/dashboard", Type: "C", Perms: "merchant:dashboard:view", Icon: "dashboard", Sort: 110, Visible: true, Roles: allMerchant},
		{Key: "m.site", Portal: merchant, Name: "站点管理", Path: "/merchant/site", Component: "merchant/site", Type: "C", Perms: "site:site:list", Icon: "system", Sort: 115, Visible: true, Roles: allMerchant},
		{Key: "m.site.add", Parent: "m.site", Portal: merchant, Name: "站点新增", Type: "F", Perms: "site:site:add", Roles: writers},
		{Key: "m.site.edit", Parent: "m.site", Portal: merchant, Name: "站点编辑", Type: "F", Perms: "site:site:edit", Roles: writers},
		{Key: "m.site.remove", Parent: "m.site", Portal: merchant, Name: "站点删除", Type: "F", Perms: "site:site:remove", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.site.status", Parent: "m.site", Portal: merchant, Name: "站点状态", Type: "F", Perms: "site:site:status", Roles: writers},
		{Key: "m.site.secret", Parent: "m.site", Portal: merchant, Name: "重置密钥", Type: "F", Perms: "site:site:secret", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.cloak", Portal: merchant, Name: "斗篷列表", Path: "/merchant/cloak", Component: "merchant/cloak", Type: "C", Perms: "cloak:cloak:list", Icon: "system", Sort: 116, Visible: true, Roles: allMerchant},
		{Key: "m.cloak.add", Parent: "m.cloak", Portal: merchant, Name: "斗篷新增", Type: "F", Perms: "cloak:cloak:add", Roles: writers},
		{Key: "m.cloak.edit", Parent: "m.cloak", Portal: merchant, Name: "斗篷编辑", Type: "F", Perms: "cloak:cloak:edit", Roles: writers},
		{Key: "m.cloak.copy", Parent: "m.cloak", Portal: merchant, Name: "斗篷复制", Type: "F", Perms: "cloak:cloak:copy", Roles: writers},
		{Key: "m.cloak.status", Parent: "m.cloak", Portal: merchant, Name: "斗篷状态", Type: "F", Perms: "cloak:cloak:status", Roles: writers},
		{Key: "m.cloak.remove", Parent: "m.cloak", Portal: merchant, Name: "斗篷删除", Type: "F", Perms: "cloak:cloak:remove", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.domain", Portal: merchant, Name: "域名列表", Path: "/merchant/domain", Component: "merchant/domain", Type: "C", Perms: "site:domain:list", Icon: "tree", Sort: 120, Visible: true, Roles: allMerchant},
		{Key: "m.domain.add", Parent: "m.domain", Portal: merchant, Name: "域名新增", Type: "F", Perms: "site:domain:add", Roles: writers},
		{Key: "m.domain.verify", Parent: "m.domain", Portal: merchant, Name: "域名校验", Type: "F", Perms: "site:domain:verify", Roles: writers},
		{Key: "m.domain.remove", Parent: "m.domain", Portal: merchant, Name: "域名删除", Type: "F", Perms: "site:domain:remove", Roles: writers},
		{Key: "m.attribution", Portal: merchant, Name: "归因规则", Path: "/merchant/attribution", Component: "merchant/attribution", Type: "C", Perms: "attribution:rule:list", Icon: "edit", Sort: 122, Visible: true, Roles: allMerchant},
		{Key: "m.attribution.edit", Parent: "m.attribution", Portal: merchant, Name: "规则编辑", Type: "F", Perms: "attribution:rule:edit", Roles: writers},
		{Key: "m.code", Portal: merchant, Name: "归因码", Path: "/merchant/code", Component: "merchant/code", Type: "C", Perms: "attribution:code:list", Icon: "number", Sort: 124, Visible: true, Roles: allMerchant},
		{Key: "m.code.add", Parent: "m.code", Portal: merchant, Name: "归因码登记", Type: "F", Perms: "attribution:code:add", Roles: writers},
		{Key: "m.code.edit", Parent: "m.code", Portal: merchant, Name: "归因码编辑", Type: "F", Perms: "attribution:code:edit", Roles: writers},
		{Key: "m.code.status", Parent: "m.code", Portal: merchant, Name: "归因码状态", Type: "F", Perms: "attribution:code:status", Roles: writers},
		{Key: "m.code.remove", Parent: "m.code", Portal: merchant, Name: "归因码删除", Type: "F", Perms: "attribution:code:remove", Roles: writers},
		{Key: "m.code.export", Parent: "m.code", Portal: merchant, Name: "归因码导出", Type: "F", Perms: "attribution:code:export", Roles: allMerchant},
		{Key: "m.dev", Portal: merchant, Name: "开发者中心", Path: "/merchant/dev", Component: "Layout", Type: "M", Perms: "dev:center:view", Icon: "code", Sort: 160, Visible: true, Roles: allMerchant},
		{Key: "m.decisionTest", Parent: "m.dev", Portal: merchant, Name: "判定测试", Path: "/merchant/decision-test", Component: "merchant/decision-test", Type: "C", Perms: "attribution:test:run", Icon: "search", Sort: 2, Visible: true, Roles: allMerchant},
		{Key: "m.decisionLog", Portal: merchant, Name: "决策日志", Path: "/merchant/decision-log", Component: "merchant/decision-log", Type: "C", Perms: "log:decision:list", Icon: "log", Sort: 127, Visible: true, Roles: allMerchant},
		{Key: "m.apiGuide", Parent: "m.dev", Portal: merchant, Name: "接入指南", Path: "/merchant/api-guide", Component: "merchant/api-guide", Type: "C", Perms: "site:site:list", Icon: "guide", Sort: 3, Visible: true, Roles: allMerchant},
		{Key: "m.apiDoc", Parent: "m.dev", Portal: merchant, Name: "接入文档", Path: "/merchant/api-key/guide", Component: "merchant/api-key/guide", Type: "C", Perms: "site:site:list", Icon: "guide", Sort: 4, Visible: true, Roles: allMerchant},
		{Key: "m.report", Portal: merchant, Name: "报表", Path: "/merchant/report", Component: "merchant/report", Type: "C", Perms: "report:view", Icon: "chart", Sort: 112, Visible: true, Roles: allMerchant},
		{Key: "m.report.export", Parent: "m.report", Portal: merchant, Name: "报表导出", Type: "F", Perms: "report:export", Roles: writers},
		{Key: "m.download", Portal: merchant, Name: "下载中心", Path: "/merchant/download", Component: "merchant/download", Type: "C", Perms: "report:view", Icon: "download", Sort: 111, Visible: false, Roles: allMerchant},
		{Key: "m.alert", Portal: merchant, Name: "告警通知", Path: "/merchant/alert", Component: "Layout", Type: "M", Perms: "alert:view", Icon: "message", Sort: 113, Visible: true, Roles: allMerchant},
		{Key: "m.alert.list", Parent: "m.alert", Portal: merchant, Name: "告警", Path: "/merchant/alert/list", Component: "merchant/alert", Type: "C", Perms: "alert:view", Icon: "message", Sort: 1, Visible: true, Roles: allMerchant},
		{Key: "m.alert.rules", Parent: "m.alert", Portal: merchant, Name: "规则", Path: "/merchant/alert/rules", Component: "merchant/alert/rules", Type: "C", Perms: "alert:view", Icon: "edit", Sort: 2, Visible: true, Roles: allMerchant},
		{Key: "m.alert.settings", Parent: "m.alert", Portal: merchant, Name: "通知设置", Path: "/merchant/alert/settings", Component: "merchant/alert/settings", Type: "C", Perms: "alert:view", Icon: "form", Sort: 3, Visible: true, Roles: allMerchant},
		{Key: "m.alert.edit", Parent: "m.alert", Portal: merchant, Name: "告警配置", Type: "F", Perms: "alert:edit", Roles: writers},
		{Key: "m.webhook", Parent: "m.dev", Portal: merchant, Name: "Webhook", Path: "/merchant/webhook", Component: "merchant/webhook", Type: "C", Perms: "webhook:view", Icon: "link", Sort: 1, Visible: true, Roles: allMerchant},
		{Key: "m.webhook.edit", Parent: "m.webhook", Portal: merchant, Name: "Webhook 配置", Type: "F", Perms: "webhook:edit", Roles: writers},
		{Key: "m.link", Portal: merchant, Name: "链接生成器", Path: "/merchant/link", Component: "merchant/link", Type: "C", Perms: "attribution:link:build", Icon: "link", Sort: 126, Visible: true, Roles: allMerchant},
		{Key: "m.apiLog", Parent: "m.dev", Portal: merchant, Name: "API 调用日志", Path: "/merchant/api-log", Component: "merchant/api-log", Type: "C", Perms: "cloak:apiLog:list", Icon: "log", Sort: 5, Visible: true, Roles: allMerchant},
		{Key: "m.access", Portal: merchant, Name: "权限管理", Path: "/merchant/access", Component: "Layout", Type: "M", Perms: "merchant:user:list", Icon: "peoples", Sort: 900, Visible: true, Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.user", Parent: "m.access", Portal: merchant, Name: "账号管理", Path: "/merchant/user", Component: "merchant/user", Type: "C", Perms: "merchant:user:list", Icon: "user", Sort: 1, Visible: true, Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.user.add", Parent: "m.user", Portal: merchant, Name: "账号新增", Type: "F", Perms: "merchant:user:add", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.user.edit", Parent: "m.user", Portal: merchant, Name: "账号编辑", Type: "F", Perms: "merchant:user:edit", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.user.status", Parent: "m.user", Portal: merchant, Name: "账号状态", Type: "F", Perms: "merchant:user:status", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.user.resetPwd", Parent: "m.user", Portal: merchant, Name: "重置密码", Type: "F", Perms: "merchant:user:resetPwd", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.user.remove", Parent: "m.user", Portal: merchant, Name: "账号删除", Type: "F", Perms: "merchant:user:remove", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.role", Parent: "m.access", Portal: merchant, Name: "角色管理", Path: "/merchant/role", Component: "merchant/role", Type: "C", Perms: "merchant:role:list", Icon: "peoples", Sort: 2, Visible: true, Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.role.edit", Parent: "m.role", Portal: merchant, Name: "角色维护", Type: "F", Perms: "merchant:role:edit", Roles: []string{identity.RoleMerchantAdmin}},
		{Key: "m.menu", Parent: "m.access", Portal: merchant, Name: "菜单管理", Path: "/merchant/menu", Component: "merchant/menu", Type: "C", Perms: "merchant:menu:list", Icon: "tree", Sort: 3, Visible: true, Roles: []string{identity.RoleMerchantAdmin}},
	}
}

// retiredMenuPaths 已下线页面（V1 遗留，前端无对应页面）。seed 时连同其按钮与角色绑定一起删除。
var retiredMenuPaths = []string{
	"/admin/cloak", "/admin/cloak-log",
	"/merchant/cloak-log", "/merchant/access-list", "/merchant/api-key",
}

func retireMenus(tx *gorm.DB) error {
	var ids []uint64
	if err := tx.Model(&identity.Menu{}).Where("path IN ?", retiredMenuPaths).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	var children []uint64
	if err := tx.Model(&identity.Menu{}).Where("parent_id IN ?", ids).Pluck("id", &children).Error; err != nil {
		return err
	}
	all := append(ids, children...)
	if err := tx.Exec("DELETE FROM role_menus WHERE menu_id IN ?", all).Error; err != nil {
		return err
	}
	return tx.Where("id IN ?", all).Delete(&identity.Menu{}).Error
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

// 目录由原来的「告警与通知」页面改成。已有角色只绑了目录，补上三个子页面；子页面已有绑定时不再覆盖。
func backfillAlertChildren(tx *gorm.DB, menus map[string]identity.Menu) error {
	parent := menus["m.alert"]
	for _, key := range []string{"m.alert.list", "m.alert.rules", "m.alert.settings"} {
		child := menus[key]
		var n int64
		if err := tx.Table("role_menus").Where("menu_id = ?", child.ID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if err := tx.Exec(
			"INSERT IGNORE INTO role_menus (role_id, menu_id) SELECT role_id, ? FROM role_menus WHERE menu_id = ?",
			child.ID, parent.ID,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

// 这些页面挂到开发者中心之后，已有角色必须同时拥有目录，否则侧边栏不显示。
func ensureDevCenterParent(tx *gorm.DB, menus map[string]identity.Menu) error {
	parent := menus["m.dev"]
	for _, key := range []string{"m.webhook", "m.decisionTest", "m.apiGuide", "m.apiDoc", "m.apiLog"} {
		child := menus[key]
		if err := tx.Exec(
			"INSERT IGNORE INTO role_menus (role_id, menu_id) SELECT role_id, ? FROM role_menus WHERE menu_id = ?",
			parent.ID, child.ID,
		).Error; err != nil {
			return err
		}
	}
	return nil
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
			DecisionQuota: 1_000_000, SiteQuota: 5, DomainQuota: 20, CodeQuota: 1000, CloakQuota: 20, UserQuota: 10,
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
