package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/repository"
)

type ActivityService interface {
	RecordActivity(ctx context.Context, req dto.RecordActivityRequest) error
	GetRecentActivities(ctx context.Context, periodID string, limit int, roles []string) ([]dto.ActivityEventResponse, error)
}

type activityService struct {
	repo repository.ActivityRepository
}

func NewActivityService(repo repository.ActivityRepository) ActivityService {
	return &activityService{repo: repo}
}

func (s *activityService) RecordActivity(ctx context.Context, req dto.RecordActivityRequest) error {
	event := &model.ActivityEvent{
		ID:          uuid.New(),
		PeriodID:    req.PeriodID,
		ActorID:     req.ActorID,
		ActorName:   req.ActorName,
		ActorRole:   req.ActorRole,
		Action:      req.Action,
		ActionLabel: req.ActionLabel,
		TargetType:  req.TargetType,
		TargetID:    req.TargetID,
		TargetLabel: req.TargetLabel,
		Service:     req.Service,
	}
	return s.repo.Create(ctx, event)
}

func (s *activityService) GetRecentActivities(ctx context.Context, periodID string, limit int, roles []string) ([]dto.ActivityEventResponse, error) {
	// Filter service berdasarkan hak akses role (PRD 2.9)
	var allowedServices []string
	isSuperOrAdmin := false
	for _, r := range roles {
		if r == "superadmin" || r == "admin_pmb" || r == "pimpinan" {
			isSuperOrAdmin = true
			break
		}
	}

	if !isSuperOrAdmin {
		for _, r := range roles {
			if r == "keuangan" {
				allowedServices = append(allowedServices, "finance")
			}
			if r == "prodi" || r == "dekan" {
				allowedServices = append(allowedServices, "admission", "selection")
			}
		}
	}

	events, err := s.repo.FindRecent(ctx, periodID, limit, allowedServices)
	if err != nil {
		return nil, err
	}

	res := make([]dto.ActivityEventResponse, len(events))
	for i, e := range events {
		res[i] = dto.ActivityEventResponse{
			ID:          e.ID,
			PeriodID:    e.PeriodID,
			ActorName:   e.ActorName,
			ActorRole:   e.ActorRole,
			Action:      e.Action,
			ActionLabel: e.ActionLabel,
			TargetType:  e.TargetType,
			TargetID:    e.TargetID,
			TargetLabel: e.TargetLabel,
			Service:     e.Service,
			CreatedAt:   e.CreatedAt,
		}
	}
	return res, nil
}
