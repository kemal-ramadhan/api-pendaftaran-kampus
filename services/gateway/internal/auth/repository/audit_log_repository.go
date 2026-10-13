package repository

import (
	"context"

	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/model"
	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	if log.Metadata == "" {
		log.Metadata = "{}"
	}
	return r.db.WithContext(ctx).Create(log).Error
}
