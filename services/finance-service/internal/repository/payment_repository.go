package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrPaymentNotFound = errors.New("data pembayaran tidak ditemukan")
)

type PaymentRepository interface {
	DB() *gorm.DB
	CreateInTx(tx *gorm.DB, payment *model.Payment) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error)
	FindByProviderRef(ctx context.Context, providerRef string) (*model.Payment, error)
	ListAwaitingVerification(ctx context.Context, limit, offset int) ([]model.Payment, int64, error)
	UpdateStatusInTx(tx *gorm.DB, paymentID uuid.UUID, status model.PaymentStatus, notes string, verifiedBy *uuid.UUID, paidAt *time.Time) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) DB() *gorm.DB {
	return r.db
}

func (r *paymentRepository) CreateInTx(tx *gorm.DB, payment *model.Payment) error {
	if payment.ID == uuid.Nil {
		payment.ID = uuid.New()
	}
	return tx.Create(payment).Error
}

func (r *paymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	var p model.Payment
	err := r.db.WithContext(ctx).
		Preload("Receipt").
		First(&p, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) FindByProviderRef(ctx context.Context, providerRef string) (*model.Payment, error) {
	var p model.Payment
	err := r.db.WithContext(ctx).
		Preload("Receipt").
		First(&p, "provider_ref = ?", providerRef).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) ListAwaitingVerification(ctx context.Context, limit, offset int) ([]model.Payment, int64, error) {
	var list []model.Payment
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Payment{}).
		Where("status = ?", model.PaymentStatusAwaitingVerification)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	err := q.Order("created_at asc").
		Limit(limit).
		Offset(offset).
		Find(&list).Error

	return list, total, err
}

func (r *paymentRepository) UpdateStatusInTx(tx *gorm.DB, paymentID uuid.UUID, status model.PaymentStatus, notes string, verifiedBy *uuid.UUID, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if notes != "" {
		updates["notes"] = notes
	}
	if verifiedBy != nil {
		updates["verified_by"] = verifiedBy
		now := time.Now()
		updates["verified_at"] = &now
	}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}
	return tx.Model(&model.Payment{}).Where("id = ?", paymentID).Updates(updates).Error
}
