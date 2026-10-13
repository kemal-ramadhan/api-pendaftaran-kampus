package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/repository"
	"github.com/stretchr/testify/assert"
)

type mockRefundReviewRepo struct {
	reviews map[uuid.UUID]*model.RefundReview
}

func newMockRefundReviewRepo() *mockRefundReviewRepo {
	return &mockRefundReviewRepo{
		reviews: make(map[uuid.UUID]*model.RefundReview),
	}
}

func (m *mockRefundReviewRepo) Create(ctx context.Context, review *model.RefundReview) error {
	m.reviews[review.ID] = review
	return nil
}

func (m *mockRefundReviewRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.RefundReview, error) {
	if r, ok := m.reviews[id]; ok {
		return r, nil
	}
	return nil, repository.ErrRefundReviewNotFound
}

func (m *mockRefundReviewRepo) FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*model.RefundReview, error) {
	for _, r := range m.reviews {
		if r.PaymentID == paymentID {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockRefundReviewRepo) FindAll(ctx context.Context, status string, limit, offset int) ([]model.RefundReview, int64, error) {
	var list []model.RefundReview
	for _, r := range m.reviews {
		if status == "" || string(r.Status) == status {
			list = append(list, *r)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockRefundReviewRepo) Update(ctx context.Context, review *model.RefundReview) error {
	m.reviews[review.ID] = review
	return nil
}

func TestRefundReviewService(t *testing.T) {
	repo := newMockRefundReviewRepo()
	svc := NewRefundReviewService(repo)
	ctx := context.Background()

	revID := uuid.New()
	rev := &model.RefundReview{
		ID:            revID,
		PaymentID:     uuid.New(),
		InvoiceID:     uuid.New(),
		ApplicationID: uuid.New(),
		Amount:        250000,
		Status:        model.RefundReviewStatusPending,
		CreatedAt:     time.Now(),
	}
	_ = repo.Create(ctx, rev)

	t.Run("list refund reviews", func(t *testing.T) {
		items, total, err := svc.ListRefundReviews(ctx, "", 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, items, 1)
	})

	t.Run("update refund review", func(t *testing.T) {
		reviewerID := uuid.New()
		updated, err := svc.UpdateRefundReview(ctx, revID, "completed", "Selesai ditransfer manual", reviewerID)
		assert.NoError(t, err)
		assert.Equal(t, model.RefundReviewStatusCompleted, updated.Status)
		assert.Equal(t, "Selesai ditransfer manual", updated.Notes)
		assert.Equal(t, &reviewerID, updated.ReviewedBy)
	})

	t.Run("update refund review with invalid status", func(t *testing.T) {
		reviewerID := uuid.New()
		_, err := svc.UpdateRefundReview(ctx, revID, "invalid_status", "note", reviewerID)
		assert.Error(t, err)
	})
}
