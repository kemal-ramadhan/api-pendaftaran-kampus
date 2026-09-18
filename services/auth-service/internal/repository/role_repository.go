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
