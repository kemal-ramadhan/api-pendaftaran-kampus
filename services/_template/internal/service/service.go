package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/pagination"
	"github.com/pkm-asia/pmb-backend/services/_template/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/_template/internal/repository"
)

type TemplateService interface {
	List(ctx context.Context, q pagination.Query) ([]dto.ItemResponse, int64, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*dto.ItemResponse, error)
}

type templateService struct {
	repo repository.TemplateRepository
}

func NewTemplateService(repo repository.TemplateRepository) TemplateService {
	return &templateService{repo: repo}
}

func (s *templateService) List(ctx context.Context, q pagination.Query) ([]dto.ItemResponse, int64, error) {
	items, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return nil, 0, err
	}

	res := make([]dto.ItemResponse, len(items))
	for i, item := range items {
		res[i] = dto.ItemResponse{
			ID:   item.ID,
			Name: item.Name,
		}
	}
	return res, total, nil
}

func (s *templateService) GetByUserID(ctx context.Context, userID uuid.UUID) (*dto.ItemResponse, error) {
	item, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.ItemResponse{
		ID:   item.ID,
		Name: item.Name,
	}, nil
}
