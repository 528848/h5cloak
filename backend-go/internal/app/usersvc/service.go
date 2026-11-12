// Package usersvc 商户端子账号管理：列表、创建、编辑、状态、重置密码、删除、角色列表。
package usersvc

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/app/merchantsvc"
	"cloak/internal/auth"
	"cloak/internal/domain/identity"
)

type Service struct {
	users     identity.UserRepository
	roles     *mysql.IdentityRepo
	merchants identity.MerchantRepository
	tokens    *auth.TokenManager
}

func New(u identity.UserRepository, r *mysql.IdentityRepo, m identity.MerchantRepository, t *auth.TokenManager) *Service {
	return &Service{users: u, roles: r, merchants: m, tokens: t}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

type Row struct {
	UserID      uint64   `json:"userId"`
	Username    string   `json:"userName"`
	Nickname    string   `json:"nickName"`
	Email       string   `json:"email"`
	Phone       string   `json:"phonenumber"`
	Status      string   `json:"status"`
	RoleKeys    []string `json:"roleKeys"`
	RoleNames   []string `json:"roleNames"`
	IsAdmin     bool     `json:"isAdmin"`
	LastLoginAt *string  `json:"lastLoginAt"`
	Remark      string   `json:"remark"`
	CreatedAt   string   `json:"createdAt"`
}

type RoleRow struct {
	RoleID  uint64 `json:"roleId"`
	RoleKey string `json:"roleKey"`
	Name    string `json:"name"`
	Remark  string `json:"remark"`
	Builtin bool   `json:"builtin"`
}

func (s *Service) Roles(ctx context.Context, merchantID uint64) ([]RoleRow, error) {
	roles, err := s.roles.ListForMerchant(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	out := make([]RoleRow, 0, len(roles))
	for _, r := range roles {
		if r.Status != identity.StatusEnabled {
			continue
		}
		out = append(out, RoleRow{
			RoleID: r.ID, RoleKey: r.RoleKey, Name: r.Name,
			Remark: identity.Deref(r.Remark), Builtin: r.MerchantID == nil,
		})
	}
	return out, nil
}

func (s *Service) List(ctx context.Context, merchantID uint64, f identity.UserFilter) ([]Row, int64, error) {
	rows, total, err := s.users.ListByMerchant(ctx, merchantID, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Row, 0, len(rows))
	for i := range rows {
		out = append(out, toRow(&rows[i]))
	}
	return out, total, nil
}

type Input struct {
	Username string
	Password string // 创建必填，编辑忽略
	Nickname string
	Email    string
	Phone    string
	RoleKey  string
	Remark   string
}

// Create 新建子账号。operator 为当前操作者，用于额度与角色上限校验。
func (s *Service) Create(ctx context.Context, operator *identity.User, in Input) (*Row, error) {
	merchantID := *operator.MerchantID
	in.Username = strings.TrimSpace(in.Username)
	if err := merchantsvc.ValidateUsername(in.Username); err != nil {
		return nil, err
	}
	if err := auth.ValidatePasswordPolicy(in.Password); err != nil {
		return nil, errf(400, err.Error())
	}
	if exists, err := s.users.UsernameExists(ctx, in.Username, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, errf(400, "账号已被使用")
	}
	role, err := s.resolveRole(ctx, merchantID, in.RoleKey)
	if err != nil {
		return nil, err
	}
	m, err := s.merchants.FindByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	n, err := s.users.CountByMerchant(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if n >= int64(m.UserQuota) {
		return nil, errf(400, "子账号数量已达上限，请联系平台调整额度")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	u := &identity.User{
		MerchantID: &merchantID,
		Username:   in.Username,
		Nickname:   identity.Ptr(firstNonEmpty(in.Nickname, in.Username)),
		Email:      identity.Ptr(in.Email),
		Phone:      identity.Ptr(in.Phone),
		Password:   hash,
		Status:     identity.StatusEnabled,
		Remark:     identity.Ptr(in.Remark),
	}
	if err := s.users.Create(ctx, u, []uint64{role.ID}); err != nil {
		return nil, err
	}
	u.Roles = []identity.Role{*role}
	r := toRow(u)
	return &r, nil
}

func (s *Service) Update(ctx context.Context, operator *identity.User, id uint64, in Input) error {
	u, err := s.mustGetOwned(ctx, operator, id)
	if err != nil {
		return err
	}
	role, err := s.resolveRole(ctx, *operator.MerchantID, in.RoleKey)
	if err != nil {
		return err
	}
	// 最后一个商户管理员不能降级
	if u.HasRole(identity.RoleMerchantAdmin) && role.RoleKey != identity.RoleMerchantAdmin {
		if err := s.ensureNotLastAdmin(ctx, *operator.MerchantID, u.ID); err != nil {
			return err
		}
	}
	u.Nickname = identity.Ptr(firstNonEmpty(in.Nickname, u.Username))
	u.Email = identity.Ptr(in.Email)
	u.Phone = identity.Ptr(in.Phone)
	u.Remark = identity.Ptr(in.Remark)
	if err := s.users.Update(ctx, u, "Nickname", "Email", "Phone", "Remark"); err != nil {
		return err
	}
	if err := s.users.SetRoles(ctx, u.ID, []uint64{role.ID}); err != nil {
		return err
	}
	s.tokens.RevokeUser(ctx, u.ID) // 角色变更后重新登录以刷新权限
	return nil
}

func (s *Service) ChangeStatus(ctx context.Context, operator *identity.User, id uint64, status string) error {
	if status != identity.StatusEnabled && status != identity.StatusDisabled {
		return errf(400, "状态无效")
	}
	u, err := s.mustGetOwned(ctx, operator, id)
	if err != nil {
		return err
	}
	if u.ID == operator.ID {
		return errf(400, "不能停用自己的账号")
	}
	if status == identity.StatusDisabled && u.HasRole(identity.RoleMerchantAdmin) {
		if err := s.ensureNotLastAdmin(ctx, *operator.MerchantID, u.ID); err != nil {
			return err
		}
	}
	u.Status = status
	if err := s.users.Update(ctx, u, "Status"); err != nil {
		return err
	}
	if status == identity.StatusDisabled {
		s.tokens.RevokeUser(ctx, u.ID)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, operator *identity.User, id uint64, newPwd string) error {
	u, err := s.mustGetOwned(ctx, operator, id)
	if err != nil {
		return err
	}
	if err := auth.ValidatePasswordPolicy(newPwd); err != nil {
		return errf(400, err.Error())
	}
	hash, err := auth.HashPassword(newPwd)
	if err != nil {
		return err
	}
	now := time.Now()
	u.Password = hash
	u.PasswordChangedAt = &now
	if err := s.users.Update(ctx, u, "Password", "PasswordChangedAt"); err != nil {
		return err
	}
	s.tokens.RevokeUser(ctx, u.ID)
	return nil
}

func (s *Service) Delete(ctx context.Context, operator *identity.User, id uint64) error {
	u, err := s.mustGetOwned(ctx, operator, id)
	if err != nil {
		return err
	}
	if u.ID == operator.ID {
		return errf(400, "不能删除自己的账号")
	}
	if u.HasRole(identity.RoleMerchantAdmin) {
		if err := s.ensureNotLastAdmin(ctx, *operator.MerchantID, u.ID); err != nil {
			return err
		}
	}
	s.tokens.RevokeUser(ctx, u.ID)
	return s.users.Delete(ctx, u.ID)
}

func (s *Service) mustGetOwned(ctx context.Context, operator *identity.User, id uint64) (*identity.User, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "账号不存在")
		}
		return nil, err
	}
	if u.MerchantID == nil || operator.MerchantID == nil || *u.MerchantID != *operator.MerchantID {
		return nil, errf(404, "账号不存在")
	}
	return u, nil
}

func (s *Service) resolveRole(ctx context.Context, merchantID uint64, key string) (*identity.Role, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errf(400, "请选择角色")
	}
	roles, err := s.roles.FindByKeys(ctx, []string{key})
	if err != nil || len(roles) == 0 || roles[0].Scope != identity.ScopeMerchant {
		return nil, errf(400, "角色无效")
	}
	role := roles[0]
	if role.MerchantID != nil && *role.MerchantID != merchantID {
		return nil, errf(400, "角色无效")
	}
	if role.Status != identity.StatusEnabled {
		return nil, errf(400, "该角色已停用")
	}
	return &role, nil
}

func (s *Service) ensureNotLastAdmin(ctx context.Context, merchantID, excludeUserID uint64) error {
	rows, _, err := s.users.ListByMerchant(ctx, merchantID, identity.UserFilter{RoleKey: identity.RoleMerchantAdmin, Status: identity.StatusEnabled, Page: 1, Size: 10})
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.ID != excludeUserID {
			return nil
		}
	}
	return errf(400, "至少保留一个启用的商户管理员")
}

func toRow(u *identity.User) Row {
	r := Row{
		UserID:    u.ID,
		Username:  u.Username,
		Nickname:  identity.Deref(u.Nickname),
		Email:     identity.Deref(u.Email),
		Phone:     identity.Deref(u.Phone),
		Status:    u.Status,
		Remark:    identity.Deref(u.Remark),
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		RoleKeys:  []string{},
		RoleNames: []string{},
	}
	for _, ro := range u.Roles {
		r.RoleKeys = append(r.RoleKeys, ro.RoleKey)
		r.RoleNames = append(r.RoleNames, ro.Name)
		if ro.RoleKey == identity.RoleMerchantAdmin {
			r.IsAdmin = true
		}
	}
	if u.LastLoginAt != nil {
		t := u.LastLoginAt.Format("2006-01-02 15:04:05")
		r.LastLoginAt = &t
	}
	return r
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
