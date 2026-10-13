package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/sender"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/template"
)

var (
	ErrInvalidChannel = errors.New("channel tidak valid, harus 'email' atau 'whatsapp'")
	ErrMissingTarget  = errors.New("email atau nomor telepon tujuan wajib diisi sesuai channel")
)

type QueueNotificationRequest struct {
	IdempotencyKey string                 `json:"idempotency_key"`
	UserID         *uuid.UUID             `json:"user_id,omitempty"`
	Email          string                 `json:"email,omitempty"`
	Phone          string                 `json:"phone,omitempty"`
	Channel        string                 `json:"channel"`
	Template       string                 `json:"template"`
	Data           map[string]interface{} `json:"data"`
}

type NotificationService interface {
	QueueNotification(ctx context.Context, req *QueueNotificationRequest) (*model.NotificationLog, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.NotificationLog, error)
	ProcessItem(ctx context.Context, item *model.NotificationLog) error
	ProcessPending(ctx context.Context, limit int) (int, error)
}

type notificationService struct {
	repo       repository.NotificationRepository
	emailSnd   sender.EmailSender
	waSnd      sender.WhatsAppSender
	tmplEngine template.TemplateEngine
}

func NewNotificationService(
	repo repository.NotificationRepository,
	emailSnd sender.EmailSender,
	waSnd sender.WhatsAppSender,
	tmplEngine template.TemplateEngine,
) NotificationService {
	return &notificationService{
		repo:       repo,
		emailSnd:   emailSnd,
		waSnd:      waSnd,
		tmplEngine: tmplEngine,
	}
}

func (s *notificationService) QueueNotification(ctx context.Context, req *QueueNotificationRequest) (*model.NotificationLog, error) {
	if req.Channel != "email" && req.Channel != "whatsapp" {
		return nil, ErrInvalidChannel
	}
	if req.Channel == "email" && req.Email == "" {
		return nil, ErrMissingTarget
	}
	if req.Channel == "whatsapp" && req.Phone == "" {
		return nil, ErrMissingTarget
	}

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = uuid.New().String()
	}

	// Cek IdempotencyKey
	existing, err := s.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	dataBytes, _ := json.Marshal(req.Data)

	notifLog := &model.NotificationLog{
		ID:             uuid.New(),
		IdempotencyKey: req.IdempotencyKey,
		UserID:         req.UserID,
		Email:          req.Email,
		Phone:          req.Phone,
		Channel:        req.Channel,
		Template:       req.Template,
		Data:           string(dataBytes),
		Status:         model.StatusQueued,
		Retries:        0,
	}

	if err := s.repo.Create(ctx, notifLog); err != nil {
		return nil, fmt.Errorf("gagal menyimpan log notifikasi: %w", err)
	}

	// Trigger kirim langsung secara asinkron
	go func(item model.NotificationLog) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.ProcessItem(bgCtx, &item); err != nil {
			log.Printf("[NOTIFICATION-WORKER] Gagal mengirim notifikasi ID %s: %v", item.ID, err)
		}
	}(*notifLog)

	return notifLog, nil
}

func (s *notificationService) GetByID(ctx context.Context, id uuid.UUID) (*model.NotificationLog, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *notificationService) ProcessItem(ctx context.Context, item *model.NotificationLog) error {
	var sendErr error

	if item.Channel == "email" {
		rendered, err := s.tmplEngine.RenderEmail(item.Template, item.Data)
		if err != nil {
			sendErr = fmt.Errorf("gagal render template email: %w", err)
		} else {
			sendErr = s.emailSnd.Send(ctx, item.Email, rendered.Subject, rendered.Body)
		}
	} else if item.Channel == "whatsapp" {
		renderedText, err := s.tmplEngine.RenderWhatsApp(item.Template, item.Data)
		if err != nil {
			sendErr = fmt.Errorf("gagal render template whatsapp: %w", err)
		} else {
			sendErr = s.waSnd.Send(ctx, item.Phone, renderedText)
		}
	} else {
		sendErr = fmt.Errorf("channel tidak didukung: %s", item.Channel)
	}

	if sendErr == nil {
		return s.repo.UpdateStatus(ctx, item.ID, model.StatusSent, item.Retries, "")
	}

	newRetries := item.Retries + 1
	newStatus := model.StatusFailed
	if newRetries < 3 {
		newStatus = model.StatusQueued
	}

	_ = s.repo.UpdateStatus(ctx, item.ID, newStatus, newRetries, sendErr.Error())
	return sendErr
}

func (s *notificationService) ProcessPending(ctx context.Context, limit int) (int, error) {
	items, err := s.repo.GetPendingQueued(ctx, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, item := range items {
		itemCopy := item
		if err := s.ProcessItem(ctx, &itemCopy); err == nil {
			processed++
		}
	}

	return processed, nil
}
