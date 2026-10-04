package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/pdf"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/xuri/excelize/v2"
)

var (
	ErrWithdrawalNotAllowed        = errors.New("WITHDRAWAL_NOT_ALLOWED: pengunduran diri tidak dapat dilakukan pada tahap ini")
	ErrWithdrawalPending           = errors.New("WITHDRAWAL_PENDING: permohonan pengunduran diri sedang dalam proses (pending)")
	ErrPasswordConfirmationFailed  = errors.New("PASSWORD_CONFIRMATION_FAILED: kata sandi konfirmasi salah")
	ErrWithdrawalStatementRequired = errors.New("pernyataan persetujuan pengunduran diri wajib dicentang")
	ErrWithdrawalNotFound          = errors.New("data pengunduran diri tidak ditemukan")
)

type WithdrawalEligibilityResponse struct {
	Allowed          bool                       `json:"allowed"`
	RequiresApproval bool                       `json:"requires_approval"`
	BlockReasonLabel string                     `json:"block_reason_label,omitempty"`
	Consequences     []string                   `json:"consequences"`
	ReasonCategories []model.ReasonCategoryItem `json:"reason_categories"`
}

type WithdrawalRequestInput struct {
	ReasonCategory    string `json:"reason_category"`
	ReasonText        string `json:"reason_text"`
	Reason            string `json:"reason,omitempty"`
	AgreeStatement    bool   `json:"agree_statement"`
	Password          string `json:"password" binding:"required"`
	BankName          string `json:"bank_name,omitempty"`
	BankAccountNumber string `json:"bank_account_number,omitempty"`
	BankAccountHolder string `json:"bank_account_holder,omitempty"`
}

type AdminWithdrawInput struct {
	ReasonCategory string `json:"reason_category"`
	ReasonText     string `json:"reason_text"`
	Reason         string `json:"reason,omitempty"`
	Note           string `json:"note,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type WithdrawalResultResponse struct {
	Withdrawal       *model.Withdrawal `json:"withdrawal"`
	Message          string            `json:"message"`
	VerificationCode string            `json:"verification_code,omitempty"`
	ID               uuid.UUID         `json:"id,omitempty"`
	Status           string            `json:"status,omitempty"`
}

type VerifyWithdrawalResponse struct {
	Valid              bool   `json:"valid"`
	LetterNumber       string `json:"letter_number"`
	FullNameMasked     string `json:"full_name_masked"`
	RegistrationNumber string `json:"registration_number"`
	ProgramName        string `json:"program_name"`
	PathName           string `json:"path_name"`
	EffectiveDate      string `json:"effective_date"`
	ReasonCategory     string `json:"reason_category"`
	Status             string `json:"status"`
}

type WithdrawalService interface {
	GetWithdrawalEligibility(ctx context.Context, userID uuid.UUID) (*WithdrawalEligibilityResponse, error)
	RequestWithdrawal(ctx context.Context, userID uuid.UUID, input *WithdrawalRequestInput) (*WithdrawalResultResponse, error)
	GetMyWithdrawal(ctx context.Context, userID uuid.UUID) (*model.Withdrawal, error)
	CancelPendingWithdrawal(ctx context.Context, userID uuid.UUID) error
	GetWithdrawalLetterPDF(ctx context.Context, userID uuid.UUID) ([]byte, string, error)
	VerifyWithdrawalPublic(ctx context.Context, code string) (*VerifyWithdrawalResponse, error)

	// Admin
	AdminListWithdrawals(ctx context.Context, status string, page, limit int) ([]model.Withdrawal, int64, error)
	AdminApproveWithdrawal(ctx context.Context, withdrawalID, reviewerID uuid.UUID, note string) error
	AdminRejectWithdrawal(ctx context.Context, withdrawalID, reviewerID uuid.UUID, note string) error
	AdminWithdrawApplication(ctx context.Context, appID, actorID uuid.UUID, input *AdminWithdrawInput, attachmentKey string) (*model.Withdrawal, error)
	AdminGetWithdrawalStats(ctx context.Context) (map[string]interface{}, error)
	AdminExportWithdrawalsXLSX(ctx context.Context, status string) ([]byte, error)
}

type withdrawalService struct {
	withdrawalRepo repository.WithdrawalRepository
	appRepo        repository.ApplicationRepository
	changeRepo     repository.ChangeRepository
	pathRepo       repository.AdmissionPathRepository
	authCli        internalclient.Client
	financeCli     internalclient.Client
	examCli        internalclient.Client
	selectionCli   internalclient.Client
	studentCli     internalclient.Client
	nimCli         internalclient.Client
	identityCli    internalclient.Client
	notifCli       internalclient.Client
}

func NewWithdrawalService(
	withdrawalRepo repository.WithdrawalRepository,
	appRepo repository.ApplicationRepository,
	changeRepo repository.ChangeRepository,
	pathRepo repository.AdmissionPathRepository,
	authCli internalclient.Client,
	financeCli internalclient.Client,
	examCli internalclient.Client,
	selectionCli internalclient.Client,
	studentCli internalclient.Client,
	nimCli internalclient.Client,
	identityCli internalclient.Client,
	notifCli internalclient.Client,
) WithdrawalService {
	return &withdrawalService{
		withdrawalRepo: withdrawalRepo,
		appRepo:        appRepo,
		changeRepo:     changeRepo,
		pathRepo:       pathRepo,
		authCli:        authCli,
		financeCli:     financeCli,
		examCli:        examCli,
		selectionCli:   selectionCli,
		studentCli:     studentCli,
		nimCli:         nimCli,
		identityCli:    identityCli,
		notifCli:       notifCli,
	}
}

func defaultConsequences() []string {
	return []string{
		"Seluruh hak Anda sebagai calon mahasiswa pada pendaftaran ini akan gugur.",
		"Tagihan pendaftaran dan biaya pendidikan yang belum dibayar akan dibatalkan.",
		"Biaya yang telah dibayarkan tidak dapat dikembalikan secara otomatis oleh sistem (akan ditinjau manual oleh bagian keuangan).",
		"Jadwal ujian, hasil seleksi, dan status daftar ulang Anda akan dibatalkan.",
		"NIM yang telah diterbitkan akan dinyatakan tidak berlaku (voided) dan tidak dapat digunakan kembali.",
	}
}

func (s *withdrawalService) GetWithdrawalEligibility(ctx context.Context, userID uuid.UUID) (*WithdrawalEligibilityResponse, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return &WithdrawalEligibilityResponse{
			Allowed:          false,
			BlockReasonLabel: "Pendaftaran aktif tidak ditemukan.",
			Consequences:     defaultConsequences(),
			ReasonCategories: model.DefaultReasonCategories,
		}, nil
	}

	// 1. Status sebelum bayar: draft atau awaiting_payment
	if app.Status == model.StatusDraft || app.Status == model.StatusAwaitingPayment {
		return &WithdrawalEligibilityResponse{
			Allowed:          false,
			BlockReasonLabel: "Pengunduran diri tidak tersedia sebelum melakukan pembayaran. Gunakan menu Batalkan Pendaftaran.",
			Consequences:     defaultConsequences(),
			ReasonCategories: model.DefaultReasonCategories,
		}, nil
	}

	// 2. Status akhir: cancelled, failed, withdrawn
	if app.Status == model.StatusCancelled || app.Status == model.StatusFailed || app.Status == model.StatusWithdrawn {
		return &WithdrawalEligibilityResponse{
			Allowed:          false,
			BlockReasonLabel: "Pendaftaran ini sudah tidak aktif atau telah dibatalkan/mengundurkan diri.",
			Consequences:     defaultConsequences(),
			ReasonCategories: model.DefaultReasonCategories,
		}, nil
	}

	// 3. Cek apakah mahasiswa sudah berstatus active di Student Service
	if s.studentCli != nil {
		var studResp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := s.studentCli.Get(ctx, fmt.Sprintf("/internal/students/by-application/%s", app.ID), nil, &studResp); err == nil {
			if studResp.Data.Status == "active" {
				return &WithdrawalEligibilityResponse{
					Allowed:          false,
					BlockReasonLabel: "Pengunduran diri tidak tersedia di sistem PMB karena Anda telah berstatus mahasiswa aktif. Silakan hubungi bagian administrasi akademik kampus.",
					Consequences:     defaultConsequences(),
					ReasonCategories: model.DefaultReasonCategories,
				}, nil
			}
		}
	}

	// 4. Cek apakah ada permohonan pending
	pending, err := s.withdrawalRepo.FindPendingByApplicationID(ctx, app.ID)
	if err == nil && pending != nil {
		return &WithdrawalEligibilityResponse{
			Allowed:          false,
			BlockReasonLabel: "Anda masih memiliki permohonan pengunduran diri yang sedang diproses (pending).",
			Consequences:     defaultConsequences(),
			ReasonCategories: model.DefaultReasonCategories,
		}, nil
	}

	// 5. Tentukan apakah memerlukan persetujuan admin
	// passed, reregistered, enrolled -> butuh persetujuan admin
	requiresApproval := false
	if app.Status == model.StatusPassed ||
		app.Status == model.StatusReregistered ||
		app.Status == model.StatusEnrolled {
		requiresApproval = true
	}

	return &WithdrawalEligibilityResponse{
		Allowed:          true,
		RequiresApproval: requiresApproval,
		Consequences:     defaultConsequences(),
		ReasonCategories: model.DefaultReasonCategories,
	}, nil
}

func (s *withdrawalService) RequestWithdrawal(ctx context.Context, userID uuid.UUID, input *WithdrawalRequestInput) (*WithdrawalResultResponse, error) {
	if strings.TrimSpace(input.ReasonText) == "" && strings.TrimSpace(input.Reason) != "" {
		input.ReasonText = strings.TrimSpace(input.Reason)
	}
	if strings.TrimSpace(input.ReasonCategory) == "" {
		input.ReasonCategory = "other"
	}
	input.AgreeStatement = true

	cleanReason := strings.TrimSpace(input.ReasonText)
	if len(cleanReason) < 5 {
		cleanReason = "Pengunduran diri calon mahasiswa"
	}

	elig, err := s.GetWithdrawalEligibility(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !elig.Allowed {
		return nil, fmt.Errorf("%w: %s", ErrWithdrawalNotAllowed, elig.BlockReasonLabel)
	}

	// 1. Verifikasi Password ke Auth Service
	if s.authCli != nil {
		var verifyResp struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		errCall := s.authCli.Post(ctx, fmt.Sprintf("/internal/users/%s/verify-password", userID), map[string]string{
			"password": input.Password,
		}, &verifyResp)
		if errCall != nil {
			var clientErr *internalclient.Error
			if errors.As(errCall, &clientErr) {
				if clientErr.StatusCode == 429 {
					return nil, fmt.Errorf("akun terkunci sementara karena terlalu banyak percobaan gagal")
				}
				if clientErr.StatusCode == 400 || clientErr.ErrorCode == "PASSWORD_CONFIRMATION_FAILED" {
					return nil, ErrPasswordConfirmationFailed
				}
			}
			return nil, ErrPasswordConfirmationFailed
		}
	}

	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil || app == nil {
		return nil, fmt.Errorf("pendaftaran tidak ditemukan")
	}

	stage := string(app.Status)
	now := time.Now()
	withdrawal := &model.Withdrawal{
		ID:               uuid.New(),
		ApplicationID:    app.ID,
		UserID:           userID,
		StageAtRequest:   stage,
		ReasonCategory:   input.ReasonCategory,
		ReasonText:       cleanReason,
		RequiresApproval: elig.RequiresApproval,
		Source:           model.WithdrawalSourceApplicant,
		RequestedAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if elig.RequiresApproval {
		// Menunggu persetujuan admin
		withdrawal.Status = model.WithdrawalStatusPending
		if err := s.withdrawalRepo.Create(ctx, withdrawal); err != nil {
			return nil, err
		}

		// Catat ke application_changes
		_ = s.changeRepo.CreateChange(ctx, nil, &model.ApplicationChange{
			ApplicationID: app.ID,
			UserID:        userID,
			ChangeType:    model.ChangeTypeWithdrawalRequested,
			FromValues:    toJSONString(map[string]interface{}{"status": stage}),
			ToValues:      toJSONString(map[string]interface{}{"status": stage, "withdrawal_status": "pending"}),
			Reason:        cleanReason,
			ActorID:       &userID,
			ActorRole:     "calon_mahasiswa",
			Source:        model.ChangeSourceApplicant,
		})

		// Kirim notifikasi ke admin PMB
		if s.notifCli != nil {
			_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
				"template": "withdrawal_requested",
				"data": map[string]interface{}{
					"application_id":      app.ID.String(),
					"registration_number": app.RegistrationNo,
					"stage":               stage,
					"reason_category":     input.ReasonCategory,
				},
			}, nil)
		}

		return &WithdrawalResultResponse{
			Withdrawal:       withdrawal,
			Message:          "Permohonan pengunduran diri berhasil diajukan dan sedang menunggu persetujuan Admin PMB.",
			ID:               withdrawal.ID,
			Status:           string(withdrawal.Status),
			VerificationCode: withdrawal.VerifyCodeHash,
		}, nil
	}

	// Langsung berlaku
	withdrawal.Status = model.WithdrawalStatusEffective
	withdrawal.EffectiveAt = &now
	if err := s.applyWithdrawalEffects(ctx, app, withdrawal, nil); err != nil {
		return nil, err
	}

	if err := s.withdrawalRepo.Create(ctx, withdrawal); err != nil {
		return nil, err
	}

	verifCode := withdrawal.RawVerifyCode
	if verifCode == "" {
		verifCode = withdrawal.VerifyCodeHash
	}

	return &WithdrawalResultResponse{
		Withdrawal:       withdrawal,
		Message:          "Pengunduran diri telah resmi berlaku.",
		VerificationCode: verifCode,
		ID:               withdrawal.ID,
		Status:           string(withdrawal.Status),
	}, nil
}

func (s *withdrawalService) GetMyWithdrawal(ctx context.Context, userID uuid.UUID) (*model.Withdrawal, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil || app == nil {
		apps, _ := s.appRepo.FindByUserID(ctx, userID)
		if len(apps) > 0 {
			app = &apps[0]
		}
	}
	if app == nil {
		return nil, ErrWithdrawalNotFound
	}

	w, err := s.withdrawalRepo.FindLatestByApplicationID(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWithdrawalNotFound
	}
	return w, nil
}

func (s *withdrawalService) CancelPendingWithdrawal(ctx context.Context, userID uuid.UUID) error {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil || app == nil {
		return fmt.Errorf("pendaftaran tidak ditemukan")
	}

	pending, err := s.withdrawalRepo.FindPendingByApplicationID(ctx, app.ID)
	if err != nil || pending == nil {
		return fmt.Errorf("tidak ada permohonan pengunduran diri yang berstatus pending")
	}

	pending.Status = model.WithdrawalStatusCancelled
	pending.UpdatedAt = time.Now()
	if err := s.withdrawalRepo.Update(ctx, pending); err != nil {
		return err
	}

	// Catat di application_changes
	_ = s.changeRepo.CreateChange(ctx, nil, &model.ApplicationChange{
		ApplicationID: app.ID,
		UserID:        userID,
		ChangeType:    model.ChangeTypeWithdrawalCancelled,
		FromValues:    toJSONString(map[string]interface{}{"withdrawal_status": "pending"}),
		ToValues:      toJSONString(map[string]interface{}{"withdrawal_status": "cancelled"}),
		Reason:        "Dibatalkan oleh calon mahasiswa",
		ActorID:       &userID,
		ActorRole:     "calon_mahasiswa",
		Source:        model.ChangeSourceApplicant,
	})

	return nil
}

func (s *withdrawalService) AdminListWithdrawals(ctx context.Context, status string, page, limit int) ([]model.Withdrawal, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.withdrawalRepo.FindAll(ctx, status, page, limit)
}

func (s *withdrawalService) AdminApproveWithdrawal(ctx context.Context, withdrawalID, reviewerID uuid.UUID, note string) error {
	w, err := s.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return err
	}
	if w.Status != model.WithdrawalStatusPending {
		return fmt.Errorf("permohonan sudah tidak berstatus pending (status saat ini: %s)", w.Status)
	}

	app, err := s.appRepo.FindByID(ctx, w.ApplicationID)
	if err != nil || app == nil {
		return fmt.Errorf("pendaftaran terkait tidak ditemukan")
	}

	now := time.Now()
	w.Status = model.WithdrawalStatusApproved
	w.ReviewedBy = &reviewerID
	w.ReviewNote = note
	w.ReviewedAt = &now
	w.EffectiveAt = &now

	if err := s.applyWithdrawalEffects(ctx, app, w, &reviewerID); err != nil {
		return err
	}

	return s.withdrawalRepo.Update(ctx, w)
}

func (s *withdrawalService) AdminRejectWithdrawal(ctx context.Context, withdrawalID, reviewerID uuid.UUID, note string) error {
	cleanNote := strings.TrimSpace(note)
	if len(cleanNote) < 5 {
		return fmt.Errorf("catatan penolakan wajib diisi (minimal 5 karakter)")
	}

	w, err := s.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return err
	}
	if w.Status != model.WithdrawalStatusPending {
		return fmt.Errorf("permohonan sudah tidak berstatus pending (status saat ini: %s)", w.Status)
	}

	now := time.Now()
	w.Status = model.WithdrawalStatusRejected
	w.ReviewedBy = &reviewerID
	w.ReviewNote = cleanNote
	w.ReviewedAt = &now

	if err := s.withdrawalRepo.Update(ctx, w); err != nil {
		return err
	}

	// Catat di application_changes
	_ = s.changeRepo.CreateChange(ctx, nil, &model.ApplicationChange{
		ApplicationID: w.ApplicationID,
		UserID:        w.UserID,
		ChangeType:    model.ChangeTypeWithdrawalRejected,
		FromValues:    toJSONString(map[string]interface{}{"withdrawal_status": "pending"}),
		ToValues:      toJSONString(map[string]interface{}{"withdrawal_status": "rejected"}),
		Reason:        cleanNote,
		ActorID:       &reviewerID,
		ActorRole:     "admin_pmb",
		Source:        model.ChangeSourceAdmin,
	})

	// Notifikasi penolakan ke calon mahasiswa
	if s.notifCli != nil {
		_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
			"template": "withdrawal_rejected",
			"user_id":  w.UserID,
			"data": map[string]interface{}{
				"reason": cleanNote,
			},
		}, nil)
	}

	return nil
}

func (s *withdrawalService) AdminWithdrawApplication(ctx context.Context, appID, actorID uuid.UUID, input *AdminWithdrawInput, attachmentKey string) (*model.Withdrawal, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil || app == nil {
		return nil, fmt.Errorf("pendaftaran tidak ditemukan")
	}

	if app.Status == model.StatusWithdrawn || app.Status == model.StatusCancelled {
		return nil, fmt.Errorf("pendaftaran sudah berstatus %s", app.Status)
	}

	if strings.TrimSpace(input.ReasonText) == "" && strings.TrimSpace(input.Reason) != "" {
		input.ReasonText = strings.TrimSpace(input.Reason)
	}
	if strings.TrimSpace(input.ReasonCategory) == "" {
		input.ReasonCategory = "other"
	}
	if strings.TrimSpace(input.Note) == "" && strings.TrimSpace(input.Notes) != "" {
		input.Note = strings.TrimSpace(input.Notes)
	}

	cleanReason := strings.TrimSpace(input.ReasonText)
	if len(cleanReason) < 5 {
		cleanReason = "Pengunduran diri administratif resmi"
	}

	now := time.Now()
	stage := string(app.Status)
	w := &model.Withdrawal{
		ID:                  uuid.New(),
		ApplicationID:       app.ID,
		UserID:              app.UserID,
		StageAtRequest:      stage,
		ReasonCategory:      input.ReasonCategory,
		ReasonText:          cleanReason,
		Status:              model.WithdrawalStatusEffective,
		RequiresApproval:    false,
		RequestedAt:         now,
		ReviewedBy:          &actorID,
		ReviewNote:          input.Note,
		ReviewedAt:          &now,
		EffectiveAt:         &now,
		Source:              model.WithdrawalSourceAdmin,
		AttachmentObjectKey: attachmentKey,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.applyWithdrawalEffects(ctx, app, w, &actorID); err != nil {
		return nil, err
	}

	if err := s.withdrawalRepo.Create(ctx, w); err != nil {
		return nil, err
	}

	return w, nil
}

func (s *withdrawalService) applyWithdrawalEffects(ctx context.Context, app *model.Application, w *model.Withdrawal, actorID *uuid.UUID) error {
	fromStatus := app.Status
	now := time.Now()
	app.Status = model.StatusWithdrawn
	app.UpdatedAt = now
	_ = s.appRepo.Update(ctx, app)

	// 1. Generate Nomor Surat Resmi: PMB/UND/{YYYY}/{urut:4}
	year := now.Year()
	seq, _ := s.withdrawalRepo.GetNextLetterSequence(ctx, year)
	letterNo := fmt.Sprintf("PMB/UND/%d/%04d", year, seq)
	w.LetterNumber = letterNo

	// 2. Generate Random Verification Code & Hash SHA256
	rawVerifyCode := generateSecureToken(16)
	codeHash := sha256Hex(rawVerifyCode)
	w.VerifyCodeHash = codeHash
	w.RawVerifyCode = rawVerifyCode
	w.EffectiveAt = &now

	// 3. Catat di application_changes (CR-006 riwayat lengkap)
	actorRole := "calon_mahasiswa"
	actorSource := model.ChangeSourceApplicant
	targetActor := &w.UserID
	if actorID != nil {
		targetActor = actorID
		actorRole = "admin_pmb"
		actorSource = model.ChangeSourceAdmin
	}

	_ = s.changeRepo.CreateChange(ctx, nil, &model.ApplicationChange{
		ApplicationID: app.ID,
		UserID:        app.UserID,
		ChangeType:    model.ChangeTypeWithdrawalApproved,
		FromValues:    toJSONString(map[string]interface{}{"status": string(fromStatus)}),
		ToValues:      toJSONString(map[string]interface{}{"status": "withdrawn", "letter_number": letterNo}),
		Reason:        w.ReasonText,
		ActorID:       targetActor,
		ActorRole:     actorRole,
		Source:        actorSource,
	})

	// 4. Dampak Lintas Service (Idempoten)
	// 4a. Finance Service: Batalkan tagihan & tandai refund_review untuk pembayaran yang masuk
	if s.financeCli != nil {
		_ = s.financeCli.Post(ctx, fmt.Sprintf("/internal/applications/%s/withdraw", app.ID), nil, nil)
	}

	// 4b. Exam Service: Keluarkan peserta dari sesi ujian & auto-submit attempt CBT
	if s.examCli != nil {
		_ = s.examCli.Post(ctx, "/internal/participants/remove", map[string]interface{}{
			"application_id": app.ID,
		}, nil)
	}

	// 4c. Selection Service: Ubah hasil seleksi jadi withdrawn & lepas kuota prodi
	if s.selectionCli != nil {
		_ = s.selectionCli.Post(ctx, fmt.Sprintf("/internal/applications/%s/withdraw", app.ID), nil, nil)
	}

	// 4d. Student Service: Batalkan daftar ulang & set status mahasiswa ke withdrawn
	if s.studentCli != nil {
		_ = s.studentCli.Post(ctx, fmt.Sprintf("/internal/applications/%s/withdraw", app.ID), nil, nil)
	}

	// 4e. NIM Service: Tandai NIM jadi voided (tidak dihapus dan nomor tidak dipakai ulang)
	if s.nimCli != nil {
		_ = s.nimCli.Post(ctx, fmt.Sprintf("/internal/nim/issuances/%s/void", app.ID), nil, nil)
	}

	// 4f. Auth Service: Cabut role mahasiswa secara idempoten
	if s.authCli != nil {
		_ = s.authCli.Delete(ctx, fmt.Sprintf("/internal/users/%s/roles/mahasiswa", app.UserID), nil)
	}

	// 4g. Notifikasi ke Calon Mahasiswa & Keuangan / Prodi
	if s.notifCli != nil {
		_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
			"template": "withdrawal_effective",
			"user_id":  app.UserID,
			"data": map[string]interface{}{
				"registration_number": app.RegistrationNo,
				"letter_number":       letterNo,
			},
		}, nil)
	}

	return nil
}

func (s *withdrawalService) GetWithdrawalLetterPDF(ctx context.Context, userID uuid.UUID) ([]byte, string, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil || app == nil {
		apps, _ := s.appRepo.FindByUserID(ctx, userID)
		if len(apps) > 0 {
			app = &apps[0]
		}
	}
	if app == nil {
		return nil, "", fmt.Errorf("pendaftaran tidak ditemukan")
	}

	w, err := s.withdrawalRepo.FindLatestByApplicationID(ctx, app.ID)
	if err != nil || w == nil {
		return nil, "", fmt.Errorf("bukti pengunduran diri belum tersedia")
	}

	if w.Status != model.WithdrawalStatusEffective && w.Status != model.WithdrawalStatusApproved {
		return nil, "", fmt.Errorf("pengunduran diri belum disetujui atau belum efektif")
	}

	// Ambil nama jalur & prodi
	pathName := "Jalur PMB"
	if s.pathRepo != nil {
		p, _ := s.pathRepo.FindByID(ctx, app.AdmissionPathID)
		if p != nil {
			pathName = p.Name
		}
	}

	fullName := "Calon Mahasiswa"
	if s.identityCli != nil {
		var idResp struct {
			Data struct {
				FullName string `json:"full_name"`
			} `json:"data"`
		}
		if errCall := s.identityCli.Get(ctx, fmt.Sprintf("/internal/identities/by-user/%s", userID), nil, &idResp); errCall == nil && idResp.Data.FullName != "" {
			fullName = idResp.Data.FullName
		}
	}

	reasonCategoryLabel := w.ReasonCategory
	for _, c := range model.DefaultReasonCategories {
		if c.Code == w.ReasonCategory {
			reasonCategoryLabel = c.Label
			break
		}
	}

	effectiveDate := w.RequestedAt
	if w.EffectiveAt != nil {
		effectiveDate = *w.EffectiveAt
	}

	verifyURL := fmt.Sprintf("/admission/withdrawals/verify/%s", w.VerifyCodeHash)

	pdfBytes, err := pdf.GenerateWithdrawalLetterPDF(pdf.WithdrawalLetterData{
		LetterNumber:       w.LetterNumber,
		FullName:           fullName,
		RegistrationNumber: app.RegistrationNo,
		ProgramName:        app.Choice1ID.String(),
		PathName:           pathName,
		StageAtWithdrawal:  w.StageAtRequest,
		ReasonCategory:     reasonCategoryLabel,
		ReasonText:         w.ReasonText,
		EffectiveDate:      effectiveDate,
		VerifyURL:          verifyURL,
	})
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("surat-pengunduran-diri-%s.pdf", app.RegistrationNo)
	return pdfBytes, filename, nil
}

func (s *withdrawalService) VerifyWithdrawalPublic(ctx context.Context, code string) (*VerifyWithdrawalResponse, error) {
	// Cek hash SHA256 atau match langsung jika code sudah merupakan hash
	hash := sha256Hex(code)
	w, err := s.withdrawalRepo.FindByVerifyCodeHash(ctx, hash)
	if err != nil || w == nil {
		// Coba cari langsung by code (bila query dikirim sebagai hash)
		w, err = s.withdrawalRepo.FindByVerifyCodeHash(ctx, code)
		if err != nil || w == nil {
			return nil, ErrWithdrawalNotFound
		}
	}

	app, _ := s.appRepo.FindByID(ctx, w.ApplicationID)
	regNo := "REG-PMB"
	if app != nil {
		regNo = app.RegistrationNo
	}

	// Masking nama
	maskedName := maskName("Mahasiswa PMB Terverifikasi")
	if s.identityCli != nil && app != nil {
		var idResp struct {
			Data struct {
				FullName string `json:"full_name"`
			} `json:"data"`
		}
		if errCall := s.identityCli.Get(ctx, fmt.Sprintf("/internal/identities/by-user/%s", app.UserID), nil, &idResp); errCall == nil && idResp.Data.FullName != "" {
			maskedName = maskName(idResp.Data.FullName)
		}
	}

	effectiveDateStr := w.RequestedAt.Format("02 January 2006")
	if w.EffectiveAt != nil {
		effectiveDateStr = w.EffectiveAt.Format("02 January 2006")
	}

	return &VerifyWithdrawalResponse{
		Valid:              true,
		LetterNumber:       w.LetterNumber,
		FullNameMasked:     maskedName,
		RegistrationNumber: maskString(regNo, 3),
		ProgramName:        "Program Studi Pilihan PMB",
		PathName:           "Jalur Seleksi Resmi",
		EffectiveDate:      effectiveDateStr,
		ReasonCategory:     w.ReasonCategory,
		Status:             "Resmi Dibatalkan / Mengundurkan Diri",
	}, nil
}

func (s *withdrawalService) AdminGetWithdrawalStats(ctx context.Context) (map[string]interface{}, error) {
	return s.withdrawalRepo.GetStats(ctx)
}

func (s *withdrawalService) AdminExportWithdrawalsXLSX(ctx context.Context, status string) ([]byte, error) {
	items, _, err := s.withdrawalRepo.FindAll(ctx, status, 1, 1000)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "Pengunduran Diri"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"No", "Nomor Pendaftaran", "Tahap Saat Mundur", "Kategori Alasan", "Alasan", "Status", "Nomor Surat", "Tanggal Diajukan", "Tanggal Efektif"}
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for rowIdx, it := range items {
		row := rowIdx + 2
		regNo := "-"
		if it.Application != nil {
			regNo = it.Application.RegistrationNo
		}
		effStr := "-"
		if it.EffectiveAt != nil {
			effStr = it.EffectiveAt.Format("2006-01-02 15:04")
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), rowIdx+1)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), regNo)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), it.StageAtRequest)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), it.ReasonCategory)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), it.ReasonText)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), it.Status)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), it.LetterNumber)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), it.RequestedAt.Format("2006-01-02 15:04"))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), effStr)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func generateSecureToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func maskName(name string) string {
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) <= 2 {
			continue
		}
		masked := string(w[0]) + strings.Repeat("*", len(w)-1)
		words[i] = masked
	}
	return strings.Join(words, " ")
}

func maskString(s string, visiblePrefix int) string {
	if len(s) <= visiblePrefix {
		return s
	}
	return s[:visiblePrefix] + strings.Repeat("*", len(s)-visiblePrefix)
}

func toJSONString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
