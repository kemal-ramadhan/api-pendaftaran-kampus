package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"gorm.io/gorm"
)

type WebhookLogRepository interface {
	Create(ctx context.Context, log *model.PaymentWebhookLog) error
	FindByProviderRef(ctx context.Context, providerRef string) (*model.PaymentWebhookLog, error)
}

type webhookLogRepository struct {
	db *gorm.DB
}

func NewWebhookLogRepository(db *gorm.DB) WebhookLogRepository {
	return &webhookLogRepository{db: db}
}

func (r *webhookLogRepository) Create(ctx context.Context, log *model.PaymentWebhookLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *webhookLogRepository) FindByProviderRef(ctx context.Context, providerRef string) (*model.PaymentWebhookLog, error) {
	var l model.PaymentWebhookLog
	err := r.db.WithContext(ctx).Where("provider_ref = ? AND status = 'processed'", providerRef).First(&l).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}
