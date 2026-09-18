package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"gorm.io/gorm"
)

type PasswordResetRepository interface {
	Create(ctx context.Context, reset *model.PasswordReset) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordReset, error)
	MarkUsed(ctx context.Context, resetID uuid.UUID) error
	InvalidateActiveTokens(ctx context.Context, userID uuid.UUID) error
}

type passwordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

func (r *passwordResetRepository) Create(ctx context.Context, reset *model.PasswordReset) error {
	return r.db.WithContext(ctx).Create(reset).Error
}

func (r *passwordResetRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordReset, error) {
	var reset model.PasswordReset
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&reset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reset, nil
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, resetID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.PasswordReset{}).
		Where("id = ?", resetID).
		Update("used_at", &now).Error
}

func (r *passwordResetRepository) InvalidateActiveTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.PasswordReset{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", &now).Error
}
