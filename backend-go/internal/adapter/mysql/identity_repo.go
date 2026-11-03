package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"cloak/internal/domain/identity"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("record not found")

// IdentityRepo 同时实现 UserRepository、MenuRepository、LogRepository。
type IdentityRepo struct{ db *gorm.DB }

func NewIdentityRepo(db *gorm.DB) *IdentityRepo { return &IdentityRepo{db: db} }

func (r *IdentityRepo) FindByUsername(ctx context.Context, username string) (*identity.User, error) {
	var u identity.User
	err := r.db.WithContext(ctx).
		Preload("Merchant").
		Preload("Roles", "status = ?", identity.StatusEnabled).
		Where("username = ?", username).
		First(&u).Error
	return wrap(&u, err)
}

func (r *IdentityRepo) FindByID(ctx context.Context, id uint64) (*identity.User, error) {
	var u identity.User
	err := r.db.WithContext(ctx).
		Preload("Merchant").
		Preload("Roles", "status = ?", identity.StatusEnabled).
		First(&u, id).Error
	return wrap(&u, err)
}

func (r *IdentityRepo) Update(ctx context.Context, user *identity.User, fields ...string) error {
	q := r.db.WithContext(ctx).Model(user)
	if len(fields) > 0 {
		q = q.Select(fields)
	}
	return q.Updates(user).Error
}

func (r *IdentityRepo) ListByRoles(ctx context.Context, roleIDs []uint64, portal string) ([]identity.Menu, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	var menus []identity.Menu
	err := r.db.WithContext(ctx).
		Table("menus").
		Select("DISTINCT menus.*").
		Joins("JOIN role_menus rm ON rm.menu_id = menus.id").
		Where("rm.role_id IN ? AND menus.status = ? AND menus.portal = ?", roleIDs, identity.StatusEnabled, portal).
		Order("menus.sort ASC, menus.id ASC").
		Find(&menus).Error
	return menus, err
}

// ListAllByPortal 超级管理员使用：该门户全部启用菜单。
func (r *IdentityRepo) ListAllByPortal(ctx context.Context, portal string) ([]identity.Menu, error) {
	var menus []identity.Menu
	err := r.db.WithContext(ctx).
		Where("status = ? AND portal = ?", identity.StatusEnabled, portal).
		Order("sort ASC, id ASC").
		Find(&menus).Error
	return menus, err
}

func (r *IdentityRepo) WriteLogin(ctx context.Context, log *identity.LoginLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *IdentityRepo) WriteOperation(ctx context.Context, log *identity.OperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func wrap[T any](v *T, err error) (*T, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
