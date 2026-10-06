package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrForbiddenAccess    = errors.New("Anda tidak memiliki akses ke data tagihan ini")
	ErrInvoiceAlreadyPaid = errors.New("tagihan ini sudah lunas")
	ErrCancelNotAllowed   = errors.New("CANCEL_NOT_ALLOWED: tagihan tidak dapat dibatalkan karena sudah dibayar atau sedang diverifikasi")
	ErrRepriceNotAllowed  = errors.New("PAYMENT_IN_PROGRESS: tagihan tidak dapat diubah nominalnya karena ada pembayaran sedang diproses atau sudah lunas")
	ErrInvoiceFree        = errors.New("INVOICE_FREE: Tagihan ini gratis (Rp 0) dan tidak memerlukan pembayaran.")
)

type CreateAdjustmentInvoiceRequest struct {
	UserID        uuid.UUID  `json:"user_id"`
	ApplicationID *uuid.UUID `json:"application_id,omitempty"`
	Amount        int64      `json:"amount"`
	Description   string     `json:"description"`
}

type CreateInvoiceRequest struct {
	UserID        uuid.UUID  `json:"user_id"`
	ApplicationID *uuid.UUID `json:"application_id,omitempty"`
	FeeType       string     `json:"fee_type"`
	Amount        int64      `json:"amount"` // Nominal integer Rupiah
	Description   string     `json:"description"`
	DueDateDays   int        `json:"due_date_days"`
}

type ActivePaymentInfo struct {
	PaymentID   uuid.UUID  `json:"payment_id"`
	Method      string     `json:"method"`
	Provider    string     `json:"provider"`
	PaymentURL  string     `json:"payment_url,omitempty"`
	VANumber    string     `json:"va_number,omitempty"`
	PaymentCode string     `json:"payment_code,omitempty"`
	Amount      int64      `json:"amount"`
	Status      string     `json:"status"`
	ExpiredAt   *time.Time `json:"expired_at,omitempty"`
	ProofURL    string     `json:"proof_url,omitempty"`
}

type InvoiceDetailResponse struct {
	ID                  uuid.UUID                  `json:"id"`
	InvoiceNumber       string                     `json:"invoice_number"`
	UserID              uuid.UUID                  `json:"user_id"`
	ApplicationID       *uuid.UUID                 `json:"application_id,omitempty"`
	FeeType             string                     `json:"fee_type"`
	Description         string                     `json:"description"`
	Amount              int64                      `json:"amount"`
	PaidAmount          int64                      `json:"paid_amount"`
	RemainingAmount     int64                      `json:"remaining_amount"`
	Status              model.InvoiceStatus        `json:"status"`
	DueDate             time.Time                  `json:"due_date"`
	PaymentScheme       string                     `json:"payment_scheme"`
	ActivePayment       *ActivePaymentInfo         `json:"active_payment,omitempty"`
	Payments            []model.Payment            `json:"payments"`
	Installments        []model.Installment        `json:"installments,omitempty"`
	InstallmentRequests []model.InstallmentRequest `json:"installment_requests,omitempty"`
	ReceiptURL          string                     `json:"receipt_url,omitempty"`
	CanCancel           bool                       `json:"can_cancel"`
	CreatedAt           time.Time                  `json:"created_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
}

type InvoiceService interface {
	CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*model.Invoice, error)
	GetMyInvoices(ctx context.Context, userID uuid.UUID) ([]model.Invoice, error)
	GetInvoiceDetail(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*model.Invoice, error)
	GetInvoiceDetailDTO(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*InvoiceDetailResponse, error)
	GetInvoiceByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Invoice, error)
	GetInvoiceByApplicationIDAndFeeType(ctx context.Context, appID uuid.UUID, feeType string) (*model.Invoice, error)
	CancelInvoice(ctx context.Context, invoiceID uuid.UUID) error
	CancelInvoiceByApplicationID(ctx context.Context, appID uuid.UUID) error
	RepriceInvoice(ctx context.Context, invoiceID uuid.UUID, newAmount int64) error
	CreateAdjustmentInvoice(ctx context.Context, req *CreateAdjustmentInvoiceRequest) (*model.Invoice, error)
	ProcessApplicationWithdrawal(ctx context.Context, appID uuid.UUID) error
}

type invoiceService struct {
	invoiceRepo repository.InvoiceRepository
}

func NewInvoiceService(invoiceRepo repository.InvoiceRepository) InvoiceService {
	return &invoiceService{
		invoiceRepo: invoiceRepo,
	}
}

func (s *invoiceService) CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*model.Invoice, error) {
	if req.Amount < 0 {
		return nil, fmt.Errorf("nominal tagihan harus lebih besar dari 0")
	}

	feeType := req.FeeType
	if feeType == "" {
		feeType = "registration"
	}

	// Hanya pendaftaran jalur gratis (memiliki ApplicationID dan feeType registration) yang boleh nominal 0
	if req.Amount == 0 && (req.ApplicationID == nil || feeType != "registration") {
		return nil, fmt.Errorf("nominal tagihan harus lebih besar dari 0")
	}

	// Cek apakah invoice untuk pendaftaran ini dengan jenis biaya yang sama sudah ada
	if req.ApplicationID != nil {
		existing, err := s.invoiceRepo.FindByApplicationIDAndFeeType(ctx, *req.ApplicationID, feeType)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	dueDays := req.DueDateDays
	if dueDays <= 0 {
		dueDays = 3
	}
	dueDate := time.Now().AddDate(0, 0, dueDays)

	var newInvoice *model.Invoice

	runInTx := func(fn func(tx *gorm.DB) error) error {
		if s.invoiceRepo.DB() != nil {
			return s.invoiceRepo.DB().WithContext(ctx).Transaction(fn)
		}
		return fn(nil)
	}

	err := runInTx(func(tx *gorm.DB) error {
		invNumber, genErr := s.invoiceRepo.GenerateInvoiceNumber(ctx, tx)
		if genErr != nil {
			return fmt.Errorf("gagal membuat nomor tagihan: %w", genErr)
		}

		status := model.InvoiceStatusUnpaid
		scheme := "full"
		if req.Amount == 0 {
			status = model.InvoiceStatusPaid
			scheme = "free"
		}

		inv := &model.Invoice{
			ID:            uuid.New(),
			InvoiceNumber: invNumber,
			UserID:        req.UserID,
			ApplicationID: req.ApplicationID,
			FeeType:       feeType,
			Description:   req.Description,
			Amount:        req.Amount,
			PaidAmount:    0,
			Status:        status,
			DueDate:       dueDate,
			PaymentScheme: scheme,
		}

		if err := s.invoiceRepo.CreateInTx(tx, inv); err != nil {
			return fmt.Errorf("gagal menyimpan invoice: %w", err)
		}

		if req.Amount == 0 && tx != nil {
			now := time.Now()
			pay := &model.Payment{
				ID:          uuid.New(),
				InvoiceID:   inv.ID,
				Method:      model.PaymentMethodFree,
				Provider:    "system",
				ProviderRef: fmt.Sprintf("FREE-%s", inv.ID.String()[:8]),
				Channel:     "free",
				Amount:      0,
				Status:      model.PaymentStatusSuccess,
				Notes:       "Pendaftaran jalur gratis",
				PaidAt:      &now,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := tx.Create(pay).Error; err != nil {
				return fmt.Errorf("gagal membuat pembayaran gratis: %w", err)
			}
		}

		newInvoice = inv
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.invoiceRepo.FindByID(ctx, newInvoice.ID)
}

func (s *invoiceService) GetMyInvoices(ctx context.Context, userID uuid.UUID) ([]model.Invoice, error) {
	return s.invoiceRepo.FindByUserID(ctx, userID)
}

func (s *invoiceService) GetInvoiceDetail(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*model.Invoice, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Keuangan dan admin dapat mengakses seluruh tagihan
	if role == "admin" || role == "keuangan" || role == "superadmin" || role == "admin_pmb" {
		return inv, nil
	}

	// Pengguna biasa hanya dapat mengakses tagihan miliknya sendiri
	if inv.UserID != requesterID {
		return nil, ErrForbiddenAccess
	}

	return inv, nil
}

func (s *invoiceService) GetInvoiceByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Invoice, error) {
	return s.invoiceRepo.FindByApplicationID(ctx, appID)
}

func (s *invoiceService) GetInvoiceByApplicationIDAndFeeType(ctx context.Context, appID uuid.UUID, feeType string) (*model.Invoice, error) {
	return s.invoiceRepo.FindByApplicationIDAndFeeType(ctx, appID, feeType)
}

func (s *invoiceService) GetInvoiceDetailDTO(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*InvoiceDetailResponse, error) {
	inv, err := s.GetInvoiceDetail(ctx, id, requesterID, role)
	if err != nil {
		return nil, err
	}

	remaining := inv.Amount - inv.PaidAmount
	if remaining < 0 {
		remaining = 0
	}

	canCancel := inv.Status == model.InvoiceStatusUnpaid
	var activePayment *ActivePaymentInfo
	var receiptURL string

	for _, p := range inv.Payments {
		if p.Status == model.PaymentStatusSuccess || p.Status == model.PaymentStatusAwaitingVerification {
			canCancel = false
		}
		if p.Status == model.PaymentStatusPending || p.Status == model.PaymentStatusAwaitingVerification {
			activePayment = &ActivePaymentInfo{
				PaymentID:   p.ID,
				Method:      string(p.Method),
				Provider:    p.Provider,
				PaymentURL:  p.PaymentURL,
				VANumber:    p.VANumber,
				PaymentCode: p.PaymentCode,
				Amount:      p.Amount,
				Status:      string(p.Status),
				ProofURL:    p.ProofURL,
			}
		}
		if p.Receipt != nil {
			receiptURL = fmt.Sprintf("/api/v1/receipts/me/%s", p.ID.String())
		}
	}

	dto := &InvoiceDetailResponse{
		ID:                  inv.ID,
		InvoiceNumber:       inv.InvoiceNumber,
		UserID:              inv.UserID,
		ApplicationID:       inv.ApplicationID,
		FeeType:             inv.FeeType,
		Description:         inv.Description,
		Amount:              inv.Amount,
		PaidAmount:          inv.PaidAmount,
		RemainingAmount:     remaining,
		Status:              inv.Status,
		DueDate:             inv.DueDate,
		PaymentScheme:       inv.PaymentScheme,
		ActivePayment:       activePayment,
		Payments:            inv.Payments,
		Installments:        inv.Installments,
		InstallmentRequests: inv.InstallmentRequests,
		ReceiptURL:          receiptURL,
		CanCancel:           canCancel,
		CreatedAt:           inv.CreatedAt,
		UpdatedAt:           inv.UpdatedAt,
	}

	return dto, nil
}

func (s *invoiceService) CancelInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	// Idempoten jika sudah cancelled
	if inv.Status == model.InvoiceStatusCancelled {
		return nil
	}

	// Tagihan lunas tidak boleh dibatalkan
	if inv.Status == model.InvoiceStatusPaid || inv.IsPaid() {
		return ErrCancelNotAllowed
	}

	// Periksa apakah ada pembayaran yang sudah sukses atau sedang diverifikasi
	// Kecuali pembayaran untuk jalur gratis (method: free atau payment_scheme: free)
	if inv.PaymentScheme != "free" {
		for _, p := range inv.Payments {
			if (p.Status == model.PaymentStatusSuccess || p.Status == model.PaymentStatusAwaitingVerification) && p.Method != model.PaymentMethodFree {
				return ErrCancelNotAllowed
			}
		}
	}

	runInTx := func(fn func(tx *gorm.DB) error) error {
		if s.invoiceRepo.DB() != nil {
			return s.invoiceRepo.DB().WithContext(ctx).Transaction(fn)
		}
		return fn(nil)
	}

	return runInTx(func(tx *gorm.DB) error {
		now := time.Now()
		if tx != nil {
			// Update status invoice menjadi cancelled
			if err := tx.Model(&model.Invoice{}).Where("id = ?", inv.ID).Updates(map[string]interface{}{
				"status":     model.InvoiceStatusCancelled,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}

			// Tandai pembayaran pending menjadi cancelled
			if err := tx.Model(&model.Payment{}).Where("invoice_id = ? AND status = ?", inv.ID, model.PaymentStatusPending).Updates(map[string]interface{}{
				"status":     model.PaymentStatusCancelled,
				"notes":      "Dibatalkan seiring pembatalan tagihan",
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
		} else {
			_ = s.invoiceRepo.UpdateStatus(ctx, inv.ID, model.InvoiceStatusCancelled, inv.PaidAmount)
		}
		return nil
	})
}

func (s *invoiceService) CancelInvoiceByApplicationID(ctx context.Context, appID uuid.UUID) error {
	inv, err := s.invoiceRepo.FindByApplicationID(ctx, appID)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			// Tidak ada invoice yang perlu dibatalkan
			return nil
		}
		return err
	}

	return s.CancelInvoice(ctx, inv.ID)
}

func (s *invoiceService) RepriceInvoice(ctx context.Context, invoiceID uuid.UUID, newAmount int64) error {
	if newAmount <= 0 {
		return fmt.Errorf("nominal tagihan baru harus lebih besar dari 0")
	}

	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	if inv.Status != model.InvoiceStatusUnpaid {
		return ErrRepriceNotAllowed
	}

	for _, p := range inv.Payments {
		if p.Status == model.PaymentStatusSuccess || p.Status == model.PaymentStatusAwaitingVerification {
			return ErrRepriceNotAllowed
		}
		if p.Status == model.PaymentStatusPending {
			return ErrRepriceNotAllowed
		}
	}

	return s.invoiceRepo.UpdateAmount(ctx, invoiceID, newAmount)
}

func (s *invoiceService) CreateAdjustmentInvoice(ctx context.Context, req *CreateAdjustmentInvoiceRequest) (*model.Invoice, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("nominal penyesuaian harus lebih besar dari 0")
	}

	tx := s.invoiceRepo.DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	invNo, err := s.invoiceRepo.GenerateAdjustmentInvoiceNumber(ctx, tx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("gagal membuat nomor invoice penyesuaian: %w", err)
	}

	dueDate := time.Now().AddDate(0, 0, 7) // default jatuh tempo penyesuaian 7 hari

	inv := &model.Invoice{
		InvoiceNumber: invNo,
		UserID:        req.UserID,
		ApplicationID: req.ApplicationID,
		FeeType:       "registration_adjustment",
		Description:   req.Description,
		Amount:        req.Amount,
		Status:        model.InvoiceStatusUnpaid,
		DueDate:       dueDate,
		PaymentScheme: "full",
	}

	if err := s.invoiceRepo.CreateInTx(tx, inv); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("gagal menyimpan invoice penyesuaian: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("gagal commit transaksi invoice penyesuaian: %w", err)
	}

	return inv, nil
}

func (s *invoiceService) ProcessApplicationWithdrawal(ctx context.Context, appID uuid.UUID) error {
	inv, err := s.invoiceRepo.FindByApplicationID(ctx, appID)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			return nil
		}
		return err
	}

	runInTx := func(fn func(tx *gorm.DB) error) error {
		if s.invoiceRepo.DB() != nil {
			return s.invoiceRepo.DB().WithContext(ctx).Transaction(fn)
		}
		return fn(nil)
	}

	return runInTx(func(tx *gorm.DB) error {
		now := time.Now()
		if tx != nil {
			if inv.Status == model.InvoiceStatusUnpaid {
				_ = tx.Model(&model.Invoice{}).Where("id = ?", inv.ID).Updates(map[string]interface{}{
					"status":     model.InvoiceStatusCancelled,
					"updated_at": now,
				}).Error
				_ = tx.Model(&model.Payment{}).Where("invoice_id = ? AND status = ?", inv.ID, model.PaymentStatusPending).Updates(map[string]interface{}{
					"status":     model.PaymentStatusCancelled,
					"notes":      "Dibatalkan karena pemohon mengundurkan diri",
					"updated_at": now,
				}).Error
			} else {
				for _, p := range inv.Payments {
					if p.Status == model.PaymentStatusSuccess {
						_ = tx.Model(&model.Payment{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
							"status":     model.PaymentStatusRefundReview,
							"notes":      "Ditandai refund_review karena pemohon mengundurkan diri",
							"updated_at": now,
						}).Error

						var existingCount int64
						_ = tx.Model(&model.RefundReview{}).Where("payment_id = ?", p.ID).Count(&existingCount).Error
						if existingCount == 0 {
							rev := &model.RefundReview{
								ID:            uuid.New(),
								PaymentID:     p.ID,
								InvoiceID:     inv.ID,
								ApplicationID: appID,
								Amount:        p.Amount,
								Status:        model.RefundReviewStatusPending,
								Notes:         "Pengunduran diri pendaftaran",
								CreatedAt:     now,
								UpdatedAt:     now,
							}
							_ = tx.Create(rev).Error
						}
					} else if p.Status == model.PaymentStatusPending {
						_ = tx.Model(&model.Payment{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
							"status":     model.PaymentStatusCancelled,
							"notes":      "Dibatalkan karena pemohon mengundurkan diri",
							"updated_at": now,
						}).Error
					}
				}
			}
		}
		return nil
	})
}


