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

type AdmissionService interface {
	CreateApplication(ctx context.Context, req *CreateApplicationRequest) (*model.Application, error)
	ChangeChoices(ctx context.Context, req *ChangeChoicesRequest) error
	GetMyApplications(ctx context.Context, userID uuid.UUID) ([]model.Application, error)
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
}

type admissionService struct {
	periodRepo  repository.PeriodRepository
	pathRepo    repository.AdmissionPathRepository
	quotaRepo   repository.QuotaRepository
	appRepo     repository.ApplicationRepository
	docRepo     repository.ApplicationDocumentRepository
	outboxRepo  outbox.Repository
	notifCli    internalclient.Client
	identityCli internalclient.Client
	financeCli  internalclient.Client
	storageCli  storage.Client
	cfg         config.Config
}

func NewAdmissionService(
	periodRepo repository.PeriodRepository,
	pathRepo repository.AdmissionPathRepository,
	quotaRepo repository.QuotaRepository,
	appRepo repository.ApplicationRepository,
	docRepo repository.ApplicationDocumentRepository,
	outboxRepo outbox.Repository,
	notifCli internalclient.Client,
	identityCli internalclient.Client,
	financeCli internalclient.Client,
	storageCli storage.Client,
	cfg config.Config,
) AdmissionService {
	return &admissionService{
		periodRepo:  periodRepo,
		pathRepo:    pathRepo,
		quotaRepo:   quotaRepo,
		appRepo:     appRepo,
		docRepo:     docRepo,
		outboxRepo:  outboxRepo,
		notifCli:    notifCli,
		identityCli: identityCli,
		financeCli:  financeCli,
		storageCli:  storageCli,
		cfg:         cfg,
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

	// 3. Cek Duplikasi Pendaftaran
	existing, err := s.appRepo.FindByUserIDAndPeriod(ctx, req.UserID, req.PeriodID)
	if err == nil && existing != nil {
		return nil, ErrDuplicateApplication
	}

	var newApp *model.Application

	// 4. Eksekusi dalam Database Transaction
	err = s.appRepo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		regNo, genErr := s.appRepo.GenerateRegistrationNo(ctx, tx, period.ID, period.AcademicYear)
		if genErr != nil {
			return fmt.Errorf("gagal membuat nomor registrasi: %w", genErr)
		}

		app := &model.Application{
			ID:              uuid.New(),
			RegistrationNo:  regNo,
			UserID:          req.UserID,
			PeriodID:        req.PeriodID,
			AdmissionPathID: req.AdmissionPathID,
			Choice1ID:       req.Choice1ID,
			Choice2ID:       req.Choice2ID,
			Status:          model.StatusAwaitingPayment,
		}

		if err := s.appRepo.CreateInTx(tx, app); err != nil {
			return fmt.Errorf("gagal menyimpan berkas pendaftaran: %w", err)
		}

		statusLog := &model.ApplicationStatusLog{
			ID:            uuid.New(),
			ApplicationID: app.ID,
			FromStatus:    model.StatusDraft,
			ToStatus:      model.StatusAwaitingPayment,
			ChangedBy:     &req.UserID,
			Notes:         "Pendaftaran berhasil dibuat dan menunggu pembayaran tagihan",
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
			var personResp struct {
				Data interface{} `json:"data"`
			}
			err := s.identityCli.Get(ctx, fmt.Sprintf("/internal/persons/%s", app.UserID.String()), nil, &personResp)
			if err == nil {
				app.PersonData = personResp.Data
			}
		}

		if s.financeCli != nil {
			var invResp struct {
				Data interface{} `json:"data"`
			}
			err := s.financeCli.Get(ctx, fmt.Sprintf("/internal/invoices?application_id=%s", app.ID.String()), nil, &invResp)
			if err == nil {
				app.InvoiceData = invResp.Data
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
		if strings.EqualFold(req.DocumentType, reqCode) {
			reqFound = &req
			break
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
			Data struct {
				Complete bool     `json:"complete"`
				Missing  []string `json:"missing"`
			} `json:"data"`
		}
		err := s.identityCli.Get(ctx, fmt.Sprintf("/internal/persons/%s/completeness", userID.String()), nil, &compResp)
		if err != nil {
			return nil, fmt.Errorf("gagal memeriksa kelengkapan biodata pendaftar: %w", err)
		}
		if !compResp.Data.Complete {
			return nil, fmt.Errorf("%w: %s", ErrBiodataIncomplete, strings.Join(compResp.Data.Missing, ", "))
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
			d, ok := docMap[strings.ToLower(req.DocumentType)]
			if !ok || d.Status == model.DocStatusInvalid {
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
	if app.Status != model.StatusSubmitted {
		return nil, ErrApplicationNotSubmitted
	}

	for _, rev := range reviews {
		doc, err := s.docRepo.FindByApplicationAndType(ctx, appID, rev.DocumentType)
		if err != nil {
			continue
		}
		_ = s.docRepo.UpdateReviewStatus(ctx, doc.ID, rev.Status, rev.Notes, &reviewerID)
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
	updatedDocMap := make(map[string]model.ApplicationDocument)
	for _, d := range updatedDocs {
		updatedDocMap[strings.ToLower(d.DocumentType)] = d
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
			d, ok := updatedDocMap[strings.ToLower(req.DocumentType)]
			if !ok || d.Status != model.DocStatusValid {
				allRequiredValid = false
				break
			}
		}
	}

	fromStatus := app.Status

	if hasInvalid {
		notesStr := "Dokumen memerlukan perbaikan: " + strings.Join(invalidNotes, "; ")
		_ = s.appRepo.UpdateStatus(ctx, appID, fromStatus, model.StatusRevisionRequired, notesStr, &reviewerID)

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
	} else if allRequiredValid {
		_ = s.appRepo.UpdateStatus(ctx, appID, fromStatus, model.StatusVerified, "Seluruh dokumen persyaratan telah diverifikasi dan valid", &reviewerID)

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
