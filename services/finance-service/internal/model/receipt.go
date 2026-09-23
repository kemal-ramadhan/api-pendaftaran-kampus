package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Receipt struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PaymentID     uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"payment_id"`
	InvoiceID     uuid.UUID `gorm:"type:uuid;not null;index" json:"invoice_id"`
	ReceiptNumber string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"receipt_number"`
	PDFObjectKey  string    `gorm:"type:varchar(255);not null" json:"pdf_object_key"`
	PDFURL        string    `gorm:"type:varchar(500)" json:"pdf_url,omitempty"`
	Amount        int64     `gorm:"type:bigint;not null" json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

func (r *Receipt) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
