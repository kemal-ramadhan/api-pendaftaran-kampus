package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"gorm.io/gorm"
)

type OnboardingRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.AdmissionOnboarding, error)
	ConfirmStep(ctx context.Context, userID uuid.UUID, stepKey string) error
	IsStepConfirmed(ctx context.Context, userID uuid.UUID, stepKey string) (bool, error)
}

type onboardingRepository struct {
	db *gorm.DB
}

func NewOnboardingRepository(db *gorm.DB) OnboardingRepository {
	return &onboardingRepository{db: db}
}

func (r *onboardingRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.AdmissionOnboarding, error) {
	var records []model.AdmissionOnboarding
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&records).Error
	return records, err
}

func (r *onboardingRepository) ConfirmStep(ctx context.Context, userID uuid.UUID, stepKey string) error {
	now := time.Now()
	var rec model.AdmissionOnboarding
	err := r.db.WithContext(ctx).Where("user_id = ? AND step_key = ?", userID, stepKey).First(&rec).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			newRec := model.AdmissionOnboarding{
				ID:          uuid.New(),
				UserID:      userID,
				StepKey:     stepKey,
				ConfirmedAt: &now,
			}
			return r.db.WithContext(ctx).Create(&newRec).Error
		}
		return err
	}
	rec.ConfirmedAt = &now
	return r.db.WithContext(ctx).Save(&rec).Error
}

func (r *onboardingRepository) IsStepConfirmed(ctx context.Context, userID uuid.UUID, stepKey string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AdmissionOnboarding{}).
		Where("user_id = ? AND step_key = ? AND confirmed_at IS NOT NULL", userID, stepKey).
		Count(&count).Error
	return count > 0, err
}
