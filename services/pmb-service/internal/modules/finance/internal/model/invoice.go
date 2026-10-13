package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceStatus string

const (
	InvoiceStatusUnpaid        InvoiceStatus = "unpaid"
	InvoiceStatusPartiallyPaid InvoiceStatus = "partially_paid"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusCancelled     InvoiceStatus = "cancelled"
	InvoiceStatusExpired       InvoiceStatus = "expired"
)

type Invoice struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	InvoiceNumber string        `gorm:"type:varchar(50);uniqueIndex;not null" json:"invoice_number"`
	UserID        uuid.UUID     `gorm:"type:uuid;index;not null" json:"user_id"`
	ApplicationID *uuid.UUID    `gorm:"type:uuid;index" json:"application_id,omitempty"`
	FeeType       string        `gorm:"type:varchar(50);not null" json:"fee_type"` // registration, reregistration
	Description   string        `gorm:"type:text" json:"description"`
	Amount        int64         `gorm:"type:bigint;not null" json:"amount"`      // Nominal integer Rupiah
	PaidAmount    int64         `gorm:"type:bigint;not null;default:0" json:"paid_amount"`
	Status        InvoiceStatus `gorm:"type:varchar(30);not null;default:'unpaid';index" json:"status"`
	DueDate       time.Time     `gorm:"not null" json:"due_date"`
	RenewalCount  int           `gorm:"not null;default:0" json:"renewal_count"`
	PeriodID      *uint         `gorm:"index" json:"period_id,omitempty"`
	PaymentScheme       string               `gorm:"type:varchar(20);not null;default:'full'" json:"payment_scheme"` // full, installment
	Payments            []Payment            `gorm:"foreignKey:InvoiceID" json:"payments,omitempty"`
	InstallmentRequests []InstallmentRequest `gorm:"foreignKey:InvoiceID" json:"installment_requests,omitempty"`
	Installments        []Installment        `gorm:"foreignKey:InvoiceID" json:"installments,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func (i *Invoice) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

func (i *Invoice) IsPaid() bool {
	return i.Status == InvoiceStatusPaid
}
