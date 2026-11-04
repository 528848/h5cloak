package platformsvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/auth"
	"cloak/internal/domain/identity"
)

func (s *Service) ListUsers(ctx context.Context, keyword, status string, page, size int) ([]map[string]any, int64, error) {
	rows, total, err := s.id.ListPlatformUsers(ctx, identity.UserFilter{Keyword: strings.TrimSpace(keyword), Status: status, Page: page, Size: size})
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		out = append(out, userRow(&rows[i]))
	}
	return out, total, nil
}

type UserInput struct {
	Username string   `json:"userName"`
	Password string   `json:"password"`
	Nickname string   `json:"nickName"`
	Remark   string   `json:"remark"`
	Status   string   `json:"status"`
	RoleIDs  []uint64 `json:"roleIds"`
}

func (s *Service) CreateUser(ctx context.Context, in UserInput) (uint64, error) {
	in.Username = strings.TrimSpace(in.Username)
	if len(in.Username) < 3 || len(in.Username) > 64 {
		return 0, errf(400, "账号长度为 3 到 64")
	}
	if len(in.Password) < 6 {
		return 0, errf(400, "密码至少 6 位")
	}
	roles, err := s.platformRoles(ctx, in.RoleIDs)
	if err != nil {
		return 0, err
	}
	exists, err := s.id.UsernameExists(ctx, in.Username, 0)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, errf(400, "账号已存在")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return 0, err
	}
	status := identity.StatusEnabled
	if in.Status == identity.StatusDisabled {
		status = identity.StatusDisabled
	}
	u := &identity.User{
		Username: in.Username, Nickname: identity.Ptr(strings.TrimSpace(in.Nickname)), Password: hash,
		Status: status, Remark: identity.Ptr(in.Remark),
	}
	ids := roleIDs(roles)
	if err := s.id.Create(ctx, u, ids); err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (s *Service) UpdateUser(ctx context.Context, actorID, id uint64, in UserInput) error {
	u, err := s.platformUser(ctx, id)
	if err != nil {
		return err
	}
	roles, err := s.platformRoles(ctx, in.RoleIDs)
	if err != nil {
		return err
	}
	if err := s.keepLastSuper(ctx, u, in.Status, roles); err != nil {
		return err
	}
	if in.Status != "" && in.Status != identity.StatusEnabled && in.Status != identity.StatusDisabled {
		return errf(400, "状态无效")
	}
	if in.Status != "" {
		if actorID == id && in.Status != identity.StatusEnabled {
			return errf(400, "不能停用当前登录账号")
		}
		u.Status = in.Status
	}
	u.Nickname = identity.Ptr(strings.TrimSpace(in.Nickname))
	u.Remark = identity.Ptr(in.Remark)
	if err := s.id.Update(ctx, u, "Nickname", "Remark", "Status"); err != nil {
		return err
	}
	nextIDs := roleIDs(roles)
	if err := s.id.SetRoles(ctx, id, nextIDs); err != nil {
		return err
	}
	if u.Status != identity.StatusEnabled || !sameIDs(roleIDsOf(u), nextIDs) {
		s.tokens.RevokeUser(ctx, id)
	}
	return nil
}

func (s *Service) ChangeUserStatus(ctx context.Context, actorID, id uint64, status string) error {
	if status != identity.StatusEnabled && status != identity.StatusDisabled {
		return errf(400, "状态无效")
	}
	u, err := s.platformUser(ctx, id)
	if err != nil {
		return err
	}
	if actorID == id && status != identity.StatusEnabled {
		return errf(400, "不能停用当前登录账号")
	}
	if err := s.keepLastSuper(ctx, u, status, nil); err != nil {
		return err
	}
	u.Status = status
	if err := s.id.Update(ctx, u, "Status"); err != nil {
		return err
	}
	if status != identity.StatusEnabled {
		s.tokens.RevokeUser(ctx, id)
	}
	return nil
}

func (s *Service) ResetUserPassword(ctx context.Context, id uint64, password string) error {
	if len(password) < 6 {
		return errf(400, "密码至少 6 位")
	}
	u, err := s.platformUser(ctx, id)
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now()
	u.Password = hash
	u.PasswordChangedAt = &now
	if err := s.id.Update(ctx, u, "Password", "PasswordChangedAt"); err != nil {
		return err
	}
	s.tokens.RevokeUser(ctx, id)
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, actorID, id uint64) error {
	if actorID == id {
		return errf(400, "不能删除当前登录账号")
	}
	u, err := s.platformUser(ctx, id)
	if err != nil {
		return err
	}
	if err := s.keepLastSuper(ctx, u, identity.StatusDisabled, nil); err != nil {
		return err
	}
	if err := s.id.Delete(ctx, id); err != nil {
		return err
	}
	s.tokens.RevokeUser(ctx, id)
	return nil
}

func (s *Service) ListRoles(ctx context.Context) ([]map[string]any, error) {
	roles, err := s.id.ListByScope(ctx, identity.ScopePlatform)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(roles))
	for i := range roles {
		ids, _ := s.id.RoleMenuIDs(ctx, roles[i].ID)
		out = append(out, map[string]any{
			"roleId": roles[i].ID, "roleKey": roles[i].RoleKey, "name": roles[i].Name,
			"status": roles[i].Status, "remark": deref(roles[i].Remark), "builtin": builtinRole(roles[i].RoleKey),
			"menuIds": ids,
		})
	}
	return out, nil
}

type RoleInput struct {
	Name    string   `json:"name"`
	RoleKey string   `json:"roleKey"`
	Status  string   `json:"status"`
	Remark  string   `json:"remark"`
	MenuIDs []uint64 `json:"menuIds"`
}

func (s *Service) CreateRole(ctx context.Context, in RoleInput) (uint64, error) {
	in.RoleKey = strings.TrimSpace(in.RoleKey)
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || !roleKeyOK(in.RoleKey) {
		return 0, errf(400, "请填写角色名称，标识为小写字母、数字和下划线")
	}
	if builtinRole(in.RoleKey) {
		return 0, errf(400, "不能使用内置角色标识")
	}
	found, err := s.id.FindByKeys(ctx, []string{in.RoleKey})
	if err != nil {
		return 0, err
	}
	if len(found) > 0 {
		return 0, errf(400, "角色标识已存在")
	}
	menuIDs, err := s.adminMenuIDs(ctx, in.MenuIDs)
	if err != nil {
		return 0, err
	}
	role := &identity.Role{
		Name: in.Name, RoleKey: in.RoleKey, Scope: identity.ScopePlatform,
		Status: identity.StatusEnabled, Remark: identity.Ptr(in.Remark),
	}
	if err := s.id.CreateRole(ctx, role); err != nil {
		return 0, err
	}
	if err := s.id.ReplaceRoleMenus(ctx, role.ID, menuIDs); err != nil {
		return 0, err
	}
	return role.ID, nil
}

func (s *Service) UpdateRole(ctx context.Context, id uint64, in RoleInput) error {
	role, err := s.id.FindRole(ctx, id)
	if err != nil || role.Scope != identity.ScopePlatform {
		return errf(404, "角色不存在")
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errf(400, "请填写角色名称")
	}
	if in.Status != "" && in.Status != identity.StatusEnabled && in.Status != identity.StatusDisabled {
		return errf(400, "状态无效")
	}
	if role.RoleKey == identity.RoleSuperAdmin && in.Status == identity.StatusDisabled {
		return errf(400, "不能停用超级管理员角色")
	}
	menuIDs, err := s.adminMenuIDs(ctx, in.MenuIDs)
	if err != nil {
		return err
	}
	role.Name = in.Name
	role.Remark = identity.Ptr(in.Remark)
	fields := []string{"Name", "Remark"}
	if in.Status != "" && !builtinRole(role.RoleKey) {
		role.Status = in.Status
		fields = append(fields, "Status")
	}
	if err := s.id.UpdateRole(ctx, role, fields...); err != nil {
		return err
	}
	return s.id.ReplaceRoleMenus(ctx, id, menuIDs)
}

func (s *Service) DeleteRole(ctx context.Context, id uint64) error {
	role, err := s.id.FindRole(ctx, id)
	if err != nil || role.Scope != identity.ScopePlatform {
		return errf(404, "角色不存在")
	}
	if builtinRole(role.RoleKey) {
		return errf(400, "不能删除内置角色")
	}
	n, err := s.id.CountRoleUsers(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errf(400, "该角色仍有用户，不能删除")
	}
	return s.id.DeleteRole(ctx, id)
}

func (s *Service) ListMenus(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.id.ListMenus(ctx, identity.PortalAdmin)
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
			"path": deref(rows[i].Path), "perms": deref(rows[i].Perms), "icon": deref(rows[i].Icon),
			"sort": rows[i].Sort, "visible": rows[i].Visible, "status": rows[i].Status,
		})
	}
	return out, nil
}

type MenuInput struct {
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Sort    uint32 `json:"sort"`
	Visible *bool  `json:"visible"`
}

func (s *Service) UpdateMenu(ctx context.Context, id uint64, in MenuInput) error {
	rows, err := s.id.ListMenus(ctx, identity.PortalAdmin)
	if err != nil {
		return err
	}
	var found bool
	for i := range rows {
		if rows[i].ID == id {
			found = true
			break
		}
	}
	if !found {
		return errf(404, "菜单不存在")
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 64 {
		return errf(400, "请填写菜单名称")
	}
	fields := map[string]any{"name": in.Name, "icon": in.Icon, "sort": in.Sort}
	if in.Visible != nil {
		fields["visible"] = *in.Visible
	}
	return s.id.UpdateMenu(ctx, id, fields)
}

// Impersonate 签发 60 秒一次性票据，返回商户端登录地址。操作本身由操作日志中间件记录。
func (s *Service) Impersonate(ctx context.Context, merchantID uint64) (string, error) {
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return "", errf(404, "商户不存在")
		}
		return "", err
	}
	if m.Status != identity.StatusEnabled {
		return "", errf(400, "商户未启用，不能模拟登录")
	}
	u, err := s.id.FindMerchantAdmin(ctx, merchantID)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return "", errf(400, "该商户没有管理员账号")
		}
		return "", err
	}
	if u.Status != identity.StatusEnabled {
		return "", errf(400, "商户管理员已停用")
	}
	if s.rdb == nil {
		return "", errf(500, "会话服务不可用")
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	ticket := hex.EncodeToString(buf)
	if err := s.rdb.Set(ctx, "imp:"+ticket, strconv.FormatUint(u.ID, 10), 60*time.Second).Err(); err != nil {
		return "", err
	}
	base := strings.TrimRight(s.consoleURL, "/")
	if base == "" {
		base = "http://localhost:5174"
	}
	return base + "/login?ticket=" + ticket, nil
}

// RedeemTicket 兑换模拟登录票据，签发商户端令牌并写登录日志。
func (s *Service) RedeemTicket(ctx context.Context, ticket, ip, uaStr string) (string, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" || s.rdb == nil {
		return "", errf(401, "登录票据无效或已过期")
	}
	raw, err := s.rdb.GetDel(ctx, "imp:"+ticket).Result()
	if err != nil || raw == "" {
		return "", errf(401, "登录票据无效或已过期")
	}
	id, _ := strconv.ParseUint(raw, 10, 64)
	u, err := s.id.FindByID(ctx, id)
	if err != nil || u.Portal() != identity.PortalMerchant || u.Status != identity.StatusEnabled {
		return "", errf(401, "登录票据无效或已过期")
	}
	var mid uint64
	if u.MerchantID != nil {
		mid = *u.MerchantID
	}
	token, err := s.tokens.Issue(ctx, u.ID, mid, identity.PortalMerchant)
	if err != nil {
		return "", err
	}
	msg := "模拟登录"
	_ = s.id.WriteLogin(ctx, &identity.LoginLog{
		Username: identity.Ptr(u.Username), UserID: &u.ID, MerchantID: u.MerchantID,
		Portal: identity.Ptr(identity.PortalMerchant), IP: identity.Ptr(ip), Browser: identity.Ptr(truncate(uaStr, 128)),
		Status: "success", Message: identity.Ptr(msg), CreatedAt: time.Now(),
	})
	return token, nil
}

func (s *Service) platformUser(ctx context.Context, id uint64) (*identity.User, error) {
	u, err := s.id.FindByID(ctx, id)
	if err != nil || u.MerchantID != nil {
		return nil, errf(404, "账号不存在")
	}
	return u, nil
}

func (s *Service) platformRoles(ctx context.Context, ids []uint64) ([]identity.Role, error) {
	uniq := uniqueIDs(ids)
	if len(uniq) == 0 {
		return nil, errf(400, "请选择角色")
	}
	roles, err := s.id.FindByIDs(ctx, uniq)
	if err != nil {
		return nil, err
	}
	if len(roles) != len(uniq) {
		return nil, errf(400, "角色无效")
	}
	for _, r := range roles {
		if r.Scope != identity.ScopePlatform {
			return nil, errf(400, "只能分配总后台角色")
		}
	}
	return roles, nil
}

// keepLastSuper 停用、删除或卸下超级管理员角色时，保证还剩至少一个启用的超级管理员。
// nextRoles 为 nil 表示角色不变，只检查状态。
func (s *Service) keepLastSuper(ctx context.Context, u *identity.User, nextStatus string, nextRoles []identity.Role) error {
	if !u.HasRole(identity.RoleSuperAdmin) || u.Status != identity.StatusEnabled {
		return nil
	}
	still := nextStatus == "" || nextStatus == identity.StatusEnabled
	if nextRoles != nil {
		still = still && hasRoleKey(nextRoles, identity.RoleSuperAdmin)
	}
	if still {
		return nil
	}
	n, err := s.id.CountEnabledSuperExcept(ctx, u.ID)
	if err != nil {
		return err
	}
	if n < 1 {
		return errf(400, "至少保留一个启用的超级管理员")
	}
	return nil
}

func (s *Service) adminMenuIDs(ctx context.Context, ids []uint64) ([]uint64, error) {
	uniq := uniqueIDs(ids)
	if len(uniq) == 0 {
		return []uint64{}, nil
	}
	rows, err := s.id.ListMenus(ctx, identity.PortalAdmin)
	if err != nil {
		return nil, err
	}
	ok := map[uint64]bool{}
	for i := range rows {
		ok[rows[i].ID] = true
	}
	for _, id := range uniq {
		if !ok[id] {
			return nil, errf(400, "菜单无效")
		}
	}
	return uniq, nil
}

func userRow(u *identity.User) map[string]any {
	ids := make([]uint64, 0, len(u.Roles))
	names := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		ids = append(ids, r.ID)
		names = append(names, r.Name)
	}
	return map[string]any{
		"userId": u.ID, "userName": u.Username, "nickName": deref(u.Nickname), "status": u.Status,
		"remark": deref(u.Remark), "roleIds": ids, "roleNames": names,
		"createdAt": u.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func roleIDsOf(u *identity.User) []uint64 {
	out := make([]uint64, 0, len(u.Roles))
	for _, r := range u.Roles {
		out = append(out, r.ID)
	}
	return out
}

func sameIDs(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[uint64]int{}
	for _, id := range a {
		seen[id]++
	}
	for _, id := range b {
		seen[id]--
		if seen[id] < 0 {
			return false
		}
	}
	return true
}

func roleIDs(roles []identity.Role) []uint64 {
	out := make([]uint64, 0, len(roles))
	for _, r := range roles {
		out = append(out, r.ID)
	}
	return out
}

func hasRoleKey(roles []identity.Role, key string) bool {
	for _, r := range roles {
		if r.RoleKey == key {
			return true
		}
	}
	return false
}

func builtinRole(key string) bool {
	return key == identity.RoleSuperAdmin || key == identity.RoleOpsAdmin
}

func roleKeyOK(key string) bool {
	if len(key) < 2 || len(key) > 32 {
		return false
	}
	for i, c := range key {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9' || c == '_':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func uniqueIDs(ids []uint64) []uint64 {
	seen := map[uint64]bool{}
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
