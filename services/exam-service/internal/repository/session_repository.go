package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"gorm.io/gorm"
)

type SessionRepository interface {
	Create(ctx context.Context, session *model.ExamSession) error
	Update(ctx context.Context, session *model.ExamSession) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.ExamSession, error)
	List(ctx context.Context, periodID, pathID *uuid.UUID, sessionType *string, isActive *bool) ([]model.ExamSession, error)
	AddExaminer(ctx context.Context, examiner *model.SessionExaminer) error
	RemoveExaminer(ctx context.Context, sessionID, examinerUserID uuid.UUID) error
	GetExaminersBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.SessionExaminer, error)
	GetSessionsByExaminerID(ctx context.Context, examinerUserID uuid.UUID) ([]model.ExamSession, error)
	IsExaminerAssigned(ctx context.Context, sessionID, examinerUserID uuid.UUID) (bool, error)
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *model.ExamSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepository) Update(ctx context.Context, session *model.ExamSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *sessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.ExamSession{}, "id = ?", id).Error
}

func (r *sessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.ExamSession, error) {
	var session model.ExamSession
	err := r.db.WithContext(ctx).
		Preload("Examiners").
		Where("id = ?", id).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var count int64
	r.db.WithContext(ctx).Model(&model.Participant{}).Where("session_id = ?", session.ID).Count(&count)
	session.ParticipantsCount = int(count)

	return &session, nil
}

func (r *sessionRepository) List(ctx context.Context, periodID, pathID *uuid.UUID, sessionType *string, isActive *bool) ([]model.ExamSession, error) {
	query := r.db.WithContext(ctx).Model(&model.ExamSession{}).Preload("Examiners")

	if periodID != nil && *periodID != uuid.Nil {
		query = query.Where("period_id = ?", *periodID)
	}
	if pathID != nil && *pathID != uuid.Nil {
		query = query.Where("path_id = ?", *pathID)
	}
	if sessionType != nil && *sessionType != "" {
		query = query.Where("type = ?", *sessionType)
	}
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	var sessions []model.ExamSession
	if err := query.Order("start_at ASC").Find(&sessions).Error; err != nil {
		return nil, err
	}

	for i := range sessions {
		var count int64
		r.db.WithContext(ctx).Model(&model.Participant{}).Where("session_id = ?", sessions[i].ID).Count(&count)
		sessions[i].ParticipantsCount = int(count)
	}

	return sessions, nil
}

func (r *sessionRepository) AddExaminer(ctx context.Context, examiner *model.SessionExaminer) error {
	return r.db.WithContext(ctx).Create(examiner).Error
}

func (r *sessionRepository) RemoveExaminer(ctx context.Context, sessionID, examinerUserID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("session_id = ? AND examiner_user_id = ?", sessionID, examinerUserID).
		Delete(&model.SessionExaminer{}).Error
}

func (r *sessionRepository) GetExaminersBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.SessionExaminer, error) {
	var examiners []model.SessionExaminer
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Find(&examiners).Error
	return examiners, err
}

func (r *sessionRepository) GetSessionsByExaminerID(ctx context.Context, examinerUserID uuid.UUID) ([]model.ExamSession, error) {
	var sessions []model.ExamSession
	err := r.db.WithContext(ctx).
		Table("exam_sessions").
		Joins("JOIN session_examiners ON session_examiners.session_id = exam_sessions.id").
		Where("session_examiners.examiner_user_id = ? AND exam_sessions.deleted_at IS NULL", examinerUserID).
		Preload("Examiners").
		Order("exam_sessions.start_at ASC").
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}

	for i := range sessions {
		var count int64
		r.db.WithContext(ctx).Model(&model.Participant{}).Where("session_id = ?", sessions[i].ID).Count(&count)
		sessions[i].ParticipantsCount = int(count)
	}

	return sessions, nil
}

func (r *sessionRepository) IsExaminerAssigned(ctx context.Context, sessionID, examinerUserID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.SessionExaminer{}).
		Where("session_id = ? AND examiner_user_id = ?", sessionID, examinerUserID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
