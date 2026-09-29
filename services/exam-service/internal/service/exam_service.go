package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/pdf"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/repository"
)

var (
	ErrSessionNotFound         = errors.New("sesi ujian tidak ditemukan")
	ErrParticipantNotFound     = errors.New("peserta ujian tidak ditemukan")
	ErrSessionCapacityExceeded = errors.New("SESSION_CAPACITY_EXCEEDED: kapasitas sesi ujian telah penuh")
	ErrApplicationNotVerified  = errors.New("APPLICATION_NOT_VERIFIED: hanya pendaftar berstatus verified yang dapat dijadwalkan ujian")
	ErrDuplicateParticipant    = errors.New("DUPLICATE_PARTICIPANT: pendaftar sudah terdaftar pada sesi ujian ini")
	ErrForbiddenExaminer       = errors.New("FORBIDDEN: Anda bukan penguji yang ditugaskan pada sesi ujian ini")
	ErrInvalidAttendance       = errors.New("status kehadiran tidak valid (harus 'present' atau 'absent')")
	ErrInvalidSessionTime      = errors.New("waktu selesai ujian harus lebih besar dari waktu mulai")
	ErrCardNotFound            = errors.New("kartu ujian belum diterbitkan atau tidak ditemukan")
)

type ExamService interface {
	CreateSession(ctx context.Context, session *model.ExamSession) (*model.ExamSession, error)
	UpdateSession(ctx context.Context, session *model.ExamSession) (*model.ExamSession, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
	GetSessionDetail(ctx context.Context, id uuid.UUID) (*model.ExamSession, error)
	ListSessions(ctx context.Context, periodID, pathID *uuid.UUID, sessionType *string, isActive *bool) ([]model.ExamSession, error)
	AssignExaminer(ctx context.Context, sessionID, examinerUserID uuid.UUID) (*model.SessionExaminer, error)
	RemoveExaminer(ctx context.Context, sessionID, examinerUserID uuid.UUID) error
	GetExaminerSessions(ctx context.Context, examinerUserID uuid.UUID) ([]model.ExamSession, error)
	AssignParticipantManual(ctx context.Context, sessionID, appID uuid.UUID) (*model.Participant, error)
	AutoAssignVerifiedParticipants(ctx context.Context, sessionID uuid.UUID, appIDs []uuid.UUID) ([]model.Participant, error)
	GetSessionParticipants(ctx context.Context, sessionID uuid.UUID, examinerUserID *uuid.UUID, role string) ([]model.Participant, error)
	UpdateAttendance(ctx context.Context, participantID uuid.UUID, attendance string, actorUserID uuid.UUID, actorRole string) error
	GetMyExamSchedule(ctx context.Context, userID uuid.UUID) ([]model.Participant, error)
	GetMyExamCardPDF(ctx context.Context, userID uuid.UUID) ([]byte, string, error)
	GetParticipantByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Participant, error)
	RemoveParticipantByApplicationID(ctx context.Context, appID uuid.UUID) error
}

type examService struct {
	cfg             *config.Config
	sessionRepo     repository.SessionRepository
	participantRepo repository.ParticipantRepository
	storageCli      storage.Client
	admissionCli    internalclient.Client
	identityCli     internalclient.Client
	notificationCli internalclient.Client
}

func NewExamService(
	cfg *config.Config,
	sessionRepo repository.SessionRepository,
	participantRepo repository.ParticipantRepository,
	storageCli storage.Client,
	admissionCli internalclient.Client,
	identityCli internalclient.Client,
	notificationCli internalclient.Client,
) ExamService {
	return &examService{
		cfg:             cfg,
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		storageCli:      storageCli,
		admissionCli:    admissionCli,
		identityCli:     identityCli,
		notificationCli: notificationCli,
	}
}

func (s *examService) CreateSession(ctx context.Context, session *model.ExamSession) (*model.ExamSession, error) {
	if session.EndAt.Before(session.StartAt) || session.EndAt.Equal(session.StartAt) {
		return nil, ErrInvalidSessionTime
	}
	if session.Capacity <= 0 {
		session.Capacity = 30
	}
	if session.Type == "" {
		session.Type = model.SessionTypeWritten
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *examService) UpdateSession(ctx context.Context, session *model.ExamSession) (*model.ExamSession, error) {
	existing, err := s.sessionRepo.FindByID(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrSessionNotFound
	}

	if !session.EndAt.IsZero() && !session.StartAt.IsZero() {
		if session.EndAt.Before(session.StartAt) || session.EndAt.Equal(session.StartAt) {
			return nil, ErrInvalidSessionTime
		}
		existing.StartAt = session.StartAt
		existing.EndAt = session.EndAt
	}

	if session.Title != "" {
		existing.Title = session.Title
	}
	if session.Type != "" {
		existing.Type = session.Type
	}
	if session.Location != nil {
		existing.Location = session.Location
	}
	if session.MeetingLink != nil {
		existing.MeetingLink = session.MeetingLink
	}
	if session.Capacity > 0 {
		existing.Capacity = session.Capacity
	}
	existing.IsActive = session.IsActive

	if err := s.sessionRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *examService) DeleteSession(ctx context.Context, id uuid.UUID) error {
	existing, err := s.sessionRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSessionNotFound
	}

	count, _ := s.participantRepo.CountBySessionID(ctx, id)
	if count > 0 {
		return errors.New("tidak dapat menghapus sesi ujian yang sudah memiliki peserta terdaftar")
	}

	return s.sessionRepo.Delete(ctx, id)
}

func (s *examService) GetSessionDetail(ctx context.Context, id uuid.UUID) (*model.ExamSession, error) {
	session, err := s.sessionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (s *examService) ListSessions(ctx context.Context, periodID, pathID *uuid.UUID, sessionType *string, isActive *bool) ([]model.ExamSession, error) {
	return s.sessionRepo.List(ctx, periodID, pathID, sessionType, isActive)
}

func (s *examService) AssignExaminer(ctx context.Context, sessionID, examinerUserID uuid.UUID) (*model.SessionExaminer, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	examiner := &model.SessionExaminer{
		SessionID:      sessionID,
		ExaminerUserID: examinerUserID,
	}

	if err := s.sessionRepo.AddExaminer(ctx, examiner); err != nil {
		return nil, err
	}
	return examiner, nil
}

func (s *examService) RemoveExaminer(ctx context.Context, sessionID, examinerUserID uuid.UUID) error {
	return s.sessionRepo.RemoveExaminer(ctx, sessionID, examinerUserID)
}

func (s *examService) GetExaminerSessions(ctx context.Context, examinerUserID uuid.UUID) ([]model.ExamSession, error) {
	return s.sessionRepo.GetSessionsByExaminerID(ctx, examinerUserID)
}

type internalAppResponse struct {
	Data struct {
		ID                string `json:"id"`
		UserID            string `json:"user_id"`
		ApplicationNumber string `json:"application_number"`
		Status            string `json:"status"`
		Period            struct {
			Name string `json:"name"`
		} `json:"period"`
		AdmissionPath struct {
			Name            string `json:"name"`
			SelectionMethod string `json:"selection_method"`
		} `json:"admission_path"`
		Choice1 struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"choice_1"`
	} `json:"data"`
}

type internalPersonResponse struct {
	Data struct {
		FullName string `json:"full_name"`
	} `json:"data"`
}

func (s *examService) AssignParticipantManual(ctx context.Context, sessionID, appID uuid.UUID) (*model.Participant, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	// 1. Validasi Kapasitas Sesi
	currentCount, err := s.participantRepo.CountBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if currentCount >= int64(session.Capacity) {
		return nil, ErrSessionCapacityExceeded
	}

	// 2. Validasi Duplikasi Sesi
	existing, _ := s.participantRepo.FindByApplicationID(ctx, appID)
	if existing != nil && existing.SessionID == sessionID {
		return nil, ErrDuplicateParticipant
	}

	// 3. Ambil data aplikasi & validasi status verified ke Admission Service
	var appResp internalAppResponse
	if s.admissionCli != nil {
		if err := s.admissionCli.Get(ctx, "/internal/applications/"+appID.String(), nil, &appResp); err != nil {
			return nil, fmt.Errorf("gagal memverifikasi pendaftaran ke admission service: %w", err)
		}
	} else {
		// Mock data untuk pengujian lokal unit test
		appResp.Data.ID = appID.String()
		appResp.Data.UserID = uuid.New().String()
		appResp.Data.ApplicationNumber = "PMB-2026-TEST"
		appResp.Data.Status = "verified"
		appResp.Data.Period.Name = "Gelombang 1"
		appResp.Data.AdmissionPath.Name = "Jalur Reguler"
		appResp.Data.Choice1.Name = "Teknik Informatika"
	}

	if appResp.Data.Status != "verified" {
		return nil, ErrApplicationNotVerified
	}

	parsedUserID, err := uuid.Parse(appResp.Data.UserID)
	if err != nil {
		parsedUserID = uuid.New()
	}

	// 4. Ambil nama pendaftar dari Identity Service
	fullName := "Calon Mahasiswa"
	if s.identityCli != nil {
		var personResp internalPersonResponse
		if err := s.identityCli.Get(ctx, "/internal/persons/"+parsedUserID.String(), nil, &personResp); err == nil && personResp.Data.FullName != "" {
			fullName = personResp.Data.FullName
		}
	}

	// 5. Generate nomor peserta ujian resmi
	year := session.StartAt.Year()
	if year == 0 {
		year = time.Now().Year()
	}
	seq, err := s.participantRepo.GetNextParticipantSequence(ctx, year)
	if err != nil {
		return nil, err
	}
	participantNumber := fmt.Sprintf("EXAM-%d-%05d", year, seq)

	// 6. Buat record peserta
	participant := &model.Participant{
		SessionID:         sessionID,
		ApplicationID:     appID,
		UserID:            parsedUserID,
		ParticipantNumber: participantNumber,
	}

	if err := s.participantRepo.AddParticipant(ctx, participant); err != nil {
		return nil, err
	}

	// 7. Terbitkan Kartu Ujian Resmi PDF & Simpan ke MinIO
	locStr := ""
	if session.Location != nil {
		locStr = *session.Location
	}
	meetStr := ""
	if session.MeetingLink != nil {
		meetStr = *session.MeetingLink
	}

	cardData := pdf.ExamCardData{
		ParticipantNumber: participantNumber,
		FullName:          fullName,
		ApplicationNumber: appResp.Data.ApplicationNumber,
		PeriodName:        appResp.Data.Period.Name,
		PathName:          appResp.Data.AdmissionPath.Name,
		StudyProgramName:  appResp.Data.Choice1.Name,
		SessionTitle:      session.Title,
		SessionType:       string(session.Type),
		StartAt:           session.StartAt,
		EndAt:             session.EndAt,
		Location:          locStr,
		MeetingLink:       meetStr,
	}

	pdfBytes, err := pdf.GenerateExamCardPDF(cardData)
	if err == nil {
		objectKey := fmt.Sprintf("cards/%s_%s.pdf", appID.String(), participantNumber)
		if s.storageCli != nil {
			_ = s.storageCli.Upload(ctx, s.cfg.ExamCardBucket, objectKey, bytes.NewReader(pdfBytes), int64(len(pdfBytes)), "application/pdf")
		}

		card := &model.ExamCard{
			ParticipantID: participant.ID,
			PdfObjectKey:  objectKey,
			GeneratedAt:   time.Now(),
		}
		_ = s.participantRepo.SaveExamCard(ctx, card)
		participant.ExamCard = card
	}

	// 8. Beri tahu Admission Service: Ubah status menjadi exam_scheduled
	if s.admissionCli != nil {
		statusPayload := map[string]interface{}{
			"status": "exam_scheduled",
			"notes":  "Jadwal ujian berhasil diterbitkan: " + participantNumber,
		}
		_ = s.admissionCli.Post(ctx, "/internal/applications/"+appID.String()+"/status", statusPayload, nil)
	}

	// 9. Beri tahu Notification Service
	if s.notificationCli != nil {
		notifPayload := map[string]interface{}{
			"user_id":  parsedUserID,
			"channel":  "email",
			"template": "exam_card_ready",
			"data": map[string]interface{}{
				"participant_number": participantNumber,
				"session_title":      session.Title,
				"start_at":           session.StartAt.Format(time.RFC3339),
			},
		}
		_ = s.notificationCli.Post(ctx, "/internal/notifications", notifPayload, nil)
	}

	participant.FullName = fullName
	participant.StudyProgramName = appResp.Data.Choice1.Name
	participant.PathName = appResp.Data.AdmissionPath.Name
	participant.PeriodName = appResp.Data.Period.Name

	return participant, nil
}

func (s *examService) AutoAssignVerifiedParticipants(ctx context.Context, sessionID uuid.UUID, appIDs []uuid.UUID) ([]model.Participant, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	currentCount, err := s.participantRepo.CountBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	remainingCapacity := session.Capacity - int(currentCount)
	if remainingCapacity <= 0 {
		return nil, ErrSessionCapacityExceeded
	}

	var assigned []model.Participant
	for _, appID := range appIDs {
		if len(assigned) >= remainingCapacity {
			break
		}

		p, err := s.AssignParticipantManual(ctx, sessionID, appID)
		if err != nil {
			continue // Lanjut ke kandidat berikutnya jika ada yang invalid
		}
		assigned = append(assigned, *p)
	}

	return assigned, nil
}

func (s *examService) GetSessionParticipants(ctx context.Context, sessionID uuid.UUID, examinerUserID *uuid.UUID, role string) ([]model.Participant, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	// Isolasi akses role dosen
	if role == "dosen" && examinerUserID != nil {
		assigned, err := s.sessionRepo.IsExaminerAssigned(ctx, sessionID, *examinerUserID)
		if err != nil {
			return nil, err
		}
		if !assigned {
			return nil, ErrForbiddenExaminer
		}
	}

	return s.participantRepo.FindBySessionID(ctx, sessionID)
}

func (s *examService) UpdateAttendance(ctx context.Context, participantID uuid.UUID, attendance string, actorUserID uuid.UUID, actorRole string) error {
	if attendance != model.AttendancePresent && attendance != model.AttendanceAbsent {
		return ErrInvalidAttendance
	}

	participant, err := s.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrParticipantNotFound
	}

	// Dosen hanya bisa mengubah presensi pada sesinya sendiri
	if actorRole == "dosen" {
		assigned, err := s.sessionRepo.IsExaminerAssigned(ctx, participant.SessionID, actorUserID)
		if err != nil {
			return err
		}
		if !assigned {
			return ErrForbiddenExaminer
		}
	}

	return s.participantRepo.UpdateAttendance(ctx, participantID, attendance)
}

func (s *examService) GetMyExamSchedule(ctx context.Context, userID uuid.UUID) ([]model.Participant, error) {
	return s.participantRepo.FindByUserID(ctx, userID)
}

func (s *examService) GetMyExamCardPDF(ctx context.Context, userID uuid.UUID) ([]byte, string, error) {
	participants, err := s.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if len(participants) == 0 {
		return nil, "", ErrParticipantNotFound
	}

	// Ambil kartu ujian dari pendaftaran pertama/terakhir
	p := participants[0]
	if p.ExamCard == nil {
		card, err := s.participantRepo.FindCardByParticipantID(ctx, p.ID)
		if err != nil || card == nil {
			return nil, "", ErrCardNotFound
		}
		p.ExamCard = card
	}

	filename := fmt.Sprintf("kartu-ujian-%s.pdf", p.ParticipantNumber)

	if s.storageCli != nil {
		reader, err := s.storageCli.Download(ctx, s.cfg.ExamCardBucket, p.ExamCard.PdfObjectKey)
		if err == nil {
			defer reader.Close()
			buf, err := io.ReadAll(reader)
			if err == nil && len(buf) > 0 {
				return buf, filename, nil
			}
		}
	}

	// Fallback jika belum tersimpan di MinIO / mode pengujian: generate on the fly
	locStr := ""
	if p.Session != nil && p.Session.Location != nil {
		locStr = *p.Session.Location
	}
	meetStr := ""
	if p.Session != nil && p.Session.MeetingLink != nil {
		meetStr = *p.Session.MeetingLink
	}
	titleStr := "Sesi Ujian Seleksi PMB"
	typeStr := "written_test"
	startAt := time.Now()
	endAt := time.Now().Add(2 * time.Hour)
	if p.Session != nil {
		titleStr = p.Session.Title
		typeStr = string(p.Session.Type)
		startAt = p.Session.StartAt
		endAt = p.Session.EndAt
	}

	cardData := pdf.ExamCardData{
		ParticipantNumber: p.ParticipantNumber,
		FullName:          "Calon Mahasiswa",
		ApplicationNumber: "PMB-OFFICIAL",
		PeriodName:        "PMB Resmi",
		PathName:          "Jalur Reguler",
		StudyProgramName:  "Program Studi Pilihan",
		SessionTitle:      titleStr,
		SessionType:       typeStr,
		StartAt:           startAt,
		EndAt:             endAt,
		Location:          locStr,
		MeetingLink:       meetStr,
	}

	pdfBytes, err := pdf.GenerateExamCardPDF(cardData)
	if err != nil {
		return nil, "", err
	}
	return pdfBytes, filename, nil
}

func (s *examService) GetParticipantByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Participant, error) {
	p, err := s.participantRepo.FindByApplicationID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrParticipantNotFound
	}
	return p, nil
}

func (s *examService) RemoveParticipantByApplicationID(ctx context.Context, appID uuid.UUID) error {
	return s.participantRepo.RemoveByApplicationID(ctx, appID)
}

