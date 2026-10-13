package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/model"
	"gorm.io/gorm"
)

type ServiceAccountRepository interface {
	// Service Accounts
	CreateServiceAccount(ctx context.Context, sa *model.ServiceAccount) error
	GetServiceAccountByCode(ctx context.Context, code string) (*model.ServiceAccount, error)
	GetServiceAccountByID(ctx context.Context, id uuid.UUID) (*model.ServiceAccount, error)
	ListServiceAccounts(ctx context.Context) ([]model.ServiceAccount, error)
	UpdateServiceAccount(ctx context.Context, sa *model.ServiceAccount) error

	// User Accounts & Permissions
	GetUserAccount(ctx context.Context, userID, serviceAccountID uuid.UUID) (*model.UserAccount, error)
	GetUserAccountsByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserAccount, error)
	CreateUserAccount(ctx context.Context, ua *model.UserAccount) error
	UpdateUserAccountStatus(ctx context.Context, uaID uuid.UUID, status string) error
	DeleteUserAccount(ctx context.Context, userID, serviceAccountID uuid.UUID) error
	AddRoleToUserAccount(ctx context.Context, uaID uuid.UUID, roleID uint, scopeType, scopeID *string, createdBy *uuid.UUID) error
	RemoveRoleFromUserAccount(ctx context.Context, uaID uuid.UUID, roleID uint) error
	GetUserAccountRoles(ctx context.Context, uaID uuid.UUID) ([]model.UserAccountRole, error)

	// Signing Keys
	GetActiveSigningKeys(ctx context.Context) ([]model.SigningKey, error)
	SaveSigningKey(ctx context.Context, key *model.SigningKey) error
}

type serviceAccountRepository struct {
	db *gorm.DB
}

func NewServiceAccountRepository(db *gorm.DB) ServiceAccountRepository {
	return &serviceAccountRepository{db: db}
}

func (r *serviceAccountRepository) CreateServiceAccount(ctx context.Context, sa *model.ServiceAccount) error {
	return r.db.WithContext(ctx).Create(sa).Error
}

func (r *serviceAccountRepository) GetServiceAccountByCode(ctx context.Context, code string) (*model.ServiceAccount, error) {
	var sa model.ServiceAccount
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&sa).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sa, nil
}

func (r *serviceAccountRepository) GetServiceAccountByID(ctx context.Context, id uuid.UUID) (*model.ServiceAccount, error) {
	var sa model.ServiceAccount
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sa).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sa, nil
}

func (r *serviceAccountRepository) ListServiceAccounts(ctx context.Context) ([]model.ServiceAccount, error) {
	var list []model.ServiceAccount
	err := r.db.WithContext(ctx).Order("name ASC").Find(&list).Error
	return list, err
}

func (r *serviceAccountRepository) UpdateServiceAccount(ctx context.Context, sa *model.ServiceAccount) error {
	return r.db.WithContext(ctx).Save(sa).Error
}

func (r *serviceAccountRepository) GetUserAccount(ctx context.Context, userID, serviceAccountID uuid.UUID) (*model.UserAccount, error) {
	var ua model.UserAccount
	err := r.db.WithContext(ctx).
		Preload("ServiceAccount").
		Preload("UserAccountRoles.Role").
		Where("user_id = ? AND service_account_id = ?", userID, serviceAccountID).
		First(&ua).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ua, nil
}

func (r *serviceAccountRepository) GetUserAccountsByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserAccount, error) {
	var list []model.UserAccount
	err := r.db.WithContext(ctx).
		Preload("ServiceAccount").
		Preload("UserAccountRoles.Role").
		Where("user_id = ?", userID).
		Find(&list).Error
	return list, err
}

func (r *serviceAccountRepository) CreateUserAccount(ctx context.Context, ua *model.UserAccount) error {
	return r.db.WithContext(ctx).Create(ua).Error
}

func (r *serviceAccountRepository) UpdateUserAccountStatus(ctx context.Context, uaID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.UserAccount{}).
		Where("id = ?", uaID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *serviceAccountRepository) DeleteUserAccount(ctx context.Context, userID, serviceAccountID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND service_account_id = ?", userID, serviceAccountID).
		Delete(&model.UserAccount{}).Error
}

func (r *serviceAccountRepository) AddRoleToUserAccount(ctx context.Context, uaID uuid.UUID, roleID uint, scopeType, scopeID *string, createdBy *uuid.UUID) error {
	uar := model.UserAccountRole{
		UserAccountID: uaID,
		RoleID:        roleID,
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}
	return r.db.WithContext(ctx).Save(&uar).Error
}

func (r *serviceAccountRepository) RemoveRoleFromUserAccount(ctx context.Context, uaID uuid.UUID, roleID uint) error {
	return r.db.WithContext(ctx).
		Where("user_account_id = ? AND role_id = ?", uaID, roleID).
		Delete(&model.UserAccountRole{}).Error
}

func (r *serviceAccountRepository) GetUserAccountRoles(ctx context.Context, uaID uuid.UUID) ([]model.UserAccountRole, error) {
	var list []model.UserAccountRole
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_account_id = ?", uaID).
		Find(&list).Error
	return list, err
}

func (r *serviceAccountRepository) GetActiveSigningKeys(ctx context.Context) ([]model.SigningKey, error) {
	var keys []model.SigningKey
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("created_at DESC").Find(&keys).Error
	return keys, err
}

func (r *serviceAccountRepository) SaveSigningKey(ctx context.Context, key *model.SigningKey) error {
	return r.db.WithContext(ctx).Save(key).Error
}
