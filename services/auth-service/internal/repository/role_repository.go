package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"gorm.io/gorm"
)

type RoleRepository interface {
	FindByCode(ctx context.Context, code string) (*model.Role, error)
	FindAll(ctx context.Context) ([]model.Role, error)
	AssignRole(ctx context.Context, userRole *model.UserRole) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error)
	RemoveUserRoles(ctx context.Context, userID uuid.UUID) error
	ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roles []model.UserRole) error
	RemoveUserRoleByCode(ctx context.Context, userID uuid.UUID, code string) error
	CountActiveSuperadmins(ctx context.Context) (int64, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetAllPermissions(ctx context.Context) ([]model.Permission, error)
	GetRolePermissions(ctx context.Context, roleCode string) ([]string, error)
	UpdateRolePermissions(ctx context.Context, roleCode string, permissionCodes []string) error
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) FindByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindAll(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Order("id asc").Find(&roles).Error
	return roles, err
}

func (r *roleRepository) AssignRole(ctx context.Context, userRole *model.UserRole) error {
	emptyScope := ""
	if userRole.ScopeType == nil {
		userRole.ScopeType = &emptyScope
	}
	if userRole.ScopeID == nil {
		userRole.ScopeID = &emptyScope
	}
	return r.db.WithContext(ctx).Create(userRole).Error
}

func (r *roleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ?", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *roleRepository) RemoveUserRoles(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.UserRole{}).Error
}

func (r *roleRepository) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roles []model.UserRole) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		for i := range roles {
			emptyScope := ""
			if roles[i].ScopeType == nil {
				roles[i].ScopeType = &emptyScope
			}
			if roles[i].ScopeID == nil {
				roles[i].ScopeID = &emptyScope
			}
			if err := tx.Create(&roles[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *roleRepository) RemoveUserRoleByCode(ctx context.Context, userID uuid.UUID, code string) error {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return r.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, role.ID).Delete(&model.UserRole{}).Error
}

func (r *roleRepository) CountActiveSuperadmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Joins("JOIN users ON users.id = user_roles.user_id").
		Where("roles.code = ? AND users.status = ? AND users.deleted_at IS NULL", model.RoleSuperadmin, model.UserStatusActive).
		Distinct("user_roles.user_id").
		Count(&count).Error
	return count, err
}

func (r *roleRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var perms []string
	err := r.db.WithContext(ctx).
		Table("role_permissions").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Distinct("role_permissions.permission_code").
		Pluck("role_permissions.permission_code", &perms).Error
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *roleRepository) GetAllPermissions(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.WithContext(ctx).Order("module asc, code asc").Find(&permissions).Error
	return permissions, err
}

func (r *roleRepository) GetRolePermissions(ctx context.Context, roleCode string) ([]string, error) {
	var perms []string
	err := r.db.WithContext(ctx).
		Table("role_permissions").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Where("roles.code = ?", roleCode).
		Order("role_permissions.permission_code asc").
		Pluck("role_permissions.permission_code", &perms).Error
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *roleRepository) UpdateRolePermissions(ctx context.Context, roleCode string, permissionCodes []string) error {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("code = ?", roleCode).First(&role).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", role.ID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}

		for _, pc := range permissionCodes {
			trimmed := pc
			if trimmed == "" {
				continue
			}
			rp := model.RolePermission{
				RoleID:         role.ID,
				PermissionCode: trimmed,
			}
			if err := tx.Create(&rp).Error; err != nil {
				return err
			}
		}
		return nil
	})
}


