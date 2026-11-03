package identity

import "context"

// UserRepository 账号读写。
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*User, error) // 预加载 Merchant 与 Roles
	FindByID(ctx context.Context, id uint64) (*User, error)
	Update(ctx context.Context, user *User, fields ...string) error
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
