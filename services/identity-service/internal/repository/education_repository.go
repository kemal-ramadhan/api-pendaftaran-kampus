package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/model"
	"gorm.io/gorm"
)

type EducationRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.EducationHistory, error)
	Save(ctx context.Context, edu *model.EducationHistory) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type educationRepository struct {
	db *gorm.DB
}

func NewEducationRepository(db *gorm.DB) EducationRepository {
	return &educationRepository{db: db}
}

func (r *educationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.EducationHistory, error) {
	var list []model.EducationHistory
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("graduation_year desc").Find(&list).Error
	return list, err
}

func (r *educationRepository) Save(ctx context.Context, edu *model.EducationHistory) error {
	return r.db.WithContext(ctx).Save(edu).Error
}

func (r *educationRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.EducationHistory{}).Error
}
