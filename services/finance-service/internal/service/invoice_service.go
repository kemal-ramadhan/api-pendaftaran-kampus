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
	ErrForbiddenAccess = errors.New("Anda tidak memiliki akses ke data tagihan ini")
	ErrInvoiceAlreadyPaid = errors.New("tagihan ini sudah lunas")
)

type CreateInvoiceRequest struct {
	UserID        uuid.UUID  `json:"user_id"`
	ApplicationID *uuid.UUID `json:"application_id,omitempty"`
	FeeType       string     `json:"fee_type"`
	Amount        int64      `json:"amount"` // Nominal integer Rupiah
	Description   string     `json:"description"`
	DueDateDays   int        `json:"due_date_days"`
}

type InvoiceService interface {
	CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*model.Invoice, error)
	GetMyInvoices(ctx context.Context, userID uuid.UUID) ([]model.Invoice, error)
	GetInvoiceDetail(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*model.Invoice, error)
	GetInvoiceByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Invoice, error)
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
	if req.Amount <= 0 {
		return nil, fmt.Errorf("nominal tagihan harus lebih besar dari 0")
	}

	// Cek apakah invoice untuk pendaftaran ini sudah ada
	if req.ApplicationID != nil {
		existing, err := s.invoiceRepo.FindByApplicationID(ctx, *req.ApplicationID)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	dueDays := req.DueDateDays
	if dueDays <= 0 {
		dueDays = 3
	}
	dueDate := time.Now().AddDate(0, 0, dueDays)

	feeType := req.FeeType
	if feeType == "" {
		feeType = "registration"
	}

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

		inv := &model.Invoice{
			ID:            uuid.New(),
			InvoiceNumber: invNumber,
			UserID:        req.UserID,
			ApplicationID: req.ApplicationID,
			FeeType:       feeType,
			Description:   req.Description,
			Amount:        req.Amount,
			PaidAmount:    0,
			Status:        model.InvoiceStatusUnpaid,
			DueDate:       dueDate,
			PaymentScheme: "full",
		}

		if err := s.invoiceRepo.CreateInTx(tx, inv); err != nil {
			return fmt.Errorf("gagal menyimpan invoice: %w", err)
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
