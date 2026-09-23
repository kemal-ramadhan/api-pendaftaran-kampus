package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecommendationRepository interface {
	SaveRecommendation(ctx context.Context, rec *model.Recommendation, items []model.RecommendationItem) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Recommendation, error)
	FindByPeriodAndProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (*model.Recommendation, error)
	ListByPeriod(ctx context.Context, periodID uuid.UUID, status *model.RecommendationStatus) ([]model.Recommendation, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.RecommendationStatus, actorID *uuid.UUID, note string) error
}

type recommendationRepository struct {
	db *gorm.DB
}

func NewRecommendationRepository(db *gorm.DB) RecommendationRepository {
	return &recommendationRepository{db: db}
}

func (r *recommendationRepository) SaveRecommendation(ctx context.Context, rec *model.Recommendation, items []model.RecommendationItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Upsert header rekomendasi
		now := time.Now()
		rec.UpdatedAt = now
		if rec.CreatedAt.IsZero() {
			rec.CreatedAt = now
		}

		err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "period_id"}, {Name: "path_id"}, {Name: "study_program_code"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"status", "note", "updated_at",
			}),
		}).Create(rec).Error
		if err != nil {
			return err
		}

		// Jika rec.ID baru ter-generate atau ambil dari record yang ada
		var existing model.Recommendation
		if err := tx.Where("period_id = ? AND path_id = ? AND study_program_code = ?", rec.PeriodID, rec.PathID, rec.StudyProgramCode).First(&existing).Error; err != nil {
			return err
		}
		rec.ID = existing.ID

		// 2. Hapus item lama dan simpan item baru
		if err := tx.Where("recommendation_id = ?", rec.ID).Delete(&model.RecommendationItem{}).Error; err != nil {
			return err
		}

		if len(items) > 0 {
			for i := range items {
				items[i].RecommendationID = rec.ID
				items[i].CreatedAt = now
				items[i].UpdatedAt = now
			}
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *recommendationRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Recommendation, error) {
	var rec model.Recommendation
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("id = ?", id).
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *recommendationRepository) FindByPeriodAndProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (*model.Recommendation, error) {
	var rec model.Recommendation
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("period_id = ? AND path_id = ? AND study_program_code = ?", periodID, pathID, programCode).
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *recommendationRepository) ListByPeriod(ctx context.Context, periodID uuid.UUID, status *model.RecommendationStatus) ([]model.Recommendation, error) {
	var list []model.Recommendation
	query := r.db.WithContext(ctx).
		Preload("Items").
		Where("period_id = ?", periodID)

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	err := query.Order("study_program_code ASC").Find(&list).Error
	return list, err
}

func (r *recommendationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.RecommendationStatus, actorID *uuid.UUID, note string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}
	if note != "" {
		updates["note"] = note
	}

	if status == model.StatusSubmitted {
		updates["submitted_by"] = actorID
		updates["submitted_at"] = &now
	} else if status == model.StatusApproved {
		updates["approved_by"] = actorID
		updates["approved_at"] = &now
	}

	return r.db.WithContext(ctx).Model(&model.Recommendation{}).Where("id = ?", id).Updates(updates).Error
}
