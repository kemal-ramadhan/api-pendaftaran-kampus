package repository

import (
	"context"

	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/model"
	"gorm.io/gorm"
)

type ActivityRepository interface {
	Create(ctx context.Context, e *model.ActivityEvent) error
	FindRecent(ctx context.Context, periodID string, limit int, allowedServices []string) ([]model.ActivityEvent, error)
}

type activityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{db: db}
}

func (r *activityRepository) Create(ctx context.Context, e *model.ActivityEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *activityRepository) FindRecent(ctx context.Context, periodID string, limit int, allowedServices []string) ([]model.ActivityEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var events []model.ActivityEvent
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)

	if periodID != "" {
		query = query.Where("period_id = ?", periodID)
	}
	if len(allowedServices) > 0 {
		query = query.Where("service IN ?", allowedServices)
	}

	err := query.Find(&events).Error
	return events, err
}
