package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/model"
	"gorm.io/gorm"
)

var (
	ErrNotificationNotFound = errors.New("log notifikasi tidak ditemukan")
)

type NotificationRepository interface {
	Create(ctx context.Context, log *model.NotificationLog) error
	FindByIdempotencyKey(ctx context.Context, key string) (*model.NotificationLog, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.NotificationLog, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, retries int, errorMsg string) error
	GetPendingQueued(ctx context.Context, limit int) ([]model.NotificationLog, error)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, log *model.NotificationLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *notificationRepository) FindByIdempotencyKey(ctx context.Context, key string) (*model.NotificationLog, error) {
	var n model.NotificationLog
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}
	return &n, nil
}

func (r *notificationRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.NotificationLog, error) {
	var n model.NotificationLog
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}
	return &n, nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, retries int, errorMsg string) error {
	return r.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        status,
			"retries":       retries,
			"error_message": errorMsg,
			"updated_at":    time.Now(),
		}).Error
}

func (r *notificationRepository) GetPendingQueued(ctx context.Context, limit int) ([]model.NotificationLog, error) {
	var list []model.NotificationLog
	err := r.db.WithContext(ctx).
		Where("status IN ?", []string{string(model.StatusQueued), string(model.StatusFailed)}).
		Where("retries < ?", 3).
		Order("created_at asc").
		Limit(limit).
		Find(&list).Error
	return list, err
}
