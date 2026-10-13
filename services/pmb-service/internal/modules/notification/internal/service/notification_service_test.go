package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/template"
	"github.com/stretchr/testify/assert"
)

type mockNotificationRepo struct {
	logs map[string]*model.NotificationLog
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{
		logs: make(map[string]*model.NotificationLog),
	}
}

func (m *mockNotificationRepo) Create(ctx context.Context, log *model.NotificationLog) error {
	m.logs[log.IdempotencyKey] = log
	return nil
}

func (m *mockNotificationRepo) FindByIdempotencyKey(ctx context.Context, key string) (*model.NotificationLog, error) {
	if l, ok := m.logs[key]; ok {
		return l, nil
	}
	return nil, nil
}

func (m *mockNotificationRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.NotificationLog, error) {
	for _, l := range m.logs {
		if l.ID == id {
			return l, nil
		}
	}
	return nil, nil
}

func (m *mockNotificationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, retries int, errorMsg string) error {
	for _, l := range m.logs {
		if l.ID == id {
			l.Status = status
			l.Retries = retries
			l.ErrorMessage = errorMsg
			return nil
		}
	}
	return nil
}

func (m *mockNotificationRepo) GetPendingQueued(ctx context.Context, limit int) ([]model.NotificationLog, error) {
	var res []model.NotificationLog
	for _, l := range m.logs {
		if (l.Status == model.StatusQueued || l.Status == model.StatusFailed) && l.Retries < 3 {
			res = append(res, *l)
		}
	}
	return res, nil
}

type mockEmailSender struct {
	sentCount int
}

func (m *mockEmailSender) Send(ctx context.Context, to string, subject string, htmlBody string) error {
	m.sentCount++
	return nil
}

type mockWASender struct {
	sentCount int
}

func (m *mockWASender) Send(ctx context.Context, phone string, message string) error {
	m.sentCount++
	return nil
}

func TestNotificationService_QueueNotification(t *testing.T) {
	repo := newMockNotificationRepo()
	emailSnd := &mockEmailSender{}
	waSnd := &mockWASender{}
	tmplEngine := template.NewTemplateEngine()

	svc := NewNotificationService(repo, emailSnd, waSnd, tmplEngine)
	ctx := context.Background()

	t.Run("valid email notification", func(t *testing.T) {
		req := &QueueNotificationRequest{
			IdempotencyKey: "test-key-1",
			Email:          "budi@example.com",
			Channel:        "email",
			Template:       "email_verification",
			Data: map[string]interface{}{
				"name":             "Budi",
				"code":             "123456",
				"verification_url": "https://pmb.ac.id/verify",
			},
		}

		res, err := svc.QueueNotification(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "test-key-1", res.IdempotencyKey)
		assert.Equal(t, model.StatusQueued, res.Status)

		// Idempotency: kirim lagi dengan key yang sama tidak boleh duplicate
		resDuplicate, err := svc.QueueNotification(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, res.ID, resDuplicate.ID)
	})

	t.Run("valid whatsapp notification", func(t *testing.T) {
		req := &QueueNotificationRequest{
			IdempotencyKey: "test-key-2",
			Phone:          "08123456789",
			Channel:        "whatsapp",
			Template:       "application_registered",
			Data: map[string]interface{}{
				"name":            "Siti",
				"registration_no": "PMB-2026-1-00001",
			},
		}

		res, err := svc.QueueNotification(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "test-key-2", res.IdempotencyKey)
	})

	t.Run("invalid channel returns error", func(t *testing.T) {
		req := &QueueNotificationRequest{
			Channel: "sms",
		}
		_, err := svc.QueueNotification(ctx, req)
		assert.ErrorIs(t, err, ErrInvalidChannel)
	})

	t.Run("missing email for email channel returns error", func(t *testing.T) {
		req := &QueueNotificationRequest{
			Channel: "email",
		}
		_, err := svc.QueueNotification(ctx, req)
		assert.ErrorIs(t, err, ErrMissingTarget)
	})
}
