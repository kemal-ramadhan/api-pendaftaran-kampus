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
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrNotPassed            = errors.New("NOT_PASSED: calon mahasiswa belum dinyatakan lulus seleksi")
	ErrPaymentRequired      = errors.New("PAYMENT_REQUIRED: tagihan daftar ulang belum lunas atau termin 1 belum dibayar")
	ErrIncompleteData       = errors.New("INCOMPLETE_DATA: ukuran jas almamater, kontak darurat, dan pernyataan persetujuan wajib diisi")
	ErrAlreadyCompleted     = errors.New("ALREADY_COMPLETED: pendaftaran ulang sudah selesai diproses")
	ErrForbiddenProdiAccess = errors.New("FORBIDDEN: Anda tidak memiliki akses ke data program studi ini")
	ErrStudentNotFound      = errors.New("data mahasiswa tidak ditemukan")
	ErrPKKMBGroupNotFound   = errors.New("gugus PKKMB tidak ditemukan")
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
	WithdrawApplication(ctx context.Context, appID uuid.UUID) error

	// Enrolment & NIM Issue (Butir h)
	GetEnrolmentList(ctx context.Context, filter model.EnrolmentFilter, userRoles, userScopes []string) ([]model.EnrolmentItem, int64, error)
	IssueNIMManual(ctx context.Context, studentIDOrAppID uuid.UUID) (*model.Student, error)
	IssueNIMBatch(ctx context.Context, req model.BatchNIMRequest) (*model.BatchNIMResponse, error)

	// PKKMB (Butir h)
	ListPKKMBGroups(ctx context.Context, periodID *uuid.UUID) ([]model.PKKMBGroup, error)
	CreatePKKMBGroup(ctx context.Context, req *model.CreatePKKMBGroupRequest) (*model.PKKMBGroup, error)
	UpdatePKKMBGroup(ctx context.Context, id uuid.UUID, req *model.UpdatePKKMBGroupRequest) (*model.PKKMBGroup, error)
	DeletePKKMBGroup(ctx context.Context, id uuid.UUID) error
	AutoAssignPKKMB(ctx context.Context, req *model.AutoAssignPKKMBRequest) (*model.AutoAssignPKKMBResponse, error)
	AssignPKKMBManual(ctx context.Context, studentID uuid.UUID, groupID uuid.UUID, adminID *uuid.UUID) error
	ExportPKKMBXLSX(ctx context.Context, periodID *uuid.UUID) ([]byte, error)

	// SIAKAD Export (Butir h)
	ExportSIAKAD(ctx context.Context, format string) ([]byte, error)
}

type studentService struct {
	cfg          *config.Config
	repo         repository.StudentRepository
	exporter     exporter.StudentExporter
	admissionCli internalclient.Client
	selectionCli internalclient.Client
	financeCli   internalclient.Client
	nimCli       internalclient.Client
	identityCli  internalclient.Client
}

func NewStudentService(
	repo repository.StudentRepository,
	exp exporter.StudentExporter,
	admissionCli internalclient.Client,
	selectionCli internalclient.Client,
	financeCli internalclient.Client,
	nimCli internalclient.Client,
	identityCli internalclient.Client,
	cfg *config.Config,
) StudentService {
	if cfg == nil {
		cfg = &config.Config{NIMIssuanceMode: "manual"}
	}
	return &studentService{
		cfg:          cfg,
		repo:         repo,
		exporter:     exp,
		admissionCli: admissionCli,
		selectionCli: selectionCli,
		financeCli:   financeCli,
		nimCli:       nimCli,
		identityCli:  identityCli,
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

	// Calon mahasiswa melihat gugusnya di reregistration-overview (field pkkmb_group)
	var pkkmbInfo *model.PKKMBGroupInfo
	st, _ := s.repo.GetStudentByApplicationID(ctx, reg.ApplicationID)
	if st != nil {
		assignment, _ := s.repo.GetPKKMBAssignmentByStudentID(ctx, st.ID)
		if assignment != nil {
			grp, _ := s.repo.GetPKKMBGroupByID(ctx, assignment.GroupID)
			if grp != nil {
				pkkmbInfo = &model.PKKMBGroupInfo{
					ID:           grp.ID,
					Name:         grp.Name,
					LeaderName:   grp.LeaderName,
					ScheduleNote: grp.ScheduleNote,
				}
			}
		}
	}

	return &model.ReregistrationDetailResponse{
		Reregistration:   reg,
		IsPaymentCleared: isCleared,
		PaymentStatus:    invStatus,
		InvoiceID:        invID,
		CanComplete:      canComplete,
		PKKMBGroup:       pkkmbInfo,
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

	isAuto := s.cfg != nil && s.cfg.NIMIssuanceMode == "auto"

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

	// Siapkan payload outbox event untuk NIM Service
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
	if err := s.repo.CreateOutboxEventInTx(tx, event); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Perbarui status di Admission Service menjadi 'reregistered' dan proses antrean outbox NIM jika mode auto
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.updateAdmissionStatus(bgCtx, reg.ApplicationID, "reregistered")
		if isAuto {
			_, _ = s.ProcessPendingOutboxEvents(bgCtx)
		}
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

func (s *studentService) WithdrawApplication(ctx context.Context, appID uuid.UUID) error {
	runInTx := func(fn func(tx *gorm.DB) error) error {
		if s.repo.DB() != nil {
			return s.repo.DB().WithContext(ctx).Transaction(fn)
		}
		return fn(nil)
	}

	return runInTx(func(tx *gorm.DB) error {
		now := time.Now()
		if tx != nil {
			_ = tx.Model(&model.Reregistration{}).Where("application_id = ?", appID).Updates(map[string]interface{}{
				"status":     model.ReregistrationStatusCancelled,
				"updated_at": now,
			}).Error
			_ = tx.Model(&model.Student{}).Where("application_id = ?", appID).Updates(map[string]interface{}{
				"status":     model.StudentStatusWithdrawn,
				"updated_at": now,
			}).Error
		} else {
			rereg, _ := s.repo.GetReregistrationByApplicationID(ctx, appID)
			if rereg != nil {
				rereg.Status = model.ReregistrationStatusCancelled
				_ = s.repo.UpdateReregistration(ctx, rereg)
			}
		}
		return nil
	})
}

// -----------------------------------------------------------------------------
// Enrolment & NIM Issuance (Butir h)
// -----------------------------------------------------------------------------

func (s *studentService) GetEnrolmentList(ctx context.Context, filter model.EnrolmentFilter, userRoles, userScopes []string) ([]model.EnrolmentItem, int64, error) {
	studFilter := model.StudentFilter{StudyProgramCode: filter.StudyProgramCode}
	if err := s.applyProdiIsolation(&studFilter, userRoles, userScopes); err != nil {
		return nil, 0, err
	}
	filter.StudyProgramCode = studFilter.StudyProgramCode

	reregs, err := s.repo.ListReregistrations(ctx, "")
	if err != nil {
		return nil, 0, err
	}

	students, _, _ := s.repo.ListStudents(ctx, model.StudentFilter{
		StudyProgramCode: filter.StudyProgramCode,
		Limit:            10000,
	})
	studentMap := make(map[uuid.UUID]model.Student)
	for _, st := range students {
		studentMap[st.ApplicationID] = st
	}

	assignments, _ := s.repo.ListAllPKKMBAssignments(ctx)
	assignmentMap := make(map[uuid.UUID]model.PKKMBAssignment)
	for _, a := range assignments {
		assignmentMap[a.StudentID] = a
	}

	groups, _ := s.repo.ListPKKMBGroups(ctx, nil)
	groupNameMap := make(map[uuid.UUID]string)
	for _, g := range groups {
		groupNameMap[g.ID] = g.Name
	}

	var allItems []model.EnrolmentItem

	for _, reg := range reregs {
		if filter.StudyProgramCode != "" && reg.AcceptedProgram != filter.StudyProgramCode {
			continue
		}

		isCleared, invStatus, _ := s.checkPaymentCleared(ctx, reg.ApplicationID)
		if invStatus == "" {
			invStatus = "unpaid"
		}

		st, hasStudent := studentMap[reg.ApplicationID]
		nim := ""
		var studentID *uuid.UUID
		pkkmbGroupName := ""
		var pkkmbGroupID *uuid.UUID

		if hasStudent {
			nim = st.NIM
			studentID = &st.ID
			if asg, ok := assignmentMap[st.ID]; ok {
				pkkmbGroupID = &asg.GroupID
				pkkmbGroupName = groupNameMap[asg.GroupID]
			}
		}

		canIssue := reg.Status == model.ReregistrationStatusCompleted && isCleared && nim == ""
		actions := []string{}
		if canIssue {
			actions = append(actions, "issue_nim")
		}

		// Filter
		switch filter.Filter {
		case "has_nim":
			if nim == "" {
				continue
			}
		case "ready_for_nim":
			if !canIssue {
				continue
			}
		case "waiting_tuition":
			if invStatus != "unpaid" && isCleared {
				continue
			}
		}

		// Ambil nama dari kontak darurat atau user ID sebagai default
		name := reg.EmergencyContactName
		if name == "" {
			name = "Calon Mahasiswa"
		}
		regNum := fmt.Sprintf("PMB-%d-%s", reg.CreatedAt.Year(), reg.ApplicationID.String()[:8])

		// Search
		if filter.Search != "" {
			term := strings.ToLower(filter.Search)
			matched := strings.Contains(strings.ToLower(regNum), term) ||
				strings.Contains(strings.ToLower(name), term) ||
				strings.Contains(strings.ToLower(reg.AcceptedProgram), term) ||
				strings.Contains(strings.ToLower(nim), term)
			if !matched {
				continue
			}
		}

		item := model.EnrolmentItem{
			StudentID:          studentID,
			ApplicationID:      reg.ApplicationID,
			UserID:             reg.UserID,
			RegistrationNumber: regNum,
			Name:               name,
			StudyProgramCode:   reg.AcceptedProgram,
			SelectionStatus:    "passed",
			TuitionStatus:      invStatus,
			NIM:                nim,
			PKKMBGroup:         pkkmbGroupName,
			PKKMBGroupID:       pkkmbGroupID,
			CanIssueNIM:        canIssue,
			Actions:            actions,
		}
		allItems = append(allItems, item)
	}

	total := int64(len(allItems))
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > len(allItems) {
		offset = len(allItems)
	}
	end := offset + limit
	if end > len(allItems) {
		end = len(allItems)
	}

	return allItems[offset:end], total, nil
}

func (s *studentService) IssueNIMManual(ctx context.Context, id uuid.UUID) (*model.Student, error) {
	st, _ := s.repo.GetStudentByID(ctx, id)
	if st == nil {
		st, _ = s.repo.GetStudentByApplicationID(ctx, id)
	}
	if st == nil {
		reg, errR := s.repo.GetReregistrationByApplicationID(ctx, id)
		if errR == nil && reg != nil && reg.Status == model.ReregistrationStatusCompleted {
			now := time.Now()
			newSt := &model.Student{
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
			_ = s.repo.CreateStudentInTx(s.repo.DB(), newSt)
			st = newSt
		}
	}
	if st == nil {
		return nil, ErrStudentNotFound
	}

	if st.NIM != "" {
		return st, nil // Idempoten
	}

	isCleared, _, _ := s.checkPaymentCleared(ctx, st.ApplicationID)
	if !isCleared {
		return nil, ErrPaymentRequired
	}

	type issueResponse struct {
		NIM string `json:"nim"`
	}
	var res issueResponse
	reqBody := map[string]interface{}{
		"application_id":     st.ApplicationID,
		"user_id":            st.UserID,
		"study_program_code": st.StudyProgramCode,
		"entry_year":         st.EntryYear,
		"selection_path":     st.EntryPath,
	}

	var nim string
	if s.nimCli != nil {
		err := s.nimCli.Post(ctx, "/internal/nim/issue", reqBody, &res)
		if err == nil && res.NIM != "" {
			nim = res.NIM
		}
	}
	if nim == "" {
		yy := fmt.Sprintf("%02d", st.EntryYear%100)
		nim = fmt.Sprintf("%s551%04d", yy, time.Now().Unix()%10000)
	}

	st.NIM = nim
	st.Status = model.StudentStatusActive
	st.UpdatedAt = time.Now()
	if err := s.repo.UpdateStudent(ctx, st); err != nil {
		return nil, err
	}

	return st, nil
}

func (s *studentService) IssueNIMBatch(ctx context.Context, req model.BatchNIMRequest) (*model.BatchNIMResponse, error) {
	students, err := s.repo.ListStudentsReadyForNIM(ctx, req.StudyProgramCode)
	if err != nil {
		return nil, err
	}

	resp := &model.BatchNIMResponse{
		Processed: 0,
		Success:   0,
		Skipped:   0,
		Failed:    0,
		Details:   []model.BatchNIMDetailItem{},
	}

	for _, st := range students {
		resp.Processed++
		studentCopy := st

		if st.NIM != "" {
			resp.Skipped++
			resp.Details = append(resp.Details, model.BatchNIMDetailItem{
				StudentID:     &studentCopy.ID,
				ApplicationID: studentCopy.ApplicationID,
				NIM:           studentCopy.NIM,
				Status:        "skipped",
				Reason:        "Sudah memiliki NIM",
			})
			continue
		}

		isCleared, _, _ := s.checkPaymentCleared(ctx, st.ApplicationID)
		if !isCleared {
			resp.Skipped++
			resp.Details = append(resp.Details, model.BatchNIMDetailItem{
				StudentID:     &studentCopy.ID,
				ApplicationID: studentCopy.ApplicationID,
				Status:        "skipped",
				Reason:        "Tagihan UKT belum lunas",
			})
			continue
		}

		updatedSt, errIssue := s.IssueNIMManual(ctx, st.ID)
		if errIssue != nil {
			resp.Failed++
			resp.Details = append(resp.Details, model.BatchNIMDetailItem{
				StudentID:     &studentCopy.ID,
				ApplicationID: studentCopy.ApplicationID,
				Status:        "failed",
				Reason:        errIssue.Error(),
			})
			continue
		}

		resp.Success++
		resp.Details = append(resp.Details, model.BatchNIMDetailItem{
			StudentID:     &studentCopy.ID,
			ApplicationID: studentCopy.ApplicationID,
			NIM:           updatedSt.NIM,
			Status:        "success",
		})
	}

	return resp, nil
}

// -----------------------------------------------------------------------------
// PKKMB Service Methods (Butir h)
// -----------------------------------------------------------------------------

func (s *studentService) ListPKKMBGroups(ctx context.Context, periodID *uuid.UUID) ([]model.PKKMBGroup, error) {
	groups, err := s.repo.ListPKKMBGroups(ctx, periodID)
	if err != nil {
		return nil, err
	}
	counts, _ := s.repo.CountAssignedPerGroup(ctx)
	for i := range groups {
		if c, ok := counts[groups[i].ID]; ok {
			groups[i].AssignedCount = c
		}
	}
	return groups, nil
}

func (s *studentService) CreatePKKMBGroup(ctx context.Context, req *model.CreatePKKMBGroupRequest) (*model.PKKMBGroup, error) {
	now := time.Now()
	group := &model.PKKMBGroup{
		ID:           uuid.New(),
		PeriodID:     req.PeriodID,
		Name:         strings.TrimSpace(req.Name),
		Capacity:     req.Capacity,
		LeaderName:   strings.TrimSpace(req.LeaderName),
		ScheduleNote: strings.TrimSpace(req.ScheduleNote),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreatePKKMBGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *studentService) UpdatePKKMBGroup(ctx context.Context, id uuid.UUID, req *model.UpdatePKKMBGroupRequest) (*model.PKKMBGroup, error) {
	grp, err := s.repo.GetPKKMBGroupByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if grp == nil {
		return nil, ErrPKKMBGroupNotFound
	}

	if req.Name != "" {
		grp.Name = strings.TrimSpace(req.Name)
	}
	if req.Capacity > 0 {
		grp.Capacity = req.Capacity
	}
	if req.LeaderName != "" {
		grp.LeaderName = strings.TrimSpace(req.LeaderName)
	}
	if req.ScheduleNote != "" {
		grp.ScheduleNote = strings.TrimSpace(req.ScheduleNote)
	}
	grp.UpdatedAt = time.Now()

	if err := s.repo.UpdatePKKMBGroup(ctx, grp); err != nil {
		return nil, err
	}
	return grp, nil
}

func (s *studentService) DeletePKKMBGroup(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePKKMBGroup(ctx, id)
}

func (s *studentService) AutoAssignPKKMB(ctx context.Context, req *model.AutoAssignPKKMBRequest) (*model.AutoAssignPKKMBResponse, error) {
	students, err := s.repo.ListStudentsWithNIM(ctx, "")
	if err != nil {
		return nil, err
	}

	groups, err := s.ListPKKMBGroups(ctx, req.PeriodID)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("belum ada gugus PKKMB yang terdaftar")
	}

	existingAssignments, _ := s.repo.ListAllPKKMBAssignments(ctx)
	assignedMap := make(map[uuid.UUID]bool)
	for _, a := range existingAssignments {
		assignedMap[a.StudentID] = true
	}

	var unassigned []model.Student
	for _, st := range students {
		if !assignedMap[st.ID] {
			unassigned = append(unassigned, st)
		}
	}

	assignedCount := 0
	groupIdx := 0

	// Distribusikan secara round-robin per kapasitas grup
	for _, st := range unassigned {
		// Cari gugus yang masih memiliki kapasitas
		found := false
		for attempts := 0; attempts < len(groups); attempts++ {
			g := &groups[groupIdx]
			groupIdx = (groupIdx + 1) % len(groups)
			if g.Capacity > g.AssignedCount {
				asg := &model.PKKMBAssignment{
					ID:        uuid.New(),
					StudentID: st.ID,
					GroupID:   g.ID,
					Method:    "auto",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				_ = s.repo.CreateOrUpdatePKKMBAssignment(ctx, asg)
				g.AssignedCount++
				assignedCount++
				found = true
				break
			}
		}
		if !found {
			// Kapasitas semua grup telah penuh
			break
		}
	}

	return &model.AutoAssignPKKMBResponse{
		AssignedCount:   assignedCount,
		UnassignedCount: len(unassigned) - assignedCount,
		TotalGroups:     len(groups),
	}, nil
}

func (s *studentService) AssignPKKMBManual(ctx context.Context, studentID uuid.UUID, groupID uuid.UUID, adminID *uuid.UUID) error {
	grp, err := s.repo.GetPKKMBGroupByID(ctx, groupID)
	if err != nil {
		return err
	}
	if grp == nil {
		return ErrPKKMBGroupNotFound
	}

	asg := &model.PKKMBAssignment{
		ID:         uuid.New(),
		StudentID:  studentID,
		GroupID:    groupID,
		AssignedBy: adminID,
		Method:     "manual",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	return s.repo.CreateOrUpdatePKKMBAssignment(ctx, asg)
}

func (s *studentService) ExportPKKMBXLSX(ctx context.Context, periodID *uuid.UUID) ([]byte, error) {
	groups, err := s.repo.ListPKKMBGroups(ctx, periodID)
	if err != nil {
		return nil, err
	}
	groupMap := make(map[uuid.UUID]model.PKKMBGroup)
	for _, g := range groups {
		groupMap[g.ID] = g
	}

	students, err := s.repo.ListStudentsWithNIM(ctx, "")
	if err != nil {
		return nil, err
	}

	assignments, _ := s.repo.ListAllPKKMBAssignments(ctx)
	asgMap := make(map[uuid.UUID]model.PKKMBAssignment)
	for _, a := range assignments {
		asgMap[a.StudentID] = a
	}

	var exportRows []exporter.PKKMBExportRow
	for _, st := range students {
		asg, ok := asgMap[st.ID]
		if !ok {
			continue
		}
		grp := groupMap[asg.GroupID]
		name, _ := s.fetchStudentProfile(ctx, st.UserID)
		if name == "" {
			name = fmt.Sprintf("Mahasiswa %s", st.NIM)
		}
		exportRows = append(exportRows, exporter.PKKMBExportRow{
			NIM:          st.NIM,
			Name:         name,
			Prodi:        st.StudyProgramCode,
			GroupName:    grp.Name,
			LeaderName:   grp.LeaderName,
			ScheduleNote: grp.ScheduleNote,
			Method:       asg.Method,
		})
	}

	return s.exporter.ExportPKKMB(exportRows)
}

func (s *studentService) ExportSIAKAD(ctx context.Context, format string) ([]byte, error) {
	students, err := s.repo.ListStudentsWithNIM(ctx, "")
	if err != nil {
		return nil, err
	}

	var studentIDs []uuid.UUID
	var items []model.SIAKADStudentItem
	for _, st := range students {
		studentIDs = append(studentIDs, st.ID)
		name, email := s.fetchStudentProfile(ctx, st.UserID)
		if name == "" {
			name = fmt.Sprintf("Mahasiswa %s", st.NIM)
		}
		if email == "" {
			email = fmt.Sprintf("%s@student.kampus.ac.id", st.NIM)
		}
		items = append(items, model.SIAKADStudentItem{
			NIM:              st.NIM,
			Name:             name,
			StudyProgramCode: st.StudyProgramCode,
			Email:            email,
			EntryPath:        st.EntryPath,
			EntryYear:        st.EntryYear,
		})
	}

	_ = s.repo.MarkSiakadExported(ctx, studentIDs)

	if strings.ToLower(format) == "xlsx" {
		return s.exporter.ExportSIAKADXLSX(items)
	}
	return s.exporter.ExportSIAKADCSV(items)
}

func (s *studentService) fetchStudentProfile(ctx context.Context, userID uuid.UUID) (name, email string) {
	if s.identityCli != nil {
		var person struct {
			Fullname string `json:"fullname"`
		}
		_ = s.identityCli.Get(ctx, fmt.Sprintf("/internal/persons/%s", userID.String()), nil, &person)
		if person.Fullname != "" {
			name = person.Fullname
		}
	}
	return name, email
}

