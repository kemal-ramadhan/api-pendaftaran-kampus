package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrInvoiceNotFound = errors.New("tagihan pembayaran tidak ditemukan")
)

type InvoiceRepository interface {
	DB() *gorm.DB
	CreateInTx(tx *gorm.DB, invoice *model.Invoice) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error)
	FindByInvoiceNumber(ctx context.Context, invNo string) (*model.Invoice, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Invoice, error)
	FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Invoice, error)
	GenerateInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error)
	GenerateAdjustmentInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.InvoiceStatus, paidAmount int64) error
	UpdateAmount(ctx context.Context, id uuid.UUID, amount int64) error
	ListAllInvoices(ctx context.Context, periodID *uint, status string, feeType string, search string, limit, offset int) ([]model.Invoice, int64, error)
	UpdateDueDateAndRenewal(ctx context.Context, id uuid.UUID, newDueDate time.Time, renewalCount int) error
}

type invoiceRepository struct {
	db *gorm.DB
}

func NewInvoiceRepository(db *gorm.DB) InvoiceRepository {
	return &invoiceRepository{db: db}
}

func (r *invoiceRepository) DB() *gorm.DB {
	return r.db
}

func (r *invoiceRepository) CreateInTx(tx *gorm.DB, invoice *model.Invoice) error {
	if invoice.ID == uuid.Nil {
		invoice.ID = uuid.New()
	}
	return tx.Create(invoice).Error
}

func (r *invoiceRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error) {
	var inv model.Invoice
	err := r.db.WithContext(ctx).
		Preload("Payments.Receipt").
		First(&inv, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *invoiceRepository) FindByInvoiceNumber(ctx context.Context, invNo string) (*model.Invoice, error) {
	var inv model.Invoice
	err := r.db.WithContext(ctx).
		Preload("Payments.Receipt").
		First(&inv, "invoice_number = ?", invNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *invoiceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Invoice, error) {
	var list []model.Invoice
	err := r.db.WithContext(ctx).
		Preload("Payments.Receipt").
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&list).Error
	return list, err
}

func (r *invoiceRepository) FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Invoice, error) {
	var inv model.Invoice
	err := r.db.WithContext(ctx).
		Preload("Payments.Receipt").
		Where("application_id = ?", appID).
		First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *invoiceRepository) GenerateInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error) {
	dateStr := time.Now().Format("20060102")
	prefix := fmt.Sprintf("INV-%s-", dateStr)

	dbRunner := r.db
	if tx != nil {
		dbRunner = tx
	}

	var count int64
	err := dbRunner.WithContext(ctx).Model(&model.Invoice{}).
		Where("invoice_number LIKE ?", prefix+"%").
		Count(&count).Error
	if err != nil {
		return "", err
	}

	seq := count + 1
	return fmt.Sprintf("INV-%s-%05d", dateStr, seq), nil
}

func (r *invoiceRepository) GenerateAdjustmentInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error) {
	yearStr := time.Now().Format("2006")
	prefix := fmt.Sprintf("INV-REG-%s-", yearStr)

	dbRunner := r.db
	if tx != nil {
		dbRunner = tx
	}

	var count int64
	err := dbRunner.WithContext(ctx).Model(&model.Invoice{}).
		Where("invoice_number LIKE ?", prefix+"%-ADJ").
		Count(&count).Error
	if err != nil {
		return "", err
	}

	seq := count + 1
	return fmt.Sprintf("INV-REG-%s-%04d-ADJ", yearStr, seq), nil
}

func (r *invoiceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.InvoiceStatus, paidAmount int64) error {
	updates := map[string]interface{}{
		"status":      status,
		"paid_amount": paidAmount,
		"updated_at":  time.Now(),
	}
	return r.db.WithContext(ctx).Model(&model.Invoice{}).Where("id = ?", id).Updates(updates).Error
}

func (r *invoiceRepository) UpdateAmount(ctx context.Context, id uuid.UUID, amount int64) error {
	updates := map[string]interface{}{
		"amount":     amount,
		"updated_at": time.Now(),
	}
	return r.db.WithContext(ctx).Model(&model.Invoice{}).Where("id = ?", id).Updates(updates).Error
}

func (r *invoiceRepository) ListAllInvoices(ctx context.Context, periodID *uint, status string, feeType string, search string, limit, offset int) ([]model.Invoice, int64, error) {
	var list []model.Invoice
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Invoice{})
	if periodID != nil && *periodID > 0 {
		q = q.Where("period_id = ?", *periodID)
	}
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	if feeType != "" && feeType != "all" {
		q = q.Where("fee_type = ?", feeType)
	}
	if search != "" {
		q = q.Where("invoice_number ILIKE ?", "%"+search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	err := q.Preload("Payments.Receipt").
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&list).Error

	return list, total, err
}

func (r *invoiceRepository) UpdateDueDateAndRenewal(ctx context.Context, id uuid.UUID, newDueDate time.Time, renewalCount int) error {
	updates := map[string]interface{}{
		"status":        model.InvoiceStatusUnpaid,
		"due_date":      newDueDate,
		"renewal_count": renewalCount,
		"updated_at":    time.Now(),
	}
	return r.db.WithContext(ctx).Model(&model.Invoice{}).Where("id = ?", id).Updates(updates).Error
}

