// Package identity 定义账号、角色、菜单、商户的领域模型。
package identity

import (
	"encoding/json"
	"time"
)

// 门户。
const (
	PortalAdmin    = "admin"
	PortalMerchant = "merchant"
)

// 角色作用域。
const (
	ScopePlatform = "platform"
	ScopeMerchant = "merchant"
)

// 内置角色 key。
const (
	RoleSuperAdmin       = "admin"
	RoleOpsAdmin         = "ops_admin"
	RoleMerchantAdmin    = "merchant_admin"
	RoleMerchantOperator = "merchant_operator"
	RoleMerchantViewer   = "merchant_viewer"
)

// 通用状态。
const (
	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
	StatusExpired  = "expired"
	StatusFrozen   = "frozen"
)

// SuperPermission 超级管理员权限标识。
const SuperPermission = "*:*:*"

type Merchant struct {
	ID               uint64  `gorm:"primaryKey"`
	Name             string  `gorm:"size:255"`
	ContactName      *string `gorm:"size:255"`
	ContactEmail     *string `gorm:"size:255"`
	ContactPhone     *string `gorm:"size:64"`
	Status           string  `gorm:"size:32"`
	ExpireAt         *time.Time
	Timezone         string `gorm:"size:64"`
	DecisionQuota    uint64
	SiteQuota        uint32
	DomainQuota      uint32
	CodeQuota        uint32
	CloakQuota       uint32
	UserQuota        uint32
	LogRetentionDays uint32
	QuotaPolicy      string          `gorm:"size:32"`
	Features         json.RawMessage `gorm:"type:json"`
	Remark           *string         `gorm:"size:500"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Active 商户是否可用于登录与判定。
func (m *Merchant) Active(now time.Time) bool {
	if m.Status != StatusEnabled {
		return false
	}
	if m.ExpireAt != nil && m.ExpireAt.Before(now) {
		return false
	}
	return true
}

type User struct {
	ID                uint64  `gorm:"primaryKey"`
	MerchantID        *uint64 `gorm:"index"`
	Username          string  `gorm:"size:64;uniqueIndex"`
	Nickname          *string `gorm:"size:255"`
	Password          string  `gorm:"size:255"`
	Avatar            *string `gorm:"size:500"`
	Email             *string `gorm:"size:255"`
	Phone             *string `gorm:"size:32"`
	Sex               *string `gorm:"size:8"`
	Status            string  `gorm:"size:32"`
	LastLoginAt       *time.Time
	LastLoginIP       *string `gorm:"size:64"`
	PasswordChangedAt *time.Time
	Remark            *string `gorm:"size:500"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	Merchant *Merchant `gorm:"foreignKey:MerchantID"`
	Roles    []Role    `gorm:"many2many:user_roles;"`
}

// Portal 用户所属门户：有商户的是商户端，否则总后台。
func (u *User) Portal() string {
	if u.MerchantID != nil && *u.MerchantID != 0 {
		return PortalMerchant
	}
	return PortalAdmin
}

// HasRole 是否拥有指定角色 key。
func (u *User) HasRole(key string) bool {
	for _, r := range u.Roles {
		if r.RoleKey == key && r.Status == StatusEnabled {
			return true
		}
	}
	return false
}

type Role struct {
	ID         uint64  `gorm:"primaryKey"`
	Name       string  `gorm:"size:64"`
	RoleKey    string  `gorm:"size:64;uniqueIndex"`
	Scope      string  `gorm:"size:32"`
	MerchantID *uint64 `gorm:"index"`
	Status     string  `gorm:"size:32"`
	Sort       uint32
	Remark     *string `gorm:"size:500"`
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Menus []Menu `gorm:"many2many:role_menus;"`
}

type Menu struct {
	ID        uint64  `gorm:"primaryKey"`
	ParentID  *uint64 `gorm:"index"`
	Portal    string  `gorm:"size:16"`
	Name      string  `gorm:"size:64"`
	Path      *string `gorm:"size:255"`
	Component *string `gorm:"size:255"`
	Query     *string `gorm:"size:255"`
	MenuType  string  `gorm:"size:1"` // M, C, F
	Visible   bool
	Status    string  `gorm:"size:32"`
	Perms     *string `gorm:"size:128"`
	Icon      *string `gorm:"size:64"`
	Sort      uint32
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LoginLog struct {
	ID         uint64 `gorm:"primaryKey"`
	Username   *string
	UserID     *uint64
	MerchantID *uint64
	Portal     *string
	IP         *string
	Location   *string
	Browser    *string
	OS         *string
	Status     string
	Message    *string
	CreatedAt  time.Time
}

type OperationLog struct {
	ID             uint64 `gorm:"primaryKey"`
	RequestID      *string
	ActorUserID    *uint64
	ActorUsername  *string
	ActorRole      *string
	Portal         *string
	MerchantID     *uint64
	Module         string
	Action         string
	Method         string
	Path           string
	ObjectID       *string
	BeforeSnapshot json.RawMessage `gorm:"type:json"`
	AfterSnapshot  json.RawMessage `gorm:"type:json"`
	RequestSummary json.RawMessage `gorm:"type:json"`
	StatusCode     int
	ResultCode     int
	IP             *string
	LatencyMs      uint32
	CreatedAt      time.Time
}

// Ptr 返回字符串指针，空串返回 nil。
func Ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Deref 安全取指针字符串。
func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
