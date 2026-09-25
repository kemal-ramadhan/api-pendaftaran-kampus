package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrQuestionBankNotFound = errors.New("bank soal tidak ditemukan")
	ErrQuestionNotFound     = errors.New("soal tidak ditemukan")
	ErrCBTExamNotFound      = errors.New("ujian CBT tidak ditemukan")
	ErrCBTAttemptNotFound   = errors.New("sesi pengerjaan CBT tidak ditemukan")
	ErrCBTAnswerNotFound    = errors.New("jawaban CBT tidak ditemukan")
)

type CBTRepository interface {
	DB() *gorm.DB

	// Bank Soal & Soal
	CreateQuestionBank(ctx context.Context, bank *model.QuestionBank) error
	FindQuestionBankByID(ctx context.Context, id uuid.UUID) (*model.QuestionBank, error)
	ListQuestionBanks(ctx context.Context) ([]model.QuestionBank, error)
	CreateQuestion(ctx context.Context, q *model.Question) error
	FindQuestionByID(ctx context.Context, id uuid.UUID) (*model.Question, error)
	ListQuestionsByBankID(ctx context.Context, bankID uuid.UUID) ([]model.Question, error)
	GetQuestionsByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Question, error)

	// Ujian CBT
	CreateCBTExam(ctx context.Context, exam *model.CBTExam) error
	UpdateCBTExam(ctx context.Context, exam *model.CBTExam) error
	FindCBTExamByID(ctx context.Context, id uuid.UUID) (*model.CBTExam, error)
	FindCBTExamBySessionID(ctx context.Context, sessionID uuid.UUID) (*model.CBTExam, error)
	ListCBTExams(ctx context.Context) ([]model.CBTExam, error)

	// Attempt (Pengerjaan)
	CreateAttempt(ctx context.Context, attempt *model.CBTAttempt) error
	UpdateAttempt(ctx context.Context, attempt *model.CBTAttempt) error
	FindAttemptByID(ctx context.Context, id uuid.UUID) (*model.CBTAttempt, error)
	FindAttemptByExamAndParticipant(ctx context.Context, examID, participantID uuid.UUID) (*model.CBTAttempt, error)
	ListAttemptsByExamID(ctx context.Context, examID uuid.UUID) ([]model.CBTAttempt, error)
	FindExpiredInProgressAttempts(ctx context.Context, now time.Time) ([]model.CBTAttempt, error)

	// Jawaban & Event
	UpsertAnswer(ctx context.Context, answer *model.CBTAnswer) error
	FindAnswersByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]model.CBTAnswer, error)
	FindUngradedEssaysByExamID(ctx context.Context, examID uuid.UUID) ([]model.CBTAnswer, error)
	GradeAnswer(ctx context.Context, answerID uuid.UUID, points float64, isCorrect bool, graderID uuid.UUID) error
	CreateEvent(ctx context.Context, event *model.CBTEvent) error
}

type cbtRepository struct {
	db *gorm.DB
}

func NewCBTRepository(db *gorm.DB) CBTRepository {
	return &cbtRepository{db: db}
}

func (r *cbtRepository) DB() *gorm.DB {
	return r.db
}

func (r *cbtRepository) CreateQuestionBank(ctx context.Context, bank *model.QuestionBank) error {
	if bank.ID == uuid.Nil {
		bank.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(bank).Error
}

func (r *cbtRepository) FindQuestionBankByID(ctx context.Context, id uuid.UUID) (*model.QuestionBank, error) {
	var bank model.QuestionBank
	err := r.db.WithContext(ctx).Preload("Questions.Options").First(&bank, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionBankNotFound
		}
		return nil, err
	}
	return &bank, nil
}

func (r *cbtRepository) ListQuestionBanks(ctx context.Context) ([]model.QuestionBank, error) {
	var banks []model.QuestionBank
	err := r.db.WithContext(ctx).Order("name asc").Find(&banks).Error
	return banks, err
}

func (r *cbtRepository) CreateQuestion(ctx context.Context, q *model.Question) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(q).Error
}

func (r *cbtRepository) FindQuestionByID(ctx context.Context, id uuid.UUID) (*model.Question, error) {
	var q model.Question
	err := r.db.WithContext(ctx).Preload("Options").First(&q, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	return &q, nil
}

func (r *cbtRepository) ListQuestionsByBankID(ctx context.Context, bankID uuid.UUID) ([]model.Question, error) {
	var qs []model.Question
	err := r.db.WithContext(ctx).Preload("Options").Where("bank_id = ? AND is_active = true", bankID).Find(&qs).Error
	return qs, err
}

func (r *cbtRepository) GetQuestionsByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Question, error) {
	if len(ids) == 0 {
		return []model.Question{}, nil
	}
	var qs []model.Question
	err := r.db.WithContext(ctx).Preload("Options").Where("id IN ?", ids).Find(&qs).Error
	return qs, err
}

func (r *cbtRepository) CreateCBTExam(ctx context.Context, exam *model.CBTExam) error {
	if exam.ID == uuid.Nil {
		exam.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(exam).Error
}

func (r *cbtRepository) UpdateCBTExam(ctx context.Context, exam *model.CBTExam) error {
	return r.db.WithContext(ctx).Save(exam).Error
}

func (r *cbtRepository) FindCBTExamByID(ctx context.Context, id uuid.UUID) (*model.CBTExam, error) {
	var exam model.CBTExam
	err := r.db.WithContext(ctx).Preload("Rules").Preload("Session").First(&exam, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCBTExamNotFound
		}
		return nil, err
	}
	return &exam, nil
}

func (r *cbtRepository) FindCBTExamBySessionID(ctx context.Context, sessionID uuid.UUID) (*model.CBTExam, error) {
	var exam model.CBTExam
	err := r.db.WithContext(ctx).Preload("Rules").Preload("Session").First(&exam, "session_id = ?", sessionID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCBTExamNotFound
		}
		return nil, err
	}
	return &exam, nil
}

func (r *cbtRepository) ListCBTExams(ctx context.Context) ([]model.CBTExam, error) {
	var exams []model.CBTExam
	err := r.db.WithContext(ctx).Preload("Session").Order("created_at desc").Find(&exams).Error
	return exams, err
}

func (r *cbtRepository) CreateAttempt(ctx context.Context, attempt *model.CBTAttempt) error {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(attempt).Error
}

func (r *cbtRepository) UpdateAttempt(ctx context.Context, attempt *model.CBTAttempt) error {
	return r.db.WithContext(ctx).Save(attempt).Error
}

func (r *cbtRepository) FindAttemptByID(ctx context.Context, id uuid.UUID) (*model.CBTAttempt, error) {
	var att model.CBTAttempt
	err := r.db.WithContext(ctx).Preload("CBTExam.Session").Preload("Participant").Preload("Answers").First(&att, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCBTAttemptNotFound
		}
		return nil, err
	}
	return &att, nil
}

func (r *cbtRepository) FindAttemptByExamAndParticipant(ctx context.Context, examID, participantID uuid.UUID) (*model.CBTAttempt, error) {
	var att model.CBTAttempt
	err := r.db.WithContext(ctx).Preload("CBTExam.Session").Preload("Participant").Preload("Answers").
		Where("cbt_exam_id = ? AND participant_id = ?", examID, participantID).First(&att).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCBTAttemptNotFound
		}
		return nil, err
	}
	return &att, nil
}

func (r *cbtRepository) ListAttemptsByExamID(ctx context.Context, examID uuid.UUID) ([]model.CBTAttempt, error) {
	var list []model.CBTAttempt
	err := r.db.WithContext(ctx).Preload("Participant").Where("cbt_exam_id = ?", examID).Find(&list).Error
	return list, err
}

func (r *cbtRepository) FindExpiredInProgressAttempts(ctx context.Context, now time.Time) ([]model.CBTAttempt, error) {
	var list []model.CBTAttempt
	err := r.db.WithContext(ctx).
		Where("status = ? AND deadline_at IS NOT NULL AND deadline_at < ?", model.CBTAttemptStatusInProgress, now).
		Find(&list).Error
	return list, err
}

func (r *cbtRepository) UpsertAnswer(ctx context.Context, answer *model.CBTAnswer) error {
	var existing model.CBTAnswer
	err := r.db.WithContext(ctx).Where("attempt_id = ? AND question_id = ?", answer.AttemptID, answer.QuestionID).First(&existing).Error
	now := time.Now()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if answer.ID == uuid.Nil {
				answer.ID = uuid.New()
			}
			answer.AnsweredAt = &now
			return r.db.WithContext(ctx).Create(answer).Error
		}
		return err
	}

	updates := map[string]interface{}{
		"selected_option_ids": answer.SelectedOptionIDs,
		"essay_text":          answer.EssayText,
		"is_flagged":          answer.IsFlagged,
		"answered_at":         now,
		"updated_at":          now,
	}
	return r.db.WithContext(ctx).Model(&model.CBTAnswer{}).Where("id = ?", existing.ID).Updates(updates).Error
}

func (r *cbtRepository) FindAnswersByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]model.CBTAnswer, error) {
	var answers []model.CBTAnswer
	err := r.db.WithContext(ctx).Preload("Question.Options").Where("attempt_id = ?", attemptID).Find(&answers).Error
	return answers, err
}

func (r *cbtRepository) FindUngradedEssaysByExamID(ctx context.Context, examID uuid.UUID) ([]model.CBTAnswer, error) {
	var answers []model.CBTAnswer
	err := r.db.WithContext(ctx).
		Joins("JOIN cbt_attempts ON cbt_attempts.id = cbt_answers.attempt_id").
		Joins("JOIN questions ON questions.id = cbt_answers.question_id").
		Preload("Question").
		Where("cbt_attempts.cbt_exam_id = ? AND questions.type = ? AND cbt_answers.graded_by IS NULL", examID, model.QuestionTypeEssay).
		Find(&answers).Error
	return answers, err
}

func (r *cbtRepository) GradeAnswer(ctx context.Context, answerID uuid.UUID, points float64, isCorrect bool, graderID uuid.UUID) error {
	updates := map[string]interface{}{
		"points_awarded": points,
		"is_correct":     isCorrect,
		"graded_by":      graderID,
		"updated_at":     time.Now(),
	}
	return r.db.WithContext(ctx).Model(&model.CBTAnswer{}).Where("id = ?", answerID).Updates(updates).Error
}

func (r *cbtRepository) CreateEvent(ctx context.Context, event *model.CBTEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(event).Error
}
