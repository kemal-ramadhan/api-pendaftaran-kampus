package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefundReviewStatus string

const (
	RefundReviewStatusPending   RefundReviewStatus = "pending"
	RefundReviewStatusCompleted RefundReviewStatus = "completed"
	RefundReviewStatusRejected  RefundReviewStatus = "rejected"
)

type RefundReview struct {
	ID            uuid.UUID          `gorm:"type:uuid;primaryKey" json:"id"`
	PaymentID     uuid.UUID          `gorm:"type:uuid;not null;uniqueIndex" json:"payment_id"`
	Payment       *Payment           `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
	InvoiceID     uuid.UUID          `gorm:"type:uuid;not null;index" json:"invoice_id"`
	Invoice       *Invoice           `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	ApplicationID uuid.UUID          `gorm:"type:uuid;not null;index" json:"application_id"`
	Amount        int64              `gorm:"type:bigint;not null" json:"amount"`
	Status        RefundReviewStatus `gorm:"type:varchar(30);not null;default:'pending';index" json:"status"`
	Notes         string             `gorm:"type:text" json:"notes,omitempty"`
	ReviewedBy    *uuid.UUID         `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time         `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func (r *RefundReview) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
