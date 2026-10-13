package finance

import (
	"context"

	"github.com/google/uuid"
)

type InvoiceDTO struct {
	ID            uuid.UUID `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	UserID        uuid.UUID `json:"user_id"`
	Amount        float64   `json:"amount"`
	FeeType       string    `json:"fee_type"`
	Status        string    `json:"status"` // pending, paid, expired, cancelled
}

type API interface {
	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*InvoiceDTO, error)
	CheckPaymentStatus(ctx context.Context, invoiceID uuid.UUID) (bool, error)
}
