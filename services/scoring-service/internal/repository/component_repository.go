package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/model"
	"gorm.io/gorm"
)

type ComponentRepository interface {
	Create(ctx context.Context, comp *model.ScoreComponent) error
	Update(ctx context.Context, comp *model.ScoreComponent) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.ScoreComponent, error)
	FindByPathID(ctx context.Context, pathID uuid.UUID) ([]model.ScoreComponent, error)
	FindByPathAndCode(ctx context.Context, pathID uuid.UUID, code string) (*model.ScoreComponent, error)
	GetTotalWeightByPath(ctx context.Context, pathID uuid.UUID, excludeID *uuid.UUID) (float64, error)
}

type componentRepository struct {
	db *gorm.DB
}

func NewComponentRepository(db *gorm.DB) ComponentRepository {
	return &componentRepository{db: db}
}

func (r *componentRepository) Create(ctx context.Context, comp *model.ScoreComponent) error {
	return r.db.WithContext(ctx).Create(comp).Error
}

func (r *componentRepository) Update(ctx context.Context, comp *model.ScoreComponent) error {
	return r.db.WithContext(ctx).Save(comp).Error
}

func (r *componentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.ScoreComponent{}, "id = ?", id).Error
}

func (r *componentRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.ScoreComponent, error) {
	var comp model.ScoreComponent
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&comp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &comp, nil
}

func (r *componentRepository) FindByPathID(ctx context.Context, pathID uuid.UUID) ([]model.ScoreComponent, error) {
	var comps []model.ScoreComponent
	err := r.db.WithContext(ctx).
		Where("path_id = ?", pathID).
		Order("code ASC").
		Find(&comps).Error
	return comps, err
}

func (r *componentRepository) FindByPathAndCode(ctx context.Context, pathID uuid.UUID, code string) (*model.ScoreComponent, error) {
	var comp model.ScoreComponent
	err := r.db.WithContext(ctx).
		Where("path_id = ? AND code = ?", pathID, code).
		First(&comp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &comp, nil
}

func (r *componentRepository) GetTotalWeightByPath(ctx context.Context, pathID uuid.UUID, excludeID *uuid.UUID) (float64, error) {
	var total float64
	query := r.db.WithContext(ctx).
		Model(&model.ScoreComponent{}).
		Where("path_id = ? AND is_active = true", pathID)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	row := query.Select("COALESCE(SUM(weight_percent), 0)").Row()
	err := row.Scan(&total)
	return total, err
}
