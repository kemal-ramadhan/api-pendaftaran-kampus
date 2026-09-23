package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/outbox"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrPeriodClosed         = errors.New("periode pendaftaran telah ditutup atau belum dibuka")
	ErrDuplicateApplication = errors.New("Anda sudah memiliki pendaftaran aktif pada periode ini")
	ErrInvalidState         = errors.New("pilihan program studi tidak dapat diubah pada status pendaftaran saat ini")
	ErrForbiddenAccess      = errors.New("Anda tidak memiliki akses ke data pendaftaran ini")
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

type AdmissionService interface {
	CreateApplication(ctx context.Context, req *CreateApplicationRequest) (*model.Application, error)
	ChangeChoices(ctx context.Context, req *ChangeChoicesRequest) error
	GetMyApplications(ctx context.Context, userID uuid.UUID) ([]model.Application, error)
	GetApplicationDetail(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string, prodiID *uuid.UUID) (*model.Application, error)
	ListApplicationsAdmin(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, int64, error)
	GetActivePeriod(ctx context.Context) (*model.Period, error)
	GetAdmissionPaths(ctx context.Context) ([]model.AdmissionPath, error)
	UpdateApplicationStatus(ctx context.Context, id uuid.UUID, newStatus model.ApplicationStatus, notes string, changedBy *uuid.UUID) error
}

type admissionService struct {
	periodRepo repository.PeriodRepository
	pathRepo   repository.AdmissionPathRepository
	quotaRepo  repository.QuotaRepository
	appRepo    repository.ApplicationRepository
	outboxRepo outbox.Repository
	notifCli   internalclient.Client
}

func NewAdmissionService(
	periodRepo repository.PeriodRepository,
	pathRepo repository.AdmissionPathRepository,
	quotaRepo repository.QuotaRepository,
	appRepo repository.ApplicationRepository,
	outboxRepo outbox.Repository,
	notifCli internalclient.Client,
) AdmissionService {
	return &admissionService{
		periodRepo: periodRepo,
		pathRepo:   pathRepo,
		quotaRepo:  quotaRepo,
		appRepo:    appRepo,
		outboxRepo: outboxRepo,
		notifCli:   notifCli,
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
		// Generate registration number: PMB-{YYYY}-{period_id}-{urut 5 digit}
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

		// Catat log status awal
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

		// Buat Transactional Outbox event untuk Finance Service (POST /internal/invoices)
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

	// 5. Trigger Notifikasi Email (Asinkron / Best-effort)
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

	// Ambil data lengkap beserta relasi
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
		return app, nil
	case "prodi":
		if prodiID == nil {
			return nil, ErrForbiddenAccess
		}
		// Staf prodi hanya boleh melihat jika aplikasi memilih prodinya
		if app.Choice1ID != *prodiID && (app.Choice2ID == nil || *app.Choice2ID != *prodiID) {
			return nil, ErrForbiddenAccess
		}
		return app, nil
	default:
		// Peserta hanya boleh melihat pendaftaran miliknya
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

// OutboxDispatcher memproses event outbox ke Finance Service (POST /internal/invoices)
// Pada Tahap 3, finance service belum aktif, pemanggilan akan mengembalikan error dan event tertahan (retry)
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
