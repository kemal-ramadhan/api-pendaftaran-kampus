package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/repository"
)

var (
	ErrForbiddenAccess            = errors.New("Anda tidak memiliki akses ke sesi ujian ini")
	ErrExamNotStarted             = errors.New("EXAM_NOT_STARTED: sesi ujian belum dimulai atau jadwal belum aktif")
	ErrExamEnded                  = errors.New("EXAM_ENDED: sesi ujian telah berakhir")
	ErrExamAccessTokenInvalid    = errors.New("EXAM_ACCESS_TOKEN_INVALID: token akses ujian tidak valid")
	ErrAttemptAlreadySubmitted   = errors.New("ATTEMPT_ALREADY_SUBMITTED: ujian ini sudah dikumpulkan sebelumnya")
	ErrAttemptTimeUp             = errors.New("ATTEMPT_TIME_UP: waktu pengerjaan ujian telah habis")
	ErrQuestionNotInAttempt      = errors.New("QUESTION_NOT_IN_ATTEMPT: soal tidak termasuk dalam paket ujian Anda")
	ErrCBTNotConfigured          = errors.New("paket ujian online CBT belum dikonfigurasi untuk sesi ini")
	ErrNotRegisteredForSession   = errors.New("Anda tidak terdaftar sebagai peserta pada sesi ujian ini")
)

type PublicQuestionOption struct {
	ID             uuid.UUID `json:"id"`
	Label          string    `json:"label"`
	Content        string    `json:"content"`
	ImageObjectKey *string   `json:"image_object_key,omitempty"`
}

type PublicQuestionItem struct {
	ID                uuid.UUID              `json:"id"`
	Number            int                    `json:"number"`
	Type              model.QuestionType     `json:"type"`
	Content           string                 `json:"content"`
	ImageObjectKey    *string                `json:"image_object_key,omitempty"`
	Points            int                    `json:"points"`
	Options           []PublicQuestionOption `json:"options,omitempty"`
	SelectedOptionIDs []uuid.UUID            `json:"selected_option_ids,omitempty"`
	EssayText         *string                `json:"essay_text,omitempty"`
	IsFlagged         bool                   `json:"is_flagged"`
}

type AttemptQuestionsResponse struct {
	AttemptID        uuid.UUID            `json:"attempt_id"`
	ExamTitle        string               `json:"exam_title"`
	Status           string               `json:"status"`
	RemainingSeconds int64                `json:"remaining_seconds"`
	ServerTime       time.Time            `json:"server_time"`
	DeadlineAt       *time.Time           `json:"deadline_at"`
	Questions        []PublicQuestionItem `json:"questions"`
}

type CBTExamDetailResponse struct {
	ExamID                  uuid.UUID `json:"id"`
	SessionID               uuid.UUID `json:"session_id"`
	Title                   string    `json:"title"`
	Instructions            string    `json:"instructions"`
	DurationMinutes         int       `json:"duration_minutes"`
	QuestionCount           int       `json:"question_count"`
	RequiresAccessToken     bool      `json:"requires_access_token"`
	StartAt                 time.Time `json:"start_at"`
	EndAt                   time.Time `json:"end_at"`
	Status                  string    `json:"status"`
	CanStart                bool      `json:"can_start"`
	Reason                  string    `json:"reason"`
	AttemptStatus           string    `json:"attempt_status,omitempty"`
	AttemptID               *uuid.UUID `json:"attempt_id,omitempty"`
	ShowResultToParticipant bool      `json:"show_result_to_participant"`
}

type CBTService interface {
	// Pendaftar / Peserta
	GetMyCBTExams(ctx context.Context, userID uuid.UUID) ([]CBTExamDetailResponse, error)
	GetCBTExamDetail(ctx context.Context, examID, userID uuid.UUID) (*CBTExamDetailResponse, error)
	StartAttempt(ctx context.Context, examID, userID uuid.UUID, accessToken string) (*model.CBTAttempt, error)
	GetAttemptQuestions(ctx context.Context, attemptID, userID uuid.UUID) (*AttemptQuestionsResponse, error)
	SaveAnswer(ctx context.Context, attemptID, questionID, userID uuid.UUID, selectedOptions []uuid.UUID, essayText *string, isFlagged bool) error
	RecordEvent(ctx context.Context, attemptID uuid.UUID, eventType string, clientTime *time.Time, metadata string) error
	SubmitAttempt(ctx context.Context, attemptID, userID uuid.UUID) (*model.CBTAttempt, error)
	GetAttemptResult(ctx context.Context, attemptID, userID uuid.UUID) (map[string]interface{}, error)

	// Admin CBT
	CreateQuestionBank(ctx context.Context, bank *model.QuestionBank) (*model.QuestionBank, error)
	ListQuestionBanks(ctx context.Context) ([]model.QuestionBank, error)
	CreateQuestion(ctx context.Context, q *model.Question) (*model.Question, error)
	ListQuestions(ctx context.Context, bankID uuid.UUID) ([]model.Question, error)
	CreateCBTExam(ctx context.Context, exam *model.CBTExam, rawToken string) (*model.CBTExam, error)
	PublishCBTExam(ctx context.Context, examID uuid.UUID) error
	ResetAttempt(ctx context.Context, attemptID uuid.UUID, reason string) error
	ExtendAttempt(ctx context.Context, attemptID uuid.UUID, minutes int, reason string) error

	// Dosen Penilai Esai
	ListUngradedEssays(ctx context.Context, examID, examinerUserID uuid.UUID) ([]model.CBTAnswer, error)
	GradeEssay(ctx context.Context, answerID uuid.UUID, score float64, examinerUserID uuid.UUID) error
}

type cbtService struct {
	cbtRepo     repository.CBTRepository
	sessionRepo repository.SessionRepository
	partRepo    repository.ParticipantRepository
	scoringCli  internalclient.Client
}

func NewCBTService(
	cbtRepo repository.CBTRepository,
	sessionRepo repository.SessionRepository,
	partRepo repository.ParticipantRepository,
	scoringCli internalclient.Client,
) CBTService {
	return &cbtService{
		cbtRepo:     cbtRepo,
		sessionRepo: sessionRepo,
		partRepo:    partRepo,
		scoringCli:  scoringCli,
	}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *cbtService) GetMyCBTExams(ctx context.Context, userID uuid.UUID) ([]CBTExamDetailResponse, error) {
	participants, err := s.partRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var results []CBTExamDetailResponse
	now := time.Now()

	for _, p := range participants {
		exam, err := s.cbtRepo.FindCBTExamBySessionID(ctx, p.SessionID)
		if err != nil {
			continue // Sesi ini mungkin bukan CBT online
		}

		session, err := s.sessionRepo.FindByID(ctx, p.SessionID)
		if err != nil {
			continue
		}

		canStart := true
		reason := "Ujian siap dimulai"

		if now.Before(session.StartAt) {
			canStart = false
			reason = "Waktu ujian belum dimulai"
		} else if now.After(session.EndAt) {
			canStart = false
			reason = "Waktu sesi ujian telah berakhir"
		}

		attempt, err := s.cbtRepo.FindAttemptByExamAndParticipant(ctx, exam.ID, p.ID)
		attemptStatus := string(model.CBTAttemptStatusNotStarted)
		var attemptID *uuid.UUID
		if err == nil && attempt != nil {
			attemptStatus = string(attempt.Status)
			attemptID = &attempt.ID
			if attempt.Status == model.CBTAttemptStatusSubmitted || attempt.Status == model.CBTAttemptStatusAutoSubmitted {
				canStart = false
				reason = "Ujian sudah dikumpulkan"
			}
		}

		results = append(results, CBTExamDetailResponse{
			ExamID:                  exam.ID,
			SessionID:               session.ID,
			Title:                   exam.Title,
			Instructions:            exam.Instructions,
			DurationMinutes:         exam.DurationMinutes,
			QuestionCount:           exam.QuestionCount,
			RequiresAccessToken:     exam.RequiresAccessToken,
			StartAt:                 session.StartAt,
			EndAt:                   session.EndAt,
			Status:                  string(exam.Status),
			CanStart:                canStart,
			Reason:                  reason,
			AttemptStatus:           attemptStatus,
			AttemptID:               attemptID,
			ShowResultToParticipant: exam.ShowResultToParticipant,
		})
	}

	return results, nil
}

func (s *cbtService) GetCBTExamDetail(ctx context.Context, examID, userID uuid.UUID) (*CBTExamDetailResponse, error) {
	exam, err := s.cbtRepo.FindCBTExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}

	session, err := s.sessionRepo.FindByID(ctx, exam.SessionID)
	if err != nil {
		return nil, err
	}

	p, err := s.partRepo.FindBySessionAndUser(ctx, exam.SessionID, userID)
	if err != nil {
		return nil, ErrNotRegisteredForSession
	}

	now := time.Now()
	canStart := true
	reason := "Ujian siap dimulai"

	if now.Before(session.StartAt) {
		canStart = false
		reason = "Waktu ujian belum dimulai"
	} else if now.After(session.EndAt) {
		canStart = false
		reason = "Waktu sesi ujian telah berakhir"
	}

	attempt, err := s.cbtRepo.FindAttemptByExamAndParticipant(ctx, exam.ID, p.ID)
	attemptStatus := string(model.CBTAttemptStatusNotStarted)
	var attemptID *uuid.UUID
	if err == nil && attempt != nil {
		attemptStatus = string(attempt.Status)
		attemptID = &attempt.ID
		if attempt.Status == model.CBTAttemptStatusSubmitted || attempt.Status == model.CBTAttemptStatusAutoSubmitted {
			canStart = false
			reason = "Ujian sudah dikumpulkan"
		}
	}

	return &CBTExamDetailResponse{
		ExamID:                  exam.ID,
		SessionID:               session.ID,
		Title:                   exam.Title,
		Instructions:            exam.Instructions,
		DurationMinutes:         exam.DurationMinutes,
		QuestionCount:           exam.QuestionCount,
		RequiresAccessToken:     exam.RequiresAccessToken,
		StartAt:                 session.StartAt,
		EndAt:                   session.EndAt,
		Status:                  string(exam.Status),
		CanStart:                canStart,
		Reason:                  reason,
		AttemptStatus:           attemptStatus,
		AttemptID:               attemptID,
		ShowResultToParticipant: exam.ShowResultToParticipant,
	}, nil
}

func (s *cbtService) StartAttempt(ctx context.Context, examID, userID uuid.UUID, accessToken string) (*model.CBTAttempt, error) {
	exam, err := s.cbtRepo.FindCBTExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}

	session, err := s.sessionRepo.FindByID(ctx, exam.SessionID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if now.Before(session.StartAt) {
		return nil, ErrExamNotStarted
	}
	if now.After(session.EndAt) {
		return nil, ErrExamEnded
	}

	if exam.RequiresAccessToken && exam.AccessTokenHash != "" {
		if hashToken(accessToken) != exam.AccessTokenHash {
			return nil, ErrExamAccessTokenInvalid
		}
	}

	p, err := s.partRepo.FindBySessionAndUser(ctx, exam.SessionID, userID)
	if err != nil {
		return nil, ErrNotRegisteredForSession
	}

	existing, err := s.cbtRepo.FindAttemptByExamAndParticipant(ctx, exam.ID, p.ID)
	if err == nil && existing != nil {
		if existing.Status == model.CBTAttemptStatusSubmitted || existing.Status == model.CBTAttemptStatusAutoSubmitted {
			return nil, ErrAttemptAlreadySubmitted
		}
		return existing, nil
	}

	// Hitung batas waktu server
	duration := time.Duration(exam.DurationMinutes) * time.Minute
	deadline := now.Add(duration)
	if deadline.After(session.EndAt) {
		deadline = session.EndAt
	}

	// Kumpulkan soal berdasarkan rules
	var selectedQuestionIDs []uuid.UUID
	for _, rule := range exam.Rules {
		qs, err := s.cbtRepo.ListQuestionsByBankID(ctx, rule.BankID)
		if err != nil || len(qs) == 0 {
			continue
		}
		if exam.RandomizeQuestions {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			r.Shuffle(len(qs), func(i, j int) { qs[i], qs[j] = qs[j], qs[i] })
		}
		count := rule.Count
		if count > len(qs) {
			count = len(qs)
		}
		for i := 0; i < count; i++ {
			selectedQuestionIDs = append(selectedQuestionIDs, qs[i].ID)
		}
	}

	// Jika tidak ada rule spesifik, ambil dari bank pertama
	if len(selectedQuestionIDs) == 0 {
		banks, _ := s.cbtRepo.ListQuestionBanks(ctx)
		if len(banks) > 0 {
			qs, _ := s.cbtRepo.ListQuestionsByBankID(ctx, banks[0].ID)
			for i := 0; i < len(qs) && i < exam.QuestionCount; i++ {
				selectedQuestionIDs = append(selectedQuestionIDs, qs[i].ID)
			}
		}
	}

	orderBytes, _ := json.Marshal(selectedQuestionIDs)

	attempt := &model.CBTAttempt{
		ID:            uuid.New(),
		CBTExamID:     exam.ID,
		ParticipantID: p.ID,
		UserID:        userID,
		Status:        model.CBTAttemptStatusInProgress,
		StartedAt:     &now,
		DeadlineAt:    &deadline,
		QuestionOrder: string(orderBytes),
	}

	if err := s.cbtRepo.CreateAttempt(ctx, attempt); err != nil {
		return nil, err
	}

	_ = s.cbtRepo.CreateEvent(ctx, &model.CBTEvent{
		AttemptID: attempt.ID,
		Type:      "start",
		ServerTime: now,
	})

	return attempt, nil
}

func (s *cbtService) GetAttemptQuestions(ctx context.Context, attemptID, userID uuid.UUID) (*AttemptQuestionsResponse, error) {
	attempt, err := s.cbtRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	if attempt.UserID != userID {
		return nil, ErrForbiddenAccess
	}

	now := time.Now()
	var remaining int64 = 0
	if attempt.DeadlineAt != nil && attempt.DeadlineAt.After(now) {
		remaining = int64(attempt.DeadlineAt.Sub(now).Seconds())
	}

	// Baca urutan soal snapshot
	var qIDs []uuid.UUID
	_ = json.Unmarshal([]byte(attempt.QuestionOrder), &qIDs)

	questions, err := s.cbtRepo.GetQuestionsByIDs(ctx, qIDs)
	if err != nil {
		return nil, err
	}

	// Map jawaban tersimpan
	answers, _ := s.cbtRepo.FindAnswersByAttemptID(ctx, attempt.ID)
	answerMap := make(map[uuid.UUID]model.CBTAnswer)
	for _, a := range answers {
		answerMap[a.QuestionID] = a
	}

	qMap := make(map[uuid.UUID]model.Question)
	for _, q := range questions {
		qMap[q.ID] = q
	}

	var publicList []PublicQuestionItem
	for i, qid := range qIDs {
		q, ok := qMap[qid]
		if !ok {
			continue
		}

		var pubOpts []PublicQuestionOption
		for _, o := range q.Options {
			// SANGAT PENTING: Jangan bocorkan IsCorrect ke peserta!
			pubOpts = append(pubOpts, PublicQuestionOption{
				ID:             o.ID,
				Label:          o.Label,
				Content:        o.Content,
				ImageObjectKey: o.ImageObjectKey,
			})
		}

		var selOpts []uuid.UUID
		var essayText *string
		isFlagged := false

		if ans, found := answerMap[q.ID]; found {
			_ = json.Unmarshal([]byte(ans.SelectedOptionIDs), &selOpts)
			essayText = ans.EssayText
			isFlagged = ans.IsFlagged
		}

		publicList = append(publicList, PublicQuestionItem{
			ID:                q.ID,
			Number:            i + 1,
			Type:              q.Type,
			Content:           q.Content,
			ImageObjectKey:    q.ImageObjectKey,
			Points:            q.Points,
			Options:           pubOpts,
			SelectedOptionIDs: selOpts,
			EssayText:         essayText,
			IsFlagged:         isFlagged,
		})
	}

	examTitle := "Ujian CBT"
	if attempt.CBTExam != nil {
		examTitle = attempt.CBTExam.Title
	}

	return &AttemptQuestionsResponse{
		AttemptID:        attempt.ID,
		ExamTitle:        examTitle,
		Status:           string(attempt.Status),
		RemainingSeconds: remaining,
		ServerTime:       now,
		DeadlineAt:       attempt.DeadlineAt,
		Questions:        publicList,
	}, nil
}

func (s *cbtService) SaveAnswer(ctx context.Context, attemptID, questionID, userID uuid.UUID, selectedOptions []uuid.UUID, essayText *string, isFlagged bool) error {
	attempt, err := s.cbtRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return err
	}

	if attempt.UserID != userID {
		return ErrForbiddenAccess
	}

	if attempt.Status != model.CBTAttemptStatusInProgress {
		return ErrAttemptAlreadySubmitted
	}

	now := time.Now()
	// Server timer enforcement
	if attempt.DeadlineAt != nil && now.After(*attempt.DeadlineAt) {
		return ErrAttemptTimeUp
	}

	optJSON, _ := json.Marshal(selectedOptions)
	answer := &model.CBTAnswer{
		AttemptID:         attempt.ID,
		QuestionID:        questionID,
		SelectedOptionIDs: string(optJSON),
		EssayText:         essayText,
		IsFlagged:         isFlagged,
	}

	return s.cbtRepo.UpsertAnswer(ctx, answer)
}

func (s *cbtService) RecordEvent(ctx context.Context, attemptID uuid.UUID, eventType string, clientTime *time.Time, metadata string) error {
	event := &model.CBTEvent{
		AttemptID:  attemptID,
		Type:       eventType,
		ClientTime: clientTime,
		ServerTime: time.Now(),
		Metadata:   metadata,
	}
	return s.cbtRepo.CreateEvent(ctx, event)
}

func (s *cbtService) SubmitAttempt(ctx context.Context, attemptID, userID uuid.UUID) (*model.CBTAttempt, error) {
	attempt, err := s.cbtRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	if attempt.UserID != userID {
		return nil, ErrForbiddenAccess
	}

	if attempt.Status == model.CBTAttemptStatusSubmitted || attempt.Status == model.CBTAttemptStatusAutoSubmitted {
		return attempt, nil
	}

	now := time.Now()
	attempt.Status = model.CBTAttemptStatusSubmitted
	attempt.SubmittedAt = &now

	// Hitung nilai objektif
	answers, _ := s.cbtRepo.FindAnswersByAttemptID(ctx, attempt.ID)
	var objScore float64 = 0
	hasEssay := false

	for _, a := range answers {
		q, qErr := s.cbtRepo.FindQuestionByID(ctx, a.QuestionID)
		if qErr != nil {
			continue
		}

		if q.Type == model.QuestionTypeEssay {
			hasEssay = true
			continue
		}

		var chosenUUIDs []uuid.UUID
		_ = json.Unmarshal([]byte(a.SelectedOptionIDs), &chosenUUIDs)

		correctUUIDs := make(map[uuid.UUID]bool)
		for _, opt := range q.Options {
			if opt.IsCorrect {
				correctUUIDs[opt.ID] = true
			}
		}

		isCorrect := false
		if len(chosenUUIDs) == len(correctUUIDs) && len(chosenUUIDs) > 0 {
			allMatch := true
			for _, cid := range chosenUUIDs {
				if !correctUUIDs[cid] {
					allMatch = false
					break
				}
			}
			isCorrect = allMatch
		}

		points := 0.0
		if isCorrect {
			points = float64(q.Points)
			objScore += points
		}

		_ = s.cbtRepo.GradeAnswer(ctx, a.ID, points, isCorrect, uuid.Nil)
	}

	attempt.ObjectiveScore = objScore
	if !hasEssay {
		attempt.FinalScore = objScore
		attempt.GradedAt = &now

		// Kirim nilai otomatis ke Scoring Service jika tidak ada soal esai
		if s.scoringCli != nil && attempt.Participant != nil {
			go func() {
				scoringPayload := map[string]interface{}{
					"session_id":     attempt.Participant.SessionID.String(),
					"application_id": attempt.Participant.ApplicationID.String(),
					"examiner_id":    uuid.Nil.String(),
					"component_name": "written_test",
					"score":          attempt.FinalScore,
					"notes":          "Penilaian CBT Otomatis",
				}
				var scoringResp interface{}
				_ = s.scoringCli.Post(context.Background(), "/internal/scores", scoringPayload, &scoringResp)
			}()
		}
	}

	_ = s.cbtRepo.UpdateAttempt(ctx, attempt)
	_ = s.cbtRepo.CreateEvent(ctx, &model.CBTEvent{
		AttemptID:  attempt.ID,
		Type:       "submit",
		ServerTime: now,
	})

	return attempt, nil
}

func (s *cbtService) GetAttemptResult(ctx context.Context, attemptID, userID uuid.UUID) (map[string]interface{}, error) {
	attempt, err := s.cbtRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	if attempt.UserID != userID {
		return nil, ErrForbiddenAccess
	}

	return map[string]interface{}{
		"attempt_id":      attempt.ID,
		"status":          attempt.Status,
		"objective_score": attempt.ObjectiveScore,
		"essay_score":     attempt.EssayScore,
		"final_score":     attempt.FinalScore,
		"submitted_at":    attempt.SubmittedAt,
	}, nil
}

// Admin Methods
func (s *cbtService) CreateQuestionBank(ctx context.Context, bank *model.QuestionBank) (*model.QuestionBank, error) {
	err := s.cbtRepo.CreateQuestionBank(ctx, bank)
	return bank, err
}

func (s *cbtService) ListQuestionBanks(ctx context.Context) ([]model.QuestionBank, error) {
	return s.cbtRepo.ListQuestionBanks(ctx)
}

func (s *cbtService) CreateQuestion(ctx context.Context, q *model.Question) (*model.Question, error) {
	err := s.cbtRepo.CreateQuestion(ctx, q)
	return q, err
}

func (s *cbtService) ListQuestions(ctx context.Context, bankID uuid.UUID) ([]model.Question, error) {
	return s.cbtRepo.ListQuestionsByBankID(ctx, bankID)
}

func (s *cbtService) CreateCBTExam(ctx context.Context, exam *model.CBTExam, rawToken string) (*model.CBTExam, error) {
	if rawToken != "" {
		exam.AccessTokenHash = hashToken(rawToken)
		exam.RequiresAccessToken = true
	}
	err := s.cbtRepo.CreateCBTExam(ctx, exam)
	return exam, err
}

func (s *cbtService) PublishCBTExam(ctx context.Context, examID uuid.UUID) error {
	exam, err := s.cbtRepo.FindCBTExamByID(ctx, examID)
	if err != nil {
		return err
	}
	exam.Status = model.CBTExamStatusPublished
	return s.cbtRepo.UpdateCBTExam(ctx, exam)
}

func (s *cbtService) ResetAttempt(ctx context.Context, attemptID uuid.UUID, reason string) error {
	attempt, err := s.cbtRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return err
	}
	now := time.Now()
	attempt.Status = model.CBTAttemptStatusInProgress
	attempt.SubmittedAt = nil
	_ = s.cbtRepo.CreateEvent(ctx, &model.CBTEvent{
		AttemptID:  attempt.ID,
		Type:       "reset",
		ServerTime: now,
		Metadata:   fmt.Sprintf(`{"reason":"%s"}`, reason),
	})
	return s.cbtRepo.UpdateAttempt(ctx, attempt)
}

func (s *cbtService) ExtendAttempt(ctx context.Context, attemptID uuid.UUID, minutes int, reason string) error {
	attempt, err := s.cbtRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return err
	}
	if attempt.DeadlineAt != nil {
		newDeadline := attempt.DeadlineAt.Add(time.Duration(minutes) * time.Minute)
		attempt.DeadlineAt = &newDeadline
	}
	_ = s.cbtRepo.CreateEvent(ctx, &model.CBTEvent{
		AttemptID:  attempt.ID,
		Type:       "extend",
		ServerTime: time.Now(),
		Metadata:   fmt.Sprintf(`{"minutes":%d,"reason":"%s"}`, minutes, reason),
	})
	return s.cbtRepo.UpdateAttempt(ctx, attempt)
}

func (s *cbtService) ListUngradedEssays(ctx context.Context, examID, examinerUserID uuid.UUID) ([]model.CBTAnswer, error) {
	return s.cbtRepo.FindUngradedEssaysByExamID(ctx, examID)
}

func (s *cbtService) GradeEssay(ctx context.Context, answerID uuid.UUID, score float64, examinerUserID uuid.UUID) error {
	return s.cbtRepo.GradeAnswer(ctx, answerID, score, score > 0, examinerUserID)
}
