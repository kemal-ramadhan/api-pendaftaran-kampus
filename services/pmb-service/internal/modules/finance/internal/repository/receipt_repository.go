package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"gorm.io/gorm"
)

var (
	ErrReceiptNotFound = errors.New("kwitansi pembayaran tidak ditemukan")
)

type ReceiptRepository interface {
	CreateInTx(tx *gorm.DB, receipt *model.Receipt) error
	FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*model.Receipt, error)
	FindByReceiptNumber(ctx context.Context, receiptNo string) (*model.Receipt, error)
	GenerateReceiptNumber(ctx context.Context, tx *gorm.DB) (string, error)
}

type receiptRepository struct {
	db *gorm.DB
}

func NewReceiptRepository(db *gorm.DB) ReceiptRepository {
	return &receiptRepository{db: db}
}

func (r *receiptRepository) CreateInTx(tx *gorm.DB, receipt *model.Receipt) error {
	if receipt.ID == uuid.Nil {
		receipt.ID = uuid.New()
	}
	return tx.Create(receipt).Error
}

func (r *receiptRepository) FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*model.Receipt, error) {
	var rc model.Receipt
	err := r.db.WithContext(ctx).First(&rc, "payment_id = ?", paymentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReceiptNotFound
		}
		return nil, err
	}
	return &rc, nil
}

func (r *receiptRepository) FindByReceiptNumber(ctx context.Context, receiptNo string) (*model.Receipt, error) {
	var rc model.Receipt
	err := r.db.WithContext(ctx).First(&rc, "receipt_number = ?", receiptNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReceiptNotFound
		}
		return nil, err
	}
	return &rc, nil
}

func (r *receiptRepository) GenerateReceiptNumber(ctx context.Context, tx *gorm.DB) (string, error) {
	dateStr := time.Now().Format("20060102")
	prefix := fmt.Sprintf("KW-%s-", dateStr)

	dbRunner := r.db
	if tx != nil {
		dbRunner = tx
	}

	var count int64
	err := dbRunner.WithContext(ctx).Model(&model.Receipt{}).
		Where("receipt_number LIKE ?", prefix+"%").
		Count(&count).Error
	if err != nil {
		return "", err
	}

	seq := count + 1
	return fmt.Sprintf("KW-%s-%05d", dateStr, seq), nil
}
