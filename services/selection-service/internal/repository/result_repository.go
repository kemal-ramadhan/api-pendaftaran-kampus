package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResultRepository interface {
	BatchCreate(ctx context.Context, results []model.Result) error
	FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Result, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Result, error)
	WithdrawApplication(ctx context.Context, appID uuid.UUID) (*model.Result, error)
}

type resultRepository struct {
	db *gorm.DB
}

func NewResultRepository(db *gorm.DB) ResultRepository {
	return &resultRepository{db: db}
}

func (r *resultRepository) BatchCreate(ctx context.Context, results []model.Result) error {
	if len(results) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "application_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "decision", "accepted_program", "announced_at", "updated_at",
		}),
	}).Create(&results).Error
}

func (r *resultRepository) FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Result, error) {
	var res model.Result
	err := r.db.WithContext(ctx).
		Where("application_id = ?", appID).
		First(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *resultRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Result, error) {
	var res model.Result
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("announced_at DESC").
		First(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *resultRepository) WithdrawApplication(ctx context.Context, appID uuid.UUID) (*model.Result, error) {
	var res model.Result
	err := r.db.WithContext(ctx).Where("application_id = ?", appID).First(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = r.db.WithContext(ctx).Model(&model.RecommendationItem{}).Where("application_id = ?", appID).Update("decision", model.DecisionWithdrawn).Error
			return nil, nil
		}
		return nil, err
	}

	res.Decision = model.DecisionWithdrawn
	res.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Save(&res).Error; err != nil {
		return nil, err
	}

	_ = r.db.WithContext(ctx).Model(&model.RecommendationItem{}).Where("application_id = ?", appID).Update("decision", model.DecisionWithdrawn).Error
	return &res, nil
}

