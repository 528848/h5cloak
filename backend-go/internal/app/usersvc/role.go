package usersvc

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"cloak/internal/domain/identity"
)

var roleKeyRE = regexp.MustCompile(`^[a-z][a-z0-9_]{1,20}$`)

type RoleDetail struct {
	RoleID  uint64   `json:"roleId"`
	RoleKey string   `json:"roleKey"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Remark  string   `json:"remark"`
	Builtin bool     `json:"builtin"`
	MenuIDs []uint64 `json:"menuIds"`
}

type RoleInput struct {
	Name    string   `json:"name"`
	RoleKey string   `json:"roleKey"`
	Status  string   `json:"status"`
	Remark  string   `json:"remark"`
	MenuIDs []uint64 `json:"menuIds"`
}

func (s *Service) ManageRoles(ctx context.Context, merchantID uint64) ([]RoleDetail, error) {
	roles, err := s.roles.ListForMerchant(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	out := make([]RoleDetail, 0, len(roles))
	for i := range roles {
		ids, _ := s.roles.RoleMenuIDs(ctx, roles[i].ID)
		if ids == nil {
			ids = []uint64{}
		}
		out = append(out, RoleDetail{
			RoleID: roles[i].ID, RoleKey: roles[i].RoleKey, Name: roles[i].Name,
			Status: roles[i].Status, Remark: identity.Deref(roles[i].Remark),
			Builtin: roles[i].MerchantID == nil, MenuIDs: ids,
		})
	}
	return out, nil
}

func (s *Service) CreateRole(ctx context.Context, merchantID uint64, in RoleInput) (uint64, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.RoleKey = strings.TrimSpace(in.RoleKey)
	if in.Name == "" || !roleKeyRE.MatchString(in.RoleKey) {
		return 0, errf(400, "请填写角色名称。标识用小写字母开头，只含字母、数字和下划线")
	}
	menuIDs, err := s.merchantMenuIDs(ctx, in.MenuIDs)
	if err != nil {
		return 0, err
	}
	key := fmt.Sprintf("m%d_%s", merchantID, in.RoleKey)
	found, err := s.roles.FindByKeys(ctx, []string{key})
	if err != nil {
		return 0, err
	}
	if len(found) > 0 {
		return 0, errf(400, "角色标识已存在")
	}
	status := identity.StatusEnabled
	if in.Status == identity.StatusDisabled {
		status = identity.StatusDisabled
	}
	role := &identity.Role{
		Name: in.Name, RoleKey: key, Scope: identity.ScopeMerchant, MerchantID: &merchantID,
		Status: status, Remark: identity.Ptr(in.Remark),
	}
	if err := s.roles.CreateRole(ctx, role); err != nil {
		return 0, err
	}
	if err := s.roles.ReplaceRoleMenus(ctx, role.ID, menuIDs); err != nil {
		return 0, err
	}
	return role.ID, nil
}

func (s *Service) UpdateRole(ctx context.Context, merchantID, id uint64, in RoleInput) error {
	role, err := s.ownRole(ctx, merchantID, id)
	if err != nil {
		return err
	}
	if role.MerchantID == nil {
		return errf(400, "内置角色不能修改")
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errf(400, "请填写角色名称")
	}
	if in.Status != "" && in.Status != identity.StatusEnabled && in.Status != identity.StatusDisabled {
		return errf(400, "状态无效")
	}
	menuIDs, err := s.merchantMenuIDs(ctx, in.MenuIDs)
	if err != nil {
		return err
	}
	role.Name = in.Name
	role.Remark = identity.Ptr(in.Remark)
	fields := []string{"Name", "Remark"}
	if in.Status != "" {
		role.Status = in.Status
		fields = append(fields, "Status")
	}
	if err := s.roles.UpdateRole(ctx, role, fields...); err != nil {
		return err
	}
	return s.roles.ReplaceRoleMenus(ctx, id, menuIDs)
}

func (s *Service) DeleteRole(ctx context.Context, merchantID, id uint64) error {
	role, err := s.ownRole(ctx, merchantID, id)
	if err != nil {
		return err
	}
	if role.MerchantID == nil {
		return errf(400, "不能删除内置角色")
	}
	n, err := s.roles.CountRoleUsers(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errf(400, "该角色仍有账号，不能删除")
	}
	return s.roles.DeleteRole(ctx, id)
}

func (s *Service) ListMenus(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.roles.ListMenus(ctx, identity.PortalMerchant)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		var parent uint64
		if rows[i].ParentID != nil {
			parent = *rows[i].ParentID
		}
		out = append(out, map[string]any{
			"menuId": rows[i].ID, "parentId": parent, "name": rows[i].Name, "menuType": rows[i].MenuType,
			"path": identity.Deref(rows[i].Path), "perms": identity.Deref(rows[i].Perms), "icon": identity.Deref(rows[i].Icon),
			"sort": rows[i].Sort, "visible": rows[i].Visible, "status": rows[i].Status,
		})
	}
	return out, nil
}

func (s *Service) ownRole(ctx context.Context, merchantID, id uint64) (*identity.Role, error) {
	role, err := s.roles.FindRole(ctx, id)
	if err != nil || role.Scope != identity.ScopeMerchant {
		return nil, errf(404, "角色不存在")
	}
	if role.MerchantID != nil && *role.MerchantID != merchantID {
		return nil, errf(404, "角色不存在")
	}
	return role, nil
}

func (s *Service) merchantMenuIDs(ctx context.Context, ids []uint64) ([]uint64, error) {
	rows, err := s.roles.ListMenus(ctx, identity.PortalMerchant)
	if err != nil {
		return nil, err
	}
	ok := map[uint64]bool{}
	for i := range rows {
		ok[rows[i].ID] = true
	}
	out := make([]uint64, 0, len(ids))
	seen := map[uint64]bool{}
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		if !ok[id] {
			return nil, errf(400, "包含无效的菜单")
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}
