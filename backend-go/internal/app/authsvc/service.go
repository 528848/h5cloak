// Package authsvc 实现登录、退出、当前用户信息、个人中心等用例。
package authsvc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"cloak/internal/adapter/mysql"
	"cloak/internal/auth"
	"cloak/internal/domain/identity"
	"cloak/internal/rbac"
)

// Error 带业务码的错误，供 HTTP 层映射到若依响应。
type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func errf(code int, msg string) error { return &Error{Code: code, Msg: msg} }

// AsError 提取业务错误。
func AsError(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}

type Service struct {
	users  identity.UserRepository
	logs   identity.LogRepository
	rbac   *rbac.Service
	tokens *auth.TokenManager
	locker *auth.LoginLocker
	log    *zap.Logger
}

func New(users identity.UserRepository, logs identity.LogRepository, r *rbac.Service, tokens *auth.TokenManager, locker *auth.LoginLocker, log *zap.Logger) *Service {
	return &Service{users: users, logs: logs, rbac: r, tokens: tokens, locker: locker, log: log}
}

// LoginInput 登录参数。
type LoginInput struct {
	Username string
	Password string
	Portal   string // admin / merchant，前端按构建传入
	IP       string
	UA       string
}

// Login 校验账号并签发 token。
func (s *Service) Login(ctx context.Context, in LoginInput) (string, error) {
	if in.Portal != identity.PortalAdmin && in.Portal != identity.PortalMerchant {
		return "", errf(400, "门户参数无效")
	}
	if remain := s.locker.Locked(ctx, in.Username); remain > 0 {
		s.writeLogin(ctx, nil, in, "locked", fmt.Sprintf("账号锁定中，%d 分钟后重试", int(remain.Minutes())+1))
		return "", errf(423, fmt.Sprintf("登录失败次数过多，请 %d 分钟后再试", int(remain.Minutes())+1))
	}

	user, err := s.users.FindByUsername(ctx, in.Username)
	if err != nil && !errors.Is(err, mysql.ErrNotFound) {
		return "", err
	}
	if user == nil || !auth.VerifyPassword(user.Password, in.Password) {
		locked, remain := s.locker.Fail(ctx, in.Username)
		msg := "账号或密码错误"
		if locked {
			msg = "登录失败次数过多，账号已锁定"
		} else if remain > 0 && remain <= 2 {
			msg = fmt.Sprintf("账号或密码错误，还可尝试 %d 次", remain)
		}
		s.writeLogin(ctx, user, in, "failed", msg)
		return "", errf(401, msg)
	}

	if user.Status != identity.StatusEnabled {
		s.writeLogin(ctx, user, in, "failed", "账号已停用")
		return "", errf(403, "账号已停用，无法登录")
	}
	if user.Portal() != in.Portal {
		s.writeLogin(ctx, user, in, "failed", "门户不匹配")
		if in.Portal == identity.PortalAdmin {
			return "", errf(403, "该账号属于商户端，请打开商户后台登录")
		}
		return "", errf(403, "该账号属于总后台，请打开总后台登录")
	}
	if user.MerchantID != nil {
		if user.Merchant == nil || !user.Merchant.Active(time.Now()) {
			s.writeLogin(ctx, user, in, "failed", "商户不可用")
			return "", errf(403, "商户已停用或已到期，无法登录")
		}
	}

	var merchantID uint64
	if user.MerchantID != nil {
		merchantID = *user.MerchantID
	}
	token, err := s.tokens.Issue(ctx, user.ID, merchantID, in.Portal)
	if err != nil {
		return "", err
	}

	s.locker.Reset(ctx, in.Username)
	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = identity.Ptr(in.IP)
	fields := []string{"LastLoginAt", "LastLoginIP"}
	if auth.NeedsRehash(user.Password) {
		if h, herr := auth.HashPassword(in.Password); herr == nil {
			user.Password = h
			fields = append(fields, "Password")
		}
	}
	if err := s.users.Update(ctx, user, fields...); err != nil {
		s.log.Warn("update last login failed", zap.Error(err))
	}
	s.writeLogin(ctx, user, in, "success", "登录成功")
	return token, nil
}

// Logout 注销会话。
func (s *Service) Logout(ctx context.Context, claims *auth.Claims, ip, ua string) {
	s.tokens.Revoke(ctx, claims)
	user, _ := s.users.FindByID(ctx, claims.UserID)
	s.writeLogin(ctx, user, LoginInput{Portal: claims.Portal, IP: ip, UA: ua}, "logout", "退出登录")
}

// Info 若依 /getInfo 结构。
type Info struct {
	User        map[string]any `json:"user"`
	Roles       []string       `json:"roles"`
	Permissions []string       `json:"permissions"`
}

// GetInfo 当前用户、角色与权限。
func (s *Service) GetInfo(ctx context.Context, u *identity.User) (*Info, error) {
	perms, err := s.rbac.Permissions(ctx, u)
	if err != nil {
		return nil, err
	}
	return &Info{
		User: map[string]any{
			"userId":     u.ID,
			"userName":   u.Username,
			"nickName":   identity.Deref(u.Nickname),
			"avatar":     identity.Deref(u.Avatar),
			"merchantId": u.MerchantID,
			"portal":     u.Portal(),
		},
		Roles:       rbac.RoleKeys(u),
		Permissions: perms,
	}, nil
}

// Routers 动态菜单。
func (s *Service) Routers(ctx context.Context, u *identity.User) ([]rbac.RouterItem, error) {
	return s.rbac.Routers(ctx, u)
}

// ProfileInput 个人资料。
type ProfileInput struct {
	NickName    string
	PhoneNumber string
	Email       string
	Sex         string
}

func (s *Service) UpdateProfile(ctx context.Context, u *identity.User, in ProfileInput) error {
	if in.NickName == "" {
		return errf(400, "昵称不能为空")
	}
	u.Nickname = identity.Ptr(in.NickName)
	u.Phone = identity.Ptr(in.PhoneNumber)
	u.Email = identity.Ptr(in.Email)
	u.Sex = identity.Ptr(in.Sex)
	return s.users.Update(ctx, u, "Nickname", "Phone", "Email", "Sex")
}

func (s *Service) UpdateAvatar(ctx context.Context, u *identity.User, url string) error {
	u.Avatar = identity.Ptr(url)
	return s.users.Update(ctx, u, "Avatar")
}

// UpdatePassword 校验旧密码、策略，写新哈希并踢出该用户其它会话。
func (s *Service) UpdatePassword(ctx context.Context, u *identity.User, oldPwd, newPwd string, keep *auth.Claims) error {
	if !auth.VerifyPassword(u.Password, oldPwd) {
		return errf(400, "旧密码错误")
	}
	if err := auth.ValidatePasswordPolicy(newPwd); err != nil {
		return errf(400, err.Error())
	}
	h, err := auth.HashPassword(newPwd)
	if err != nil {
		return err
	}
	now := time.Now()
	u.Password = h
	u.PasswordChangedAt = &now
	if err := s.users.Update(ctx, u, "Password", "PasswordChangedAt"); err != nil {
		return err
	}
	s.tokens.RevokeUser(ctx, u.ID)
	if keep != nil {
		// 当前会话保留，重新登记
		_, _ = s.tokens.Issue(ctx, u.ID, keep.MerchantID, keep.Portal)
	}
	return nil
}

func (s *Service) writeLogin(ctx context.Context, u *identity.User, in LoginInput, status, msg string) {
	entry := &identity.LoginLog{
		Username:  identity.Ptr(in.Username),
		Portal:    identity.Ptr(in.Portal),
		IP:        identity.Ptr(in.IP),
		Browser:   identity.Ptr(truncate(in.UA, 128)),
		Status:    status,
		Message:   identity.Ptr(msg),
		CreatedAt: time.Now(),
	}
	if u != nil {
		entry.UserID = &u.ID
		entry.MerchantID = u.MerchantID
		if entry.Username == nil {
			entry.Username = &u.Username
		}
	}
	if err := s.logs.WriteLogin(ctx, entry); err != nil {
		s.log.Warn("write login log failed", zap.Error(err))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
