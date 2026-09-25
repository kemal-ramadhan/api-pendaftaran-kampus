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

type PaymentRequest struct {
	InvoiceID       uuid.UUID
	PaymentID       uuid.UUID
	MerchantOrderID string
	Amount          int64
	ProductDetails  string
	Email           string
	PhoneNumber     string
	CustomerName    string
	PaymentMethod   string
}

type PaymentResult struct {
	ProviderRef      string
	PaymentURL       string
	VANumber         string
	PaymentCode      string
	PublisherOrderID string
	Amount           int64
	ExpiresAt        time.Time
}

type PaymentStatusResult struct {
	MerchantOrderID string
	StatusCode      string // "00" sukses, "01" pending, "02" gagal/expired
	StatusMessage   string
	Amount          int64
	Reference       string
	PaidAt          time.Time
}

type WebhookPayload struct {
	Provider         string    `json:"provider"`
	ProviderRef      string    `json:"provider_ref"`
	VANumber         string    `json:"va_number"`
	Amount           int64     `json:"amount"`
	Status           string    `json:"status"` // success, failed
	PaymentCode      string    `json:"payment_code,omitempty"`
	PublisherOrderID string    `json:"publisher_order_id,omitempty"`
	PaidAt           time.Time `json:"paid_at"`
}

type PaymentProvider interface {
	CreateVA(ctx context.Context, invoiceID uuid.UUID, amount int64) (*VACreationResult, error)
	CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResult, error)
	ParseWebhook(ctx context.Context, body []byte) (*WebhookPayload, error)
	VerifySignature(secret string, signatureHeader string, body []byte) bool
	CheckStatus(ctx context.Context, merchantOrderID string) (*PaymentStatusResult, error)
}
