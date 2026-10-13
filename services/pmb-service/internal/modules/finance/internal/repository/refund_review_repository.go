package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"gorm.io/gorm"
)

var (
	ErrRefundReviewNotFound = errors.New("tinjauan pengembalian dana tidak ditemukan")
)

type RefundReviewRepository interface {
	Create(ctx context.Context, review *model.RefundReview) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.RefundReview, error)
	FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*model.RefundReview, error)
	FindAll(ctx context.Context, status string, limit, offset int) ([]model.RefundReview, int64, error)
	Update(ctx context.Context, review *model.RefundReview) error
}

type refundReviewRepository struct {
	db *gorm.DB
}

func NewRefundReviewRepository(db *gorm.DB) RefundReviewRepository {
	return &refundReviewRepository{db: db}
}

func (r *refundReviewRepository) Create(ctx context.Context, review *model.RefundReview) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *refundReviewRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.RefundReview, error) {
	var review model.RefundReview
	err := r.db.WithContext(ctx).
		Preload("Payment").
		Preload("Invoice").
		First(&review, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRefundReviewNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *refundReviewRepository) FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*model.RefundReview, error) {
	var review model.RefundReview
	err := r.db.WithContext(ctx).
		Preload("Payment").
		Preload("Invoice").
		First(&review, "payment_id = ?", paymentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &review, nil
}

func (r *refundReviewRepository) FindAll(ctx context.Context, status string, limit, offset int) ([]model.RefundReview, int64, error) {
	var reviews []model.RefundReview
	var total int64

	q := r.db.WithContext(ctx).Model(&model.RefundReview{})
	if status != "" {
		q = q.Where("status = ?", status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := q.Preload("Payment").
		Preload("Invoice").
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&reviews).Error
	if err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *refundReviewRepository) Update(ctx context.Context, review *model.RefundReview) error {
	return r.db.WithContext(ctx).Save(review).Error
}
