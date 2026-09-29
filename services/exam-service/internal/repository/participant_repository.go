package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrParticipantNotFound = errors.New("peserta ujian tidak ditemukan")
)

type ParticipantRepository interface {
	AddParticipant(ctx context.Context, p *model.Participant) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Participant, error)
	FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Participant, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Participant, error)
	FindBySessionAndUser(ctx context.Context, sessionID, userID uuid.UUID) (*model.Participant, error)
	FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.Participant, error)
	CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error)
	UpdateAttendance(ctx context.Context, participantID uuid.UUID, attendance string) error
	GetNextParticipantSequence(ctx context.Context, year int) (int64, error)
	SaveExamCard(ctx context.Context, card *model.ExamCard) error
	FindCardByParticipantID(ctx context.Context, participantID uuid.UUID) (*model.ExamCard, error)
	RemoveByApplicationID(ctx context.Context, appID uuid.UUID) error
}

type participantRepository struct {
	db *gorm.DB
}

func NewParticipantRepository(db *gorm.DB) ParticipantRepository {
	return &participantRepository{db: db}
}

func (r *participantRepository) AddParticipant(ctx context.Context, p *model.Participant) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *participantRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Participant, error) {
	var p model.Participant
	err := r.db.WithContext(ctx).
		Preload("Session").
		Preload("ExamCard").
		Where("id = ?", id).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *participantRepository) FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Participant, error) {
	var p model.Participant
	err := r.db.WithContext(ctx).
		Preload("Session").
		Preload("ExamCard").
		Where("application_id = ?", appID).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *participantRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Participant, error) {
	var participants []model.Participant
	err := r.db.WithContext(ctx).
		Preload("Session").
		Preload("ExamCard").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&participants).Error
	return participants, err
}

func (r *participantRepository) FindBySessionAndUser(ctx context.Context, sessionID, userID uuid.UUID) (*model.Participant, error) {
	var p model.Participant
	err := r.db.WithContext(ctx).
		Preload("Session").
		Preload("ExamCard").
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParticipantNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *participantRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.Participant, error) {
	var participants []model.Participant
	err := r.db.WithContext(ctx).
		Preload("Session").
		Preload("ExamCard").
		Where("session_id = ?", sessionID).
		Order("participant_number ASC").
		Find(&participants).Error
	return participants, err
}

func (r *participantRepository) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Participant{}).
		Where("session_id = ?", sessionID).
		Count(&count).Error
	return count, err
}

func (r *participantRepository) UpdateAttendance(ctx context.Context, participantID uuid.UUID, attendance string) error {
	return r.db.WithContext(ctx).
		Model(&model.Participant{}).
		Where("id = ?", participantID).
		Update("attendance", attendance).Error
}

func (r *participantRepository) GetNextParticipantSequence(ctx context.Context, year int) (int64, error) {
	prefix := fmt.Sprintf("EXAM-%d-%%", year)
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Participant{}).
		Where("participant_number LIKE ?", prefix).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count + 1, nil
}

func (r *participantRepository) SaveExamCard(ctx context.Context, card *model.ExamCard) error {
	return r.db.WithContext(ctx).Save(card).Error
}

func (r *participantRepository) FindCardByParticipantID(ctx context.Context, participantID uuid.UUID) (*model.ExamCard, error) {
	var card model.ExamCard
	err := r.db.WithContext(ctx).
		Where("participant_id = ?", participantID).
		First(&card).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &card, nil
}

func (r *participantRepository) RemoveByApplicationID(ctx context.Context, appID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p model.Participant
		err := tx.Where("application_id = ?", appID).First(&p).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		// Auto-submit any in-progress CBT attempts
		now := time.Now()
		_ = tx.Model(&model.CBTAttempt{}).
			Where("participant_id = ? AND status = ?", p.ID, model.CBTAttemptStatusInProgress).
			Updates(map[string]interface{}{
				"status":       model.CBTAttemptStatusAutoSubmitted,
				"submitted_at": now,
				"updated_at":   now,
			})

		// Hapus kartu ujian bila ada
		_ = tx.Where("participant_id = ?", p.ID).Delete(&model.ExamCard{})

		// Hapus peserta
		return tx.Delete(&p).Error
	})
}

