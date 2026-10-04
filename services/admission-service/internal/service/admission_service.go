package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/outbox"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrPeriodClosed            = errors.New("periode pendaftaran telah ditutup atau belum dibuka")
	ErrDuplicateApplication    = errors.New("Anda sudah memiliki pendaftaran aktif pada periode ini")
	ErrInvalidState            = errors.New("pilihan program studi tidak dapat diubah pada status pendaftaran saat ini")
	ErrForbiddenAccess         = errors.New("Anda tidak memiliki akses ke data pendaftaran ini")
	ErrDocUploadNotAllowed     = errors.New("dokumen tidak dapat diunggah pada status pendaftaran saat ini")
	ErrDocTypeNotSupported     = errors.New("jenis dokumen tidak valid untuk jalur pendaftaran ini")
	ErrDocTooLarge             = errors.New("ukuran dokumen melebihi batas maksimal yang ditentukan")
	ErrSubmitNotAllowed        = errors.New("pendaftaran tidak dapat disubmit pada status saat ini")
	ErrBiodataIncomplete       = errors.New("BIODATA_INCOMPLETE: biodata diri calon mahasiswa belum lengkap")
	ErrDocumentRequired        = errors.New("DOCUMENT_REQUIRED: seluruh dokumen persyaratan wajib harus diunggah")
	ErrApplicationNotSubmitted = errors.New("berkas pendaftaran belum disubmit untuk verifikasi")
	ErrActiveApplicationExists = errors.New("ACTIVE_APPLICATION_EXISTS: Anda memiliki pendaftaran yang sedang aktif")
	ErrCancelNotAllowed        = errors.New("CANCEL_NOT_ALLOWED: pendaftaran tidak dapat dibatalkan pada status saat ini atau biaya sudah dibayar")
	ErrSameChoiceAlreadyFailed = errors.New("SAME_CHOICE_ALREADY_FAILED: Anda tidak dapat memilih jalur dan program studi yang sama setelah dinyatakan tidak lulus pada gelombang ini")
	ErrAlreadyAccepted         = errors.New("ALREADY_ACCEPTED: Anda sudah dinyatakan lulus atau terdaftar pada seleksi sebelumnya")
	ErrProgramClosed           = errors.New("PROGRAM_CLOSED: pendaftaran ke program studi yang dipilih sedang ditutup")
)

type CreateApplicationRequest struct {
	UserID          uuid.UUID  `json:"user_id"`
	PeriodID        uint       `json:"period_id"`
	AdmissionPathID uint       `json:"admission_path_id"`
	Choice1ID       uuid.UUID  `json:"choice_1_id"`
	Choice2ID       *uuid.UUID `json:"choice_2_id,omitempty"`
	UserEmail       string     `json:"user_email,omitempty"`
	UserName        string     `json:"user_name,omitempty"`
}

type ChangeChoicesRequest struct {
	ApplicationID uuid.UUID  `json:"application_id"`
	UserID        uuid.UUID  `json:"user_id"`
	Choice1ID     uuid.UUID  `json:"choice_1_id"`
	Choice2ID     *uuid.UUID `json:"choice_2_id,omitempty"`
}

type DocumentReviewItem struct {
	DocumentType string `json:"document_type" binding:"required"`
	Status       string `json:"status" binding:"required,oneof=valid invalid"`
	Notes        string `json:"notes"`
}

type BlockedChoice struct {
	PeriodID           uint      `json:"period_id"`
	AdmissionPathID    uint      `json:"path_id"`
	FirstChoiceProgram uuid.UUID `json:"first_choice_program"`
}

type EligibilityResponse struct {
	CanCreateNew   bool            `json:"can_create_new"`
	Reason         string          `json:"reason"`
	BlockedChoices []BlockedChoice `json:"blocked_choices"`
}

type StepStatus string

const (
	StepDone    StepStatus = "done"
	StepCurrent StepStatus = "current"
	StepLocked  StepStatus = "locked"
)

type ProgressStep struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      StepStatus `json:"status"`
	ActionURL   string     `json:"action_url,omitempty"`
}

type ProgressResponse struct {
	ActiveApplicationID *uuid.UUID     `json:"active_application_id,omitempty"`
	Status              string         `json:"status,omitempty"`
	CurrentStep         string         `json:"current_step"`
	Steps               []ProgressStep `json:"steps"`
}

type AdmissionService interface {
	CreateApplication(ctx context.Context, req *CreateApplicationRequest) (*model.Application, error)
	ChangeChoices(ctx context.Context, req *ChangeChoicesRequest) error
	GetMyApplications(ctx context.Context, userID uuid.UUID) ([]model.Application, error)
	GetMyActiveApplication(ctx context.Context, userID uuid.UUID) (*model.Application, error)
	CancelMyApplication(ctx context.Context, userID uuid.UUID, reason string) error
	GetEligibility(ctx context.Context, userID uuid.UUID) (*EligibilityResponse, error)
	GetProgress(ctx context.Context, userID uuid.UUID) (*ProgressResponse, error)
	GetApplicationDetail(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string, prodiID *uuid.UUID) (*model.Application, error)
	ListApplicationsAdmin(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, int64, error)
	GetActivePeriod(ctx context.Context) (*model.Period, error)
	GetAdmissionPaths(ctx context.Context) ([]model.AdmissionPath, error)
	UpdateApplicationStatus(ctx context.Context, id uuid.UUID, newStatus model.ApplicationStatus, notes string, changedBy *uuid.UUID) error

	// Tahap 5 - F2-M6 (Dokumen, Submit, Verifikasi, Export, Stats)
	UploadDocument(ctx context.Context, appID, userID uuid.UUID, reqCode string, reader io.Reader, size int64, originalName, declaredMime string) (*model.ApplicationDocument, error)
	DeleteDocument(ctx context.Context, appID, userID uuid.UUID, reqCode string) error
	SubmitApplication(ctx context.Context, appID, userID uuid.UUID) (*model.Application, error)
	VerifyApplicationDocuments(ctx context.Context, appID, reviewerID uuid.UUID, reviews []DocumentReviewItem) (*model.Application, error)
	GetDocumentPresignedURL(ctx context.Context, appID uuid.UUID, reqCode string, requesterID uuid.UUID, role string) (string, error)
	ExportApplicationsXLSX(ctx context.Context, filter repository.ApplicationFilter) ([]byte, error)
	GetStats(ctx context.Context) (*model.AdmissionStats, error)

	// CR-009 & CR-010 Admin Portal & Camaba v2
	GetHeaderSummary(ctx context.Context, periodID *uint, scope rbac.ScopeResult) (*HeaderSummaryResponse, error)
	GetOverview(ctx context.Context, periodID *uint, scope rbac.ScopeResult, role string) (*OverviewResponse, error)
	GetDocumentQueue(ctx context.Context, periodID *uint, status, q string, page, limit int, scope rbac.ScopeResult) (*DocumentQueueResponse, error)
	GetApplicationDocumentsModal(ctx context.Context, appID uuid.UUID, scope rbac.ScopeResult) (*ApplicationDocumentsModalResponse, error)
	GetInstitutionDashboard(ctx context.Context, periodID *uint, level, code string, scope rbac.ScopeResult) (*InstitutionDashboardResponse, error)
	GetReportsPreview(ctx context.Context, periodID *uint, programCode string, scope rbac.ScopeResult) (*ReportPreviewResponse, error)
	GenerateOfficialReportPDF(ctx context.Context, docType string, periodID *uint, actorID uuid.UUID, actorName string) ([]byte, string, error)
	GetCamabaDashboardV2(ctx context.Context, userID uuid.UUID) (*CamabaDashboardV2Response, error)
	GetAdmissionOptions(ctx context.Context) (*AdmissionOptionsResponse, error)
}

type admissionService struct {
	periodRepo      repository.PeriodRepository
	pathRepo        repository.AdmissionPathRepository
	quotaRepo       repository.QuotaRepository
	appRepo         repository.ApplicationRepository
	docRepo         repository.ApplicationDocumentRepository
	outboxRepo      outbox.Repository
	officialDocRepo repository.OfficialDocumentRepository
	notifCli        internalclient.Client
	identityCli     internalclient.Client
	financeCli      internalclient.Client
	orgCli          internalclient.Client
	storageCli      storage.Client
	cfg             config.Config
}

func NewAdmissionService(
	periodRepo repository.PeriodRepository,
	pathRepo repository.AdmissionPathRepository,
	quotaRepo repository.QuotaRepository,
	appRepo repository.ApplicationRepository,
	docRepo repository.ApplicationDocumentRepository,
	outboxRepo outbox.Repository,
	officialDocRepo repository.OfficialDocumentRepository,
	notifCli internalclient.Client,
	identityCli internalclient.Client,
	financeCli internalclient.Client,
	orgCli internalclient.Client,
	storageCli storage.Client,
	cfg config.Config,
) AdmissionService {
	return &admissionService{
		periodRepo:      periodRepo,
		pathRepo:        pathRepo,
		quotaRepo:       quotaRepo,
		appRepo:         appRepo,
		docRepo:         docRepo,
		outboxRepo:      outboxRepo,
		officialDocRepo: officialDocRepo,
		notifCli:        notifCli,
		identityCli:     identityCli,
		financeCli:      financeCli,
		orgCli:          orgCli,
		storageCli:      storageCli,
		cfg:             cfg,
	}
}

func (s *admissionService) GetActivePeriod(ctx context.Context) (*model.Period, error) {
	return s.periodRepo.FindActive(ctx)
}

func (s *admissionService) GetAdmissionPaths(ctx context.Context) ([]model.AdmissionPath, error) {
	return s.pathRepo.FindAllActive(ctx)
}

func (s *admissionService) CreateApplication(ctx context.Context, req *CreateApplicationRequest) (*model.Application, error) {
	// 1. Cek Periode
	period, err := s.periodRepo.FindByID(ctx, req.PeriodID)
	if err != nil {
		return nil, ErrPeriodClosed
	}
	if !period.IsOpenNow() {
		return nil, ErrPeriodClosed
	}

	// 2. Cek Jalur Masuk
	path, err := s.pathRepo.FindByID(ctx, req.AdmissionPathID)
	if err != nil || !path.IsActive {
		return nil, errors.New("jalur pendaftaran tidak valid atau sedang nonaktif")
	}

	// 2b. Cek Status Buka/Tutup Program Studi (CR-009/CR-010)
	if s.orgCli != nil {
		type orgProgItem struct {
			Code          string `json:"code"`
			IsActive      bool   `json:"is_active"`
			AdmissionOpen bool   `json:"admission_open"`
		}
		var orgResp struct {
			Success bool          `json:"success"`
			Data    []orgProgItem `json:"data"`
		}
		if err := s.orgCli.Get(ctx, "/internal/study-programs", nil, &orgResp); err == nil && orgResp.Success {
			for _, item := range orgResp.Data {
				progUUID := uuid.NewMD5(uuid.NameSpaceDNS, []byte("prodi:"+item.Code))
				if (req.Choice1ID == progUUID || (req.Choice2ID != nil && *req.Choice2ID == progUUID)) && !item.AdmissionOpen {
					return nil, ErrProgramClosed
				}
			}
		}
	}

	// 3. Cek Duplikasi & Validasi Pendaftaran Aktif (CR-002)
	activeApp, err := s.appRepo.FindActiveByUserID(ctx, req.UserID)
	if err == nil && activeApp != nil {
		return nil, ErrActiveApplicationExists
	}

	// Cek Riwayat Pendaftaran Sebelumnya (CR-002 D.5 point 6 & 8)
	history, err := s.appRepo.FindByUserID(ctx, req.UserID)
	if err == nil {
		for _, prevApp := range history {
			// Jika sudah pernah dinyatakan lulus / daftar ulang / mahasiswa, tidak boleh mendaftar lagi
			if prevApp.Status == model.StatusPassed || prevApp.Status == model.StatusReregistered || prevApp.Status == model.StatusEnrolled {
				return nil, ErrAlreadyAccepted
			}
			// Jika gagal di gelombang yang sama, tidak boleh pilih kombinasi jalur + prodi 1 yang sama
			if prevApp.Status == model.StatusFailed && prevApp.PeriodID == req.PeriodID && prevApp.AdmissionPathID == req.AdmissionPathID && prevApp.Choice1ID == req.Choice1ID {
				return nil, ErrSameChoiceAlreadyFailed
			}
		}
	}

	var newApp *model.Application

	// 4. Eksekusi dalam Database Transaction
	err = s.appRepo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		regNo, genErr := s.appRepo.GenerateRegistrationNo(ctx, tx, period.ID, period.AcademicYear)
		if genErr != nil {
			return fmt.Errorf("gagal membuat nomor registrasi: %w", genErr)
		}

		initialStatus := model.StatusAwaitingPayment
		initialNotes := "Pendaftaran berhasil dibuat dan menunggu pembayaran tagihan"
		if path.RegistrationFee == 0 {
			initialStatus = model.StatusFormOpen
			initialNotes = "Pendaftaran jalur gratis berhasil dibuat - otomatis aktif tanpa pembayaran formulir"
		}

		app := &model.Application{
			ID:              uuid.New(),
			RegistrationNo:  regNo,
			UserID:          req.UserID,
			PeriodID:        req.PeriodID,
			AdmissionPathID: req.AdmissionPathID,
			Choice1ID:       req.Choice1ID,
			Choice2ID:       req.Choice2ID,
			Status:          initialStatus,
		}

		if err := s.appRepo.CreateInTx(tx, app); err != nil {
			return fmt.Errorf("gagal menyimpan berkas pendaftaran: %w", err)
		}

		statusLog := &model.ApplicationStatusLog{
			ID:            uuid.New(),
			ApplicationID: app.ID,
			FromStatus:    model.StatusDraft,
			ToStatus:      initialStatus,
			ChangedBy:     &req.UserID,
			Notes:         initialNotes,
		}
		if err := s.appRepo.CreateStatusLog(tx, statusLog); err != nil {
			return fmt.Errorf("gagal mencatat riwayat status pendaftaran: %w", err)
		}

		// Transactional Outbox event untuk Finance Service
		outboxPayload := map[string]interface{}{
			"user_id":        app.UserID.String(),
			"application_id": app.ID.String(),
			"amount":         path.RegistrationFee,
			"description":    fmt.Sprintf("Biaya Pendaftaran PMB Jalur %s (%s)", path.Name, regNo),
			"due_date_days":  3,
		}

		idempotencyKey := fmt.Sprintf("invoice:application:%s", app.ID.String())
		outboxEvent, evErr := outbox.NewEvent("application", app.ID.String(), "invoice.create", idempotencyKey, outboxPayload)
		if evErr != nil {
			return fmt.Errorf("gagal membuat objek outbox event: %w", evErr)
		}

		if err := s.outboxRepo.CreateInTx(tx, outboxEvent); err != nil {
			return fmt.Errorf("gagal menyimpan outbox event: %w", err)
		}

		newApp = app
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sinkronkan pembuatan tagihan ke Finance Service agar langsung tersedia tanpa jeda outbox
	if s.financeCli != nil {
		go func() {
			var resp interface{}
			_ = s.financeCli.Post(context.Background(), "/internal/invoices", map[string]interface{}{
				"user_id":        newApp.UserID.String(),
				"application_id": newApp.ID.String(),
				"amount":         path.RegistrationFee,
				"description":    fmt.Sprintf("Biaya Pendaftaran PMB Jalur %s (%s)", path.Name, newApp.RegistrationNo),
				"due_date_days":  3,
			}, &resp)
		}()
	}

	// 5. Trigger Notifikasi Email
	if req.UserEmail != "" && s.notifCli != nil {
		go func() {
			notifPayload := map[string]interface{}{
				"channel":  "email",
				"email":    req.UserEmail,
				"template": "application_registered",
				"data": map[string]interface{}{
					"name":             req.UserName,
					"registration_no":  newApp.RegistrationNo,
					"path_name":        path.Name,
					"period_name":      period.Name,
					"bill_amount":      fmt.Sprintf("%.0f", path.RegistrationFee),
					"va_number":        "- (Menunggu invoice Finance)",
					"payment_deadline": "3 hari setelah pendaftaran",
				},
			}
			var resp interface{}
			_ = s.notifCli.Post(context.Background(), "/internal/notifications", notifPayload, &resp)
		}()
	}

	return s.appRepo.FindByID(ctx, newApp.ID)
}

func (s *admissionService) ChangeChoices(ctx context.Context, req *ChangeChoicesRequest) error {
	app, err := s.appRepo.FindByID(ctx, req.ApplicationID)
	if err != nil {
		return err
	}

	if app.UserID != req.UserID {
		return ErrForbiddenAccess
	}

	if !app.CanChangeMajor() {
		return ErrInvalidState
	}

	return s.appRepo.UpdateChoices(ctx, req.ApplicationID, req.Choice1ID, req.Choice2ID)
}

func (s *admissionService) GetMyApplications(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	return s.appRepo.FindByUserID(ctx, userID)
}

func (s *admissionService) GetMyActiveApplication(ctx context.Context, userID uuid.UUID) (*model.Application, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	app.CanCancel = app.IsCancellable()
	return app, nil
}

func (s *admissionService) CancelMyApplication(ctx context.Context, userID uuid.UUID, reason string) error {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if !app.IsCancellable() {
		return ErrCancelNotAllowed
	}

	// Pembatalan tagihan di Finance Service secara sinkron (CR-002 D.5 point 2)
	if s.financeCli != nil {
		var cancelResp struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		cancelPayload := map[string]string{
			"reason": reason,
		}
		// Panggil pembatalan berdasarkan application_id
		err := s.financeCli.Post(ctx, fmt.Sprintf("/internal/invoices/cancel-by-application/%s", app.ID.String()), cancelPayload, &cancelResp)
		if err != nil {
			log.Printf("[ADMISSION] Finance cancel returned error for app %s: %v", app.ID, err)
			return fmt.Errorf("%w: pembayaran sudah berhasil atau sedang diverifikasi", ErrCancelNotAllowed)
		}
	}

	// Update status pendaftaran menjadi cancelled di basis data
	err = s.appRepo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(&model.Application{}).Where("id = ?", app.ID).Updates(map[string]interface{}{
			"status":     model.StatusCancelled,
			"notes":      reason,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}

		statusLog := &model.ApplicationStatusLog{
			ID:            uuid.New(),
			ApplicationID: app.ID,
			FromStatus:    app.Status,
			ToStatus:      model.StatusCancelled,
			ChangedBy:     &userID,
			Notes:         "Pendaftaran dibatalkan oleh calon mahasiswa. Alasan: " + reason,
		}
		if err := s.appRepo.CreateStatusLog(tx, statusLog); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Kirim notifikasi async pembatalan (CR-002)
	if s.notifCli != nil {
		go func() {
			notifPayload := map[string]interface{}{
				"channel":  "email",
				"template": "application_cancelled",
				"data": map[string]interface{}{
					"registration_no": app.RegistrationNo,
					"reason":          reason,
				},
			}
			var resp interface{}
			_ = s.notifCli.Post(context.Background(), "/internal/notifications", notifPayload, &resp)
		}()
	}

	return nil
}

func (s *admissionService) GetEligibility(ctx context.Context, userID uuid.UUID) (*EligibilityResponse, error) {
	resp := &EligibilityResponse{
		CanCreateNew:   true,
		BlockedChoices: []BlockedChoice{},
	}

	// 1. Cek pendaftaran aktif
	activeApp, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err == nil && activeApp != nil {
		resp.CanCreateNew = false
		resp.Reason = "Anda memiliki pendaftaran yang sedang aktif"
		return resp, nil
	}

	// 2. Cek apakah ada gelombang pendaftaran buka
	period, err := s.periodRepo.FindActive(ctx)
	if err != nil || !period.IsOpenNow() {
		resp.CanCreateNew = false
		resp.Reason = "Tidak ada gelombang pendaftaran yang sedang dibuka"
		return resp, nil
	}

	// 3. Cek riwayat pendaftaran
	history, err := s.appRepo.FindByUserID(ctx, userID)
	if err == nil {
		for _, prevApp := range history {
			if prevApp.Status == model.StatusPassed || prevApp.Status == model.StatusReregistered || prevApp.Status == model.StatusEnrolled {
				resp.CanCreateNew = false
				resp.Reason = "Anda sudah dinyatakan lulus seleksi atau terdaftar sebagai mahasiswa"
				return resp, nil
			}
			if prevApp.Status == model.StatusFailed && prevApp.PeriodID == period.ID {
				resp.BlockedChoices = append(resp.BlockedChoices, BlockedChoice{
					PeriodID:           prevApp.PeriodID,
					AdmissionPathID:    prevApp.AdmissionPathID,
					FirstChoiceProgram: prevApp.Choice1ID,
				})
			}
		}
	}

	resp.Reason = "Anda memenuhi syarat untuk membuat pendaftaran baru"
	return resp, nil
}

func (s *admissionService) GetProgress(ctx context.Context, userID uuid.UUID) (*ProgressResponse, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrApplicationNotFound) {
		return nil, err
	}

	resp := &ProgressResponse{
		CurrentStep: "step_choice",
		Steps: []ProgressStep{
			{
				ID:          "step_account",
				Title:       "Pendaftaran Akun",
				Description: "Pembuatan akun pendaftaran dan verifikasi email",
				Status:      StepDone,
			},
			{
				ID:          "step_choice",
				Title:       "Pilihan Program Studi",
				Description: "Pemilihan gelombang, jalur, dan program studi yang dituju",
				Status:      StepLocked,
			},
			{
				ID:          "step_payment",
				Title:       "Pembayaran Biaya Pendaftaran",
				Description: "Pembayaran tagihan pendaftaran melalui Duitku / VA",
				Status:      StepLocked,
			},
			{
				ID:          "step_biodata_docs",
				Title:       "Isi Biodata & Unggah Berkas",
				Description: "Pengisian formulir biodata lengkap dan unggah dokumen persyaratan",
				Status:      StepLocked,
			},
			{
				ID:          "step_verification",
				Title:       "Verifikasi Berkas",
				Description: "Pemeriksaan dan validasi berkas oleh tim administrasi PMB",
				Status:      StepLocked,
			},
			{
				ID:          "step_exam",
				Title:       "Ujian Seleksi / CBT",
				Description: "Pelaksanaan tes tertulis online (CBT) atau wawancara seleksi",
				Status:      StepLocked,
			},
			{
				ID:          "step_announcement",
				Title:       "Pengumuman Kelulusan",
				Description: "Pemberitahuan hasil seleksi penerimaan mahasiswa baru",
				Status:      StepLocked,
			},
			{
				ID:          "step_reregistration",
				Title:       "Daftar Ulang & NIM",
				Description: "Pembayaran uang kuliah dan penerbitan Nomor Induk Mahasiswa",
				Status:      StepLocked,
			},
		},
	}

	if app == nil {
		resp.Steps[1].Status = StepCurrent
		resp.CurrentStep = "step_choice"
		return resp, nil
	}

	resp.ActiveApplicationID = &app.ID
	resp.Status = string(app.Status)
	resp.Steps[1].Status = StepDone

	switch app.Status {
	case model.StatusDraft, model.StatusAwaitingPayment:
		resp.Steps[2].Status = StepCurrent
		resp.CurrentStep = "step_payment"
	case model.StatusFormOpen, model.StatusRevisionRequired:
		resp.Steps[2].Status = StepDone
		resp.Steps[3].Status = StepCurrent
		resp.CurrentStep = "step_biodata_docs"
	case model.StatusSubmitted:
		resp.Steps[2].Status = StepDone
		resp.Steps[3].Status = StepDone
		resp.Steps[4].Status = StepCurrent
		resp.CurrentStep = "step_verification"
	case model.StatusVerified, model.StatusExamScheduled, model.StatusInterviewScheduled:
		resp.Steps[2].Status = StepDone
		resp.Steps[3].Status = StepDone
		resp.Steps[4].Status = StepDone
		resp.Steps[5].Status = StepCurrent
		resp.CurrentStep = "step_exam"
	case model.StatusDirectDecision:
		resp.Steps[2].Status = StepDone
		resp.Steps[3].Status = StepDone
		resp.Steps[4].Status = StepDone
		resp.Steps[5].Status = StepDone
		resp.Steps[6].Status = StepCurrent
		resp.CurrentStep = "step_announcement"
	case model.StatusPassed:
		resp.Steps[2].Status = StepDone
		resp.Steps[3].Status = StepDone
		resp.Steps[4].Status = StepDone
		resp.Steps[5].Status = StepDone
		resp.Steps[6].Status = StepDone
		resp.Steps[7].Status = StepCurrent
		resp.CurrentStep = "step_reregistration"
	case model.StatusReregistered, model.StatusEnrolled:
		for i := range resp.Steps {
			resp.Steps[i].Status = StepDone
		}
		resp.CurrentStep = "completed"
	default:
		resp.CurrentStep = string(app.Status)
	}

	return resp, nil
}

func (s *admissionService) GetApplicationDetail(
	ctx context.Context,
	id uuid.UUID,
	requesterID uuid.UUID,
	role string,
	prodiID *uuid.UUID,
) (*model.Application, error) {
	app, err := s.appRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	switch role {
	case "superadmin", "admin_pmb", "keuangan", "pewawancara", "penguji":
		// Pengayaan data untuk admin: ambil identitas & invoice
		if s.identityCli != nil {
			var personData interface{}
			err := s.identityCli.Get(ctx, fmt.Sprintf("/internal/persons/%s", app.UserID.String()), nil, &personData)
			if err == nil && personData != nil {
				if m, ok := personData.(map[string]interface{}); ok {
					if innerData, exists := m["data"]; exists && innerData != nil {
						app.PersonData = innerData
					} else {
						app.PersonData = personData
					}
				} else {
					app.PersonData = personData
				}
			}
		}

		if s.financeCli != nil {
			var invData interface{}
			err := s.financeCli.Get(ctx, fmt.Sprintf("/internal/invoices?application_id=%s", app.ID.String()), nil, &invData)
			if err == nil && invData != nil {
				if m, ok := invData.(map[string]interface{}); ok {
					if innerData, exists := m["data"]; exists && innerData != nil {
						app.InvoiceData = innerData
					} else {
						app.InvoiceData = invData
					}
				} else {
					app.InvoiceData = invData
				}
			}
		}

		return app, nil
	case "prodi":
		if prodiID == nil {
			return nil, ErrForbiddenAccess
		}
		if app.Choice1ID != *prodiID && (app.Choice2ID == nil || *app.Choice2ID != *prodiID) {
			return nil, ErrForbiddenAccess
		}
		return app, nil
	default:
		if app.UserID != requesterID {
			return nil, ErrForbiddenAccess
		}
		return app, nil
	}
}

func (s *admissionService) ListApplicationsAdmin(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, int64, error) {
	return s.appRepo.ListApplications(ctx, filter)
}

func (s *admissionService) UpdateApplicationStatus(ctx context.Context, id uuid.UUID, newStatus model.ApplicationStatus, notes string, changedBy *uuid.UUID) error {
	app, err := s.appRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	fromStatus := app.Status
	return s.appRepo.UpdateStatus(ctx, id, fromStatus, newStatus, notes, changedBy)
}

func isMatchDocType(reqDocType, userDocType string) bool {
	reqNorm := strings.ToUpper(strings.TrimSpace(reqDocType))
	userNorm := strings.ToUpper(strings.TrimSpace(userDocType))
	if reqNorm == userNorm {
		return true
	}
	switch userNorm {
	case "IJAZAH", "IJAZAH_SKL", "SKL":
		return reqNorm == "IJAZAH_SKL" || reqNorm == "IJAZAH"
	case "KTP", "KTP_KIA_KP", "KIA", "KARTU_PELAJAR":
		return reqNorm == "KTP_KIA_KP" || reqNorm == "KTP"
	case "KK", "KARTU_KELUARGA":
		return reqNorm == "KK" || reqNorm == "KARTU_KELUARGA"
	case "PAS_FOTO", "FOTO", "PASFOTO":
		return reqNorm == "PAS_FOTO" || reqNorm == "FOTO"
	case "SURAT_SEHAT", "SEHAT":
		return reqNorm == "SURAT_SEHAT"
	case "RAPOR", "NILAI_RAPOR":
		return reqNorm == "RAPOR" || reqNorm == "NILAI_RAPOR"
	case "SERTIFIKAT_PRESTASI", "PRESTASI", "SERTIFIKAT":
		return reqNorm == "SERTIFIKAT_PRESTASI" || reqNorm == "PRESTASI"
	}
	switch reqNorm {
	case "IJAZAH", "IJAZAH_SKL", "SKL":
		return userNorm == "IJAZAH_SKL" || userNorm == "IJAZAH"
	case "KTP", "KTP_KIA_KP", "KIA", "KARTU_PELAJAR":
		return userNorm == "KTP_KIA_KP" || userNorm == "KTP"
	case "KK", "KARTU_KELUARGA":
		return userNorm == "KK" || userNorm == "KARTU_KELUARGA"
	case "PAS_FOTO", "FOTO", "PASFOTO":
		return userNorm == "PAS_FOTO" || userNorm == "FOTO"
	case "SURAT_SEHAT", "SEHAT":
		return userNorm == "SURAT_SEHAT"
	case "RAPOR", "NILAI_RAPOR":
		return userNorm == "RAPOR" || userNorm == "NILAI_RAPOR"
	case "SERTIFIKAT_PRESTASI", "PRESTASI", "SERTIFIKAT":
		return userNorm == "SERTIFIKAT_PRESTASI" || userNorm == "PRESTASI"
	}
	return false
}

// UploadDocument memproses upload berkas persyaratan dengan validasi magic bytes MIME
func (s *admissionService) UploadDocument(
	ctx context.Context,
	appID, userID uuid.UUID,
	reqCode string,
	reader io.Reader,
	size int64,
	originalName, declaredMime string,
) (*model.ApplicationDocument, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.UserID != userID {
		return nil, ErrForbiddenAccess
	}
	if !app.CanUploadDocument() {
		return nil, ErrDocUploadNotAllowed
	}

	path, err := s.pathRepo.FindByID(ctx, app.AdmissionPathID)
	if err != nil {
		return nil, err
	}

	var reqFound *model.DocumentRequirement
	for _, req := range path.Requirements {
		if isMatchDocType(req.DocumentType, reqCode) {
			reqFound = &req
			break
		}
	}
	if reqFound == nil {
		reqNorm := strings.ToUpper(strings.TrimSpace(reqCode))
		if reqNorm == "RAPOR" || reqNorm == "NILAI_RAPOR" || reqNorm == "IJAZAH" || reqNorm == "KTP" || reqNorm == "KK" || reqNorm == "PAS_FOTO" || reqNorm == "SURAT_SEHAT" {
			reqFound = &model.DocumentRequirement{
				AdmissionPathID: app.AdmissionPathID,
				DocumentType:    reqCode,
				Name:            reqCode,
				IsRequired:      true,
				MaxSizeMB:       5,
				AllowedFormats:  "pdf,jpg,jpeg,png",
			}
		}
	}
	if reqFound == nil {
		return nil, fmt.Errorf("%w: %s", ErrDocTypeNotSupported, reqCode)
	}

	// Validasi ukuran berkas
	maxBytes := int64(reqFound.MaxSizeMB) * 1024 * 1024
	if reqFound.MaxSizeMB <= 0 {
		maxBytes = 5 * 1024 * 1024
	}
	if size > maxBytes {
		return nil, fmt.Errorf("%w: maksimal %d MB", ErrDocTooLarge, reqFound.MaxSizeMB)
	}

	// Konversi allowed formats ke daftar MIME types
	formats := strings.Split(reqFound.AllowedFormats, ",")
	var allowedMimes []string
	for _, f := range formats {
		fClean := strings.ToLower(strings.TrimSpace(f))
		switch fClean {
		case "pdf":
			allowedMimes = append(allowedMimes, "application/pdf")
		case "jpg", "jpeg":
			allowedMimes = append(allowedMimes, "image/jpeg")
		case "png":
			allowedMimes = append(allowedMimes, "image/png")
		}
	}

	var validatedReader io.Reader = reader
	var validatedMime = declaredMime

	if s.storageCli != nil {
		// Validasi Magic Bytes mendalam menggunakan SniffAndValidate
		vr, st, err := s.storageCli.SniffAndValidate(reader, allowedMimes)
		if err != nil {
			return nil, err // Mengembalikan storage.ErrFileTypeNotAllowed jika MIME palsu/tidak diizinkan
		}
		validatedReader = vr
		validatedMime = st
		_ = s.storageCli.EnsureBucket(ctx, s.cfg.DocumentBucket)
	}

	ext := "bin"
	if strings.Contains(validatedMime, "pdf") {
		ext = "pdf"
	} else if strings.Contains(validatedMime, "png") {
		ext = "png"
	} else if strings.Contains(validatedMime, "jpeg") {
		ext = "jpg"
	}

	cleanFileName := filepath.Base(originalName)
	if cleanFileName == "" || cleanFileName == "." {
		cleanFileName = fmt.Sprintf("%s.%s", reqCode, ext)
	}
	objectKey := fmt.Sprintf("documents/%s/%s_%d_%s", appID.String(), reqCode, time.Now().Unix(), cleanFileName)

	var fileURL string
	if s.storageCli != nil {
		err = s.storageCli.Upload(ctx, s.cfg.DocumentBucket, objectKey, validatedReader, size, validatedMime)
		if err != nil {
			return nil, fmt.Errorf("gagal mengunggah dokumen ke penyimpanan: %w", err)
		}
		url, err := s.storageCli.PresignedGetObject(ctx, s.cfg.DocumentBucket, objectKey, 15*time.Minute)
		if err == nil {
			fileURL = url
		}
	}

	doc := &model.ApplicationDocument{
		ID:            uuid.New(),
		ApplicationID: appID,
		DocumentType:  reqCode,
		FileName:      originalName,
		FileURL:       fileURL,
		ObjectKey:     objectKey,
		MimeType:      validatedMime,
		FileSize:      size,
		Status:        model.DocStatusUploaded,
	}

	if err := s.docRepo.Save(ctx, doc); err != nil {
		return nil, fmt.Errorf("gagal menyimpan data dokumen: %w", err)
	}

	return s.docRepo.FindByApplicationAndType(ctx, appID, reqCode)
}

// DeleteDocument menghapus dokumen yang telah diunggah
func (s *admissionService) DeleteDocument(ctx context.Context, appID, userID uuid.UUID, reqCode string) error {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return err
	}
	if app.UserID != userID {
		return ErrForbiddenAccess
	}
	if !app.CanUploadDocument() {
		return ErrDocUploadNotAllowed
	}

	doc, err := s.docRepo.FindByApplicationAndType(ctx, appID, reqCode)
	if err != nil {
		return err
	}

	if s.storageCli != nil && doc.ObjectKey != "" {
		_ = s.storageCli.DeleteObject(ctx, s.cfg.DocumentBucket, doc.ObjectKey)
	}

	return s.docRepo.DeleteByApplicationAndType(ctx, appID, reqCode)
}

// SubmitApplication memvalidasi kelengkapan biodata dan dokumen wajib sebelum mengubah status ke submitted
func (s *admissionService) SubmitApplication(ctx context.Context, appID, userID uuid.UUID) (*model.Application, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.UserID != userID {
		return nil, ErrForbiddenAccess
	}
	if !app.CanSubmit() {
		return nil, ErrSubmitNotAllowed
	}

	// 1. Validasi Kelengkapan Biodata ke Identity Service
	if s.identityCli != nil {
		var compResp struct {
			Complete bool     `json:"complete"`
			Missing  []string `json:"missing"`
			Data     *struct {
				Complete bool     `json:"complete"`
				Missing  []string `json:"missing"`
			} `json:"data"`
		}
		err := s.identityCli.Get(ctx, fmt.Sprintf("/internal/persons/%s/completeness", userID.String()), nil, &compResp)
		if err != nil {
			return nil, fmt.Errorf("gagal memeriksa kelengkapan biodata pendaftar: %w", err)
		}
		isComplete := compResp.Complete
		missing := compResp.Missing
		if compResp.Data != nil {
			isComplete = compResp.Data.Complete
			missing = compResp.Data.Missing
		}
		if !isComplete {
			return nil, fmt.Errorf("%w: %s", ErrBiodataIncomplete, strings.Join(missing, ", "))
		}
	}

	// 2. Validasi Seluruh Dokumen Wajib
	path, err := s.pathRepo.FindByID(ctx, app.AdmissionPathID)
	if err != nil {
		return nil, err
	}

	uploadedDocs, err := s.docRepo.FindByApplicationID(ctx, appID)
	if err != nil {
		return nil, err
	}

	docMap := make(map[string]model.ApplicationDocument)
	for _, d := range uploadedDocs {
		docMap[strings.ToLower(d.DocumentType)] = d
	}

	var missingDocs []string
	for _, req := range path.Requirements {
		if req.IsRequired {
			reqUpper := strings.ToUpper(strings.TrimSpace(req.DocumentType))
			if reqUpper == "SURAT_SEHAT" {
				continue
			}
			found := false
			for _, d := range uploadedDocs {
				if isMatchDocType(req.DocumentType, d.DocumentType) {
					if d.Status != model.DocStatusInvalid {
						found = true
					}
					break
				}
			}
			if !found {
				missingDocs = append(missingDocs, req.Name)
			}
		}
	}

	if len(missingDocs) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrDocumentRequired, strings.Join(missingDocs, ", "))
	}

	// 3. Update Status Pendaftaran ke submitted
	fromStatus := app.Status
	err = s.appRepo.UpdateStatus(ctx, appID, fromStatus, model.StatusSubmitted, "Pendaftaran telah disubmit oleh peserta dan menunggu verifikasi berkas", &userID)
	if err != nil {
		return nil, err
	}

	// 4. Kirim Notifikasi Submit Berhasil
	if s.notifCli != nil {
		go func() {
			notifPayload := map[string]interface{}{
				"channel":  "email",
				"user_id":  userID.String(),
				"template": "application_submitted",
				"data": map[string]interface{}{
					"registration_no": app.RegistrationNo,
					"notes":           "Pendaftaran Anda telah berhasil disubmit dan sedang menunggu verifikasi oleh panitia PMB.",
				},
			}
			var notifResp interface{}
			_ = s.notifCli.Post(context.Background(), "/internal/notifications", notifPayload, &notifResp)
		}()
	}

	return s.appRepo.FindByID(ctx, appID)
}

// VerifyApplicationDocuments memverifikasi setiap berkas pendaftar dan mengevaluasi status kelulusan berkas
func (s *admissionService) VerifyApplicationDocuments(
	ctx context.Context,
	appID, reviewerID uuid.UUID,
	reviews []DocumentReviewItem,
) (*model.Application, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.Status != model.StatusSubmitted && app.Status != model.StatusFormOpen && app.Status != model.StatusAwaitingPayment && app.Status != model.StatusDraft {
		return nil, ErrApplicationNotSubmitted
	}

	currentDocs, err := s.docRepo.FindByApplicationID(ctx, appID)
	if err != nil {
		return nil, err
	}

	for _, rev := range reviews {
		for _, d := range currentDocs {
			if isMatchDocType(rev.DocumentType, d.DocumentType) {
				_ = s.docRepo.UpdateReviewStatus(ctx, d.ID, rev.Status, rev.Notes, &reviewerID)
				break
			}
		}
	}

	updatedDocs, err := s.docRepo.FindByApplicationID(ctx, appID)
	if err != nil {
		return nil, err
	}
	path, err := s.pathRepo.FindByID(ctx, app.AdmissionPathID)
	if err != nil {
		return nil, err
	}

	hasInvalid := false
	var invalidNotes []string
	for _, d := range updatedDocs {
		if d.Status == model.DocStatusInvalid {
			hasInvalid = true
			if d.Notes != "" {
				invalidNotes = append(invalidNotes, fmt.Sprintf("%s: %s", d.DocumentType, d.Notes))
			} else {
				invalidNotes = append(invalidNotes, d.DocumentType)
			}
		}
	}

	allRequiredValid := true
	for _, req := range path.Requirements {
		if req.IsRequired {
			reqUpper := strings.ToUpper(strings.TrimSpace(req.DocumentType))
			if reqUpper == "SURAT_SEHAT" {
				continue
			}
			valid := false
			for _, d := range updatedDocs {
				if isMatchDocType(req.DocumentType, d.DocumentType) {
					if d.Status == model.DocStatusValid {
						valid = true
					}
					break
				}
			}
			if !valid {
				allRequiredValid = false
				break
			}
		}
	}

	fromStatus := app.Status

	if hasInvalid {
		notesStr := "Dokumen memerlukan perbaikan: " + strings.Join(invalidNotes, "; ")
		_ = s.appRepo.UpdateStatus(ctx, appID, fromStatus, model.StatusRevisionRequired, notesStr, &reviewerID)
		app.Status = model.StatusRevisionRequired

		if s.notifCli != nil {
			go func() {
				notifPayload := map[string]interface{}{
					"channel":  "email",
					"user_id":  app.UserID.String(),
					"template": "application_revision_required",
					"data": map[string]interface{}{
						"registration_no": app.RegistrationNo,
						"notes":           notesStr,
					},
				}
				var notifResp interface{}
				_ = s.notifCli.Post(context.Background(), "/internal/notifications", notifPayload, &notifResp)
			}()
		}
	} else if allRequiredValid || len(reviews) == 0 {
		_ = s.appRepo.UpdateStatus(ctx, appID, fromStatus, model.StatusVerified, "Seluruh dokumen persyaratan telah diverifikasi dan valid", &reviewerID)
		app.Status = model.StatusVerified

		if s.notifCli != nil {
			go func() {
				notifPayload := map[string]interface{}{
					"channel":  "email",
					"user_id":  app.UserID.String(),
					"template": "application_verified",
					"data": map[string]interface{}{
						"registration_no": app.RegistrationNo,
						"notes":           "Selamat! Berkas pendaftaran Anda telah berhasil diverifikasi oleh panitia PMB.",
					},
				}
				var notifResp interface{}
				_ = s.notifCli.Post(context.Background(), "/internal/notifications", notifPayload, &notifResp)
			}()
		}
	}

	return s.appRepo.FindByID(ctx, appID)
}

// GetDocumentPresignedURL mengembalikan presigned URL (5 menit / 300 detik untuk admin)
func (s *admissionService) GetDocumentPresignedURL(ctx context.Context, appID uuid.UUID, reqCode string, requesterID uuid.UUID, role string) (string, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return "", err
	}

	expiry := 15 * time.Minute
	if role == "superadmin" || role == "admin_pmb" {
		expiry = 300 * time.Second // Kedaluwarsa tepat 5 menit (300 detik) sesuai requirement
	} else {
		if app.UserID != requesterID {
			return "", ErrForbiddenAccess
		}
	}

	doc, err := s.docRepo.FindByApplicationAndType(ctx, appID, reqCode)
	if err != nil {
		return "", err
	}

	if s.storageCli == nil {
		return doc.FileURL, nil
	}

	return s.storageCli.PresignedGetObject(ctx, s.cfg.DocumentBucket, doc.ObjectKey, expiry)
}

// ExportApplicationsXLSX mengekspor seluruh data pendaftar ke spreadsheet Excel
func (s *admissionService) ExportApplicationsXLSX(ctx context.Context, filter repository.ApplicationFilter) ([]byte, error) {
	apps, err := s.appRepo.ListAllForExport(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data pendaftaran untuk ekspor: %w", err)
	}

	return exporter.GenerateApplicationsExcel(apps)
}

// GetStats mengembalikan agregasi data statistik pendaftaran
func (s *admissionService) GetStats(ctx context.Context) (*model.AdmissionStats, error) {
	return s.appRepo.GetStats(ctx)
}

// OutboxDispatcher memproses event outbox ke Finance Service (POST /internal/invoices)
func OutboxDispatcher(financeCli internalclient.Client) outbox.DispatcherFunc {
	return func(ctx context.Context, event *outbox.Event) error {
		if event.EventType == "invoice.create" {
			if financeCli == nil {
				return errors.New("finance client belum siap atau belum aktif")
			}
			var resp interface{}
			err := financeCli.Post(ctx, "/internal/invoices", event.Payload, &resp)
			if err != nil {
				log.Printf("[OUTBOX-WORKER] Finance service belum tersedia, event %s ditahan untuk retry: %v", event.ID, err)
				return err
			}
			log.Printf("[OUTBOX-WORKER] Berhasil mengirim invoice untuk event %s", event.ID)
			return nil
		}
		return nil
	}
}
