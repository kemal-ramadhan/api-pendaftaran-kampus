package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type mockInvoiceRepo struct {
	invoices map[uuid.UUID]*model.Invoice
	seq      int
}

func newMockInvoiceRepo() *mockInvoiceRepo {
	return &mockInvoiceRepo{
		invoices: make(map[uuid.UUID]*model.Invoice),
	}
}

func (m *mockInvoiceRepo) Create(ctx context.Context, inv *model.Invoice) error {
	m.invoices[inv.ID] = inv
	return nil
}

func (m *mockInvoiceRepo) CreateInTx(tx *gorm.DB, inv *model.Invoice) error {
	m.invoices[inv.ID] = inv
	return nil
}

func (m *mockInvoiceRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error) {
	if inv, ok := m.invoices[id]; ok {
		return inv, nil
	}
	return nil, repository.ErrInvoiceNotFound
}

func (m *mockInvoiceRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Invoice, error) {
	var list []model.Invoice
	for _, inv := range m.invoices {
		if inv.UserID == userID {
			list = append(list, *inv)
		}
	}
	return list, nil
}

func (m *mockInvoiceRepo) FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Invoice, error) {
	for _, inv := range m.invoices {
		if inv.ApplicationID != nil && *inv.ApplicationID == appID {
			return inv, nil
		}
	}
	return nil, repository.ErrInvoiceNotFound
}

func (m *mockInvoiceRepo) FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*model.Invoice, error) {
	for _, inv := range m.invoices {
		if inv.InvoiceNumber == invoiceNumber {
			return inv, nil
		}
	}
	return nil, repository.ErrInvoiceNotFound
}

func (m *mockInvoiceRepo) GenerateInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error) {
	m.seq++
	return "INV-2026-00001", nil
}

func (m *mockInvoiceRepo) GenerateAdjustmentInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error) {
	m.seq++
	return "INV-REG-2026-0001-ADJ", nil
}

func (m *mockInvoiceRepo) UpdateAmount(ctx context.Context, id uuid.UUID, amount int64) error {
	if inv, ok := m.invoices[id]; ok {
		inv.Amount = amount
		return nil
	}
	return repository.ErrInvoiceNotFound
}

func (m *mockInvoiceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.InvoiceStatus, paidAmount int64) error {
	if inv, ok := m.invoices[id]; ok {
		inv.Status = status
		inv.PaidAmount = paidAmount
		return nil
	}
	return repository.ErrInvoiceNotFound
}

func (m *mockInvoiceRepo) UpdateStatusInTx(tx *gorm.DB, id uuid.UUID, status model.InvoiceStatus, paidAmount int64) error {
	return m.UpdateStatus(context.Background(), id, status, paidAmount)
}

func (m *mockInvoiceRepo) ListAllInvoices(ctx context.Context, periodID *uint, status string, feeType string, search string, limit, offset int) ([]model.Invoice, int64, error) {
	var list []model.Invoice
	for _, inv := range m.invoices {
		list = append(list, *inv)
	}
	return list, int64(len(list)), nil
}

func (m *mockInvoiceRepo) UpdateDueDateAndRenewal(ctx context.Context, id uuid.UUID, newDueDate time.Time, renewalCount int) error {
	if inv, ok := m.invoices[id]; ok {
		inv.DueDate = newDueDate
		inv.RenewalCount = renewalCount
		return nil
	}
	return repository.ErrInvoiceNotFound
}

func (m *mockInvoiceRepo) DB() *gorm.DB {
	return nil
}

func TestInvoiceService_ValidationAndAccess(t *testing.T) {
	repo := newMockInvoiceRepo()
	svc := NewInvoiceService(repo)
	ctx := context.Background()

	t.Run("create invoice with zero or negative amount should fail", func(t *testing.T) {
		req := &CreateInvoiceRequest{
			UserID: uuid.New(),
			Amount: 0,
		}
		_, err := svc.CreateInvoice(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nominal tagihan harus lebih besar dari 0")
	})

	t.Run("create invoice with zero amount for free registration path should succeed", func(t *testing.T) {
		appID := uuid.New()
		req := &CreateInvoiceRequest{
			UserID:        uuid.New(),
			ApplicationID: &appID,
			FeeType:       "registration",
			Amount:        0,
		}
		inv, err := svc.CreateInvoice(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, inv)
		assert.Equal(t, model.InvoiceStatusPaid, inv.Status)
		assert.Equal(t, "free", inv.PaymentScheme)
	})

	t.Run("invoice access control - user isolation", func(t *testing.T) {
		ownerID := uuid.New()
		otherUserID := uuid.New()
		invID := uuid.New()

		inv := &model.Invoice{
			ID:            invID,
			InvoiceNumber: "INV-2026-00001",
			UserID:        ownerID,
			Amount:        250000,
			PaidAmount:    0,
			Status:        model.InvoiceStatusUnpaid,
			DueDate:       time.Now().Add(24 * time.Hour),
		}
		_ = repo.Create(ctx, inv)

		// 1. Owner can access invoice
		res, err := svc.GetInvoiceDetail(ctx, invID, ownerID, "applicant")
		assert.NoError(t, err)
		assert.Equal(t, invID, res.ID)

		// 2. Other applicant cannot access invoice (403 forbidden)
		_, err = svc.GetInvoiceDetail(ctx, invID, otherUserID, "applicant")
		assert.Error(t, err)
		assert.Equal(t, ErrForbiddenAccess, err)

		// 3. Admin can access any invoice
		resAdmin, err := svc.GetInvoiceDetail(ctx, invID, otherUserID, "admin")
		assert.NoError(t, err)
		assert.Equal(t, invID, resAdmin.ID)

		// 4. Finance staff can access any invoice
		resStaff, err := svc.GetInvoiceDetail(ctx, invID, otherUserID, "keuangan")
		assert.NoError(t, err)
		assert.Equal(t, invID, resStaff.ID)
	})

	t.Run("get my invoices lists only user's invoices", func(t *testing.T) {
		user1 := uuid.New()
		user2 := uuid.New()

		_ = repo.Create(ctx, &model.Invoice{ID: uuid.New(), UserID: user1, Amount: 100000})
		_ = repo.Create(ctx, &model.Invoice{ID: uuid.New(), UserID: user1, Amount: 200000})
		_ = repo.Create(ctx, &model.Invoice{ID: uuid.New(), UserID: user2, Amount: 300000})

		invoices, err := svc.GetMyInvoices(ctx, user1)
		assert.NoError(t, err)
		assert.Len(t, invoices, 2)
	})

	t.Run("process application withdrawal cancels unpaid invoice", func(t *testing.T) {
		appID := uuid.New()
		inv := &model.Invoice{
			ID:            uuid.New(),
			UserID:        uuid.New(),
			ApplicationID: &appID,
			Amount:        250000,
			Status:        model.InvoiceStatusUnpaid,
			Payments: []model.Payment{
				{
					ID:     uuid.New(),
					Status: model.PaymentStatusPending,
					Amount: 250000,
				},
			},
		}
		_ = repo.Create(ctx, inv)

		err := svc.ProcessApplicationWithdrawal(ctx, appID)
		assert.NoError(t, err)
	})
}

