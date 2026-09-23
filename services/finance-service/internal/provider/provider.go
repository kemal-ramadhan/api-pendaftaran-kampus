package provider

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VACreationResult struct {
	ProviderRef string
	VANumber    string
	ExpiresAt   time.Time
}

type WebhookPayload struct {
	Provider    string    `json:"provider"`
	ProviderRef string    `json:"provider_ref"`
	VANumber    string    `json:"va_number"`
	Amount      int64     `json:"amount"`
	Status      string    `json:"status"` // success, failed
	PaidAt      time.Time `json:"paid_at"`
}

type PaymentProvider interface {
	CreateVA(ctx context.Context, invoiceID uuid.UUID, amount int64) (*VACreationResult, error)
	ParseWebhook(ctx context.Context, body []byte) (*WebhookPayload, error)
	VerifySignature(secret string, signatureHeader string, body []byte) bool
}
