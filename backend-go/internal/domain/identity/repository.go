package identity

import "context"

// UserRepository 账号读写。
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*User, error) // 预加载 Merchant 与 Roles
	FindByID(ctx context.Context, id uint64) (*User, error)
	Update(ctx context.Context, user *User, fields ...string) error
	Create(ctx context.Context, user *User, roleIDs []uint64) error
	Delete(ctx context.Context, id uint64) error
	SetRoles(ctx context.Context, userID uint64, roleIDs []uint64) error
	UsernameExists(ctx context.Context, username string, excludeID uint64) (bool, error)
	ListByMerchant(ctx context.Context, merchantID uint64, f UserFilter) ([]User, int64, error)
	CountByMerchant(ctx context.Context, merchantID uint64) (int64, error)
	// FindMerchantAdmin 返回商户的管理员账号（首个 merchant_admin 角色用户）。
	FindMerchantAdmin(ctx context.Context, merchantID uint64) (*User, error)
}

// UserFilter 子账号列表筛选。
type UserFilter struct {
	Keyword string // 账号或昵称
	Status  string
	RoleKey string
	Page    int
	Size    int
}

// MerchantRepository 商户读写。
type MerchantRepository interface {
	FindByID(ctx context.Context, id uint64) (*Merchant, error)
	Create(ctx context.Context, m *Merchant) error
	Update(ctx context.Context, m *Merchant, fields ...string) error
	List(ctx context.Context, f MerchantFilter) ([]Merchant, int64, error)
	CountByStatus(ctx context.Context) (map[string]int64, error)
	NameExists(ctx context.Context, name string, excludeID uint64) (bool, error)
}

// MerchantFilter 商户列表筛选。
type MerchantFilter struct {
	Keyword string // 名称或联系人
	Status  string
	Page    int
	Size    int
}

// RoleRepository 角色读取与商户自建角色维护。
type RoleRepository interface {
	ListByScope(ctx context.Context, scope string) ([]Role, error)
	// ListForMerchant 内置商户角色，加上该商户自己创建的角色（含停用）。
	ListForMerchant(ctx context.Context, merchantID uint64) ([]Role, error)
	FindByKeys(ctx context.Context, keys []string) ([]Role, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]Role, error)
	FindRole(ctx context.Context, id uint64) (*Role, error)
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role, fields ...string) error
	DeleteRole(ctx context.Context, id uint64) error
	CountRoleUsers(ctx context.Context, roleID uint64) (int64, error)
	RoleMenuIDs(ctx context.Context, roleID uint64) ([]uint64, error)
	ReplaceRoleMenus(ctx context.Context, roleID uint64, menuIDs []uint64) error
	ListMenus(ctx context.Context, portal string) ([]Menu, error)
}

// MenuRepository 菜单读取。
type MenuRepository interface {
	// ListByRoles 返回角色集合可见的启用菜单（含按钮），按 sort 排序、去重。
	ListByRoles(ctx context.Context, roleIDs []uint64, portal string) ([]Menu, error)
}

// LogRepository 登录日志与操作日志写入。
type LogRepository interface {
	WriteLogin(ctx context.Context, log *LoginLog) error
	WriteOperation(ctx context.Context, log *OperationLog) error
}
