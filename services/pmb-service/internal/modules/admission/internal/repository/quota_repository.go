package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"gorm.io/gorm"
)

type QuotaRepository interface {
	FindByPeriodAndPath(ctx context.Context, periodID, pathID uint) ([]model.Quota, error)
	IncrementFilled(ctx context.Context, periodID, pathID uint, studyProgramID uuid.UUID) error
	FindAll(ctx context.Context) ([]model.Quota, error)
	Create(ctx context.Context, q *model.Quota) error
}

type quotaRepository struct {
	db *gorm.DB
}

func NewQuotaRepository(db *gorm.DB) QuotaRepository {
	return &quotaRepository{db: db}
}

func (r *quotaRepository) FindByPeriodAndPath(ctx context.Context, periodID, pathID uint) ([]model.Quota, error) {
	var list []model.Quota
	err := r.db.WithContext(ctx).
		Where("period_id = ? AND admission_path_id = ?", periodID, pathID).
		Find(&list).Error
	return list, err
}

func (r *quotaRepository) IncrementFilled(ctx context.Context, periodID, pathID uint, studyProgramID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Quota{}).
		Where("period_id = ? AND admission_path_id = ? AND study_program_id = ?", periodID, pathID, studyProgramID).
		UpdateColumn("filled_count", gorm.Expr("filled_count + 1")).Error
}

func (r *quotaRepository) FindAll(ctx context.Context) ([]model.Quota, error) {
	var list []model.Quota
	err := r.db.WithContext(ctx).Order("id asc").Find(&list).Error
	return list, err
}

func (r *quotaRepository) Create(ctx context.Context, q *model.Quota) error {
	return r.db.WithContext(ctx).Create(q).Error
}

