package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/model"
	"gorm.io/gorm"
)

type BroadcastRepository interface {
	FindAllTemplates(ctx context.Context) ([]model.BroadcastTemplate, error)
	FindTemplateByID(ctx context.Context, id uint) (*model.BroadcastTemplate, error)
	FindTemplateByCode(ctx context.Context, code string) (*model.BroadcastTemplate, error)
	CreateTemplate(ctx context.Context, t *model.BroadcastTemplate) error
	UpdateTemplate(ctx context.Context, t *model.BroadcastTemplate) error
	DeleteTemplate(ctx context.Context, id uint) error

	CreateLog(ctx context.Context, log *model.BroadcastLog) error
	FindLogs(ctx context.Context, periodID string, limit int) ([]model.BroadcastLog, error)
	FindLogByID(ctx context.Context, id uuid.UUID) (*model.BroadcastLog, error)
}

type broadcastRepository struct {
	db *gorm.DB
}

func NewBroadcastRepository(db *gorm.DB) BroadcastRepository {
	return &broadcastRepository{db: db}
}

func (r *broadcastRepository) FindAllTemplates(ctx context.Context) ([]model.BroadcastTemplate, error) {
	var list []model.BroadcastTemplate
	err := r.db.WithContext(ctx).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *broadcastRepository) FindTemplateByID(ctx context.Context, id uint) (*model.BroadcastTemplate, error) {
	var t model.BroadcastTemplate
	err := r.db.WithContext(ctx).First(&t, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *broadcastRepository) FindTemplateByCode(ctx context.Context, code string) (*model.BroadcastTemplate, error) {
	var t model.BroadcastTemplate
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *broadcastRepository) CreateTemplate(ctx context.Context, t *model.BroadcastTemplate) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *broadcastRepository) UpdateTemplate(ctx context.Context, t *model.BroadcastTemplate) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *broadcastRepository) DeleteTemplate(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.BroadcastTemplate{}, id).Error
}

func (r *broadcastRepository) CreateLog(ctx context.Context, log *model.BroadcastLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *broadcastRepository) FindLogs(ctx context.Context, periodID string, limit int) ([]model.BroadcastLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var logs []model.BroadcastLog
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if periodID != "" {
		query = query.Where("period_id = ?", periodID)
	}
	err := query.Find(&logs).Error
	return logs, err
}

func (r *broadcastRepository) FindLogByID(ctx context.Context, id uuid.UUID) (*model.BroadcastLog, error) {
	var l model.BroadcastLog
	err := r.db.WithContext(ctx).First(&l, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}
