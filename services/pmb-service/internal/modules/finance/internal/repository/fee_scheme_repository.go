package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"gorm.io/gorm"
)

type FeeSchemeRepository interface {
	FindForRegistration(ctx context.Context, periodID, pathID uint, prodiID *uuid.UUID) (*model.FeeScheme, error)
	Create(ctx context.Context, scheme *model.FeeScheme) error
	FindAll(ctx context.Context) ([]model.FeeScheme, error)
}

type feeSchemeRepository struct {
	db *gorm.DB
}

func NewFeeSchemeRepository(db *gorm.DB) FeeSchemeRepository {
	return &feeSchemeRepository{db: db}
}

func (r *feeSchemeRepository) FindForRegistration(ctx context.Context, periodID, pathID uint, prodiID *uuid.UUID) (*model.FeeScheme, error) {
	var scheme model.FeeScheme
	q := r.db.WithContext(ctx).
		Where("period_id = ? AND admission_path_id = ? AND fee_type = ?", periodID, pathID, "registration")

	if prodiID != nil {
		q = q.Where("study_program_id = ? OR study_program_id IS NULL", *prodiID).
			Order("study_program_id DESC NULLS LAST")
	} else {
		q = q.Where("study_program_id IS NULL")
	}

	err := q.First(&scheme).Error
	if err != nil {
		return nil, err
	}
	return &scheme, nil
}

func (r *feeSchemeRepository) Create(ctx context.Context, scheme *model.FeeScheme) error {
	if scheme.ID == uuid.Nil {
		scheme.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(scheme).Error
}

func (r *feeSchemeRepository) FindAll(ctx context.Context) ([]model.FeeScheme, error) {
	var list []model.FeeScheme
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&list).Error
	return list, err
}
