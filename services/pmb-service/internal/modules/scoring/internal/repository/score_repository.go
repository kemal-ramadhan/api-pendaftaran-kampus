package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScoreRepository interface {
	UpsertScore(ctx context.Context, score *model.Score) error
	FindScoresByApplicationID(ctx context.Context, appID uuid.UUID) ([]model.Score, error)
	FindScoreByAppAndComponent(ctx context.Context, appID uuid.UUID, componentCode string) (*model.Score, error)
	UpsertSummary(ctx context.Context, summary *model.ScoreSummary) error
	FindSummaryByApplicationID(ctx context.Context, appID uuid.UUID) (*model.ScoreSummary, error)
	ListSummariesByProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) ([]model.ScoreSummary, error)
	RecalculateRanksInProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) error
	IsProgramLocked(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (bool, error)
	SetProgramLock(ctx context.Context, lock *model.ProgramLock) error
	GetProgramLock(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (*model.ProgramLock, error)
	ListSummariesInternal(ctx context.Context, periodID uuid.UUID, programCode *string) ([]model.ScoreSummary, error)
}

type scoreRepository struct {
	db *gorm.DB
}

func NewScoreRepository(db *gorm.DB) ScoreRepository {
	return &scoreRepository{db: db}
}

func (r *scoreRepository) UpsertScore(ctx context.Context, score *model.Score) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "application_id"}, {Name: "component_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"score", "input_by", "input_at", "note", "updated_at",
		}),
	}).Create(score).Error
}

func (r *scoreRepository) FindScoresByApplicationID(ctx context.Context, appID uuid.UUID) ([]model.Score, error) {
	var scores []model.Score
	err := r.db.WithContext(ctx).
		Where("application_id = ?", appID).
		Order("component_code ASC").
		Find(&scores).Error
	return scores, err
}

func (r *scoreRepository) FindScoreByAppAndComponent(ctx context.Context, appID uuid.UUID, componentCode string) (*model.Score, error) {
	var score model.Score
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND component_code = ?", appID, componentCode).
		First(&score).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &score, nil
}

func (r *scoreRepository) UpsertSummary(ctx context.Context, summary *model.ScoreSummary) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "application_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"period_id", "path_id", "study_program_code", "final_score", "computed_at", "updated_at",
		}),
	}).Create(summary).Error
}

func (r *scoreRepository) FindSummaryByApplicationID(ctx context.Context, appID uuid.UUID) (*model.ScoreSummary, error) {
	var summary model.ScoreSummary
	err := r.db.WithContext(ctx).
		Where("application_id = ?", appID).
		First(&summary).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &summary, nil
}

func (r *scoreRepository) ListSummariesByProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) ([]model.ScoreSummary, error) {
	var summaries []model.ScoreSummary
	query := r.db.WithContext(ctx).
		Where("period_id = ?", periodID)

	if pathID != uuid.Nil {
		query = query.Where("path_id = ?", pathID)
	}
	if programCode != "" {
		query = query.Where("study_program_code = ?", programCode)
	}

	err := query.Order("rank_in_program ASC, final_score DESC").Find(&summaries).Error
	return summaries, err
}

func (r *scoreRepository) RecalculateRanksInProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) error {
	// Jalankan query perankingan dinamis
	rawSQL := `
		WITH ranked AS (
			SELECT id, ROW_NUMBER() OVER (
				PARTITION BY period_id, path_id, study_program_code 
				ORDER BY final_score DESC, computed_at ASC
			) as new_rank
			FROM score_summaries
			WHERE period_id = ? AND path_id = ? AND study_program_code = ?
		)
		UPDATE score_summaries s
		SET rank_in_program = ranked.new_rank
		FROM ranked
		WHERE s.id = ranked.id;
	`
	return r.db.WithContext(ctx).Exec(rawSQL, periodID, pathID, programCode).Error
}

func (r *scoreRepository) IsProgramLocked(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (bool, error) {
	var lock model.ProgramLock
	err := r.db.WithContext(ctx).
		Where("period_id = ? AND path_id = ? AND study_program_code = ?", periodID, pathID, programCode).
		First(&lock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return lock.IsLocked, nil
}

func (r *scoreRepository) SetProgramLock(ctx context.Context, lock *model.ProgramLock) error {
	now := time.Now()
	lock.UpdatedAt = now
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "period_id"}, {Name: "path_id"}, {Name: "study_program_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"is_locked", "locked_by", "locked_at", "updated_at",
		}),
	}).Create(lock).Error
}

func (r *scoreRepository) GetProgramLock(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (*model.ProgramLock, error) {
	var lock model.ProgramLock
	err := r.db.WithContext(ctx).
		Where("period_id = ? AND path_id = ? AND study_program_code = ?", periodID, pathID, programCode).
		First(&lock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &lock, nil
}

func (r *scoreRepository) ListSummariesInternal(ctx context.Context, periodID uuid.UUID, programCode *string) ([]model.ScoreSummary, error) {
	var summaries []model.ScoreSummary
	query := r.db.WithContext(ctx).
		Where("period_id = ?", periodID)

	if programCode != nil && *programCode != "" {
		query = query.Where("study_program_code = ?", *programCode)
	}

	err := query.Order("study_program_code ASC, rank_in_program ASC, final_score DESC").Find(&summaries).Error
	return summaries, err
}
