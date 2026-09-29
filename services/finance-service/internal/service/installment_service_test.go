package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Mock repository in-memory untuk installment testing
type mockInstallmentRepo struct {
	requests     map[uuid.UUID]*model.InstallmentRequest
	installments map[uuid.UUID][]model.Installment
}

func newMockInstallmentRepo() *mockInstallmentRepo {
	return &mockInstallmentRepo{
		requests:     make(map[uuid.UUID]*model.InstallmentRequest),
		installments: make(map[uuid.UUID][]model.Installment),
	}
}

func (m *mockInstallmentRepo) CreateRequest(ctx context.Context, req *model.InstallmentRequest) error {
	m.requests[req.ID] = req
	return nil
}

func (m *mockInstallmentRepo) FindRequestByID(ctx context.Context, id uuid.UUID) (*model.InstallmentRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return nil, repository.ErrInstallmentRequestNotFound
	}
	return req, nil
}

func (m *mockInstallmentRepo) FindRequestByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*model.InstallmentRequest, error) {
	for _, req := range m.requests {
		if req.InvoiceID == invoiceID {
			return req, nil
		}
	}
	return nil, nil
}

func (m *mockInstallmentRepo) ListRequests(ctx context.Context, status string) ([]model.InstallmentRequest, error) {
	var list []model.InstallmentRequest
	for _, req := range m.requests {
		if status == "" || string(req.Status) == status {
			list = append(list, *req)
		}
	}
	return list, nil
}

func (m *mockInstallmentRepo) UpdateRequest(ctx context.Context, req *model.InstallmentRequest) error {
	m.requests[req.ID] = req
	return nil
}

func (m *mockInstallmentRepo) CreateInstallments(ctx context.Context, tx *gorm.DB, installments []model.Installment) error {
	for _, inst := range installments {
		m.installments[inst.InvoiceID] = append(m.installments[inst.InvoiceID], inst)
	}
	return nil
}

func (m *mockInstallmentRepo) FindInstallmentsByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]model.Installment, error) {
	return m.installments[invoiceID], nil
}

func (m *mockInstallmentRepo) FindInstallmentByID(ctx context.Context, id uuid.UUID) (*model.Installment, error) {
	for _, list := range m.installments {
		for _, inst := range list {
			if inst.ID == id {
				return &inst, nil
			}
		}
	}
	return nil, repository.ErrInstallmentNotFound
}

func (m *mockInstallmentRepo) UpdateInstallment(ctx context.Context, inst *model.Installment) error {
	list := m.installments[inst.InvoiceID]
	for idx, itm := range list {
		if itm.ID == inst.ID {
			list[idx] = *inst
			m.installments[inst.InvoiceID] = list
			return nil
		}
	}
	return nil
}

func (m *mockInstallmentRepo) FindDueInstallments(ctx context.Context, startDate, endDate time.Time) ([]model.Installment, error) {
	var res []model.Installment
	for _, list := range m.installments {
		for _, inst := range list {
			if inst.Status == model.InstallmentStatusUnpaid && (inst.DueDate.After(startDate) || inst.DueDate.Equal(startDate)) && (inst.DueDate.Before(endDate) || inst.DueDate.Equal(endDate)) {
				res = append(res, inst)
			}
		}
	}
	return res, nil
}

func (m *mockInstallmentRepo) FindExpiredInstallments(ctx context.Context, now time.Time) ([]model.Installment, error) {
	var res []model.Installment
	for _, list := range m.installments {
		for _, inst := range list {
			if inst.Status == model.InstallmentStatusUnpaid && inst.DueDate.Before(now) {
				res = append(res, inst)
			}
		}
	}
	return res, nil
}

// Mock Invoice Repo
type mockInvoiceRepoForInstallment struct {
	invoices map[uuid.UUID]*model.Invoice
}

func newMockInvoiceRepoForInstallment() *mockInvoiceRepoForInstallment {
	return &mockInvoiceRepoForInstallment{
		invoices: make(map[uuid.UUID]*model.Invoice),
	}
}

func (m *mockInvoiceRepoForInstallment) DB() *gorm.DB {
	return nil
}

func (m *mockInvoiceRepoForInstallment) CreateInTx(tx *gorm.DB, invoice *model.Invoice) error {
	m.invoices[invoice.ID] = invoice
	return nil
}

func (m *mockInvoiceRepoForInstallment) FindByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, repository.ErrInvoiceNotFound
	}
	return inv, nil
}

func (m *mockInvoiceRepoForInstallment) FindByInvoiceNumber(ctx context.Context, invNo string) (*model.Invoice, error) {
	for _, inv := range m.invoices {
		if inv.InvoiceNumber == invNo {
			return inv, nil
		}
	}
	return nil, repository.ErrInvoiceNotFound
}

func (m *mockInvoiceRepoForInstallment) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Invoice, error) {
	var list []model.Invoice
	for _, inv := range m.invoices {
		if inv.UserID == userID {
			list = append(list, *inv)
		}
	}
	return list, nil
}

func (m *mockInvoiceRepoForInstallment) FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Invoice, error) {
	for _, inv := range m.invoices {
		if inv.ApplicationID != nil && *inv.ApplicationID == appID {
			return inv, nil
		}
	}
	return nil, repository.ErrInvoiceNotFound
}

func (m *mockInvoiceRepoForInstallment) GenerateInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error) {
	return "INV-TEST-001", nil
}

func (m *mockInvoiceRepoForInstallment) GenerateAdjustmentInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error) {
	return "INV-REG-2026-0001-ADJ", nil
}

func (m *mockInvoiceRepoForInstallment) UpdateAmount(ctx context.Context, id uuid.UUID, amount int64) error {
	inv, ok := m.invoices[id]
	if !ok {
		return repository.ErrInvoiceNotFound
	}
	inv.Amount = amount
	return nil
}

func (m *mockInvoiceRepoForInstallment) UpdateStatus(ctx context.Context, id uuid.UUID, status model.InvoiceStatus, paidAmount int64) error {
	inv, ok := m.invoices[id]
	if !ok {
		return repository.ErrInvoiceNotFound
	}
	inv.Status = status
	inv.PaidAmount = paidAmount
	return nil
}

func TestInstallmentValidationAndPrecisionMath(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	adminID := uuid.New()

	mockInvRepo := newMockInvoiceRepoForInstallment()
	mockInstRepo := newMockInstallmentRepo()

	svc := service.NewInstallmentService(mockInstRepo, mockInvRepo, nil, nil)

	// 1. Tagihan Biaya Pendaftaran (Registration) -> Ditolak INSTALLMENT_NOT_ALLOWED
	regInvID := uuid.New()
	mockInvRepo.invoices[regInvID] = &model.Invoice{
		ID:            regInvID,
		UserID:        userID,
		FeeType:       "registration",
		Amount:        250000,
		Status:        model.InvoiceStatusUnpaid,
		DueDate:       time.Now().AddDate(0, 0, 3),
		PaymentScheme: "full",
	}

	_, err := svc.RequestInstallment(ctx, regInvID, userID, 3, "Ingin mencicil uang pendaftaran")
	assert.Error(t, err)
	assert.ErrorIs(t, err, service.ErrInstallmentNotAllowed)

	// 2. Tagihan Daftar Ulang (Reregistration) dengan Nominal Ganjil Rp 3.333.334
	reregInvID := uuid.New()
	mockInvRepo.invoices[reregInvID] = &model.Invoice{
		ID:            reregInvID,
		UserID:        userID,
		InvoiceNumber: "INV-REREG-001",
		FeeType:       "reregistration",
		Amount:        3333334,
		Status:        model.InvoiceStatusUnpaid,
		DueDate:       time.Now().AddDate(0, 0, 14),
		PaymentScheme: "full",
	}

	// Coba ajukan dengan jumlah termin tidak valid (mis. 5 termin) -> ditolak
	_, err = svc.RequestInstallment(ctx, reregInvID, userID, 5, "Minta 5 termin")
	assert.Error(t, err)
	assert.ErrorIs(t, err, service.ErrInvalidInstallmentTerms)

	// Ajukan 3 termin yang valid
	req, err := svc.RequestInstallment(ctx, reregInvID, userID, 3, "Pengajuan cicilan 3 termin")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, model.InstallmentRequestPending, req.Status)
	assert.Equal(t, 3, req.RequestedTerms)

	// 3. Staf Keuangan Menyetujui (Approve) Pengajuan Cicilan
	reviewedReq, err := svc.ReviewInstallmentRequest(ctx, req.ID, adminID, "approve", "Disetujui keuangan")
	assert.NoError(t, err)
	assert.Equal(t, model.InstallmentRequestApproved, reviewedReq.Status)

	// Periksa Jadwal Termin yang Terbentuk
	installments, err := mockInstRepo.FindInstallmentsByInvoiceID(ctx, reregInvID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(installments))

	// Verifikasi Matematika Integer Tanpa Selisih:
	// 3.333.334 / 3 = 1.111.111 sisa 1
	// Termin 1: 1.111.112
	// Termin 2: 1.111.111
	// Termin 3: 1.111.111
	assert.Equal(t, int64(1111112), installments[0].Amount)
	assert.Equal(t, int64(1111111), installments[1].Amount)
	assert.Equal(t, int64(1111111), installments[2].Amount)

	var totalSum int64
	for _, inst := range installments {
		totalSum += inst.Amount
	}
	assert.Equal(t, int64(3333334), totalSum, "Total ketiga termin harus persis sama dengan total tagihan tanpa selisih pembulatan")

	// Verifikasi Skema Pembayaran Invoice Berubah Menjadi 'installment'
	invUpdated, _ := mockInvRepo.FindByID(ctx, reregInvID)
	assert.Equal(t, "installment", invUpdated.PaymentScheme)
}
