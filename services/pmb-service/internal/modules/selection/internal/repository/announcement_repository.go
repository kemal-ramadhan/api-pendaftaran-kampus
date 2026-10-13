package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/model"
	"gorm.io/gorm"
)

type AnnouncementRepository interface {
	Create(ctx context.Context, ann *model.Announcement) error
	FindByPeriodID(ctx context.Context, periodID uuid.UUID) (*model.Announcement, error)
}

type announcementRepository struct {
	db *gorm.DB
}

func NewAnnouncementRepository(db *gorm.DB) AnnouncementRepository {
	return &announcementRepository{db: db}
}

func (r *announcementRepository) Create(ctx context.Context, ann *model.Announcement) error {
	return r.db.WithContext(ctx).Create(ann).Error
}

func (r *announcementRepository) FindByPeriodID(ctx context.Context, periodID uuid.UUID) (*model.Announcement, error) {
	var ann model.Announcement
	err := r.db.WithContext(ctx).
		Where("period_id = ?", periodID).
		Order("published_at DESC").
		First(&ann).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ann, nil
}
