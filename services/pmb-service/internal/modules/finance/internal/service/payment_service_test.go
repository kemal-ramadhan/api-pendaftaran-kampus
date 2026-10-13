package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/pdf"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/provider"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type mockPaymentRepo struct {
	payments map[uuid.UUID]*model.Payment
}

func newMockPaymentRepo() *mockPaymentRepo {
	return &mockPaymentRepo{
		payments: make(map[uuid.UUID]*model.Payment),
	}
}

func (m *mockPaymentRepo) Create(ctx context.Context, p *model.Payment) error {
	m.payments[p.ID] = p
	return nil
}

func (m *mockPaymentRepo) CreateInTx(tx *gorm.DB, p *model.Payment) error {
	m.payments[p.ID] = p
	return nil
}

func (m *mockPaymentRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	if p, ok := m.payments[id]; ok {
		return p, nil
	}
	return nil, repository.ErrPaymentNotFound
}

func (m *mockPaymentRepo) FindByProviderRef(ctx context.Context, ref string) (*model.Payment, error) {
	for _, p := range m.payments {
		if p.ProviderRef == ref {
			return p, nil
		}
	}
	return nil, repository.ErrPaymentNotFound
}

func (m *mockPaymentRepo) Update(ctx context.Context, p *model.Payment) error {
	m.payments[p.ID] = p
	return nil
}

func (m *mockPaymentRepo) UpdateInTx(tx *gorm.DB, p *model.Payment) error {
	m.payments[p.ID] = p
	return nil
}

func (m *mockPaymentRepo) ListByStatus(ctx context.Context, status model.PaymentStatus, limit, offset int) ([]model.Payment, int64, error) {
	var list []model.Payment
	for _, p := range m.payments {
		if p.Status == status {
			list = append(list, *p)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockPaymentRepo) DB() *gorm.DB {
	return nil
}

func (m *mockPaymentRepo) ListAwaitingVerification(ctx context.Context, limit, offset int) ([]model.Payment, int64, error) {
	return m.ListByStatus(ctx, model.PaymentStatusAwaitingVerification, limit, offset)
}

func (m *mockPaymentRepo) FindPendingWithProviderRef(ctx context.Context, limit int) ([]model.Payment, error) {
	var list []model.Payment
	for _, p := range m.payments {
		if p.Status == model.PaymentStatusPending && p.ProviderRef != "" {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockPaymentRepo) FindByBankReference(ctx context.Context, bankRef string) (*model.Payment, error) {
	for _, p := range m.payments {
		if p.BankReference == bankRef {
			return p, nil
		}
	}
	return nil, repository.ErrPaymentNotFound
}

func (m *mockPaymentRepo) FindSuccessfulPayments(ctx context.Context, periodID *uint, channel string, from, to *time.Time) ([]model.Payment, error) {
	var list []model.Payment
	for _, p := range m.payments {
		if p.Status == model.PaymentStatusSuccess {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockPaymentRepo) UpdateStatusInTx(tx *gorm.DB, paymentID uuid.UUID, status model.PaymentStatus, notes string, verifiedBy *uuid.UUID, paidAt *time.Time) error {
	if p, ok := m.payments[paymentID]; ok {
		p.Status = status
		p.Notes = notes
		p.VerifiedBy = verifiedBy
		p.PaidAt = paidAt
		return nil
	}
	return repository.ErrPaymentNotFound
}

type mockReceiptRepo struct {
	receipts map[uuid.UUID]*model.Receipt
}

func newMockReceiptRepo() *mockReceiptRepo {
	return &mockReceiptRepo{
		receipts: make(map[uuid.UUID]*model.Receipt),
	}
}

func (m *mockReceiptRepo) Create(ctx context.Context, r *model.Receipt) error {
	m.receipts[r.ID] = r
	return nil
}

func (m *mockReceiptRepo) CreateInTx(tx *gorm.DB, r *model.Receipt) error {
	m.receipts[r.ID] = r
	return nil
}

func (m *mockReceiptRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Receipt, error) {
	if r, ok := m.receipts[id]; ok {
		return r, nil
	}
	return nil, repository.ErrReceiptNotFound
}

func (m *mockReceiptRepo) FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*model.Receipt, error) {
	for _, r := range m.receipts {
		if r.PaymentID == paymentID {
			return r, nil
		}
	}
	return nil, repository.ErrReceiptNotFound
}

func (m *mockReceiptRepo) FindByReceiptNumber(ctx context.Context, receiptNo string) (*model.Receipt, error) {
	for _, r := range m.receipts {
		if r.ReceiptNumber == receiptNo {
			return r, nil
		}
	}
	return nil, repository.ErrReceiptNotFound
}

func (m *mockReceiptRepo) GenerateReceiptNumber(ctx context.Context, tx *gorm.DB) (string, error) {
	return "KW-2026-00001", nil
}

type mockWebhookLogRepo struct {
	logs map[string]*model.PaymentWebhookLog
}

func newMockWebhookLogRepo() *mockWebhookLogRepo {
	return &mockWebhookLogRepo{
		logs: make(map[string]*model.PaymentWebhookLog),
	}
}

func (m *mockWebhookLogRepo) Create(ctx context.Context, l *model.PaymentWebhookLog) error {
	m.logs[l.ProviderRef] = l
	return nil
}

func (m *mockWebhookLogRepo) FindByProviderRef(ctx context.Context, providerRef string) (*model.PaymentWebhookLog, error) {
	if l, ok := m.logs[providerRef]; ok {
		return l, nil
	}
	return nil, gorm.ErrRecordNotFound
}

type mockStorageClient struct {
	files map[string][]byte
}

func newMockStorageClient() *mockStorageClient {
	return &mockStorageClient{
		files: make(map[string][]byte),
	}
}

func (m *mockStorageClient) EnsureBucket(ctx context.Context, bucketName string) error {
	return nil
}

func (m *mockStorageClient) Upload(ctx context.Context, bucketName, objectKey string, reader io.Reader, size int64, contentType string) error {
	buf := new(strings.Builder)
	_, _ = io.Copy(buf, reader)
	url := fmt.Sprintf("http://minio:9000/%s/%s", bucketName, objectKey)
	m.files[url] = []byte(buf.String())
	return nil
}

func (m *mockStorageClient) Download(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error) {
	url := fmt.Sprintf("http://minio:9000/%s/%s", bucketName, objectKey)
	if b, ok := m.files[url]; ok {
		return io.NopCloser(bytes.NewReader(b)), nil
	}
	return io.NopCloser(bytes.NewReader([]byte("dummy-content"))), nil
}

func (m *mockStorageClient) PresignedGetObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("http://minio:9000/%s/%s", bucketName, objectKey), nil
}

func (m *mockStorageClient) PresignedPutObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("http://minio:9000/%s/%s", bucketName, objectKey), nil
}

func (m *mockStorageClient) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	url := fmt.Sprintf("http://minio:9000/%s/%s", bucketName, objectKey)
	delete(m.files, url)
	return nil
}

func (m *mockStorageClient) SniffAndValidate(reader io.Reader, allowedTypes []string) (io.Reader, string, error) {
	return reader, "image/jpeg", nil
}

type mockInternalClient struct {
	mu    sync.Mutex
	calls []string
}

func (m *mockInternalClient) Get(ctx context.Context, path string, query map[string]string, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, path)
	return nil
}

func (m *mockInternalClient) Post(ctx context.Context, path string, body interface{}, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, path)
	return nil
}

func (m *mockInternalClient) PostMultipart(ctx context.Context, path string, fieldName, fileName string, fileReader io.Reader, extraFields map[string]string, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, path)
	return nil
}

func (m *mockInternalClient) Put(ctx context.Context, path string, body interface{}, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, path)
	return nil
}

func (m *mockInternalClient) Patch(ctx context.Context, path string, body interface{}, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, path)
	return nil
}

func (m *mockInternalClient) Delete(ctx context.Context, path string, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, path)
	return nil
}

func (m *mockInternalClient) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]string, len(m.calls))
	copy(copied, m.calls)
	return copied
}

func TestPaymentService_VirtualAccountAndDevSimulate(t *testing.T) {
	ctx := context.Background()
	invRepo := newMockInvoiceRepo()
	paymentRepo := newMockPaymentRepo()
	receiptRepo := newMockReceiptRepo()
	webhookRepo := newMockWebhookLogRepo()
	storageCli := newMockStorageClient()
	admissionCli := &mockInternalClient{}
	notifCli := &mockInternalClient{}
	mockProv := provider.NewMockProvider()

	cfg := &config.Config{
		AppEnv: "development",
	}

	svc := NewPaymentService(
		paymentRepo,
		invRepo,
		receiptRepo,
		webhookRepo,
		mockProv,
		storageCli,
		admissionCli,
		notifCli,
		cfg,
	)

	userID := uuid.New()
	appID := uuid.New()
	invID := uuid.New()

	inv := &model.Invoice{
		ID:            invID,
		InvoiceNumber: "INV-2026-00001",
		UserID:        userID,
		ApplicationID: &appID,
		FeeType:       "registration",
		Amount:        250000,
		PaidAmount:    0,
		Status:        model.InvoiceStatusUnpaid,
		DueDate:       time.Now().Add(24 * time.Hour),
	}
	_ = invRepo.Create(ctx, inv)

	t.Run("success initiate virtual account", func(t *testing.T) {
		pmt, err := svc.PayWithVirtualAccount(ctx, invID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, pmt)
		assert.Equal(t, model.PaymentMethodVirtualAccount, pmt.Method)
		assert.Equal(t, model.PaymentStatusPending, pmt.Status)
		assert.Equal(t, int64(250000), pmt.Amount)
		assert.NotEmpty(t, pmt.VANumber)
		assert.NotEmpty(t, pmt.ProviderRef)

		// Calling again returns the existing pending payment
		inv.Payments = []model.Payment{*pmt}
		pmt2, err := svc.PayWithVirtualAccount(ctx, invID, userID)
		assert.NoError(t, err)
		assert.Equal(t, pmt.ID, pmt2.ID)
	})

	t.Run("cannot initiate payment for another user invoice", func(t *testing.T) {
		otherUser := uuid.New()
		_, err := svc.PayWithVirtualAccount(ctx, invID, otherUser)
		assert.Error(t, err)
		assert.Equal(t, ErrForbiddenAccess, err)
	})

	t.Run("simulate-paid succeeds in development and updates admission and invoice", func(t *testing.T) {
		pmt, err := paymentRepo.FindByProviderRef(ctx, inv.Payments[0].ProviderRef)
		assert.NoError(t, err)

		paidPmt, err := svc.SimulatePaid(ctx, pmt.ID)
		assert.NoError(t, err)
		assert.Equal(t, model.PaymentStatusSuccess, paidPmt.Status)
		assert.NotNil(t, paidPmt.PaidAt)

		// Invoice status updated to paid
		updatedInv, err := invRepo.FindByID(ctx, invID)
		assert.NoError(t, err)
		assert.Equal(t, model.InvoiceStatusPaid, updatedInv.Status)
		assert.Equal(t, int64(250000), updatedInv.PaidAmount)

		// Receipt created
		rcp, err := receiptRepo.FindByPaymentID(ctx, pmt.ID)
		assert.NoError(t, err)
		assert.NotNil(t, rcp)
		assert.NotEmpty(t, rcp.ReceiptNumber)

		// Admission status callback triggered (form_open) asynchronously
		assert.Eventually(t, func() bool {
			for _, call := range admissionCli.GetCalls() {
				if call == "/internal/applications/"+appID.String()+"/status" {
					return true
				}
			}
			return false
		}, 500*time.Millisecond, 10*time.Millisecond)
	})

	t.Run("simulate-paid fails in production", func(t *testing.T) {
		prodCfg := &config.Config{AppEnv: "production"}
		prodSvc := NewPaymentService(paymentRepo, invRepo, receiptRepo, webhookRepo, mockProv, storageCli, admissionCli, notifCli, prodCfg)

		_, err := prodSvc.SimulatePaid(ctx, uuid.New())
		assert.Error(t, err)
		assert.Equal(t, ErrDevEndpointDisabled, err)
	})
}

func TestPDFReceiptGenerator(t *testing.T) {
	data := pdf.ReceiptData{
		ReceiptNumber: "KW-2026-00001",
		InvoiceNumber: "INV-2026-00001",
		UserName:      "Budi Prakoso",
		Amount:        250000,
		Method:        "VIRTUAL_ACCOUNT",
		PaidAt:        time.Now(),
		Description:   "Biaya Formulir Registrasi PMB 2026/2027",
	}

	pdfBytes, err := pdf.GenerateReceiptPDF(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Validasi header PDF standar
	assert.True(t, strings.HasPrefix(string(pdfBytes), "%PDF-1.4"))
	// Validasi EOF PDF standar
	assert.True(t, strings.Contains(string(pdfBytes), "%%EOF"))
}
