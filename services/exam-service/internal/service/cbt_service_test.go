package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type mockCBTRepo struct {
	banks     map[uuid.UUID]*model.QuestionBank
	questions map[uuid.UUID]*model.Question
	exams     map[uuid.UUID]*model.CBTExam
	attempts  map[uuid.UUID]*model.CBTAttempt
	answers   map[string]*model.CBTAnswer
}

func newMockCBTRepo() *mockCBTRepo {
	return &mockCBTRepo{
		banks:     make(map[uuid.UUID]*model.QuestionBank),
		questions: make(map[uuid.UUID]*model.Question),
		exams:     make(map[uuid.UUID]*model.CBTExam),
		attempts:  make(map[uuid.UUID]*model.CBTAttempt),
		answers:   make(map[string]*model.CBTAnswer),
	}
}

func (m *mockCBTRepo) DB() *gorm.DB { return nil }

func (m *mockCBTRepo) CreateQuestionBank(ctx context.Context, bank *model.QuestionBank) error {
	if bank.ID == uuid.Nil {
		bank.ID = uuid.New()
	}
	m.banks[bank.ID] = bank
	return nil
}

func (m *mockCBTRepo) FindQuestionBankByID(ctx context.Context, id uuid.UUID) (*model.QuestionBank, error) {
	if b, ok := m.banks[id]; ok {
		return b, nil
	}
	return nil, repository.ErrQuestionBankNotFound
}

func (m *mockCBTRepo) ListQuestionBanks(ctx context.Context) ([]model.QuestionBank, error) {
	var list []model.QuestionBank
	for _, b := range m.banks {
		list = append(list, *b)
	}
	return list, nil
}

func (m *mockCBTRepo) UpdateQuestionBank(ctx context.Context, bank *model.QuestionBank) error {
	m.banks[bank.ID] = bank
	return nil
}

func (m *mockCBTRepo) DeleteQuestionBank(ctx context.Context, id uuid.UUID) error {
	delete(m.banks, id)
	return nil
}

func (m *mockCBTRepo) CreateQuestion(ctx context.Context, q *model.Question) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	m.questions[q.ID] = q
	return nil
}

func (m *mockCBTRepo) UpdateQuestion(ctx context.Context, q *model.Question) error {
	m.questions[q.ID] = q
	return nil
}

func (m *mockCBTRepo) DeleteQuestion(ctx context.Context, id uuid.UUID) error {
	delete(m.questions, id)
	return nil
}

func (m *mockCBTRepo) FindQuestionByID(ctx context.Context, id uuid.UUID) (*model.Question, error) {
	if q, ok := m.questions[id]; ok {
		return q, nil
	}
	return nil, repository.ErrQuestionNotFound
}

func (m *mockCBTRepo) ListQuestionsByBankID(ctx context.Context, bankID uuid.UUID) ([]model.Question, error) {
	var list []model.Question
	for _, q := range m.questions {
		if q.BankID == bankID {
			list = append(list, *q)
		}
	}
	return list, nil
}

func (m *mockCBTRepo) GetQuestionsByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Question, error) {
	var list []model.Question
	for _, id := range ids {
		if q, ok := m.questions[id]; ok {
			list = append(list, *q)
		}
	}
	return list, nil
}

func (m *mockCBTRepo) CreateCBTExam(ctx context.Context, exam *model.CBTExam) error {
	if exam.ID == uuid.Nil {
		exam.ID = uuid.New()
	}
	m.exams[exam.ID] = exam
	return nil
}

func (m *mockCBTRepo) UpdateCBTExam(ctx context.Context, exam *model.CBTExam) error {
	m.exams[exam.ID] = exam
	return nil
}

func (m *mockCBTRepo) FindCBTExamByID(ctx context.Context, id uuid.UUID) (*model.CBTExam, error) {
	if e, ok := m.exams[id]; ok {
		return e, nil
	}
	return nil, repository.ErrCBTExamNotFound
}

func (m *mockCBTRepo) FindCBTExamBySessionID(ctx context.Context, sessionID uuid.UUID) (*model.CBTExam, error) {
	for _, e := range m.exams {
		if e.SessionID == sessionID {
			return e, nil
		}
	}
	return nil, repository.ErrCBTExamNotFound
}

func (m *mockCBTRepo) ListCBTExams(ctx context.Context) ([]model.CBTExam, error) {
	var list []model.CBTExam
	for _, e := range m.exams {
		list = append(list, *e)
	}
	return list, nil
}

func (m *mockCBTRepo) CreateAttempt(ctx context.Context, attempt *model.CBTAttempt) error {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *mockCBTRepo) UpdateAttempt(ctx context.Context, attempt *model.CBTAttempt) error {
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *mockCBTRepo) FindAttemptByID(ctx context.Context, id uuid.UUID) (*model.CBTAttempt, error) {
	if a, ok := m.attempts[id]; ok {
		return a, nil
	}
	return nil, repository.ErrCBTAttemptNotFound
}

func (m *mockCBTRepo) FindAttemptByExamAndParticipant(ctx context.Context, examID, participantID uuid.UUID) (*model.CBTAttempt, error) {
	for _, a := range m.attempts {
		if a.CBTExamID == examID && a.ParticipantID == participantID {
			return a, nil
		}
	}
	return nil, repository.ErrCBTAttemptNotFound
}

func (m *mockCBTRepo) ListAttemptsByExamID(ctx context.Context, examID uuid.UUID) ([]model.CBTAttempt, error) {
	var list []model.CBTAttempt
	for _, a := range m.attempts {
		if a.CBTExamID == examID {
			list = append(list, *a)
		}
	}
	return list, nil
}

func (m *mockCBTRepo) FindExpiredInProgressAttempts(ctx context.Context, now time.Time) ([]model.CBTAttempt, error) {
	return nil, nil
}

func (m *mockCBTRepo) UpsertAnswer(ctx context.Context, answer *model.CBTAnswer) error {
	key := answer.AttemptID.String() + ":" + answer.QuestionID.String()
	m.answers[key] = answer
	return nil
}

func (m *mockCBTRepo) FindAnswersByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]model.CBTAnswer, error) {
	var list []model.CBTAnswer
	for _, a := range m.answers {
		if a.AttemptID == attemptID {
			list = append(list, *a)
		}
	}
	return list, nil
}

func (m *mockCBTRepo) FindUngradedEssaysByExamID(ctx context.Context, examID uuid.UUID) ([]model.CBTAnswer, error) {
	return nil, nil
}

func (m *mockCBTRepo) GradeAnswer(ctx context.Context, answerID uuid.UUID, points float64, isCorrect bool, graderID uuid.UUID) error {
	for _, a := range m.answers {
		if a.ID == answerID {
			a.PointsAwarded = points
			a.IsCorrect = &isCorrect
			a.GradedBy = &graderID
			return nil
		}
	}
	return nil
}

func (m *mockCBTRepo) CreateEvent(ctx context.Context, event *model.CBTEvent) error {
	return nil
}

func TestCBTService_TimerEnforcementAndAntiLeak(t *testing.T) {
	cbtRepo := newMockCBTRepo()
	sessRepo := &mockSessionRepo{
		sessions:  make(map[uuid.UUID]*model.ExamSession),
		examiners: make(map[uuid.UUID][]uuid.UUID),
	}
	partRepo := newMockParticipantRepo()

	svc := NewCBTService(cbtRepo, sessRepo, partRepo, nil)
	ctx := context.Background()

	// 1. Setup Session dan Exam
	sessID := uuid.New()
	userID := uuid.New()
	partID := uuid.New()

	now := time.Now()
	sess := &model.ExamSession{
		ID:      sessID,
		Title:   "Ujian Masuk CBT Gelombang 1",
		StartAt: now.Add(-10 * time.Minute),
		EndAt:   now.Add(2 * time.Hour),
		Type:    model.SessionTypeWritten,
	}
	sessRepo.sessions[sessID] = sess

	part := &model.Participant{
		ID:                partID,
		SessionID:         sessID,
		UserID:            userID,
		ParticipantNumber: "2026-TEST-001",
	}
	partRepo.participants[partID] = part

	bankID := uuid.New()
	cbtRepo.banks[bankID] = &model.QuestionBank{
		ID:      bankID,
		Name:    "Matematika Dasar",
		Subject: "Matematika",
	}

	opt1ID := uuid.New()
	opt2ID := uuid.New()
	qID := uuid.New()

	q := &model.Question{
		ID:      qID,
		BankID:  bankID,
		Type:    model.QuestionTypeSingleChoice,
		Content: "Berapa 10 + 15?",
		Points:  10,
		Options: []model.QuestionOption{
			{ID: opt1ID, Label: "A", Content: "25", IsCorrect: true},
			{ID: opt2ID, Label: "B", Content: "30", IsCorrect: false},
		},
	}
	cbtRepo.questions[qID] = q

	examID := uuid.New()
	exam := &model.CBTExam{
		ID:                  examID,
		SessionID:           sessID,
		Title:               "Ujian Matematika CBT",
		DurationMinutes:     60,
		QuestionCount:       1,
		Status:              model.CBTExamStatusPublished,
		RequiresAccessToken: false,
		Rules: []model.CBTExamRule{
			{CBTExamID: examID, BankID: bankID, Count: 1},
		},
	}
	cbtRepo.exams[examID] = exam

	// 2. Mulai Attempt
	att, err := svc.StartAttempt(ctx, examID, userID, "")
	assert.NoError(t, err)
	assert.NotNil(t, att)

	// 3. Uji Anti-Leak: Kunci jawaban tidak pernah dikirim ke peserta
	qResp, err := svc.GetAttemptQuestions(ctx, att.ID, userID)
	assert.NoError(t, err)
	assert.Len(t, qResp.Questions, 1)

	// Serialisasi ke JSON untuk memverifikasi tidak ada field is_correct yang bocor
	jsonBytes, err := json.Marshal(qResp)
	assert.NoError(t, err)
	assert.NotContains(t, string(jsonBytes), "is_correct", "Kunci jawaban is_correct tidak boleh bocor ke respons peserta")

	// 4. Uji Autosave Jawaban
	err = svc.SaveAnswer(ctx, att.ID, qID, userID, []uuid.UUID{opt1ID}, nil, false)
	assert.NoError(t, err)

	// 5. Uji Server Timer Enforcement: Jika waktu habis, autosave ditolak
	expiredDeadline := now.Add(-5 * time.Minute)
	att.DeadlineAt = &expiredDeadline
	err = svc.SaveAnswer(ctx, att.ID, qID, userID, []uuid.UUID{opt2ID}, nil, false)
	assert.ErrorIs(t, err, ErrAttemptTimeUp, "Autosave setelah batas waktu harus ditolak")

	// 6. Uji Submit Ujian & Penilaian Otomatis
	att.DeadlineAt = nil // Reset deadline untuk submit
	submittedAtt, err := svc.SubmitAttempt(ctx, att.ID, userID)
	assert.NoError(t, err)
	assert.Equal(t, model.CBTAttemptStatusSubmitted, submittedAtt.Status)
	assert.Equal(t, float64(10), submittedAtt.ObjectiveScore, "Skor objektif harus 10 untuk jawaban benar")
}
