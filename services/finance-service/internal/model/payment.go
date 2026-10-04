package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentStatusPending             PaymentStatus = "pending"
	PaymentStatusSuccess             PaymentStatus = "success"
	PaymentStatusFailed              PaymentStatus = "failed"
	PaymentStatusExpired             PaymentStatus = "expired"
	PaymentStatusAwaitingVerification PaymentStatus = "awaiting_verification"
	PaymentStatusRejected            PaymentStatus = "rejected"
	PaymentStatusCancelled           PaymentStatus = "cancelled"
	PaymentStatusPaidAfterCancel     PaymentStatus = "paid_after_cancel"
	PaymentStatusRefundReview        PaymentStatus = "refund_review"
)

type PaymentMethod string

const (
	PaymentMethodVirtualAccount    PaymentMethod = "virtual_account"
	PaymentMethodManualTransfer    PaymentMethod = "manual_transfer"
	PaymentMethodManualConfirmation PaymentMethod = "manual_confirmation"
	PaymentMethodQRIS              PaymentMethod = "qris"
	PaymentMethodFree              PaymentMethod = "free"
)

type Payment struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	InvoiceID      uuid.UUID     `gorm:"type:uuid;not null;index" json:"invoice_id"`
	Method         PaymentMethod `gorm:"type:varchar(30);not null" json:"method"`
	Provider       string        `gorm:"type:varchar(50);not null;default:'mock'" json:"provider"`
	ProviderRef    string        `gorm:"type:varchar(100);uniqueIndex;not null" json:"provider_ref"`
	BankReference  string        `gorm:"type:varchar(100);index" json:"bank_reference,omitempty"`
	Channel        string        `gorm:"type:varchar(50)" json:"channel,omitempty"`
	VANumber         string        `gorm:"type:varchar(50)" json:"va_number,omitempty"`
	PaymentURL       string        `gorm:"type:varchar(500)" json:"payment_url,omitempty"`
	PaymentCode      string        `gorm:"type:varchar(50)" json:"payment_code,omitempty"`
	PublisherOrderID string        `gorm:"type:varchar(100)" json:"publisher_order_id,omitempty"`
	Amount           int64         `gorm:"type:bigint;not null" json:"amount"` // Nominal integer Rupiah
	Status           PaymentStatus `gorm:"type:varchar(30);not null;default:'pending';index" json:"status"`
	ProofObjectKey   string        `gorm:"type:varchar(255)" json:"proof_object_key,omitempty"`
	ProofURL         string        `gorm:"type:varchar(500)" json:"proof_url,omitempty"`
	Notes            string        `gorm:"type:text" json:"notes,omitempty"`
	InstallmentID    *uuid.UUID    `gorm:"type:uuid;index" json:"installment_id,omitempty"`
	Installment      *Installment  `gorm:"foreignKey:InstallmentID" json:"installment,omitempty"`
	VerifiedBy       *uuid.UUID    `gorm:"type:uuid" json:"verified_by,omitempty"`
	VerifiedAt       *time.Time    `json:"verified_at,omitempty"`
	PaidAt           *time.Time    `json:"paid_at,omitempty"`
	SettlementDate   *time.Time    `json:"settlement_date,omitempty"`
	Receipt        *Receipt      `gorm:"foreignKey:PaymentID" json:"receipt,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
