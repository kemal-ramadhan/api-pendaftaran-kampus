package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/repository"
)

var (
	ErrQuotaExceeded                 = errors.New("QUOTA_EXCEEDED: jumlah peserta yang direkomendasikan lulus melebihi kuota program studi")
	ErrResultNotPublished            = errors.New("RESULT_NOT_PUBLISHED: pengumuman hasil seleksi belum dipublikasikan oleh panitia")
	ErrRecommendationNotFound        = errors.New("rekomendasi kelulusan tidak ditemukan")
	ErrInvalidRecommendationState    = errors.New("INVALID_STATE: status rekomendasi tidak mengizinkan operasi ini")
	ErrForbiddenProgramAccess        = errors.New("FORBIDDEN: Anda tidak memiliki hak akses ke program studi ini")
	ErrAnnouncementAlreadyPublished  = errors.New("pengumuman untuk periode ini sudah pernah dipublikasikan")
	ErrPendingApprovalsRemaining     = errors.New("masih terdapat usulan rekomendasi program studi yang belum disetujui pimpinan")
	ErrUnverifiedDocumentsRemaining = errors.New("PUBLISH_REQUIRES_VERIFIED_DOCUMENTS: calon mahasiswa yang lulus wajib diverifikasi seluruh berkasnya sebelum pengumuman dipublikasikan")
)

type SummaryParticipantItem struct {
	ApplicationID      uuid.UUID `json:"application_id"`
	RegistrationNumber string    `json:"registration_number"`
	FullName           string    `json:"full_name"`
	StudyProgramCode   string    `json:"study_program_code"`
	FinalScore         float64   `json:"final_score"`
	RankInProgram      int       `json:"rank_in_program"`
	Decision           string    `json:"decision,omitempty"`
}

type SelectionService interface {
	GetProgramRecommendation(ctx context.Context, periodID, pathID uuid.UUID, programCode string, userRoles, userScopes []string) (*model.Recommendation, []SummaryParticipantItem, error)
	SaveProgramRecommendation(ctx context.Context, payload *model.SaveRecommendationPayload, actorUserID uuid.UUID, userRoles, userScopes []string) (*model.Recommendation, error)
	SubmitProgramRecommendation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID, userRoles, userScopes []string) (*model.Recommendation, error)
	ListPendingApprovals(ctx context.Context, periodID uuid.UUID) ([]model.Recommendation, error)
	ApproveRecommendation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID) (*model.Recommendation, error)
	ReturnRecommendation(ctx context.Context, id uuid.UUID, note string, actorUserID uuid.UUID) (*model.Recommendation, error)
	PublishAnnouncement(ctx context.Context, payload *model.PublishAnnouncementPayload, actorUserID uuid.UUID) (*model.Announcement, error)
	GetMyResult(ctx context.Context, userID uuid.UUID) (*model.MyResultResponse, error)
	WithdrawApplication(ctx context.Context, appID uuid.UUID) error
	GetScoreboard(ctx context.Context, periodID uuid.UUID, decisionFilter string, search string, userRoles, userScopes []string) (*model.ScoreboardResponse, error)
}

type selectionService struct {
	cfg             *config.Config
	recRepo         repository.RecommendationRepository
	annRepo         repository.AnnouncementRepository
	resultRepo      repository.ResultRepository
	scoringCli      internalclient.Client
	admissionCli    internalclient.Client
	financeCli      internalclient.Client
	notificationCli internalclient.Client
	identityCli     internalclient.Client
}

func NewSelectionService(
	cfg *config.Config,
	recRepo repository.RecommendationRepository,
	annRepo repository.AnnouncementRepository,
	resultRepo repository.ResultRepository,
	scoringCli internalclient.Client,
	admissionCli internalclient.Client,
	financeCli internalclient.Client,
	notificationCli internalclient.Client,
	identityCli internalclient.Client,
) SelectionService {
	return &selectionService{
		cfg:             cfg,
		recRepo:         recRepo,
		annRepo:         annRepo,
		resultRepo:      resultRepo,
		scoringCli:      scoringCli,
		admissionCli:    admissionCli,
		financeCli:      financeCli,
		notificationCli: notificationCli,
		identityCli:     identityCli,
	}
}

// -------------------------------------------------------------
// Rekomendasi Program Studi
// -------------------------------------------------------------

func (s *selectionService) GetProgramRecommendation(
	ctx context.Context,
	periodID, pathID uuid.UUID,
	programCode string,
	userRoles, userScopes []string,
) (*model.Recommendation, []SummaryParticipantItem, error) {
	if err := s.validateProdiScope(programCode, userRoles, userScopes); err != nil {
		return nil, nil, err
	}

	// 1. Ambil rekomendasi yang tersimpan jika ada
	rec, err := s.recRepo.FindByPeriodAndProgram(ctx, periodID, pathID, programCode)
	if err != nil {
		return nil, nil, err
	}

	decisionMap := make(map[uuid.UUID]model.DecisionType)
	if rec != nil {
		for _, item := range rec.Items {
			decisionMap[item.ApplicationID] = item.Decision
		}
	}

	// 2. Ambil ringkasan nilai pendaftar dari Scoring Service
	summaries, err := s.fetchScoringSummaries(ctx, periodID, programCode)
	if err != nil {
		// Fallback empty
		summaries = nil
	}

	var items []SummaryParticipantItem
	for _, sm := range summaries {
		regNo := sm.ApplicationID.String()[:8]
		fullName := "Calon Mahasiswa"

		appInfo, errApp := s.fetchApplication(ctx, sm.ApplicationID)
		if errApp == nil && appInfo != nil {
			if appInfo.RegistrationNumber != "" {
				regNo = appInfo.RegistrationNumber
			}
			personName, errP := s.fetchPersonName(ctx, appInfo.UserID)
			if errP == nil && personName != "" {
				fullName = personName
			}
		}

		dec := ""
		if d, ok := decisionMap[sm.ApplicationID]; ok {
			dec = string(d)
		}

		items = append(items, SummaryParticipantItem{
			ApplicationID:      sm.ApplicationID,
			RegistrationNumber: regNo,
			FullName:           fullName,
			StudyProgramCode:   sm.StudyProgramCode,
			FinalScore:         sm.FinalScore,
			RankInProgram:      sm.RankInProgram,
			Decision:           dec,
		})
	}

	return rec, items, nil
}

func (s *selectionService) SaveProgramRecommendation(
	ctx context.Context,
	payload *model.SaveRecommendationPayload,
	actorUserID uuid.UUID,
	userRoles, userScopes []string,
) (*model.Recommendation, error) {
	if err := s.validateProdiScope(payload.StudyProgramCode, userRoles, userScopes); err != nil {
		return nil, err
	}

	// Cek rekomendasi existing
	existing, err := s.recRepo.FindByPeriodAndProgram(ctx, payload.PeriodID, payload.PathID, payload.StudyProgramCode)
	if err != nil {
		return nil, err
	}
	if existing != nil && (existing.Status == model.StatusSubmitted || existing.Status == model.StatusApproved) {
		return nil, ErrInvalidRecommendationState
	}

	// Validasi Kuota: hitung pendaftar dengan status 'pass'
	passCount := 0
	for _, it := range payload.Items {
		if it.Decision == model.DecisionPass {
			passCount++
		}
	}

	quotaCapacity, err := s.fetchQuotaCapacity(ctx, payload.PeriodID, payload.PathID, payload.StudyProgramCode)
	if err == nil && quotaCapacity > 0 {
		if passCount > quotaCapacity {
			return nil, ErrQuotaExceeded
		}
	}

	now := time.Now()
	rec := &model.Recommendation{
		PeriodID:         payload.PeriodID,
		PathID:           payload.PathID,
		StudyProgramCode: payload.StudyProgramCode,
		Status:           model.StatusDraft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if existing != nil {
		rec.ID = existing.ID
		if existing.Status == model.StatusReturned {
			rec.Status = model.StatusReturned
		}
	}

	var items []model.RecommendationItem
	for _, it := range payload.Items {
		items = append(items, model.RecommendationItem{
			ApplicationID: it.ApplicationID,
			Decision:      it.Decision,
			FinalScore:    it.FinalScore,
			Rank:          it.Rank,
		})
	}

	if err := s.recRepo.SaveRecommendation(ctx, rec, items); err != nil {
		return nil, err
	}

	return s.recRepo.FindByID(ctx, rec.ID)
}

func (s *selectionService) SubmitProgramRecommendation(
	ctx context.Context,
	id uuid.UUID,
	actorUserID uuid.UUID,
	userRoles, userScopes []string,
) (*model.Recommendation, error) {
	rec, err := s.recRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrRecommendationNotFound
	}
	if err := s.validateProdiScope(rec.StudyProgramCode, userRoles, userScopes); err != nil {
		return nil, err
	}
	if rec.Status != model.StatusDraft && rec.Status != model.StatusReturned {
		return nil, ErrInvalidRecommendationState
	}

	// Validasi kuota kembali sebelum submit
	passCount := 0
	for _, it := range rec.Items {
		if it.Decision == model.DecisionPass {
			passCount++
		}
	}
	quotaCapacity, _ := s.fetchQuotaCapacity(ctx, rec.PeriodID, rec.PathID, rec.StudyProgramCode)
	if quotaCapacity > 0 && passCount > quotaCapacity {
		return nil, ErrQuotaExceeded
	}

	// 1. Kunci nilai di Scoring Service
	_ = s.lockScoring(ctx, rec.PeriodID, rec.PathID, rec.StudyProgramCode, true, &actorUserID)

	// 2. Ubah status rekomendasi menjadi 'submitted'
	if err := s.recRepo.UpdateStatus(ctx, id, model.StatusSubmitted, &actorUserID, ""); err != nil {
		return nil, err
	}

	return s.recRepo.FindByID(ctx, id)
}

// -------------------------------------------------------------
// Persetujuan Pimpinan
// -------------------------------------------------------------

func (s *selectionService) ListPendingApprovals(ctx context.Context, periodID uuid.UUID) ([]model.Recommendation, error) {
	status := model.StatusSubmitted
	return s.recRepo.ListByPeriod(ctx, periodID, &status)
}

func (s *selectionService) ApproveRecommendation(ctx context.Context, id uuid.UUID, actorUserID uuid.UUID) (*model.Recommendation, error) {
	rec, err := s.recRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrRecommendationNotFound
	}
	if rec.Status != model.StatusSubmitted {
		return nil, ErrInvalidRecommendationState
	}

	if err := s.recRepo.UpdateStatus(ctx, id, model.StatusApproved, &actorUserID, ""); err != nil {
		return nil, err
	}
	return s.recRepo.FindByID(ctx, id)
}

func (s *selectionService) ReturnRecommendation(ctx context.Context, id uuid.UUID, note string, actorUserID uuid.UUID) (*model.Recommendation, error) {
	rec, err := s.recRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrRecommendationNotFound
	}
	if rec.Status != model.StatusSubmitted {
		return nil, ErrInvalidRecommendationState
	}

	// Buka kunci nilai di Scoring Service agar prodi dapat menyesuaikan
	_ = s.lockScoring(ctx, rec.PeriodID, rec.PathID, rec.StudyProgramCode, false, nil)

	if err := s.recRepo.UpdateStatus(ctx, id, model.StatusReturned, &actorUserID, note); err != nil {
		return nil, err
	}
	return s.recRepo.FindByID(ctx, id)
}

// -------------------------------------------------------------
// Publikasi Pengumuman & Dampak Antarlayanan
// -------------------------------------------------------------

func (s *selectionService) PublishAnnouncement(
	ctx context.Context,
	payload *model.PublishAnnouncementPayload,
	actorUserID uuid.UUID,
) (*model.Announcement, error) {
	now := time.Now()

	// 1. Ambil seluruh rekomendasi yang berstatus 'approved' pada periode ini
	approvedStatus := model.StatusApproved
	recs, err := s.recRepo.ListByPeriod(ctx, payload.PeriodID, &approvedStatus)
	if err != nil {
		return nil, err
	}

	// 2. Validasi larangan publikasi bila berkas belum terverifikasi (PRD 2.4 & K11)
	for _, r := range recs {
		for _, it := range r.Items {
			if it.Decision == model.DecisionPass {
				appInfo, errApp := s.fetchApplication(ctx, it.ApplicationID)
				if errApp == nil && appInfo != nil {
					if !appInfo.IsDocumentVerified() {
						return nil, ErrUnverifiedDocumentsRemaining
					}
				}
			}
		}
	}

	decreeNum := payload.DecreeNumber
	if decreeNum == "" {
		decreeNum = fmt.Sprintf("SK/%d/REK/PMB", now.Year())
	}
	decreeDate := payload.DecreeDate
	if decreeDate == nil {
		decreeDate = &now
	}

	ann := &model.Announcement{
		PeriodID:     payload.PeriodID,
		Title:        payload.Title,
		Content:      payload.Content,
		DecreeNumber: decreeNum,
		DecreeDate:   decreeDate,
		PublishedAt:  now,
		PublishedBy:  actorUserID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Simpan pengumuman
	if err := s.annRepo.Create(ctx, ann); err != nil {
		return nil, err
	}

	var results []model.Result
	for _, r := range recs {
		for _, it := range r.Items {
			appInfo, _ := s.fetchApplication(ctx, it.ApplicationID)
			userID := uuid.Nil
			if appInfo != nil {
				userID = appInfo.GetUserID()
			}

			acceptedProg := ""
			if it.Decision == model.DecisionPass {
				acceptedProg = r.StudyProgramCode
			}

			results = append(results, model.Result{
				ApplicationID:   it.ApplicationID,
				UserID:          userID,
				Decision:        it.Decision,
				AcceptedProgram: acceptedProg,
				AnnouncedAt:     now,
				CreatedAt:       now,
				UpdatedAt:       now,
			})

			// Mutasi status pendaftaran di Admission Service
			admStatus := "passed"
			if it.Decision != model.DecisionPass {
				admStatus = "failed"
			}
			_ = s.updateAdmissionStatus(ctx, it.ApplicationID, admStatus)

			// Terbitkan tagihan daftar ulang ke Finance Service untuk peserta yang lulus
			if it.Decision == model.DecisionPass {
				_ = s.createReregistrationInvoice(ctx, it.ApplicationID, userID, 5000000)
			}

			// Kirim notifikasi pengumuman ke Notification Service
			_ = s.sendAnnouncementNotification(ctx, userID, string(it.Decision), acceptedProg)
		}
	}

	// 3. Batch simpan results ke database
	if len(results) > 0 {
		_ = s.resultRepo.BatchCreate(ctx, results)
	}

	return ann, nil
}

func (s *selectionService) GetMyResult(ctx context.Context, userID uuid.UUID) (*model.MyResultResponse, error) {
	res, err := s.resultRepo.FindByUserID(ctx, userID)
	if err != nil || res == nil {
		return nil, ErrResultNotPublished
	}

	// Periksa apakah pengumuman periode telah terbit
	appInfo, _ := s.fetchApplication(ctx, res.ApplicationID)
	regNo := res.ApplicationID.String()[:8]
	fullName := "Calon Mahasiswa"
	if appInfo != nil {
		if rNo := appInfo.GetRegistrationNumber(); rNo != "" {
			regNo = rNo
		}
		personName, errP := s.fetchPersonName(ctx, appInfo.GetUserID())
		if errP == nil && personName != "" {
			fullName = personName
		}
	}

	resp := &model.MyResultResponse{
		ApplicationID:      res.ApplicationID,
		RegistrationNumber: regNo,
		FullName:           fullName,
		Decision:           res.Decision,
		AcceptedProgram:    res.AcceptedProgram,
		AnnouncedAt:        res.AnnouncedAt,
	}

	// Jika lulus, cari rincian tagihan daftar ulang dari Finance Service
	if res.Decision == model.DecisionPass {
		inv, errInv := s.fetchInvoiceByApplication(ctx, res.ApplicationID)
		if errInv == nil && inv != nil {
			resp.InvoiceID = &inv.ID
			resp.InvoiceNumber = inv.GetNumber()
			resp.Amount = inv.Amount
			resp.PaymentStatus = inv.Status
		}
	}

	return resp, nil
}

// -------------------------------------------------------------
// Inter-Service Helpers
// -------------------------------------------------------------

type scoringSummaryItem struct {
	ApplicationID    uuid.UUID `json:"application_id"`
	StudyProgramCode string    `json:"study_program_code"`
	FinalScore       float64   `json:"final_score"`
	RankInProgram    int       `json:"rank_in_program"`
}

type admissionChoice struct {
	ChoiceNumber     int    `json:"choice_number"`
	StudyProgramCode string `json:"study_program_code"`
	StudyProgramName string `json:"study_program_name"`
}

type admissionAppResponse struct {
	ID                 uuid.UUID             `json:"id"`
	PeriodID           interface{}           `json:"period_id"`
	PathID             interface{}           `json:"path_id"`
	RegistrationNumber string                `json:"registration_number"`
	RegistrationNo     string                `json:"registration_no"`
	UserID             uuid.UUID             `json:"user_id"`
	FullName           string                `json:"full_name"`
	Status             string                `json:"status"`
	DocumentStatus     string                `json:"document_status"`
	VerifiedAt         *time.Time            `json:"verified_at"`
	Choices            []admissionChoice     `json:"choices"`
	Data               *admissionAppResponse `json:"data,omitempty"`
}

func (a *admissionAppResponse) GetStatus() string {
	if a == nil {
		return ""
	}
	if a.Data != nil && a.Data.Status != "" {
		return a.Data.Status
	}
	return a.Status
}

func (a *admissionAppResponse) GetUserID() uuid.UUID {
	if a == nil {
		return uuid.Nil
	}
	if a.Data != nil && a.Data.UserID != uuid.Nil {
		return a.Data.UserID
	}
	return a.UserID
}

func (a *admissionAppResponse) GetRegistrationNumber() string {
	if a == nil {
		return ""
	}
	if a.Data != nil {
		if a.Data.RegistrationNo != "" {
			return a.Data.RegistrationNo
		}
		if a.Data.RegistrationNumber != "" {
			return a.Data.RegistrationNumber
		}
	}
	if a.RegistrationNo != "" {
		return a.RegistrationNo
	}
	return a.RegistrationNumber
}

func (a *admissionAppResponse) GetPathID() uuid.UUID {
	if a == nil {
		return uuid.Nil
	}
	if a.PathID != nil {
		switch v := a.PathID.(type) {
		case uuid.UUID:
			return v
		case string:
			if u, err := uuid.Parse(v); err == nil {
				return u
			}
		}
	}
	if a.Data != nil {
		return a.Data.GetPathID()
	}
	return uuid.Nil
}

func (a *admissionAppResponse) IsDocumentVerified() bool {
	if a == nil {
		return false
	}
	st := a.GetStatus()
	switch st {
	case "verified", "exam_scheduled", "interview_scheduled", "direct_decision", "passed", "reregistered", "enrolled":
		return true
	}
	if a.DocumentStatus == "verified" || a.DocumentStatus == "all_verified" {
		return true
	}
	if a.Data != nil && (a.Data.DocumentStatus == "verified" || a.Data.DocumentStatus == "all_verified") {
		return true
	}
	if a.VerifiedAt != nil || (a.Data != nil && a.Data.VerifiedAt != nil) {
		return true
	}
	return false
}

type internalPersonDetailResponse struct {
	Success bool `json:"success"`
	Data    struct {
		FullName string `json:"full_name"`
		EducationHistories []struct {
			ReportAverage *float64 `json:"report_average"`
		} `json:"education_histories"`
	} `json:"data"`
}

type financeInvoiceResponse struct {
	ID            uuid.UUID `json:"id"`
	Number        string    `json:"number"`
	InvoiceNumber string    `json:"invoice_number"`
	FeeType       string    `json:"fee_type"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
}

func (f *financeInvoiceResponse) GetNumber() string {
	if f == nil {
		return ""
	}
	if f.InvoiceNumber != "" {
		return f.InvoiceNumber
	}
	return f.Number
}

func (s *selectionService) fetchScoringSummaries(ctx context.Context, periodID uuid.UUID, programCode string) ([]scoringSummaryItem, error) {
	if s.scoringCli == nil {
		return nil, nil
	}
	var summaries []scoringSummaryItem
	err := s.scoringCli.Get(ctx, "/internal/summaries", map[string]string{
		"period_id": periodID.String(),
		"program":   programCode,
	}, &summaries)
	return summaries, err
}

func (s *selectionService) lockScoring(ctx context.Context, periodID, pathID uuid.UUID, programCode string, isLocked bool, actorUserID *uuid.UUID) error {
	if s.scoringCli == nil {
		return nil
	}
	payload := map[string]interface{}{
		"period_id":          periodID,
		"path_id":            pathID,
		"study_program_code": programCode,
		"is_locked":          isLocked,
		"locked_by":          actorUserID,
	}
	return s.scoringCli.Post(ctx, "/internal/locks", payload, nil)
}

func (s *selectionService) fetchQuotaCapacity(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (int, error) {
	if s.admissionCli == nil {
		return 5, nil // Default kuota development
	}
	var data struct {
		Capacity int `json:"capacity"`
	}
	err := s.admissionCli.Get(ctx, "/internal/quotas", map[string]string{
		"study_program_code": programCode,
	}, &data)
	if err != nil || data.Capacity <= 0 {
		return 5, nil
	}
	return data.Capacity, nil
}

func (s *selectionService) fetchApplication(ctx context.Context, appID uuid.UUID) (*admissionAppResponse, error) {
	if s.admissionCli == nil {
		return &admissionAppResponse{
			ID:                 appID,
			RegistrationNumber: "REG-APP-001",
			UserID:             uuid.New(),
			Status:             "verified",
			DocumentStatus:     "verified",
		}, nil
	}
	var appData admissionAppResponse
	if err := s.admissionCli.Get(ctx, fmt.Sprintf("/internal/applications/%s", appID), nil, &appData); err != nil {
		return nil, err
	}
	return &appData, nil
}

func (s *selectionService) updateAdmissionStatus(ctx context.Context, appID uuid.UUID, status string) error {
	if s.admissionCli == nil {
		return nil
	}
	return s.admissionCli.Post(ctx, fmt.Sprintf("/internal/applications/%s/status", appID), map[string]string{
		"status": status,
		"notes":  "Status dimutasi berdasarkan hasil pengumuman resmi seleksi",
	}, nil)
}

func (s *selectionService) createReregistrationInvoice(ctx context.Context, appID, userID uuid.UUID, amount int64) error {
	if s.financeCli == nil {
		return nil
	}
	return s.financeCli.Post(ctx, "/internal/invoices", map[string]interface{}{
		"application_id": appID.String(),
		"user_id":        userID.String(),
		"amount":         amount,
		"type":           "reregistration",
		"fee_type":       "reregistration",
		"description":    "Biaya Pendaftaran Ulang Mahasiswa Baru",
		"due_date_days":  14,
	}, nil)
}

func (s *selectionService) fetchInvoiceByApplication(ctx context.Context, appID uuid.UUID) (*financeInvoiceResponse, error) {
	if s.financeCli == nil {
		return nil, nil
	}
	var single financeInvoiceResponse
	err := s.financeCli.Get(ctx, "/internal/invoices", map[string]string{
		"application_id": appID.String(),
		"fee_type":       "reregistration",
	}, &single)
	if err == nil && single.ID != uuid.Nil {
		return &single, nil
	}
	var invoices []financeInvoiceResponse
	errList := s.financeCli.Get(ctx, "/internal/invoices", map[string]string{
		"application_id": appID.String(),
	}, &invoices)
	if errList == nil && len(invoices) > 0 {
		for i := range invoices {
			if invoices[i].FeeType == "reregistration" {
				return &invoices[i], nil
			}
		}
		return &invoices[0], nil
	}
	return nil, err
}

func (s *selectionService) sendAnnouncementNotification(ctx context.Context, userID uuid.UUID, decision, programCode string) error {
	if s.notificationCli == nil {
		return nil
	}
	return s.notificationCli.Post(ctx, "/internal/notifications", map[string]interface{}{
		"user_id":  userID,
		"channel":  "email",
		"template": "announcement_published",
		"data": map[string]interface{}{
			"decision":     decision,
			"program_code": programCode,
		},
	}, nil)
}

func (s *selectionService) fetchPersonName(ctx context.Context, userID uuid.UUID) (string, error) {
	if s.identityCli == nil {
		return "", nil
	}
	var data struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	}
	if err := s.identityCli.Get(ctx, fmt.Sprintf("/internal/persons/%s", userID), nil, &data); err != nil {
		return "", err
	}
	if data.FullName != "" {
		return data.FullName, nil
	}
	return data.Name, nil
}

func (s *selectionService) validateProdiScope(programCode string, userRoles, userScopes []string) error {
	isProdiOnly := hasRole(userRoles, "prodi") && !hasAnyRole(userRoles, "admin_pmb", "superadmin", "pimpinan")
	if isProdiOnly {
		allowed := false
		for _, sc := range userScopes {
			if sc == programCode {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrForbiddenProgramAccess
		}
	}
	return nil
}

func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func hasAnyRole(roles []string, targets ...string) bool {
	for _, r := range roles {
		for _, t := range targets {
			if r == t {
				return true
			}
		}
	}
	return false
}

func (s *selectionService) WithdrawApplication(ctx context.Context, appID uuid.UUID) error {
	res, err := s.resultRepo.WithdrawApplication(ctx, appID)
	if err != nil {
		return err
	}
	if res != nil && res.AcceptedProgram != "" {
		_ = s.sendSeatReleasedNotification(ctx, appID, res.AcceptedProgram)
	}
	return nil
}

func (s *selectionService) sendSeatReleasedNotification(ctx context.Context, appID uuid.UUID, programCode string) error {
	if s.notificationCli == nil {
		return nil
	}
	return s.notificationCli.Post(ctx, "/internal/notifications", map[string]interface{}{
		"template": "seat_released",
		"data": map[string]interface{}{
			"application_id": appID.String(),
			"program_code":   programCode,
			"message":        fmt.Sprintf("Kursi kosong tersedia di program studi %s karena pengunduran diri pendaftar", programCode),
		},
	}, nil)
}

func (s *selectionService) GetScoreboard(
	ctx context.Context,
	periodID uuid.UUID,
	decisionFilter string,
	search string,
	userRoles, userScopes []string,
) (*model.ScoreboardResponse, error) {
	passingThreshold := 70.0 // Default K14

	// 1. Ambil rangkuman nilai dari scoring service
	summaries, err := s.fetchScoringSummaries(ctx, periodID, "")
	if err != nil && s.scoringCli != nil {
		// Log error jika ada masalah tapi lanjutkan dengan data yang tersedia
	}

	var allItems []model.ScoreboardItem

	counts := model.ScoreboardCounts{
		All:     0,
		Passed:  0,
		Plenary: 0,
		Failed:  0,
	}

	prodiPassCount := make(map[string]int)

	for _, sum := range summaries {
		// Dapatkan data pendaftaran dari admission
		appInfo, errApp := s.fetchApplication(ctx, sum.ApplicationID)
		fullName := "Calon Mahasiswa"
		regNumber := "REG-00000"
		progCode := sum.StudyProgramCode
		progName := sum.StudyProgramCode
		var choice2Code string

		if errApp == nil && appInfo != nil {
			if appInfo.RegistrationNumber != "" {
				regNumber = appInfo.RegistrationNumber
			}
			if appInfo.FullName != "" {
				fullName = appInfo.FullName
			}
			for _, ch := range appInfo.Choices {
				if ch.ChoiceNumber == 1 && ch.StudyProgramName != "" {
					progName = ch.StudyProgramName
				}
				if ch.ChoiceNumber == 2 {
					choice2Code = ch.StudyProgramCode
				}
			}
		}

		cbtScore := sum.FinalScore
		reportAvg := 0.0
		interviewScore := 0.0

		// Jika ada identity service, coba ambil report_average dari education history
		if s.identityCli != nil && appInfo != nil && appInfo.UserID != uuid.Nil {
			var pResp internalPersonDetailResponse
			if pErr := s.identityCli.Get(ctx, fmt.Sprintf("/internal/persons/%s", appInfo.UserID), nil, &pResp); pErr == nil {
				for _, edu := range pResp.Data.EducationHistories {
					if edu.ReportAverage != nil && *edu.ReportAverage > 0 {
						reportAvg = *edu.ReportAverage
						break
					}
				}
			}
		}

		// Keputusan awal K14 & aturan pilihan kedua K11/PRD 2.4
		decision := model.DecisionPlenary
		decisionLabel := "Evaluasi Pleno"
		isSecondChoice := false

		pathID := uuid.Nil
		if appInfo != nil {
			pathID = appInfo.GetPathID()
		}

		if sum.FinalScore >= passingThreshold {
			quotaCap, _ := s.fetchQuotaCapacity(ctx, periodID, pathID, progCode)
			currentPass := prodiPassCount[progCode]

			if currentPass < quotaCap {
				decision = model.DecisionPass
				decisionLabel = "Lulus Seleksi"
				prodiPassCount[progCode]++
			} else if choice2Code != "" {
				choice2Cap, _ := s.fetchQuotaCapacity(ctx, periodID, pathID, choice2Code)
				currentPass2 := prodiPassCount[choice2Code]
				if currentPass2 < choice2Cap {
					decision = model.DecisionPass
					decisionLabel = "Lulus Seleksi (Pilihan Kedua)"
					isSecondChoice = true
					progCode = choice2Code
					prodiPassCount[choice2Code]++
				} else {
					decision = model.DecisionPlenary
					decisionLabel = "Evaluasi Pleno"
				}
			} else {
				decision = model.DecisionPlenary
				decisionLabel = "Evaluasi Pleno"
			}
		} else {
			decision = model.DecisionPlenary
			decisionLabel = "Evaluasi Pleno"
		}

		// Update counts
		counts.All++
		if decision == model.DecisionPass {
			counts.Passed++
		} else if decision == model.DecisionPlenary {
			counts.Plenary++
		} else {
			counts.Failed++
		}

		item := model.ScoreboardItem{
			ApplicationID:     sum.ApplicationID,
			ParticipantNumber: regNumber,
			FullName:          fullName,
			StudyProgramCode:  progCode,
			StudyProgramName:  progName,
			CBTScore:          cbtScore,
			ReportAverage:     reportAvg,
			InterviewScore:    interviewScore,
			FinalScore:        sum.FinalScore,
			Decision:          decision,
			DecisionLabel:     decisionLabel,
			IsSecondChoice:    isSecondChoice,
		}

		// Filter pencarian
		if search != "" {
			qLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(item.FullName), qLower) &&
				!strings.Contains(strings.ToLower(item.ParticipantNumber), qLower) &&
				!strings.Contains(strings.ToLower(item.StudyProgramCode), qLower) {
				continue
			}
		}

		// Filter keputusan
		if decisionFilter != "" && decisionFilter != "all" {
			if strings.ToLower(decisionFilter) == "passed" && decision != model.DecisionPass {
				continue
			}
			if strings.ToLower(decisionFilter) == "plenary" && decision != model.DecisionPlenary {
				continue
			}
			if strings.ToLower(decisionFilter) == "failed" && decision != model.DecisionFail {
				continue
			}
		}

		allItems = append(allItems, item)
	}

	return &model.ScoreboardResponse{
		PeriodID:         periodID,
		PassingThreshold: passingThreshold,
		Counts:           counts,
		Items:            allItems,
	}, nil
}

