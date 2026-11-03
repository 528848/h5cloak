// Package merchantsvc 总后台的商户管理用例：开通、编辑、状态、额度与开关、重置管理员密码。
package merchantsvc

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloak/internal/adapter/mysql"
	"cloak/internal/app/authsvc"
	"cloak/internal/auth"
	"cloak/internal/domain/identity"
)

type Service struct {
	merchants identity.MerchantRepository
	users     identity.UserRepository
	roles     identity.RoleRepository
	tokens    *auth.TokenManager
}

func New(m identity.MerchantRepository, u identity.UserRepository, r identity.RoleRepository, t *auth.TokenManager) *Service {
	return &Service{merchants: m, users: u, roles: r, tokens: t}
}

func errf(code int, msg string) error { return &authsvc.Error{Code: code, Msg: msg} }

// Row 列表与详情的输出结构。
type Row struct {
	MerchantID      uint64            `json:"merchantId"`
	Name            string            `json:"merchantName"`
	ContactName     string            `json:"contactName"`
	ContactEmail    string            `json:"contactEmail"`
	ContactPhone    string            `json:"contactPhone"`
	Status          string            `json:"status"`
	EffectiveStatus string            `json:"effectiveStatus"` // 到期后为 expired
	ExpireAt        *string           `json:"expireAt"`
	Timezone        string            `json:"timezone"`
	Quota           identity.Quota    `json:"quota"`
	Features        identity.Features `json:"features"`
	Remark          string            `json:"remark"`
	AdminUsername   string            `json:"adminUsername"`
	AdminUserID     uint64            `json:"adminUserId"`
	UserCount       int64             `json:"userCount"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
}

func (s *Service) List(ctx context.Context, f identity.MerchantFilter) ([]Row, int64, error) {
	rows, total, err := s.merchants.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Row, 0, len(rows))
	for i := range rows {
		out = append(out, s.toRow(ctx, &rows[i], false))
	}
	return out, total, nil
}

func (s *Service) Get(ctx context.Context, id uint64) (*Row, error) {
	m, err := s.merchants.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "商户不存在")
		}
		return nil, err
	}
	r := s.toRow(ctx, m, true)
	return &r, nil
}

// Stats 概览卡片数据。
func (s *Service) Stats(ctx context.Context) (map[string]int64, error) {
	counts, err := s.merchants.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}
	var total int64
	for _, c := range counts {
		total += c
	}
	return map[string]int64{
		"total":    total,
		"enabled":  counts[identity.StatusEnabled],
		"disabled": counts[identity.StatusDisabled],
		"frozen":   counts[identity.StatusFrozen],
		"expired":  counts[identity.StatusExpired],
	}, nil
}

// CreateInput 开通商户。
type CreateInput struct {
	Name          string
	ContactName   string
	ContactEmail  string
	ContactPhone  string
	ExpireAt      *time.Time
	Timezone      string
	Remark        string
	Quota         *identity.Quota
	Features      *identity.Features
	AdminUsername string
	AdminPassword string
	AdminNickname string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Row, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.AdminUsername = strings.TrimSpace(in.AdminUsername)
	if in.Name == "" {
		return nil, errf(400, "商户名称不能为空")
	}
	if exists, err := s.merchants.NameExists(ctx, in.Name, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, errf(400, "商户名称已存在")
	}
	if in.AdminUsername == "" || in.AdminPassword == "" {
		return nil, errf(400, "请填写管理员账号和初始密码")
	}
	if err := validateUsername(in.AdminUsername); err != nil {
		return nil, err
	}
	if err := auth.ValidatePasswordPolicy(in.AdminPassword); err != nil {
		return nil, errf(400, err.Error())
	}
	if exists, err := s.users.UsernameExists(ctx, in.AdminUsername, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, errf(400, "管理员账号已被使用")
	}

	m := &identity.Merchant{
		Name:         in.Name,
		ContactName:  identity.Ptr(in.ContactName),
		ContactEmail: identity.Ptr(in.ContactEmail),
		ContactPhone: identity.Ptr(in.ContactPhone),
		Status:       identity.StatusEnabled,
		ExpireAt:     in.ExpireAt,
		Timezone:     firstNonEmpty(in.Timezone, "Asia/Shanghai"),
		Remark:       identity.Ptr(in.Remark),
	}
	q := identity.DefaultQuota()
	if in.Quota != nil {
		q = *in.Quota
	}
	if err := validateQuota(&q); err != nil {
		return nil, err
	}
	m.SetQuota(q)
	f := identity.DefaultFeatures()
	if in.Features != nil {
		f = *in.Features
	}
	m.SetFeatures(f)

	if err := s.merchants.Create(ctx, m); err != nil {
		return nil, err
	}

	roles, err := s.roles.FindByKeys(ctx, []string{identity.RoleMerchantAdmin})
	if err != nil || len(roles) == 0 {
		return nil, errf(500, "商户管理员角色不存在")
	}
	hash, err := auth.HashPassword(in.AdminPassword)
	if err != nil {
		return nil, err
	}
	admin := &identity.User{
		MerchantID: &m.ID,
		Username:   in.AdminUsername,
		Nickname:   identity.Ptr(firstNonEmpty(in.AdminNickname, "商户管理员")),
		Password:   hash,
		Status:     identity.StatusEnabled,
	}
	if err := s.users.Create(ctx, admin, []uint64{roles[0].ID}); err != nil {
		return nil, err
	}
	r := s.toRow(ctx, m, true)
	return &r, nil
}

// UpdateInput 编辑基础信息。
type UpdateInput struct {
	Name         string
	ContactName  string
	ContactEmail string
	ContactPhone string
	ExpireAt     *time.Time
	ClearExpire  bool
	Timezone     string
	Remark       string
}

func (s *Service) Update(ctx context.Context, id uint64, in UpdateInput) error {
	m, err := s.mustGet(ctx, id)
	if err != nil {
		return err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errf(400, "商户名称不能为空")
	}
	if exists, err := s.merchants.NameExists(ctx, in.Name, id); err != nil {
		return err
	} else if exists {
		return errf(400, "商户名称已存在")
	}
	m.Name = in.Name
	m.ContactName = identity.Ptr(in.ContactName)
	m.ContactEmail = identity.Ptr(in.ContactEmail)
	m.ContactPhone = identity.Ptr(in.ContactPhone)
	m.Timezone = firstNonEmpty(in.Timezone, m.Timezone)
	m.Remark = identity.Ptr(in.Remark)
	if in.ClearExpire {
		m.ExpireAt = nil
	} else if in.ExpireAt != nil {
		m.ExpireAt = in.ExpireAt
	}
	// 延长到期后自动从 expired 恢复
	if m.Status == identity.StatusExpired && (m.ExpireAt == nil || m.ExpireAt.After(time.Now())) {
		m.Status = identity.StatusEnabled
	}
	return s.merchants.Update(ctx, m, "Name", "ContactName", "ContactEmail", "ContactPhone", "Timezone", "Remark", "ExpireAt", "Status")
}

// ChangeStatus 启用 / 停用 / 冻结。停用与冻结会踢出该商户全部会话。
func (s *Service) ChangeStatus(ctx context.Context, id uint64, status string) error {
	switch status {
	case identity.StatusEnabled, identity.StatusDisabled, identity.StatusFrozen:
	default:
		return errf(400, "状态无效")
	}
	m, err := s.mustGet(ctx, id)
	if err != nil {
		return err
	}
	if status == identity.StatusEnabled && m.ExpireAt != nil && m.ExpireAt.Before(time.Now()) {
		return errf(400, "商户已到期，请先延长到期时间")
	}
	m.Status = status
	if err := s.merchants.Update(ctx, m, "Status"); err != nil {
		return err
	}
	if status != identity.StatusEnabled {
		s.revokeMerchantSessions(ctx, id)
	}
	return nil
}

// UpdateQuota 额度与功能开关。
func (s *Service) UpdateQuota(ctx context.Context, id uint64, q identity.Quota, f identity.Features) error {
	m, err := s.mustGet(ctx, id)
	if err != nil {
		return err
	}
	if err := validateQuota(&q); err != nil {
		return err
	}
	m.SetQuota(q)
	m.SetFeatures(f)
	return s.merchants.Update(ctx, m,
		"DecisionQuota", "SiteQuota", "DomainQuota", "CodeQuota", "UserQuota", "LogRetentionDays", "QuotaPolicy", "Features")
}

// ResetAdminPassword 重置商户管理员密码，返回管理员账号。
func (s *Service) ResetAdminPassword(ctx context.Context, id uint64, newPwd string) (string, error) {
	if _, err := s.mustGet(ctx, id); err != nil {
		return "", err
	}
	if err := auth.ValidatePasswordPolicy(newPwd); err != nil {
		return "", errf(400, err.Error())
	}
	admin, err := s.users.FindMerchantAdmin(ctx, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return "", errf(404, "该商户没有管理员账号")
		}
		return "", err
	}
	hash, err := auth.HashPassword(newPwd)
	if err != nil {
		return "", err
	}
	now := time.Now()
	admin.Password = hash
	admin.PasswordChangedAt = &now
	if err := s.users.Update(ctx, admin, "Password", "PasswordChangedAt"); err != nil {
		return "", err
	}
	s.tokens.RevokeUser(ctx, admin.ID)
	return admin.Username, nil
}

func (s *Service) mustGet(ctx context.Context, id uint64) (*identity.Merchant, error) {
	m, err := s.merchants.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, mysql.ErrNotFound) {
			return nil, errf(404, "商户不存在")
		}
		return nil, err
	}
	return m, nil
}

func (s *Service) revokeMerchantSessions(ctx context.Context, merchantID uint64) {
	users, _, err := s.users.ListByMerchant(ctx, merchantID, identity.UserFilter{Page: 1, Size: 200})
	if err != nil {
		return
	}
	for i := range users {
		s.tokens.RevokeUser(ctx, users[i].ID)
	}
}

func (s *Service) toRow(ctx context.Context, m *identity.Merchant, detail bool) Row {
	r := Row{
		MerchantID:      m.ID,
		Name:            m.Name,
		ContactName:     identity.Deref(m.ContactName),
		ContactEmail:    identity.Deref(m.ContactEmail),
		ContactPhone:    identity.Deref(m.ContactPhone),
		Status:          m.Status,
		EffectiveStatus: m.Status,
		Timezone:        m.Timezone,
		Quota:           m.QuotaOf(),
		Features:        m.FeaturesOf(),
		Remark:          identity.Deref(m.Remark),
		CreatedAt:       m.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       m.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if m.ExpireAt != nil {
		t := m.ExpireAt.Format("2006-01-02 15:04:05")
		r.ExpireAt = &t
		if m.Status == identity.StatusEnabled && m.ExpireAt.Before(time.Now()) {
			r.EffectiveStatus = identity.StatusExpired
		}
	}
	if admin, err := s.users.FindMerchantAdmin(ctx, m.ID); err == nil && admin != nil {
		r.AdminUsername = admin.Username
		r.AdminUserID = admin.ID
	}
	if detail {
		r.UserCount, _ = s.users.CountByMerchant(ctx, m.ID)
	}
	return r
}

func validateQuota(q *identity.Quota) error {
	if q.QuotaPolicy == "" {
		q.QuotaPolicy = "alert"
	}
	if !identity.ValidQuotaPolicies[q.QuotaPolicy] {
		return errf(400, "额度策略无效")
	}
	if q.LogRetentionDays < 7 || q.LogRetentionDays > 365 {
		return errf(400, "日志保留天数须在 7-365 之间")
	}
	if q.UserQuota < 1 {
		return errf(400, "子账号数至少为 1")
	}
	return nil
}

func validateUsername(u string) error {
	if len(u) < 3 || len(u) > 32 {
		return errf(400, "账号长度须为 3-32 位")
	}
	for _, r := range u {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' || r == '@'
		if !ok {
			return errf(400, "账号只能包含字母、数字、_ . - @")
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ValidateUsername 导出给子账号服务复用。
func ValidateUsername(u string) error { return validateUsername(u) }
