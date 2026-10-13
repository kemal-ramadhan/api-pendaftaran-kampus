package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/repository"
)

type RefundReviewService interface {
	ListRefundReviews(ctx context.Context, status string, limit, offset int) ([]model.RefundReview, int64, error)
	UpdateRefundReview(ctx context.Context, id uuid.UUID, status string, notes string, reviewerID uuid.UUID) (*model.RefundReview, error)
}

type refundReviewService struct {
	repo repository.RefundReviewRepository
}

func NewRefundReviewService(repo repository.RefundReviewRepository) RefundReviewService {
	return &refundReviewService{repo: repo}
}

func (s *refundReviewService) ListRefundReviews(ctx context.Context, status string, limit, offset int) ([]model.RefundReview, int64, error) {
	return s.repo.FindAll(ctx, status, limit, offset)
}

func (s *refundReviewService) UpdateRefundReview(ctx context.Context, id uuid.UUID, status string, notes string, reviewerID uuid.UUID) (*model.RefundReview, error) {
	review, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if status != string(model.RefundReviewStatusPending) &&
		status != string(model.RefundReviewStatusCompleted) &&
		status != string(model.RefundReviewStatusRejected) {
		return nil, fmt.Errorf("status tinjauan tidak valid: %s", status)
	}

	now := time.Now()
	review.Status = model.RefundReviewStatus(status)
	review.Notes = notes
	review.ReviewedBy = &reviewerID
	review.ReviewedAt = &now
	review.UpdatedAt = now

	if err := s.repo.Update(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}
