package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestFinanceAdminService_ManualConfirmPayment(t *testing.T) {
	ctx := context.Background()
	invRepo := newMockInvoiceRepo()
	payRepo := newMockPaymentRepo()
	rcRepo := newMockReceiptRepo()
	admCli := &mockInternalClient{}
	notifCli := &mockInternalClient{}

	svc := NewFinanceAdminService(invRepo, payRepo, rcRepo, admCli, notifCli)

	actorID := uuid.New()
	userID := uuid.New()

	// 1. Sukses konfirmasi manual
	invID := uuid.New()
	invRepo.invoices[invID] = &model.Invoice{
		ID:            invID,
		UserID:        userID,
		FeeType:       "registration",
		Amount:        250000,
		PaidAmount:    0,
		Status:        model.InvoiceStatusUnpaid,
		InvoiceNumber: "INV-2026-001",
	}

	req := ManualConfirmRequest{
		Amount:          250000,
		TransactionTime: time.Now(),
		Channel:         "manual_transfer",
		BankReference:   "TRX-BCA-12345",
		ProofURL:        "https://storage.local/proofs/123.jpg",
		Reason:          "Transfer manual diverifikasi staf keuangan",
	}

	payment, err := svc.ManualConfirmPayment(ctx, invID, actorID, "Admin Keuangan", req)
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, model.PaymentStatusSuccess, payment.Status)
	assert.Equal(t, "TRX-BCA-12345", payment.BankReference)
	assert.Equal(t, model.InvoiceStatusPaid, invRepo.invoices[invID].Status)

	// Pastikan kuitansi terbuat
	receipt, rErr := rcRepo.FindByPaymentID(ctx, payment.ID)
	assert.NoError(t, rErr)
	assert.NotNil(t, receipt)

	// 2. Gagal jika tagihan sudah lunas
	_, errPaid := svc.ManualConfirmPayment(ctx, invID, actorID, "Admin Keuangan", req)
	assert.ErrorIs(t, errPaid, ErrInvoiceAlreadyPaid)

	// 3. Gagal jika duplicate bank reference
	invID2 := uuid.New()
	invRepo.invoices[invID2] = &model.Invoice{
		ID:            invID2,
		UserID:        userID,
		FeeType:       "registration",
		Amount:        250000,
		PaidAmount:    0,
		Status:        model.InvoiceStatusUnpaid,
		InvoiceNumber: "INV-2026-002",
	}
	_, errDup := svc.ManualConfirmPayment(ctx, invID2, actorID, "Admin Keuangan", req)
	assert.ErrorIs(t, errDup, ErrDuplicateBankReference)

	// 4. Gagal jika nominal tidak cocok
	reqMismatch := req
	reqMismatch.BankReference = "TRX-BCA-99999"
	reqMismatch.Amount = 200000 // Kurang dari 250000
	_, errAmt := svc.ManualConfirmPayment(ctx, invID2, actorID, "Admin Keuangan", reqMismatch)
	assert.ErrorIs(t, errAmt, ErrInvalidAmount)
}

func TestFinanceAdminService_RenewInvoiceDueDate(t *testing.T) {
	ctx := context.Background()
	invRepo := newMockInvoiceRepo()
	payRepo := newMockPaymentRepo()
	rcRepo := newMockReceiptRepo()

	svc := NewFinanceAdminService(invRepo, payRepo, rcRepo, nil, nil)
	userID := uuid.New()

	invID := uuid.New()
	invRepo.invoices[invID] = &model.Invoice{
		ID:           invID,
		UserID:       userID,
		Status:       model.InvoiceStatusUnpaid,
		DueDate:      time.Now().Add(-1 * time.Hour), // sudah expired
		RenewalCount: 0,
	}

	// Perpanjangan pertama (renewal 1)
	inv1, err := svc.RenewInvoiceDueDate(ctx, invID, userID)
	assert.NoError(t, err)
	assert.Equal(t, 1, inv1.RenewalCount)
	assert.True(t, inv1.DueDate.After(time.Now()))

	// Perpanjangan kedua (renewal 2)
	inv2, err := svc.RenewInvoiceDueDate(ctx, invID, userID)
	assert.NoError(t, err)
	assert.Equal(t, 2, inv2.RenewalCount)

	// Perpanjangan ketiga (harus ditolak karena maks 2 kali)
	_, errLimit := svc.RenewInvoiceDueDate(ctx, invID, userID)
	assert.ErrorIs(t, errLimit, ErrRenewalLimitExceeded)
}

func TestFinanceAdminService_ReceiptViewAndPaymentMethods(t *testing.T) {
	ctx := context.Background()
	invRepo := newMockInvoiceRepo()
	payRepo := newMockPaymentRepo()
	rcRepo := newMockReceiptRepo()

	svc := NewFinanceAdminService(invRepo, payRepo, rcRepo, nil, nil)
	userID := uuid.New()

	// 1. Grouped Payment Methods
	grouped, err := svc.GetPaymentMethodsGrouped(ctx, 250000)
	assert.NoError(t, err)
	assert.NotNil(t, grouped)
	assert.NotEmpty(t, grouped.VirtualAccount)
	assert.NotEmpty(t, grouped.QRIS)

	// 2. Receipt View
	invID := uuid.New()
	invRepo.invoices[invID] = &model.Invoice{
		ID:            invID,
		UserID:        userID,
		Status:        model.InvoiceStatusPaid,
		Amount:        250000,
		PaidAmount:    250000,
		InvoiceNumber: "INV-2026-999",
	}

	payID := uuid.New()
	now := time.Now()
	p := &model.Payment{
		ID:            payID,
		InvoiceID:     invID,
		Amount:        250000,
		Status:        model.PaymentStatusSuccess,
		Method:        model.PaymentMethodQRIS,
		Channel:       "qris",
		BankReference: "QRIS-REF-123",
		PaidAt:        &now,
	}
	payRepo.payments[payID] = p

	rcRepo.receipts[uuid.New()] = &model.Receipt{
		ID:            uuid.New(),
		PaymentID:     payID,
		ReceiptNumber: "KW-2026-0001",
		CreatedAt:     now,
	}

	view, vErr := svc.GetReceiptView(ctx, invID, userID)
	assert.NoError(t, vErr)
	assert.NotNil(t, view)
	assert.Contains(t, view.LockNoticeLabel, "terkunci")
	assert.NotNil(t, view.NextAction)
	assert.Equal(t, "/exam", view.NextAction.ActionURL)
}
