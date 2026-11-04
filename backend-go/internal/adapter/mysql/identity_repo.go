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

func (r *IdentityRepo) Create(ctx context.Context, user *identity.User, roleIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Roles", "Merchant").Create(user).Error; err != nil {
			return err
		}
		return setRoles(tx, user.ID, roleIDs)
	})
}

func (r *IdentityRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&identity.User{}, id).Error
}

func (r *IdentityRepo) SetRoles(ctx context.Context, userID uint64, roleIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return setRoles(tx, userID, roleIDs)
	})
}

func setRoles(tx *gorm.DB, userID uint64, roleIDs []uint64) error {
	if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", userID).Error; err != nil {
		return err
	}
	for _, rid := range roleIDs {
		if err := tx.Exec("INSERT IGNORE INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, rid).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *IdentityRepo) UsernameExists(ctx context.Context, username string, excludeID uint64) (bool, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&identity.User{}).Where("username = ?", username)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&n).Error
	return n > 0, err
}

func (r *IdentityRepo) ListByMerchant(ctx context.Context, merchantID uint64, f identity.UserFilter) ([]identity.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&identity.User{}).Where("users.merchant_id = ?", merchantID)
	if f.Keyword != "" {
		like := "%" + escapeLike(f.Keyword) + "%"
		q = q.Where("users.username LIKE ? OR users.nickname LIKE ?", like, like)
	}
	if f.Status != "" {
		q = q.Where("users.status = ?", f.Status)
	}
	if f.RoleKey != "" {
		q = q.Where("EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id WHERE ur.user_id = users.id AND ro.role_key = ?)", f.RoleKey)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []identity.User
	err := q.Preload("Roles").Order("users.id DESC").Offset(offset(f.Page, f.Size)).Limit(limit(f.Size)).Find(&rows).Error
	return rows, total, err
}

func (r *IdentityRepo) CountByMerchant(ctx context.Context, merchantID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&identity.User{}).Where("merchant_id = ?", merchantID).Count(&n).Error
	return n, err
}

func (r *IdentityRepo) FindMerchantAdmin(ctx context.Context, merchantID uint64) (*identity.User, error) {
	var u identity.User
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles ro ON ro.id = ur.role_id AND ro.role_key = ?", identity.RoleMerchantAdmin).
		Where("users.merchant_id = ?", merchantID).
		Order("users.id ASC").
		First(&u).Error
	return wrap(&u, err)
}

// ListByScope 角色列表。
func (r *IdentityRepo) ListByScope(ctx context.Context, scope string) ([]identity.Role, error) {
	var roles []identity.Role
	err := r.db.WithContext(ctx).Where("scope = ? AND status = ?", scope, identity.StatusEnabled).Order("sort ASC").Find(&roles).Error
	return roles, err
}

func (r *IdentityRepo) FindByKeys(ctx context.Context, keys []string) ([]identity.Role, error) {
	var roles []identity.Role
	err := r.db.WithContext(ctx).Where("role_key IN ?", keys).Find(&roles).Error
	return roles, err
}

func (r *IdentityRepo) FindByIDs(ctx context.Context, ids []uint64) ([]identity.Role, error) {
	var roles []identity.Role
	if len(ids) == 0 {
		return roles, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&roles).Error
	return roles, err
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

// ListPlatformUsers 总后台账号（merchant_id 为空）。
func (r *IdentityRepo) ListPlatformUsers(ctx context.Context, f identity.UserFilter) ([]identity.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&identity.User{}).Where("users.merchant_id IS NULL")
	if f.Keyword != "" {
		like := "%" + escapeLike(f.Keyword) + "%"
		q = q.Where("users.username LIKE ? OR users.nickname LIKE ?", like, like)
	}
	if f.Status != "" {
		q = q.Where("users.status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []identity.User
	err := q.Preload("Roles").Order("users.id ASC").Offset(offset(f.Page, f.Size)).Limit(limit(f.Size)).Find(&rows).Error
	return rows, total, err
}

// CountEnabledSuperExcept 除指定用户外，仍启用的超级管理员数量。
func (r *IdentityRepo) CountEnabledSuperExcept(ctx context.Context, exceptID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&identity.User{}).
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles ro ON ro.id = ur.role_id AND ro.role_key = ?", identity.RoleSuperAdmin).
		Where("users.merchant_id IS NULL AND users.status = ? AND users.id <> ?", identity.StatusEnabled, exceptID).
		Count(&n).Error
	return n, err
}

func (r *IdentityRepo) FindRole(ctx context.Context, id uint64) (*identity.Role, error) {
	var role identity.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	return wrap(&role, err)
}

func (r *IdentityRepo) CreateRole(ctx context.Context, role *identity.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *IdentityRepo) UpdateRole(ctx context.Context, role *identity.Role, fields ...string) error {
	q := r.db.WithContext(ctx).Model(role)
	if len(fields) > 0 {
		q = q.Select(fields)
	}
	return q.Updates(role).Error
}

func (r *IdentityRepo) DeleteRole(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM role_menus WHERE role_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_roles WHERE role_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&identity.Role{}, id).Error
	})
}

func (r *IdentityRepo) CountRoleUsers(ctx context.Context, roleID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("user_roles").Where("role_id = ?", roleID).Count(&n).Error
	return n, err
}

func (r *IdentityRepo) RoleMenuIDs(ctx context.Context, roleID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).Table("role_menus").Where("role_id = ?", roleID).Pluck("menu_id", &ids).Error
	if ids == nil {
		ids = []uint64{}
	}
	return ids, err
}

func (r *IdentityRepo) ReplaceRoleMenus(ctx context.Context, roleID uint64, menuIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM role_menus WHERE role_id = ?", roleID).Error; err != nil {
			return err
		}
		for _, mid := range menuIDs {
			if err := tx.Exec("INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (?, ?)", roleID, mid).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListMenus 某门户的全部菜单（含停用），供总后台菜单管理。
func (r *IdentityRepo) ListMenus(ctx context.Context, portal string) ([]identity.Menu, error) {
	var menus []identity.Menu
	err := r.db.WithContext(ctx).Where("portal = ?", portal).Order("sort ASC, id ASC").Find(&menus).Error
	return menus, err
}

func (r *IdentityRepo) UpdateMenu(ctx context.Context, id uint64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&identity.Menu{}).Where("id = ?", id).Updates(fields).Error
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
