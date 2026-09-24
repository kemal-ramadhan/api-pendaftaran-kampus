package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/repository"
)

var (
	ErrNotPassed            = errors.New("NOT_PASSED: calon mahasiswa belum dinyatakan lulus seleksi")
	ErrPaymentRequired      = errors.New("PAYMENT_REQUIRED: tagihan daftar ulang belum lunas atau termin 1 belum dibayar")
	ErrIncompleteData       = errors.New("INCOMPLETE_DATA: ukuran jas almamater, kontak darurat, dan pernyataan persetujuan wajib diisi")
	ErrAlreadyCompleted     = errors.New("ALREADY_COMPLETED: pendaftaran ulang sudah selesai diproses")
	ErrForbiddenProdiAccess = errors.New("FORBIDDEN: Anda tidak memiliki akses ke data program studi ini")
	ErrStudentNotFound      = errors.New("data mahasiswa tidak ditemukan")
)

type StudentService interface {
	GetMyReregistration(ctx context.Context, userID uuid.UUID) (*model.ReregistrationDetailResponse, error)
	UpdateMyReregistration(ctx context.Context, userID uuid.UUID, req *model.UpdateReregistrationRequest) (*model.ReregistrationDetailResponse, error)
	CompleteReregistration(ctx context.Context, userID uuid.UUID) (*model.Student, error)
	ListStudents(ctx context.Context, filter model.StudentFilter, userRoles, userScopes []string) ([]model.Student, int64, error)
	ExportStudentsXLSX(ctx context.Context, filter model.StudentFilter, userRoles, userScopes []string) ([]byte, error)
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*model.Student, error)
	GetStudentByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Student, error)
	UpdateStudentNIM(ctx context.Context, appID uuid.UUID, nim string) error
	ProcessPendingOutboxEvents(ctx context.Context) (int, error)
}

type studentService struct {
	repo         repository.StudentRepository
	exporter     exporter.StudentExporter
	admissionCli internalclient.Client
	selectionCli internalclient.Client
	financeCli   internalclient.Client
	nimCli       internalclient.Client
}

func NewStudentService(
	repo repository.StudentRepository,
	exp exporter.StudentExporter,
	admissionCli internalclient.Client,
	selectionCli internalclient.Client,
	financeCli internalclient.Client,
	nimCli internalclient.Client,
) StudentService {
	return &studentService{
		repo:         repo,
		exporter:     exp,
		admissionCli: admissionCli,
		selectionCli: selectionCli,
		financeCli:   financeCli,
		nimCli:       nimCli,
	}
}

// -----------------------------------------------------------------------------
// Inter-Service Response Types
// -----------------------------------------------------------------------------

type admissionAppItem struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	Status             string    `json:"status"`
	StudyProgramCode   string    `json:"study_program_code"`
	AdmissionPathID    uint      `json:"admission_path_id"`
	RegistrationNumber string    `json:"registration_number"`
}

type selectionResultItem struct {
	ApplicationID   uuid.UUID `json:"application_id"`
	UserID          uuid.UUID `json:"user_id"`
	Decision        string    `json:"decision"`
	AcceptedProgram string    `json:"accepted_program"`
}

type financeInvoiceItem struct {
	ID      uuid.UUID `json:"id"`
	FeeType string    `json:"fee_type"`
	Amount  int64     `json:"amount"`
	Status  string    `json:"status"`
}

// -----------------------------------------------------------------------------
// Core Business Methods
// -----------------------------------------------------------------------------

func (s *studentService) GetMyReregistration(ctx context.Context, userID uuid.UUID) (*model.ReregistrationDetailResponse, error) {
	reg, err := s.repo.GetReregistrationByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Jika belum ada record reregistrations, periksa kelulusan dan buat record awal
	if reg == nil {
		app, errApp := s.fetchUserActiveApplication(ctx, userID)
		if errApp != nil || app == nil {
			return nil, ErrNotPassed
		}

		// Validasi status kelulusan (passed atau reregistered)
		if app.Status != "passed" && app.Status != "reregistered" {
			return nil, ErrNotPassed
		}

		acceptedProgram := app.StudyProgramCode
		if acceptedProgram == "" {
			acceptedProgram = "TI"
		}

		// Cari accepted_program dari Selection Service jika tersedia
		selRes, errSel := s.fetchSelectionResult(ctx, app.ID)
		if errSel == nil && selRes != nil {
			if selRes.Decision != "pass" {
				return nil, ErrNotPassed
			}
			if selRes.AcceptedProgram != "" {
				acceptedProgram = selRes.AcceptedProgram
			}
		}

		newReg := &model.Reregistration{
			ID:              uuid.New(),
			ApplicationID:   app.ID,
			UserID:          userID,
			AcceptedProgram: acceptedProgram,
			Status:          model.ReregistrationStatusWaitingPayment,
			StatementAgreed: false,
		}

		if err := s.repo.CreateReregistration(ctx, newReg); err != nil {
			return nil, err
		}
		reg = newReg
	}

	// Periksa kliring pembayaran tagihan daftar ulang ke Finance Service
	isCleared, invStatus, invID := s.checkPaymentCleared(ctx, reg.ApplicationID)

	canComplete := isCleared &&
		reg.StatementAgreed &&
		reg.JacketSize != "" &&
		reg.EmergencyContactName != "" &&
		reg.EmergencyContactPhone != "" &&
		reg.Status != model.ReregistrationStatusCompleted

	return &model.ReregistrationDetailResponse{
		Reregistration:   reg,
		IsPaymentCleared: isCleared,
		PaymentStatus:    invStatus,
		InvoiceID:        invID,
		CanComplete:      canComplete,
	}, nil
}

func (s *studentService) UpdateMyReregistration(ctx context.Context, userID uuid.UUID, req *model.UpdateReregistrationRequest) (*model.ReregistrationDetailResponse, error) {
	detail, err := s.GetMyReregistration(ctx, userID)
	if err != nil {
		return nil, err
	}

	reg := detail.Reregistration
	if reg.Status == model.ReregistrationStatusCompleted {
		return nil, ErrAlreadyCompleted
	}

	// Validasi pilihan ukuran jaket almamater
	validSizes := map[string]bool{"S": true, "M": true, "L": true, "XL": true, "XXL": true, "XXXL": true}
	upperSize := strings.ToUpper(strings.TrimSpace(req.JacketSize))
	if !validSizes[upperSize] {
		return nil, fmt.Errorf("ukuran jas almamater tidak valid. Pilihan: S, M, L, XL, XXL, XXXL")
	}

	reg.JacketSize = upperSize
	reg.EmergencyContactName = strings.TrimSpace(req.EmergencyContactName)
	reg.EmergencyContactPhone = strings.TrimSpace(req.EmergencyContactPhone)
	if req.StatementAgreed != nil {
		reg.StatementAgreed = *req.StatementAgreed
	}

	if reg.Status == model.ReregistrationStatusWaitingPayment {
		reg.Status = model.ReregistrationStatusInProgress
	}

	if err := s.repo.UpdateReregistration(ctx, reg); err != nil {
		return nil, err
	}

	return s.GetMyReregistration(ctx, userID)
}

func (s *studentService) CompleteReregistration(ctx context.Context, userID uuid.UUID) (*model.Student, error) {
	detail, err := s.GetMyReregistration(ctx, userID)
	if err != nil {
		return nil, err
	}

	reg := detail.Reregistration

	// Jika sudah pernah selesai, kembalikan data mahasiswa yang sudah ada
	if reg.Status == model.ReregistrationStatusCompleted {
		existingStudent, errS := s.repo.GetStudentByApplicationID(ctx, reg.ApplicationID)
		if errS == nil && existingStudent != nil {
			return existingStudent, nil
		}
	}

	// 1. Verifikasi kliring pembayaran (lunas atau partially_paid termin 1)
	if !detail.IsPaymentCleared {
		return nil, ErrPaymentRequired
	}

	// 2. Verifikasi kelengkapan data tambahan
	if !reg.StatementAgreed || reg.JacketSize == "" || reg.EmergencyContactName == "" || reg.EmergencyContactPhone == "" {
		return nil, ErrIncompleteData
	}

	now := time.Now()
	reg.Status = model.ReregistrationStatusCompleted
	reg.CompletedAt = &now

	student := &model.Student{
		ID:               uuid.New(),
		UserID:           reg.UserID,
		ApplicationID:    reg.ApplicationID,
		StudyProgramCode: reg.AcceptedProgram,
		EntryYear:        now.Year(),
		EntryPath:        "Reguler",
		Status:           model.StudentStatusReregistered,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Siapkan payload outbox event untuk NIM Service di Tahap 11
	outboxPayload := model.NIMIssuancePayload{
		ApplicationID:    reg.ApplicationID,
		UserID:           reg.UserID,
		StudyProgramCode: reg.AcceptedProgram,
		EntryYear:        now.Year(),
		EntryPath:        "Reguler",
	}
	payloadBytes, _ := json.Marshal(outboxPayload)

	event := &model.OutboxEvent{
		ID:        uuid.New(),
		EventType: "nim_issuance_requested",
		Payload:   string(payloadBytes),
		Status:    "pending",
		CreatedAt: now,
	}

	// Simpan secara transaksional
	tx := s.repo.DB().Begin()
	if err := s.repo.UpdateReregistrationInTx(tx, reg); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := s.repo.CreateStudentInTx(tx, student); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := s.repo.CreateOutboxEventInTx(tx, event); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Perbarui status di Admission Service menjadi 'reregistered' dan proses antrean outbox NIM
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.updateAdmissionStatus(bgCtx, reg.ApplicationID, "reregistered")
		_, _ = s.ProcessPendingOutboxEvents(bgCtx)
	}()

	return student, nil
}

func (s *studentService) ListStudents(ctx context.Context, filter model.StudentFilter, userRoles, userScopes []string) ([]model.Student, int64, error) {
	if err := s.applyProdiIsolation(&filter, userRoles, userScopes); err != nil {
		return nil, 0, err
	}
	return s.repo.ListStudents(ctx, filter)
}

func (s *studentService) ExportStudentsXLSX(ctx context.Context, filter model.StudentFilter, userRoles, userScopes []string) ([]byte, error) {
	if err := s.applyProdiIsolation(&filter, userRoles, userScopes); err != nil {
		return nil, err
	}
	filter.Limit = 10000 // Ekspor seluruh baris yang cocok
	filter.Offset = 0
	students, _, err := s.repo.ListStudents(ctx, filter)
	if err != nil {
		return nil, err
	}
	return s.exporter.ExportStudents(students)
}

func (s *studentService) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*model.Student, error) {
	st, err := s.repo.GetStudentByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrStudentNotFound
	}
	return st, nil
}

func (s *studentService) GetStudentByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Student, error) {
	st, err := s.repo.GetStudentByApplicationID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrStudentNotFound
	}
	return st, nil
}

func (s *studentService) UpdateStudentNIM(ctx context.Context, appID uuid.UUID, nim string) error {
	tx := s.repo.DB().Begin()
	if err := s.repo.UpdateStudentNIMInTx(tx, appID, nim); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// -----------------------------------------------------------------------------
// Helper Methods & Inter-Service Integration
// -----------------------------------------------------------------------------

func (s *studentService) applyProdiIsolation(filter *model.StudentFilter, userRoles, userScopes []string) error {
	isSuperOrAdminOrPimpinan := false
	for _, r := range userRoles {
		if r == "superadmin" || r == "admin_pmb" || r == "pimpinan" {
			isSuperOrAdminOrPimpinan = true
			break
		}
	}
	if isSuperOrAdminOrPimpinan {
		return nil
	}

	isProdi := false
	for _, r := range userRoles {
		if r == "prodi" {
			isProdi = true
			break
		}
	}

	if isProdi {
		if len(userScopes) == 0 {
			return ErrForbiddenProdiAccess
		}
		prodiScope := userScopes[0]
		if filter.StudyProgramCode != "" && filter.StudyProgramCode != prodiScope {
			return ErrForbiddenProdiAccess
		}
		filter.StudyProgramCode = prodiScope
		return nil
	}

	return nil
}

func (s *studentService) fetchUserActiveApplication(ctx context.Context, userID uuid.UUID) (*admissionAppItem, error) {
	if s.admissionCli == nil {
		return nil, nil
	}
	var apps []admissionAppItem
	err := s.admissionCli.Get(ctx, fmt.Sprintf("/internal/applications/user/%s", userID.String()), nil, &apps)
	if err != nil || len(apps) == 0 {
		return nil, err
	}
	return &apps[0], nil
}

func (s *studentService) fetchSelectionResult(ctx context.Context, appID uuid.UUID) (*selectionResultItem, error) {
	if s.selectionCli == nil {
		return nil, nil
	}
	var res selectionResultItem
	err := s.selectionCli.Get(ctx, fmt.Sprintf("/internal/results/%s", appID.String()), nil, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *studentService) checkPaymentCleared(ctx context.Context, appID uuid.UUID) (bool, string, *uuid.UUID) {
	if s.financeCli == nil {
		return true, "paid", nil // Default kliring jika mock/development tanpa client
	}

	var inv financeInvoiceItem
	err := s.financeCli.Get(ctx, "/internal/invoices", map[string]string{
		"application_id": appID.String(),
	}, &inv)

	if err != nil {
		return false, "unpaid", nil
	}

	// Status yang sah: paid (lunas penuh) atau partially_paid (termin 1 cicilan lunas)
	if inv.Status == "paid" || inv.Status == "partially_paid" {
		return true, inv.Status, &inv.ID
	}

	return false, inv.Status, &inv.ID
}

func (s *studentService) updateAdmissionStatus(ctx context.Context, appID uuid.UUID, status string) error {
	if s.admissionCli == nil {
		return nil
	}
	payload := map[string]interface{}{
		"status": status,
		"notes":  "Pendaftaran ulang selesai, siap penerbitan NIM",
	}
	return s.admissionCli.Put(ctx, fmt.Sprintf("/internal/applications/%s/status", appID.String()), payload, nil)
}

func (s *studentService) ProcessPendingOutboxEvents(ctx context.Context) (int, error) {
	if s.nimCli == nil {
		return 0, nil
	}

	events, err := s.repo.GetPendingOutboxEvents(ctx, 20)
	if err != nil {
		return 0, err
	}

	processedCount := 0
	for _, evt := range events {
		if evt.EventType == "nim_issuance_requested" {
			var payload model.NIMIssuancePayload
			if errUnmarshal := json.Unmarshal([]byte(evt.Payload), &payload); errUnmarshal != nil {
				evt.RetryCount++
				_ = s.repo.UpdateOutboxEvent(ctx, &evt)
				continue
			}

			reqBody := map[string]interface{}{
				"application_id":     payload.ApplicationID,
				"user_id":            payload.UserID,
				"study_program_code": payload.StudyProgramCode,
				"entry_year":         payload.EntryYear,
				"selection_path":     payload.EntryPath,
			}

			errCall := s.nimCli.Post(ctx, "/internal/nim/issue", reqBody, nil)
			if errCall != nil {
				evt.RetryCount++
				_ = s.repo.UpdateOutboxEvent(ctx, &evt)
				continue
			}

			now := time.Now()
			evt.Status = "processed"
			evt.ProcessedAt = &now
			if errUpdate := s.repo.UpdateOutboxEvent(ctx, &evt); errUpdate == nil {
				processedCount++
			}
		}
	}

	return processedCount, nil
}
