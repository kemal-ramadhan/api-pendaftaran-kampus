package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrInstallmentNotAllowed          = errors.New("INSTALLMENT_NOT_ALLOWED: pengajuan cicilan hanya diizinkan untuk tagihan biaya daftar ulang")
	ErrInvalidInstallmentTerms        = errors.New("jumlah termin cicilan tidak valid (minimal 2 dan maksimal 4 termin)")
	ErrInstallmentRequestAlreadyExist = errors.New("pengajuan cicilan untuk tagihan ini sudah ada dan sedang diproses atau telah disetujui")
	ErrInstallmentAlreadyProcessed    = errors.New("pengajuan cicilan ini sudah selesai diproses")
	ErrInstallmentTermNotFound        = errors.New("termin cicilan yang dipilih tidak ditemukan")
	ErrInstallmentAlreadyPaid         = errors.New("termin cicilan ini sudah terbayar")
)

type InstallmentService interface {
	RequestInstallment(ctx context.Context, invoiceID, userID uuid.UUID, requestedTerms int, reason string) (*model.InstallmentRequest, error)
	ReviewInstallmentRequest(ctx context.Context, requestID, reviewerID uuid.UUID, action string, reviewNote string) (*model.InstallmentRequest, error)
	ListInstallmentRequests(ctx context.Context, status string) ([]model.InstallmentRequest, error)
	GetInstallmentsByInvoiceID(ctx context.Context, invoiceID, requesterID uuid.UUID, role string) ([]model.Installment, error)
	ProcessDueRemindersAndExpirations(ctx context.Context) (remindersSent int, expiredMarked int, err error)
}

type installmentService struct {
	installmentRepo repository.InstallmentRepository
	invoiceRepo     repository.InvoiceRepository
	feeSchemeRepo   repository.FeeSchemeRepository
	notifCli        internalclient.Client
}

func NewInstallmentService(
	installmentRepo repository.InstallmentRepository,
	invoiceRepo repository.InvoiceRepository,
	feeSchemeRepo repository.FeeSchemeRepository,
	notifCli internalclient.Client,
) InstallmentService {
	return &installmentService{
		installmentRepo: installmentRepo,
		invoiceRepo:     invoiceRepo,
		feeSchemeRepo:   feeSchemeRepo,
		notifCli:        notifCli,
	}
}

func (s *installmentService) RequestInstallment(ctx context.Context, invoiceID, userID uuid.UUID, requestedTerms int, reason string) (*model.InstallmentRequest, error) {
	// 1. Validasi Keberadaan dan Kepemilikan Invoice
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.UserID != userID {
		return nil, ErrForbiddenAccess
	}

	// 2. Validasi Jenis Biaya (Hanya Reregistration yang Boleh Dicicil)
	if inv.FeeType != "reregistration" {
		return nil, ErrInstallmentNotAllowed
	}

	// 3. Validasi Status Tagihan (Hanya tagihan belum lunas)
	if inv.Status == model.InvoiceStatusPaid || inv.Status == model.InvoiceStatusCancelled {
		return nil, ErrInvoiceAlreadyPaid
	}

	// 4. Validasi Batasan Termin (Default 2 - 4 Termin)
	if requestedTerms < 2 || requestedTerms > 4 {
		return nil, ErrInvalidInstallmentTerms
	}

	// 5. Cek apakah sudah pernah mengajukan cicilan
	existingReq, err := s.installmentRepo.FindRequestByInvoiceID(ctx, invoiceID)
	if err == nil && existingReq != nil {
		if existingReq.Status == model.InstallmentRequestPending || existingReq.Status == model.InstallmentRequestApproved {
			return nil, ErrInstallmentRequestAlreadyExist
		}
	}

	// 6. Buat pengajuan cicilan baru
	req := &model.InstallmentRequest{
		ID:             uuid.New(),
		InvoiceID:      invoiceID,
		RequestedTerms: requestedTerms,
		Reason:         reason,
		Status:         model.InstallmentRequestPending,
	}

	if err := s.installmentRepo.CreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("gagal menyimpan pengajuan cicilan: %w", err)
	}

	return req, nil
}

func (s *installmentService) ReviewInstallmentRequest(ctx context.Context, requestID, reviewerID uuid.UUID, action string, reviewNote string) (*model.InstallmentRequest, error) {
	req, err := s.installmentRepo.FindRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if req.Status != model.InstallmentRequestPending {
		return nil, ErrInstallmentAlreadyProcessed
	}

	inv, err := s.invoiceRepo.FindByID(ctx, req.InvoiceID)
	if err != nil {
		return nil, err
	}

	if action == "reject" {
		req.Status = model.InstallmentRequestRejected
		req.ReviewedBy = &reviewerID
		req.ReviewNote = reviewNote
		if err := s.installmentRepo.UpdateRequest(ctx, req); err != nil {
			return nil, err
		}

		// Kirim notifikasi penolakan cicilan
		if s.notifCli != nil {
			_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
				"recipient_user_id": inv.UserID,
				"title":             "Pemberitahuan Pengajuan Cicilan Biaya Daftar Ulang",
				"message":           fmt.Sprintf("Pengajuan cicilan untuk tagihan %s belum disetujui: %s", inv.InvoiceNumber, reviewNote),
				"type":              "installment_rejected",
			}, nil)
		}
		return req, nil
	}

	// Approval Flow: Hitung Pembagian Termin Integer Presisi Tanpa Selisih
	req.Status = model.InstallmentRequestApproved
	req.ReviewedBy = &reviewerID
	req.ReviewNote = reviewNote

	totalAmount := inv.Amount
	numTerms := req.RequestedTerms
	baseTermAmount := totalAmount / int64(numTerms)
	remainder := totalAmount % int64(numTerms)

	now := time.Now()
	baseDueDate := inv.DueDate
	if baseDueDate.Before(now) {
		baseDueDate = now.AddDate(0, 0, 7) // Berikan toleransi 7 hari jika tagihan lama telah lewat
	}

	installments := make([]model.Installment, numTerms)
	for i := 0; i < numTerms; i++ {
		amount := baseTermAmount
		// Sisa pembulatan integer rupiah ditambahkan ke termin pertama agar total akumulasi persis sama
		if i == 0 {
			amount += remainder
		}

		dueDate := baseDueDate.AddDate(0, 0, i*30) // Jatuh tempo bertahap per 30 hari

		installments[i] = model.Installment{
			ID:        uuid.New(),
			InvoiceID: inv.ID,
			TermNo:    i + 1,
			Amount:    amount,
			DueDate:   dueDate,
			Status:    model.InstallmentStatusUnpaid,
		}
	}

	// Jalankan dalam transaksi: update request, buat installments, ubah payment_scheme invoice
	runInTx := func(fn func(tx *gorm.DB) error) error {
		if s.invoiceRepo.DB() != nil {
			return s.invoiceRepo.DB().WithContext(ctx).Transaction(fn)
		}
		return fn(nil)
	}

	txErr := runInTx(func(tx *gorm.DB) error {
		if tx != nil {
			if err := tx.Save(req).Error; err != nil {
				return err
			}

			if err := s.installmentRepo.CreateInstallments(ctx, tx, installments); err != nil {
				return err
			}

			if err := tx.Model(&model.Invoice{}).Where("id = ?", inv.ID).
				Updates(map[string]interface{}{
					"payment_scheme": "installment",
					"updated_at":     time.Now(),
				}).Error; err != nil {
				return err
			}
		} else {
			if err := s.installmentRepo.UpdateRequest(ctx, req); err != nil {
				return err
			}
			if err := s.installmentRepo.CreateInstallments(ctx, nil, installments); err != nil {
				return err
			}
			inv.PaymentScheme = "installment"
			inv.UpdatedAt = time.Now()
			_ = s.invoiceRepo.UpdateStatus(ctx, inv.ID, inv.Status, inv.PaidAmount)
		}

		return nil
	})

	if txErr != nil {
		return nil, fmt.Errorf("gagal memproses persetujuan cicilan: %w", txErr)
	}

	// Kirim notifikasi persetujuan cicilan
	if s.notifCli != nil {
		_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
			"recipient_user_id": inv.UserID,
			"title":             "Persetujuan Jadwal Cicilan Biaya Daftar Ulang",
			"message":           fmt.Sprintf("Pengajuan cicilan untuk tagihan %s telah disetujui sebanyak %d termin.", inv.InvoiceNumber, numTerms),
			"type":              "installment_approved",
		}, nil)
	}

	return req, nil
}

func (s *installmentService) ListInstallmentRequests(ctx context.Context, status string) ([]model.InstallmentRequest, error) {
	return s.installmentRepo.ListRequests(ctx, status)
}

func (s *installmentService) GetInstallmentsByInvoiceID(ctx context.Context, invoiceID, requesterID uuid.UUID, role string) ([]model.Installment, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	if role != "admin_pmb" && role != "superadmin" && role != "keuangan" && role != "pimpinan" {
		if inv.UserID != requesterID {
			return nil, ErrForbiddenAccess
		}
	}

	return s.installmentRepo.FindInstallmentsByInvoiceID(ctx, invoiceID)
}

func (s *installmentService) ProcessDueRemindersAndExpirations(ctx context.Context) (remindersSent int, expiredMarked int, err error) {
	now := time.Now()

	// 1. Reminder H-3 (Antara 2 s/d 3 hari ke depan)
	h3Start := now.AddDate(0, 0, 2)
	h3End := now.AddDate(0, 0, 3)
	dueH3List, _ := s.installmentRepo.FindDueInstallments(ctx, h3Start, h3End)
	for _, inst := range dueH3List {
		inv, err := s.invoiceRepo.FindByID(ctx, inst.InvoiceID)
		if err == nil && inv != nil && s.notifCli != nil {
			_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
				"recipient_user_id": inv.UserID,
				"title":             fmt.Sprintf("Pengingat Jatuh Tempo Cicilan Termin %d (H-3)", inst.TermNo),
				"message":           fmt.Sprintf("Cicilan termin %d sebesar Rp %d untuk tagihan %s akan jatuh tempo pada %s.", inst.TermNo, inst.Amount, inv.InvoiceNumber, inst.DueDate.Format("02-01-2006")),
				"type":              "installment_reminder_h3",
			}, nil)
			remindersSent++
		}
	}

	// 2. Reminder H-1 (Antara 0 s/d 1 hari ke depan)
	h1Start := now
	h1End := now.AddDate(0, 0, 1)
	dueH1List, _ := s.installmentRepo.FindDueInstallments(ctx, h1Start, h1End)
	for _, inst := range dueH1List {
		inv, err := s.invoiceRepo.FindByID(ctx, inst.InvoiceID)
		if err == nil && inv != nil && s.notifCli != nil {
			_ = s.notifCli.Post(ctx, "/internal/notifications", map[string]interface{}{
				"recipient_user_id": inv.UserID,
				"title":             fmt.Sprintf("Pengingat Jatuh Tempo Cicilan Termin %d (H-1)", inst.TermNo),
				"message":           fmt.Sprintf("Cicilan termin %d sebesar Rp %d untuk tagihan %s akan jatuh tempo BESOK (%s). Segera lakukan pembayaran.", inst.TermNo, inst.Amount, inv.InvoiceNumber, inst.DueDate.Format("02-01-2006")),
				"type":              "installment_reminder_h1",
			}, nil)
			remindersSent++
		}
	}

	// 3. Tandai Expired untuk yang telah melewati batas due date
	expiredList, _ := s.installmentRepo.FindExpiredInstallments(ctx, now)
	for _, inst := range expiredList {
		inst.Status = model.InstallmentStatusExpired
		if err := s.installmentRepo.UpdateInstallment(ctx, &inst); err == nil {
			expiredMarked++
		}
	}

	return remindersSent, expiredMarked, nil
}
