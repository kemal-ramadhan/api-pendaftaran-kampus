package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/xuri/excelize/v2"
)

var (
	ErrPathChangeNotAllowed   = errors.New("PATH_CHANGE_NOT_ALLOWED: pindah jalur tidak diizinkan pada status pendaftaran saat ini")
	ErrPathChangeLimitReached = errors.New("PATH_CHANGE_LIMIT_REACHED: batas pindah jalur setelah pembayaran (maksimal 1 kali) telah tercapai")
	ErrPathChangePending      = errors.New("PATH_CHANGE_PENDING: masih ada permohonan pindah jalur yang sedang menunggu persetujuan (pending)")
	ErrPaymentInProgress      = errors.New("PAYMENT_IN_PROGRESS: pembayaran tagihan sedang dalam proses atau belum kedaluwarsa")
	ErrReasonRequired         = errors.New("alasan pindah jalur wajib diisi (minimal 10 dan maksimal 500 karakter)")
	ErrRequestNotFound        = errors.New("permohonan pindah jalur tidak ditemukan")
)

type PathChangeInput struct {
	PeriodID       *uint      `json:"period_id,omitempty"`
	PathID         uint       `json:"path_id"`
	TargetPathID   uint       `json:"target_path_id"`
	NewPathID      uint       `json:"new_path_id"`
	FirstChoiceID  uuid.UUID  `json:"first_choice_program"`
	Choice1ID      uuid.UUID  `json:"choice1_id"`
	Choice1IDAlt   uuid.UUID  `json:"choice_1_id"`
	SecondChoiceID *uuid.UUID `json:"second_choice_program,omitempty"`
	Choice2ID      *uuid.UUID `json:"choice2_id,omitempty"`
	Choice2IDAlt   *uuid.UUID `json:"choice_2_id,omitempty"`
	Reason         string     `json:"reason,omitempty"`
}

type AdminMoveApplicationInput struct {
	PeriodID       *uint      `json:"period_id,omitempty"`
	PathID         uint       `json:"path_id"`
	TargetPathID   uint       `json:"target_path_id"`
	FirstChoiceID  uuid.UUID  `json:"first_choice_program"`
	SecondChoiceID *uuid.UUID `json:"second_choice_program,omitempty"`
	Action         string     `json:"action,omitempty"`
	Reason         string     `json:"reason"`
}

type PathChangeOptionsResponse struct {
	Allowed          bool                 `json:"allowed"`
	BlockReasonLabel string               `json:"block_reason_label,omitempty"`
	CurrentPathID    uint                 `json:"current_path_id"`
	CurrentPathName  string               `json:"current_path_name"`
	CurrentFee       int64                `json:"current_fee"`
	Options          []PathOptionItem     `json:"options"`
}

type PathOptionItem struct {
	PathID           uint   `json:"path_id"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	RegistrationFee  int64  `json:"registration_fee"`
	FeeDifference    int64  `json:"fee_difference"`
	FeeDifferenceText string `json:"fee_difference_text"`
	RequiresExam     bool   `json:"requires_exam"`
	IsActive         bool   `json:"is_active"`
}

type PathChangeResult struct {
	AppliedDirectly bool       `json:"applied_directly"`
	RequestID       *uuid.UUID `json:"request_id,omitempty"`
	Status          string     `json:"status"`
	Message         string     `json:"message"`
}

type PendingRequestSummary struct {
	Type        string    `json:"type"` // "path_change" | "withdrawal"
	StatusLabel string    `json:"status_label"`
	CreatedAt   time.Time `json:"created_at"`
}

type DashboardSummaryResponse struct {
	Application    *model.Application     `json:"application,omitempty"`
	Progress       *ProgressResponse      `json:"progress,omitempty"`
	PendingRequest *PendingRequestSummary `json:"pending_request"`
}

type ChangeService interface {
	GetApplicationChanges(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error)
	GetPathChangeOptions(ctx context.Context, userID uuid.UUID) (*PathChangeOptionsResponse, error)
	RequestOrApplyPathChange(ctx context.Context, userID uuid.UUID, input *PathChangeInput) (*PathChangeResult, error)
	CancelPendingPathChange(ctx context.Context, userID uuid.UUID) error
	GetMyPendingPathChange(ctx context.Context, userID uuid.UUID) (*model.PathChangeRequest, error)

	// Admin
	AdminListPathChangeRequests(ctx context.Context, status string, limit, offset int) ([]model.PathChangeRequest, int64, error)
	AdminApprovePathChangeRequest(ctx context.Context, requestID, reviewerID uuid.UUID, note string) error
	AdminRejectPathChangeRequest(ctx context.Context, requestID, reviewerID uuid.UUID, note string) error
	AdminMoveApplication(ctx context.Context, appID, actorID uuid.UUID, input *AdminMoveApplicationInput) error
	AdminGetApplicationChanges(ctx context.Context, appID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error)
	AdminExportPathChangesXLSX(ctx context.Context, status string) ([]byte, error)

	// Kompatibilitas CR-005: Dashboard dengan pending_request
	GetDashboardSummary(ctx context.Context, userID uuid.UUID) (*DashboardSummaryResponse, error)
}

type changeService struct {
	changeRepo repository.ChangeRepository
	appRepo    repository.ApplicationRepository
	pathRepo   repository.AdmissionPathRepository
	periodRepo repository.PeriodRepository
	quotaRepo  repository.QuotaRepository
	docRepo    repository.ApplicationDocumentRepository
	reqRepo    repository.DocumentRequirementRepository
	financeCli internalclient.Client
	examCli    internalclient.Client
	notifCli       internalclient.Client
	admService     AdmissionService
	withdrawalRepo repository.WithdrawalRepository
}

func NewChangeService(
	changeRepo repository.ChangeRepository,
	appRepo repository.ApplicationRepository,
	pathRepo repository.AdmissionPathRepository,
	periodRepo repository.PeriodRepository,
	quotaRepo repository.QuotaRepository,
	docRepo repository.ApplicationDocumentRepository,
	reqRepo repository.DocumentRequirementRepository,
	financeCli internalclient.Client,
	examCli internalclient.Client,
	notifCli internalclient.Client,
	admService AdmissionService,
	withdrawalRepo repository.WithdrawalRepository,
) ChangeService {
	return &changeService{
		changeRepo:     changeRepo,
		appRepo:        appRepo,
		pathRepo:       pathRepo,
		periodRepo:     periodRepo,
		quotaRepo:      quotaRepo,
		docRepo:        docRepo,
		reqRepo:        reqRepo,
		financeCli:     financeCli,
		examCli:        examCli,
		notifCli:       notifCli,
		admService:     admService,
		withdrawalRepo: withdrawalRepo,
	}
}

// 1. GetApplicationChanges: Linimasa riwayat pendaftaran calon mahasiswa
func (s *changeService) GetApplicationChanges(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	changes, total, err := s.changeRepo.GetChangesByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	for i := range changes {
		enrichChangeMetadata(&changes[i])
	}

	return changes, total, nil
}

// 2. GetPathChangeOptions: Pilihan jalur & prodi yang boleh dituju + perkiraan fee difference
func (s *changeService) GetPathChangeOptions(ctx context.Context, userID uuid.UUID) (*PathChangeOptionsResponse, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentPath, err := s.pathRepo.FindByID(ctx, app.AdmissionPathID)
	if err != nil {
		return nil, err
	}

	allowed := true
	blockReason := ""

	switch app.Status {
	case model.StatusDraft, model.StatusAwaitingPayment:
		allowed = true
	case model.StatusFormOpen, model.StatusSubmitted, model.StatusRevisionRequired, model.StatusVerified:
		if app.PathChangeCountAfterPayment >= 1 {
			allowed = false
			blockReason = "Batas maksimal pindah jalur setelah pembayaran (1 kali) telah tercapai"
		} else {
			pending, _ := s.changeRepo.GetPendingPathChangeRequest(ctx, app.ID)
			if pending != nil {
				allowed = false
				blockReason = "Masih ada permohonan pindah jalur yang sedang diproses (pending)"
			}
		}
	case model.StatusExamScheduled:
		allowed = false
		blockReason = "Pendaftar yang telah dijadwalkan ujian tidak dapat mengajukan pindah jalur secara mandiri (hubungi panitia PMB)"
	default:
		allowed = false
		blockReason = fmt.Sprintf("Pindah jalur tidak diizinkan pada status pendaftaran saat ini (%s)", app.Status)
	}

	allPaths, err := s.pathRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}

	currentFee := int64(currentPath.RegistrationFee)
	var options []PathOptionItem

	for _, p := range allPaths {
		pathFee := int64(p.RegistrationFee)
		diff := pathFee - currentFee
		diffText := "Biaya sama"
		if diff > 0 {
			diffText = fmt.Sprintf("+Rp %d (perlu bayar selisih)", diff)
		} else if diff < 0 {
			diffText = fmt.Sprintf("-Rp %d (tidak ada refund)", -diff)
		}

		options = append(options, PathOptionItem{
			PathID:            p.ID,
			Code:              p.Code,
			Name:              p.Name,
			RegistrationFee:   pathFee,
			FeeDifference:     diff,
			FeeDifferenceText: diffText,
			RequiresExam:      p.RequiresExam,
			IsActive:          p.IsActive,
		})
	}

	return &PathChangeOptionsResponse{
		Allowed:          allowed,
		BlockReasonLabel: blockReason,
		CurrentPathID:    currentPath.ID,
		CurrentPathName:  currentPath.Name,
		CurrentFee:       currentFee,
		Options:          options,
	}, nil
}

// 3. RequestOrApplyPathChange: Pindah langsung atau ajukan permohonan
func (s *changeService) RequestOrApplyPathChange(ctx context.Context, userID uuid.UUID, input *PathChangeInput) (*PathChangeResult, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.PathID == 0 {
		if input.TargetPathID > 0 {
			input.PathID = input.TargetPathID
		} else if input.NewPathID > 0 {
			input.PathID = input.NewPathID
		}
	}
	if input.FirstChoiceID == uuid.Nil {
		if input.Choice1ID != uuid.Nil {
			input.FirstChoiceID = input.Choice1ID
		} else if input.Choice1IDAlt != uuid.Nil {
			input.FirstChoiceID = input.Choice1IDAlt
		} else {
			input.FirstChoiceID = app.Choice1ID
		}
	}
	if input.SecondChoiceID == nil {
		if input.Choice2ID != nil {
			input.SecondChoiceID = input.Choice2ID
		} else if input.Choice2IDAlt != nil {
			input.SecondChoiceID = input.Choice2IDAlt
		} else {
			input.SecondChoiceID = app.Choice2ID
		}
	}

	targetPath, err := s.pathRepo.FindByID(ctx, input.PathID)
	if err != nil {
		return nil, err
	}

	currentPath, err := s.pathRepo.FindByID(ctx, app.AdmissionPathID)
	if err != nil {
		return nil, err
	}

	targetPeriodID := app.PeriodID
	if input.PeriodID != nil && *input.PeriodID > 0 {
		targetPeriodID = *input.PeriodID
	}

	// 3a. Alur langsung saat draft / awaiting_payment
	if app.Status == model.StatusDraft || app.Status == model.StatusAwaitingPayment {
		targetFee := int64(targetPath.RegistrationFee)

		// Reprice invoice bila ada invoice dan fee berbeda
		if app.InvoiceID != nil && s.financeCli != nil {
			var repriceResp struct {
				Success bool `json:"success"`
			}
			err := s.financeCli.Post(ctx, fmt.Sprintf("/internal/invoices/%s/reprice", app.InvoiceID.String()), map[string]interface{}{
				"amount": targetFee,
			}, &repriceResp)
			if err != nil {
				if strings.Contains(err.Error(), "PAYMENT_IN_PROGRESS") || strings.Contains(err.Error(), "409") {
					return nil, ErrPaymentInProgress
				}
				return nil, fmt.Errorf("gagal memperbarui nominal tagihan pendaftaran: %w", err)
			}
		}

		fromValues := map[string]interface{}{
			"period_id": app.PeriodID,
			"path_id":   app.AdmissionPathID,
			"path_name": currentPath.Name,
			"choice_1":  app.Choice1ID,
			"choice_2":  app.Choice2ID,
		}

		toValues := map[string]interface{}{
			"period_id": targetPeriodID,
			"path_id":   targetPath.ID,
			"path_name": targetPath.Name,
			"choice_1":  input.FirstChoiceID,
			"choice_2":  input.SecondChoiceID,
		}

		app.PeriodID = targetPeriodID
		app.AdmissionPathID = targetPath.ID
		app.Choice1ID = input.FirstChoiceID
		app.Choice2ID = input.SecondChoiceID

		if err := s.appRepo.Update(ctx, app); err != nil {
			return nil, err
		}

		changeType := model.ChangeTypeChoicesUpdated
		if currentPath.ID != targetPath.ID {
			changeType = model.ChangeType("path_change")
		}

		// Simpan riwayat immutable
		change := &model.ApplicationChange{
			ApplicationID: app.ID,
			UserID:        userID,
			ChangeType:    changeType,
			Reason:        input.Reason,
			FeeDifference: targetFee - int64(currentPath.RegistrationFee),
			ActorRole:     "calon_mahasiswa",
			Source:        model.ChangeSourceApplicant,
		}
		_ = change.SetFromValues(fromValues)
		_ = change.SetToValues(toValues)
		_ = s.changeRepo.CreateChange(ctx, nil, change)

		return &PathChangeResult{
			AppliedDirectly: true,
			Status:          string(app.Status),
			Message:         "Pilihan jalur dan program studi berhasil langsung diperbarui",
		}, nil
	}

	// 3b. Alur permohonan setelah bayar (form_open .. verified)
	if app.Status == model.StatusFormOpen || app.Status == model.StatusSubmitted || app.Status == model.StatusRevisionRequired || app.Status == model.StatusVerified {
		if app.PathChangeCountAfterPayment >= 1 {
			return nil, ErrPathChangeLimitReached
		}

		pending, _ := s.changeRepo.GetPendingPathChangeRequest(ctx, app.ID)
		if pending != nil {
			return nil, ErrPathChangePending
		}

		reason := strings.TrimSpace(input.Reason)
		if len(reason) < 10 || len(reason) > 500 {
			return nil, ErrReasonRequired
		}

		feeDiff := int64(targetPath.RegistrationFee) - int64(currentPath.RegistrationFee)

		req := &model.PathChangeRequest{
			ApplicationID: app.ID,
			UserID:        userID,
			FromPeriodID:  app.PeriodID,
			FromPathID:    app.AdmissionPathID,
			FromChoice1ID: app.Choice1ID,
			FromChoice2ID: app.Choice2ID,
			ToPeriodID:    targetPeriodID,
			ToPathID:      targetPath.ID,
			ToChoice1ID:   input.FirstChoiceID,
			ToChoice2ID:   input.SecondChoiceID,
			Reason:        reason,
			Status:        model.PathChangePending,
			FeeDifference: feeDiff,
		}

		if err := s.changeRepo.CreatePathChangeRequest(ctx, nil, req); err != nil {
			return nil, err
		}

		// Riwayat: path_change_requested
		change := &model.ApplicationChange{
			ApplicationID:    app.ID,
			UserID:           userID,
			ChangeType:       model.ChangeTypePathChangeRequested,
			Reason:           reason,
			FeeDifference:    feeDiff,
			ActorRole:        "calon_mahasiswa",
			Source:           model.ChangeSourceApplicant,
			RelatedRequestID: &req.ID,
		}
		_ = change.SetFromValues(map[string]interface{}{
			"path_id":   app.AdmissionPathID,
			"path_name": currentPath.Name,
			"choice_1":  app.Choice1ID,
			"choice_2":  app.Choice2ID,
		})
		_ = change.SetToValues(map[string]interface{}{
			"path_id":   targetPath.ID,
			"path_name": targetPath.Name,
			"choice_1":  input.FirstChoiceID,
			"choice_2":  input.SecondChoiceID,
		})
		_ = s.changeRepo.CreateChange(ctx, nil, change)

		// Notifikasi ke admin PMB
		if s.notifCli != nil {
			_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
				"recipient_role": "admin_pmb",
				"template_code":  "path_change_requested",
				"title":          "Permohonan Pindah Jalur Baru",
				"message":        fmt.Sprintf("Pendaftar %s mengajukan pindah jalur dari %s ke %s", app.RegistrationNo, currentPath.Name, targetPath.Name),
				"reference_id":   req.ID.String(),
			}, nil)
		}

		return &PathChangeResult{
			AppliedDirectly: false,
			RequestID:       &req.ID,
			Status:          "pending",
			Message:         "Permohonan pindah jalur berhasil diajukan dan sedang menunggu persetujuan Admin PMB",
		}, nil
	}

	return nil, ErrPathChangeNotAllowed
}

// 4. CancelPendingPathChange: Batalkan permohonan pending milik sendiri
func (s *changeService) CancelPendingPathChange(ctx context.Context, userID uuid.UUID) error {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}

	req, err := s.changeRepo.GetPendingPathChangeRequest(ctx, app.ID)
	if err != nil || req == nil {
		return errors.New("tidak ada permohonan pindah jalur yang berstatus pending")
	}

	req.Status = model.PathChangeCancelled
	if err := s.changeRepo.UpdatePathChangeRequest(ctx, nil, req); err != nil {
		return err
	}

	// Riwayat: path_change_cancelled
	change := &model.ApplicationChange{
		ApplicationID:    app.ID,
		UserID:           userID,
		ChangeType:       model.ChangeTypePathChangeCancelled,
		Reason:           "Dibatalkan oleh pemohon",
		ActorRole:        "calon_mahasiswa",
		Source:           model.ChangeSourceApplicant,
		RelatedRequestID: &req.ID,
	}
	_ = s.changeRepo.CreateChange(ctx, nil, change)

	return nil
}

// 4b. GetMyPendingPathChange: Ambil permohonan pending milik sendiri
func (s *changeService) GetMyPendingPathChange(ctx context.Context, userID uuid.UUID) (*model.PathChangeRequest, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil || app == nil {
		return nil, errors.New("pendaftaran aktif tidak ditemukan")
	}

	req, err := s.changeRepo.GetPendingPathChangeRequest(ctx, app.ID)
	if err != nil || req == nil {
		return nil, errors.New("tidak ada permohonan pindah jalur yang berstatus pending")
	}

	return req, nil
}

// 5. AdminListPathChangeRequests
func (s *changeService) AdminListPathChangeRequests(ctx context.Context, status string, limit, offset int) ([]model.PathChangeRequest, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.changeRepo.ListPathChangeRequests(ctx, status, limit, offset)
}

// 6. AdminApprovePathChangeRequest: Setujui permohonan
func (s *changeService) AdminApprovePathChangeRequest(ctx context.Context, requestID, reviewerID uuid.UUID, note string) error {
	req, err := s.changeRepo.GetPathChangeRequestByID(ctx, requestID)
	if err != nil || req == nil {
		return ErrRequestNotFound
	}

	if req.Status != model.PathChangePending {
		return errors.New("hanya permohonan berstatus pending yang dapat disetujui")
	}

	app, err := s.appRepo.FindByID(ctx, req.ApplicationID)
	if err != nil || app == nil {
		return errors.New("pendaftaran terkait tidak ditemukan")
	}

	targetPath, err := s.pathRepo.FindByID(ctx, req.ToPathID)
	if err != nil {
		return err
	}

	// Buat tagihan penyesuaian jika lebih mahal
	if req.FeeDifference > 0 && s.financeCli != nil {
		var adjResp struct {
			Success bool `json:"success"`
			Data    struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		err := s.financeCli.Post(ctx, "/internal/invoices/adjustment", map[string]interface{}{
			"user_id":        app.UserID.String(),
			"application_id": app.ID.String(),
			"amount":         req.FeeDifference,
			"description":    fmt.Sprintf("Selisih biaya pindah jalur ke %s", targetPath.Name),
		}, &adjResp)
		if err != nil {
			return fmt.Errorf("gagal membuat tagihan selisih penyesuaian: %w", err)
		}
	} else if req.FeeDifference < 0 && s.notifCli != nil {
		// Notifikasi bagian keuangan bila jalur baru lebih murah
		_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
			"recipient_role": "keuangan",
			"template_code":  "path_change_refund_notice",
			"title":          "Informasi Selisih Biaya Pindah Jalur Lebih Murah",
			"message":        fmt.Sprintf("Pendaftar %s pindah ke jalur lebih murah dengan selisih Rp %d", app.RegistrationNo, -req.FeeDifference),
			"reference_id":   req.ID.String(),
		}, nil)
	}

	// Update data aplikasi
	app.PeriodID = req.ToPeriodID
	app.AdmissionPathID = req.ToPathID
	app.Choice1ID = req.ToChoice1ID
	app.Choice2ID = req.ToChoice2ID
	app.PathChangeCountAfterPayment += 1

	// Rekalkulasi dokumen
	if s.reqRepo != nil && s.docRepo != nil {
		requirements, _ := s.reqRepo.FindByPathID(ctx, req.ToPathID)
		uploadedDocs, _ := s.docRepo.FindByApplicationID(ctx, app.ID)

		uploadedMap := make(map[string]bool)
		for _, d := range uploadedDocs {
			if d.Status == "valid" || d.Status == "uploaded" {
				uploadedMap[d.DocumentType] = true
			}
		}

		hasMissingRequiredDoc := false
		for _, reqDoc := range requirements {
			if reqDoc.IsRequired && !uploadedMap[reqDoc.DocumentType] {
				hasMissingRequiredDoc = true
				break
			}
		}

		if hasMissingRequiredDoc {
			app.Status = model.StatusFormOpen
		} else {
			app.Status = model.StatusSubmitted
		}
	}

	if err := s.appRepo.Update(ctx, app); err != nil {
		return err
	}

	// Update status permohonan
	now := time.Now()
	req.Status = model.PathChangeApproved
	req.ReviewedBy = &reviewerID
	req.ReviewNote = note
	req.ReviewedAt = &now
	_ = s.changeRepo.UpdatePathChangeRequest(ctx, nil, req)

	// Catat riwayat: path_change_approved
	change := &model.ApplicationChange{
		ApplicationID:    app.ID,
		UserID:           app.UserID,
		ChangeType:       model.ChangeTypePathChangeApproved,
		Reason:           note,
		FeeDifference:    req.FeeDifference,
		ActorID:          &reviewerID,
		ActorRole:        "admin_pmb",
		Source:           model.ChangeSourceAdmin,
		RelatedRequestID: &req.ID,
	}
	_ = change.SetToValues(map[string]interface{}{
		"path_id":  req.ToPathID,
		"choice_1": req.ToChoice1ID,
		"choice_2": req.ToChoice2ID,
	})
	_ = s.changeRepo.CreateChange(ctx, nil, change)

	// Notifikasi ke calon mahasiswa
	if s.notifCli != nil {
		_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
			"user_id":       app.UserID.String(),
			"template_code": "path_change_approved",
			"title":         "Permohonan Pindah Jalur Disetujui",
			"message":       fmt.Sprintf("Permohonan pindah jalur Anda ke %s telah disetujui oleh panitia PMB", targetPath.Name),
			"reference_id":  req.ID.String(),
		}, nil)
	}

	return nil
}

// 7. AdminRejectPathChangeRequest: Tolak permohonan
func (s *changeService) AdminRejectPathChangeRequest(ctx context.Context, requestID, reviewerID uuid.UUID, note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return errors.New("catatan penolakan (note) wajib diisi")
	}

	req, err := s.changeRepo.GetPathChangeRequestByID(ctx, requestID)
	if err != nil || req == nil {
		return ErrRequestNotFound
	}

	if req.Status != model.PathChangePending {
		return errors.New("hanya permohonan berstatus pending yang dapat ditolak")
	}

	now := time.Now()
	req.Status = model.PathChangeRejected
	req.ReviewedBy = &reviewerID
	req.ReviewNote = note
	req.ReviewedAt = &now
	_ = s.changeRepo.UpdatePathChangeRequest(ctx, nil, req)

	// Catat riwayat: path_change_rejected
	change := &model.ApplicationChange{
		ApplicationID:    req.ApplicationID,
		UserID:           req.UserID,
		ChangeType:       model.ChangeTypePathChangeRejected,
		Reason:           note,
		ActorID:          &reviewerID,
		ActorRole:        "admin_pmb",
		Source:           model.ChangeSourceAdmin,
		RelatedRequestID: &req.ID,
	}
	_ = s.changeRepo.CreateChange(ctx, nil, change)

	// Notifikasi ke calon mahasiswa
	if s.notifCli != nil {
		_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
			"user_id":       req.UserID.String(),
			"template_code": "path_change_rejected",
			"title":         "Permohonan Pindah Jalur Ditolak",
			"message":       fmt.Sprintf("Permohonan pindah jalur Anda belum dapat disetujui: %s", note),
			"reference_id":  req.ID.String(),
		}, nil)
	}

	return nil
}

// 8. AdminMoveApplication: Pindah langsung oleh admin
func (s *changeService) AdminMoveApplication(ctx context.Context, appID, actorID uuid.UUID, input *AdminMoveApplicationInput) error {
	if input.PathID == 0 && input.TargetPathID > 0 {
		input.PathID = input.TargetPathID
	}
	if input.Reason == "" && input.Action != "" {
		input.Reason = fmt.Sprintf("Aksi %s oleh administrator", input.Action)
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" {
		return errors.New("alasan pemindahan oleh admin wajib diisi")
	}

	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil || app == nil {
		return errors.New("pendaftaran tidak ditemukan")
	}

	if input.FirstChoiceID == uuid.Nil {
		input.FirstChoiceID = app.Choice1ID
	}
	if input.SecondChoiceID == nil {
		input.SecondChoiceID = app.Choice2ID
	}

	switch app.Status {
	case model.StatusPassed, model.StatusFailed, model.StatusReregistered, model.StatusEnrolled, model.StatusCancelled, model.StatusWithdrawn:
		return ErrPathChangeNotAllowed
	}

	// Bila status exam_scheduled, panggil Exam Service untuk mengeluarkan peserta dari sesi ujian lama
	if app.Status == model.StatusExamScheduled && s.examCli != nil {
		_ = s.examCli.Post(ctx, "/internal/participants/remove", map[string]interface{}{
			"application_id": app.ID.String(),
		}, nil)
	}

	targetPath, err := s.pathRepo.FindByID(ctx, input.PathID)
	if err != nil {
		return err
	}

	currentPath, err := s.pathRepo.FindByID(ctx, app.AdmissionPathID)
	if err != nil {
		return err
	}

	feeDiff := int64(targetPath.RegistrationFee) - int64(currentPath.RegistrationFee)

	// Jika setelah bayar dan lebih mahal, terbitkan tagihan adjustment
	if app.Status != model.StatusDraft && app.Status != model.StatusAwaitingPayment && feeDiff > 0 && s.financeCli != nil {
		_ = s.financeCli.Post(ctx, "/internal/invoices/adjustment", map[string]interface{}{
			"user_id":        app.UserID.String(),
			"application_id": app.ID.String(),
			"amount":         feeDiff,
			"description":    fmt.Sprintf("Selisih biaya pindah jalur ke %s oleh admin", targetPath.Name),
		}, nil)
	}

	fromValues := map[string]interface{}{
		"period_id": app.PeriodID,
		"path_id":   app.AdmissionPathID,
		"path_name": currentPath.Name,
		"choice_1":  app.Choice1ID,
		"choice_2":  app.Choice2ID,
	}

	toPeriodID := app.PeriodID
	if input.PeriodID != nil && *input.PeriodID > 0 {
		toPeriodID = *input.PeriodID
	}

	toValues := map[string]interface{}{
		"period_id": toPeriodID,
		"path_id":   targetPath.ID,
		"path_name": targetPath.Name,
		"choice_1":  input.FirstChoiceID,
		"choice_2":  input.SecondChoiceID,
	}

	app.PeriodID = toPeriodID
	app.AdmissionPathID = targetPath.ID
	app.Choice1ID = input.FirstChoiceID
	app.Choice2ID = input.SecondChoiceID
	app.PathChangeCountAfterPayment += 1

	if app.Status == model.StatusExamScheduled {
		app.Status = model.StatusVerified
	}

	if err := s.appRepo.Update(ctx, app); err != nil {
		return err
	}

	// Update pending request bila ada
	if pendingReq, err := s.changeRepo.GetPendingPathChangeRequest(ctx, app.ID); err == nil && pendingReq != nil {
		now := time.Now()
		pendingReq.Status = model.PathChangeApproved
		pendingReq.ReviewedBy = &actorID
		pendingReq.ReviewNote = input.Reason
		pendingReq.ReviewedAt = &now
		_ = s.changeRepo.UpdatePathChangeRequest(ctx, nil, pendingReq)
	}

	// Catat riwayat: admin_moved
	change := &model.ApplicationChange{
		ApplicationID: app.ID,
		UserID:        app.UserID,
		ChangeType:    model.ChangeTypeAdminMoved,
		Reason:        input.Reason,
		FeeDifference: feeDiff,
		ActorID:       &actorID,
		ActorRole:     "admin_pmb",
		Source:        model.ChangeSourceAdmin,
	}
	_ = change.SetFromValues(fromValues)
	_ = change.SetToValues(toValues)
	_ = s.changeRepo.CreateChange(ctx, nil, change)

	// Notifikasi ke calon mahasiswa
	if s.notifCli != nil {
		_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
			"user_id":       app.UserID.String(),
			"template_code": "admin_moved",
			"title":         "Pemberitahuan Pindah Jalur oleh Panitia",
			"message":       fmt.Sprintf("Pilihan jalur pendaftaran Anda telah dialihkan ke %s oleh panitia PMB: %s", targetPath.Name, input.Reason),
			"reference_id":  app.ID.String(),
		}, nil)
	}

	return nil
}

// 9. AdminGetApplicationChanges: Riwayat lengkap pendaftar (termasuk riwayat pendaftaran sebelumnya bila ada)
func (s *changeService) AdminGetApplicationChanges(ctx context.Context, appID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error) {
	if limit <= 0 {
		limit = 50
	}

	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil || app == nil {
		return nil, 0, errors.New("pendaftaran tidak ditemukan")
	}

	// Ambil riwayat seluruh pendaftaran milik user ini agar linimasa komprehensif
	changes, total, err := s.changeRepo.GetChangesByUserID(ctx, app.UserID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	for i := range changes {
		enrichChangeMetadata(&changes[i])
		changes[i].RegistrationNo = app.RegistrationNo
	}

	return changes, total, nil
}

// 10. AdminExportPathChangesXLSX: Ekspor spreadsheet Excel
func (s *changeService) AdminExportPathChangesXLSX(ctx context.Context, status string) ([]byte, error) {
	requests, _, err := s.changeRepo.ListPathChangeRequests(ctx, status, 1000, 0)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheetName := "Permohonan Pindah Jalur"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{
		"No", "ID Permohonan", "No. Pendaftaran", "Jalur Asal", "Jalur Tujuan",
		"Alasan", "Selisih Biaya (Rp)", "Status", "Tanggal Pengajuan", "Catatan Review",
	}

	// Header styling navy
	styleHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1A365D"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheetName, cell, h)
		f.SetCellStyle(sheetName, cell, cell, styleHeader)
	}

	for rowIdx, req := range requests {
		row := rowIdx + 2
		regNo := ""
		if req.Application != nil {
			regNo = req.Application.RegistrationNo
		}
		fromPathName := ""
		if req.FromPath != nil {
			fromPathName = req.FromPath.Name
		}
		toPathName := ""
		if req.ToPath != nil {
			toPathName = req.ToPath.Name
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), rowIdx+1)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), req.ID.String())
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), regNo)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), fromPathName)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), toPathName)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), req.Reason)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), req.FeeDifference)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), string(req.Status))
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), req.CreatedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), req.ReviewNote)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// 11. GetDashboardSummary: Kompatibilitas CR-005 dengan field pending_request
func (s *changeService) GetDashboardSummary(ctx context.Context, userID uuid.UUID) (*DashboardSummaryResponse, error) {
	app, _ := s.appRepo.FindActiveByUserID(ctx, userID)
	prog, _ := s.admService.GetProgress(ctx, userID)

	var pendingSummary *PendingRequestSummary

	if app != nil {
		// Cek permohonan pindah jalur pending
		if pendingPath, _ := s.changeRepo.GetPendingPathChangeRequest(ctx, app.ID); pendingPath != nil {
			pendingSummary = &PendingRequestSummary{
				Type:        "path_change",
				StatusLabel: "Permohonan Pindah Jalur Menunggu Persetujuan",
				CreatedAt:   pendingPath.CreatedAt,
			}
		} else if s.withdrawalRepo != nil {
			if pendingWith, _ := s.withdrawalRepo.FindPendingByApplicationID(ctx, app.ID); pendingWith != nil {
				pendingSummary = &PendingRequestSummary{
					Type:        "withdrawal",
					StatusLabel: "Permohonan Pengunduran Diri Menunggu Persetujuan",
					CreatedAt:   pendingWith.RequestedAt,
				}
			}
		}
	}

	return &DashboardSummaryResponse{
		Application:    app,
		Progress:       prog,
		PendingRequest: pendingSummary,
	}, nil
}

// Helper memperkaya label riwayat untuk kemudahan frontend
func enrichChangeMetadata(c *model.ApplicationChange) {
	c.CreatedAtLabel = c.CreatedAt.Format("02 Jan 2006 15:04 WIB")

	switch c.Source {
	case model.ChangeSourceApplicant:
		c.ActorLabel = "Anda"
	case model.ChangeSourceAdmin:
		c.ActorLabel = "Admin PMB"
	default:
		c.ActorLabel = "Sistem"
	}

	switch c.ChangeType {
	case model.ChangeTypeCreated:
		c.TitleLabel = "Pendaftaran Baru Dibuat"
		c.Description = "Formulir pendaftaran mahasiswa baru berhasil dibuat"
	case model.ChangeTypeChoicesUpdated, model.ChangeType("path_change"):
		c.TitleLabel = "Pindah Jalur / Pilihan Program Studi Diperbarui"
		c.Description = "Pilihan jalur pendaftaran dan program studi langsung diperbarui"
	case model.ChangeTypePathChangeRequested:
		c.TitleLabel = "Permohonan Pindah Jalur Diajukan"
		c.Description = "Permohonan perubahan jalur pendaftaran diajukan ke panitia PMB"
	case model.ChangeTypePathChangeApproved:
		c.TitleLabel = "Permohonan Pindah Jalur Disetujui"
		c.Description = "Permohonan perubahan jalur pendaftaran telah disetujui panitia PMB"
	case model.ChangeTypePathChangeRejected:
		c.TitleLabel = "Permohonan Pindah Jalur Ditolak"
		c.Description = "Permohonan perubahan jalur pendaftaran ditolak oleh panitia PMB"
	case model.ChangeTypePathChangeCancelled:
		c.TitleLabel = "Permohonan Pindah Jalur Dibatalkan"
		c.Description = "Permohonan perubahan jalur pendaftaran dibatalkan oleh pemohon"
	case model.ChangeTypeAdminMoved:
		c.TitleLabel = "Pindah Jalur oleh Administrator PMB"
		c.Description = "Pilihan jalur pendaftaran dialihkan secara administratif oleh panitia PMB"
	case model.ChangeTypeCancelled:
		c.TitleLabel = "Pendaftaran Dibatalkan"
		c.Description = "Pendaftaran mahasiswa baru dibatalkan"
	case model.ChangeTypeReapplied:
		c.TitleLabel = "Pendaftaran Ulang Berhasil"
		c.Description = "Pendaftaran baru dibuat setelah pembatalan atau hasil sebelumnya"
	case model.ChangeTypeWithdrawalRequested:
		c.TitleLabel = "Permohonan Pengunduran Diri Diajukan"
		c.Description = "Pengunduran diri resmi diajukan dan sedang ditinjau"
	case model.ChangeTypeWithdrawalApproved:
		c.TitleLabel = "Pengunduran Diri Disetujui"
		c.Description = "Pengunduran diri telah disetujui secara resmi"
	case model.ChangeTypeWithdrawalRejected:
		c.TitleLabel = "Permohonan Pengunduran Diri Ditolak"
		c.Description = "Permohonan pengunduran diri ditolak oleh panitia PMB"
	case model.ChangeTypeWithdrawalCancelled:
		c.TitleLabel = "Permohonan Pengunduran Diri Dibatalkan"
		c.Description = "Permohonan pengunduran diri dibatalkan oleh pemohon"
	default:
		c.TitleLabel = string(c.ChangeType)
		c.Description = "Perubahan data pendaftaran"
	}
}
