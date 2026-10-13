package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/model"
	"gorm.io/gorm"
)

type OTPRepository interface {
	Create(ctx context.Context, otp *model.UserOTP) error
	FindLatestActive(ctx context.Context, userID uuid.UUID, purpose string) (*model.UserOTP, error)
	IncrementAttempts(ctx context.Context, otpID uuid.UUID) (int16, error)
	MarkConsumed(ctx context.Context, otpID uuid.UUID) error
	InvalidateActiveOTPs(ctx context.Context, userID uuid.UUID, purpose string) error
}

type otpRepository struct {
	db *gorm.DB
}

func NewOTPRepository(db *gorm.DB) OTPRepository {
	return &otpRepository{db: db}
}

func (r *otpRepository) Create(ctx context.Context, otp *model.UserOTP) error {
	return r.db.WithContext(ctx).Create(otp).Error
}

func (r *otpRepository) FindLatestActive(ctx context.Context, userID uuid.UUID, purpose string) (*model.UserOTP, error) {
	var otp model.UserOTP
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND purpose = ? AND consumed_at IS NULL", userID, purpose).
		Order("created_at DESC").
		First(&otp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &otp, nil
}

func (r *otpRepository) IncrementAttempts(ctx context.Context, otpID uuid.UUID) (int16, error) {
	var otp model.UserOTP
	err := r.db.WithContext(ctx).Model(&otp).
		Where("id = ?", otpID).
		Update("attempts", gorm.Expr("attempts + ?", 1)).
		Error
	if err != nil {
		return 0, err
	}

	err = r.db.WithContext(ctx).Select("attempts").Where("id = ?", otpID).First(&otp).Error
	if err != nil {
		return 0, err
	}
	return otp.Attempts, nil
}

func (r *otpRepository) MarkConsumed(ctx context.Context, otpID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.UserOTP{}).
		Where("id = ?", otpID).
		Update("consumed_at", &now).Error
}

func (r *otpRepository) InvalidateActiveOTPs(ctx context.Context, userID uuid.UUID, purpose string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.UserOTP{}).
		Where("user_id = ? AND purpose = ? AND consumed_at IS NULL", userID, purpose).
		Update("consumed_at", &now).Error
}
