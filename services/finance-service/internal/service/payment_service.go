package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/pdf"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/provider"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrInvalidSignature    = errors.New("signature webhook tidak valid")
	ErrDevEndpointDisabled = errors.New("endpoint simulasi bayar hanya tersedia pada lingkungan development")
	ErrPaymentAlreadyPaid  = errors.New("pembayaran ini sudah sukses diselesaikan")
)

type PaymentService interface {
	SetInstallmentRepo(repo repository.InstallmentRepository)
	PayWithVirtualAccount(ctx context.Context, invoiceID, userID uuid.UUID) (*model.Payment, error)
	PayWithManualTransfer(ctx context.Context, invoiceID, userID uuid.UUID, proofBytes []byte, fileName string) (*model.Payment, error)
	ProcessWebhook(ctx context.Context, providerName string, signature string, payloadBytes []byte) error
	SimulatePaid(ctx context.Context, paymentID uuid.UUID) (*model.Payment, error)
	VerifyManualTransfer(ctx context.Context, paymentID uuid.UUID, action, notes string, verifiedBy uuid.UUID) error
	GetPaymentByID(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*model.Payment, error)
	GetReceiptByPaymentID(ctx context.Context, paymentID uuid.UUID, requesterID uuid.UUID, role string) (*model.Receipt, []byte, error)
	ListAwaitingVerification(ctx context.Context, limit, offset int) ([]model.Payment, int64, error)
	ReconcilePendingPayments(ctx context.Context) (int, error)
	GetPaymentMethods(ctx context.Context, amount int64) ([]provider.PaymentMethodInfo, error)
}

type paymentService struct {
	paymentRepo     repository.PaymentRepository
	invoiceRepo     repository.InvoiceRepository
	receiptRepo     repository.ReceiptRepository
	webhookLogRepo  repository.WebhookLogRepository
	installmentRepo repository.InstallmentRepository
	provider        provider.PaymentProvider
	storageCli      storage.Client
	admissionCli    internalclient.Client
	notifCli        internalclient.Client
	cfg             *config.Config
}

func NewPaymentService(
	paymentRepo repository.PaymentRepository,
	invoiceRepo repository.InvoiceRepository,
	receiptRepo repository.ReceiptRepository,
	webhookLogRepo repository.WebhookLogRepository,
	paymentProvider provider.PaymentProvider,
	storageCli storage.Client,
	admissionCli internalclient.Client,
	notifCli internalclient.Client,
	cfg *config.Config,
) PaymentService {
	return &paymentService{
		paymentRepo:    paymentRepo,
		invoiceRepo:    invoiceRepo,
		receiptRepo:    receiptRepo,
		webhookLogRepo: webhookLogRepo,
		provider:       paymentProvider,
		storageCli:     storageCli,
		admissionCli:   admissionCli,
		notifCli:       notifCli,
		cfg:            cfg,
	}
}

func (s *paymentService) SetInstallmentRepo(repo repository.InstallmentRepository) {
	s.installmentRepo = repo
}

func (s *paymentService) PayWithVirtualAccount(ctx context.Context, invoiceID, userID uuid.UUID) (*model.Payment, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	if inv.UserID != userID {
		return nil, ErrForbiddenAccess
	}

	if inv.IsPaid() {
		return nil, ErrInvoiceAlreadyPaid
	}

	// Cek apakah invoice adalah skema cicilan
	var targetInstallment *model.Installment
	payAmount := inv.Amount
	if inv.PaymentScheme == "installment" && s.installmentRepo != nil {
		installments, err := s.installmentRepo.FindInstallmentsByInvoiceID(ctx, inv.ID)
		if err == nil {
			for _, inst := range installments {
				if inst.Status == model.InstallmentStatusUnpaid {
					targetInstallment = &inst
					payAmount = inst.Amount
					break
				}
			}
		}
	}

	// Cek apakah sudah ada payment pending dengan VA untuk termin/invoice ini
	for _, p := range inv.Payments {
		if p.Method == model.PaymentMethodVirtualAccount && p.Status == model.PaymentStatusPending {
			if targetInstallment != nil {
				if p.InstallmentID != nil && *p.InstallmentID == targetInstallment.ID {
					return &p, nil
				}
			} else {
				return &p, nil
			}
		}
	}

	// Inisiasi VA baru via Provider
	vaResult, err := s.provider.CreateVA(ctx, invoiceID, payAmount)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat Virtual Account: %w", err)
	}

	payment := &model.Payment{
		ID:        uuid.New(),
		InvoiceID: inv.ID,
		InstallmentID: func() *uuid.UUID {
			if targetInstallment != nil {
				return &targetInstallment.ID
			}
			return nil
		}(),
		Method:      model.PaymentMethodVirtualAccount,
		Provider:    "mock",
		ProviderRef: vaResult.ProviderRef,
		VANumber:    vaResult.VANumber,
		Amount:      payAmount,
		Status:      model.PaymentStatusPending,
	}

	if err := s.paymentRepo.CreateInTx(s.paymentRepo.DB(), payment); err != nil {
		return nil, fmt.Errorf("gagal menyimpan data pembayaran: %w", err)
	}

	return s.paymentRepo.FindByID(ctx, payment.ID)
}

func (s *paymentService) PayWithManualTransfer(ctx context.Context, invoiceID, userID uuid.UUID, proofBytes []byte, fileName string) (*model.Payment, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	if inv.UserID != userID {
		return nil, ErrForbiddenAccess
	}

	if inv.IsPaid() {
		return nil, ErrInvoiceAlreadyPaid
	}

	var targetInstallment *model.Installment
	payAmount := inv.Amount
	if inv.PaymentScheme == "installment" && s.installmentRepo != nil {
		installments, err := s.installmentRepo.FindInstallmentsByInvoiceID(ctx, inv.ID)
		if err == nil {
			for _, inst := range installments {
				if inst.Status == model.InstallmentStatusUnpaid {
					targetInstallment = &inst
					payAmount = inst.Amount
					break
				}
			}
		}
	}

	objectKey := fmt.Sprintf("proofs/%s-%s", uuid.New().String()[:8], fileName)
	var proofURL string

	if s.storageCli != nil {
		bucket := "pmb-finance-proofs"
		uploadErr := s.storageCli.Upload(ctx, bucket, objectKey, bytes.NewReader(proofBytes), int64(len(proofBytes)), "image/jpeg")
		if uploadErr != nil {
			log.Printf("[PAYMENT-SERVICE] Peringatan: gagal unggah bukti transfer ke MinIO: %v", uploadErr)
		} else {
			proofURL, _ = s.storageCli.PresignedGetObject(ctx, bucket, objectKey, 1*time.Hour)
		}
	}

	providerRef := fmt.Sprintf("MANUAL-%s", uuid.New().String()[:8])

	payment := &model.Payment{
		ID:        uuid.New(),
		InvoiceID: inv.ID,
		InstallmentID: func() *uuid.UUID {
			if targetInstallment != nil {
				return &targetInstallment.ID
			}
			return nil
		}(),
		Method:         model.PaymentMethodManualTransfer,
		Provider:       "manual",
		ProviderRef:    providerRef,
		Amount:         payAmount,
		Status:         model.PaymentStatusAwaitingVerification,
		ProofObjectKey: objectKey,
		ProofURL:       proofURL,
	}

	if err := s.paymentRepo.CreateInTx(s.paymentRepo.DB(), payment); err != nil {
		return nil, fmt.Errorf("gagal menyimpan pembayaran transfer manual: %w", err)
	}

	return s.paymentRepo.FindByID(ctx, payment.ID)
}

func (s *paymentService) ProcessWebhook(ctx context.Context, providerName string, signature string, payloadBytes []byte) error {
	// 1. Verifikasi Signature Provider
	if !s.provider.VerifySignature(s.cfg.PaymentWebhookSecret, signature, payloadBytes) {
		_ = s.webhookLogRepo.Create(ctx, &model.PaymentWebhookLog{
			Provider:     providerName,
			ProviderRef:  "-",
			Payload:      string(payloadBytes),
			Signature:    signature,
			Status:       "rejected",
			ErrorMessage: "Signature tidak valid",
		})
		return ErrInvalidSignature
	}

	// 2. Parse Webhook
	payload, err := s.provider.ParseWebhook(ctx, payloadBytes)
	if err != nil {
		return err
	}

	// 3. Idempotency Check: Cek apakah provider_ref sudah pernah sukses diproses
	existingLog, _ := s.webhookLogRepo.FindByProviderRef(ctx, payload.ProviderRef)
	if existingLog != nil {
		log.Printf("[WEBHOOK] Ref %s sudah pernah diproses sebelumnya (idempoten)", payload.ProviderRef)
		return nil
	}

	// 4. Cari Payment
	payment, err := s.paymentRepo.FindByProviderRef(ctx, payload.ProviderRef)
	if err != nil {
		return fmt.Errorf("pembayaran dengan referensi %s tidak ditemukan: %w", payload.ProviderRef, err)
	}

	if payment.Status == model.PaymentStatusSuccess {
		return nil
	}

	inv, err := s.invoiceRepo.FindByID(ctx, payment.InvoiceID)
	if err != nil {
		return err
	}

	// 5. Penanganan Pembayaran Pasca Pembatalan (CR-002 D.5 point 3)
	if payment.Status == model.PaymentStatusCancelled || inv.Status == model.InvoiceStatusCancelled {
		log.Printf("[WEBHOOK] Pembayaran %s diterima setelah pembatalan. Menandai paid_after_cancel.", payment.ID)
		now := time.Now()
		payment.Status = model.PaymentStatusPaidAfterCancel
		payment.PaidAt = &now
		payment.Notes = "Pembayaran sukses diterima setelah tagihan dibatalkan (perlu penanganan manual/refund)"
		_ = s.paymentRepo.Update(ctx, payment)

		_ = s.webhookLogRepo.Create(ctx, &model.PaymentWebhookLog{
			Provider:    providerName,
			ProviderRef: payload.ProviderRef,
			Payload:     string(payloadBytes),
			Signature:   signature,
			Status:      "paid_after_cancel",
		})

		// Kirim notifikasi ke tim keuangan untuk ditindaklanjuti secara manual
		if s.notifCli != nil {
			go func() {
				notifPayload := map[string]interface{}{
					"channel":  "email",
					"template": "payment_paid_after_cancel",
					"data": map[string]interface{}{
						"invoice_number": inv.InvoiceNumber,
						"payment_id":     payment.ID.String(),
						"amount":         payment.Amount,
						"provider_ref":   payment.ProviderRef,
					},
				}
				var resp interface{}
				_ = s.notifCli.Post(context.Background(), "/internal/notifications", notifPayload, &resp)
			}()
		}
		return nil
	}

	// 6. Finalisasi Pelunasan Pembayaran Normal
	_, err = s.finalizePaymentSuccess(ctx, payment, inv)
	if err != nil {
		return err
	}

	// 6. Catat Webhook Log Sukses
	_ = s.webhookLogRepo.Create(ctx, &model.PaymentWebhookLog{
		Provider:    providerName,
		ProviderRef: payload.ProviderRef,
		Payload:     string(payloadBytes),
		Signature:   signature,
		Status:      "processed",
	})

	return nil
}

func (s *paymentService) SimulatePaid(ctx context.Context, paymentID uuid.UUID) (*model.Payment, error) {
	if !s.cfg.IsDevelopment() {
		return nil, ErrDevEndpointDisabled
	}

	payment, err := s.paymentRepo.FindByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if payment.Status == model.PaymentStatusSuccess {
		return payment, nil
	}

	inv, err := s.invoiceRepo.FindByID(ctx, payment.InvoiceID)
	if err != nil {
		return nil, err
	}

	_, err = s.finalizePaymentSuccess(ctx, payment, inv)
	if err != nil {
		return nil, err
	}

	return s.paymentRepo.FindByID(ctx, paymentID)
}

func (s *paymentService) VerifyManualTransfer(ctx context.Context, paymentID uuid.UUID, action, notes string, verifiedBy uuid.UUID) error {
	payment, err := s.paymentRepo.FindByID(ctx, paymentID)
	if err != nil {
		return err
	}

	if payment.Status == model.PaymentStatusSuccess {
		return ErrPaymentAlreadyPaid
	}

	inv, err := s.invoiceRepo.FindByID(ctx, payment.InvoiceID)
	if err != nil {
		return err
	}

	if action == "approve" {
		_, err := s.finalizePaymentSuccess(ctx, payment, inv)
		return err
	} else if action == "reject" {
		now := time.Now()
		var tx *gorm.DB
		if s.paymentRepo.DB() != nil {
			tx = s.paymentRepo.DB()
		}
		return s.paymentRepo.UpdateStatusInTx(tx, payment.ID, model.PaymentStatusRejected, notes, &verifiedBy, &now)
	}

	return fmt.Errorf("aksi tidak valid: %s (gunakan 'approve' atau 'reject')", action)
}

func (s *paymentService) finalizePaymentSuccess(ctx context.Context, payment *model.Payment, inv *model.Invoice) (*model.Receipt, error) {
	now := time.Now()
	var receipt *model.Receipt

	runInTx := func(fn func(tx *gorm.DB) error) error {
		if s.paymentRepo.DB() != nil {
			return s.paymentRepo.DB().WithContext(ctx).Transaction(fn)
		}
		return fn(nil)
	}

	err := runInTx(func(tx *gorm.DB) error {
		// 1. Update Payment Status
		if err := s.paymentRepo.UpdateStatusInTx(tx, payment.ID, model.PaymentStatusSuccess, "Pembayaran berhasil diverifikasi", nil, &now); err != nil {
			return err
		}

		// Update Installment status jika pembayaran termin
		if payment.InstallmentID != nil && s.installmentRepo != nil {
			inst, err := s.installmentRepo.FindInstallmentByID(ctx, *payment.InstallmentID)
			if err == nil && inst != nil {
				inst.Status = model.InstallmentStatusPaid
				inst.PaidAt = &now
				_ = s.installmentRepo.UpdateInstallment(ctx, inst)
			}
		}

		// 2. Update Invoice Status & Paid Amount
		newPaidAmount := inv.PaidAmount + payment.Amount
		newStatus := model.InvoiceStatusPaid
		if newPaidAmount < inv.Amount {
			newStatus = model.InvoiceStatusPartiallyPaid
		}

		if err := s.invoiceRepo.UpdateStatus(ctx, inv.ID, newStatus, newPaidAmount); err != nil {
			return err
		}

		// 3. Generate Receipt Number
		rcNo, err := s.receiptRepo.GenerateReceiptNumber(ctx, tx)
		if err != nil {
			return err
		}

		// 4. Generate Kwitansi PDF
		pdfData := pdf.ReceiptData{
			ReceiptNumber: rcNo,
			InvoiceNumber: inv.InvoiceNumber,
			Description:   inv.Description,
			Amount:        payment.Amount,
			PaidAt:        now,
			Method:        string(payment.Method),
		}

		pdfBytes, err := pdf.GenerateReceiptPDF(pdfData)
		if err != nil {
			return fmt.Errorf("gagal membuat PDF kwitansi: %w", err)
		}

		// 5. Simpan PDF ke MinIO (jika storage client aktif)
		objectKey := fmt.Sprintf("receipts/%s.pdf", rcNo)
		var pdfURL string
		if s.storageCli != nil {
			bucket := "pmb-finance-receipts"
			_ = s.storageCli.Upload(ctx, bucket, objectKey, bytes.NewReader(pdfBytes), int64(len(pdfBytes)), "application/pdf")
			pdfURL, _ = s.storageCli.PresignedGetObject(ctx, bucket, objectKey, 24*time.Hour)
		}

		rc := &model.Receipt{
			ID:            uuid.New(),
			PaymentID:     payment.ID,
			InvoiceID:     inv.ID,
			ReceiptNumber: rcNo,
			PDFObjectKey:  objectKey,
			PDFURL:        pdfURL,
			Amount:        payment.Amount,
			CreatedAt:     now,
		}

		if err := s.receiptRepo.CreateInTx(tx, rc); err != nil {
			return err
		}

		receipt = rc
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 6. Integrasi Antar-Service (Asinkron / Non-blocking terhadap response)
	go func(appID *uuid.UUID, invNumber string, amount int64) {
		// A. Update status aplikasi pendaftaran di Admission Service -> form_open
		if appID != nil && s.admissionCli != nil {
			updatePayload := map[string]interface{}{
				"status": "form_open",
				"notes":  fmt.Sprintf("Pembayaran pendaftaran lunas via invoice %s", invNumber),
			}
			var resp interface{}
			path := fmt.Sprintf("/internal/applications/%s/status", appID.String())
			if err := s.admissionCli.Post(context.Background(), path, updatePayload, &resp); err != nil {
				log.Printf("[FINANCE-SERVICE] Gagal update status Admission ke form_open: %v", err)
			} else {
				log.Printf("[FINANCE-SERVICE] Berhasil update status Admission %s ke form_open", appID.String())
			}
		}

		// B. Kirim notifikasi via Notification Service
		if s.notifCli != nil {
			notifPayload := map[string]interface{}{
				"channel":  "email",
				"template": "payment_received",
				"data": map[string]interface{}{
					"name":            "Pendaftar",
					"registration_no": invNumber,
					"amount":          fmt.Sprintf("%d", amount),
					"payment_date":    now.Format("02 January 2006 15:04 WIB"),
				},
			}
			var resp interface{}
			_ = s.notifCli.Post(context.Background(), "/internal/notifications", notifPayload, &resp)
		}
	}(inv.ApplicationID, inv.InvoiceNumber, inv.Amount)

	return receipt, nil
}

func (s *paymentService) GetPaymentByID(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*model.Payment, error) {
	payment, err := s.paymentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	inv, err := s.invoiceRepo.FindByID(ctx, payment.InvoiceID)
	if err != nil {
		return nil, err
	}

	if role != "keuangan" && role != "superadmin" && role != "admin_pmb" {
		if inv.UserID != requesterID {
			return nil, ErrForbiddenAccess
		}
	}

	return payment, nil
}

func (s *paymentService) GetReceiptByPaymentID(ctx context.Context, paymentID uuid.UUID, requesterID uuid.UUID, role string) (*model.Receipt, []byte, error) {
	rc, err := s.receiptRepo.FindByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, nil, err
	}

	inv, err := s.invoiceRepo.FindByID(ctx, rc.InvoiceID)
	if err != nil {
		return nil, nil, err
	}

	if role != "keuangan" && role != "superadmin" && role != "admin_pmb" {
		if inv.UserID != requesterID {
			return nil, nil, ErrForbiddenAccess
		}
	}

	// Generate bytes PDF secara langsung
	pdfData := pdf.ReceiptData{
		ReceiptNumber: rc.ReceiptNumber,
		InvoiceNumber: inv.InvoiceNumber,
		Description:   inv.Description,
		Amount:        rc.Amount,
		PaidAt:        rc.CreatedAt,
		Method:        "Pembayaran Lunas",
	}

	pdfBytes, err := pdf.GenerateReceiptPDF(pdfData)
	return rc, pdfBytes, err
}

func (s *paymentService) ListAwaitingVerification(ctx context.Context, limit, offset int) ([]model.Payment, int64, error) {
	return s.paymentRepo.ListAwaitingVerification(ctx, limit, offset)
}

func (s *paymentService) ReconcilePendingPayments(ctx context.Context) (int, error) {
	pendingPayments, err := s.paymentRepo.FindPendingWithProviderRef(ctx, 50)
	if err != nil {
		return 0, err
	}

	reconciled := 0
	for _, p := range pendingPayments {
		statusResult, err := s.provider.CheckStatus(ctx, p.ProviderRef)
		if err != nil || statusResult == nil {
			continue
		}

		if statusResult.StatusCode == "00" { // Sukses terbayar di provider
			inv, err := s.invoiceRepo.FindByID(ctx, p.InvoiceID)
			if err != nil {
				continue
			}
			pCopy := p
			_, err = s.finalizePaymentSuccess(ctx, &pCopy, inv)
			if err == nil {
				reconciled++
			}
		}
	}
	return reconciled, nil
}

func (s *paymentService) GetPaymentMethods(ctx context.Context, amount int64) ([]provider.PaymentMethodInfo, error) {
	if s.provider == nil {
		return nil, errors.New("penyedia pembayaran tidak tersedia")
	}
	return s.provider.GetPaymentMethods(ctx, amount)
}

