package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InstallmentRequestStatus string

const (
	InstallmentRequestPending  InstallmentRequestStatus = "pending"
	InstallmentRequestApproved InstallmentRequestStatus = "approved"
	InstallmentRequestRejected InstallmentRequestStatus = "rejected"
)

type InstallmentStatus string

const (
	InstallmentStatusUnpaid  InstallmentStatus = "unpaid"
	InstallmentStatusPaid    InstallmentStatus = "paid"
	InstallmentStatusExpired InstallmentStatus = "expired"
)

type InstallmentRequest struct {
	ID             uuid.UUID                `gorm:"type:uuid;primaryKey" json:"id"`
	InvoiceID      uuid.UUID                `gorm:"type:uuid;not null;index" json:"invoice_id"`
	Invoice        *Invoice                 `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	RequestedTerms int                      `gorm:"not null" json:"requested_terms"`
	Reason         string                   `gorm:"type:text" json:"reason"`
	Status         InstallmentRequestStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	ReviewedBy     *uuid.UUID               `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewNote     string                   `gorm:"type:text" json:"review_note,omitempty"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

func (r *InstallmentRequest) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

func (InstallmentRequest) TableName() string {
	return "installment_requests"
}

type Installment struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	InvoiceID uuid.UUID         `gorm:"type:uuid;not null;index" json:"invoice_id"`
	TermNo    int               `gorm:"not null" json:"term_no"`
	Amount    int64             `gorm:"type:bigint;not null" json:"amount"` // Nominal Rupiah integer
	DueDate   time.Time         `gorm:"not null;index" json:"due_date"`
	Status    InstallmentStatus `gorm:"type:varchar(20);not null;default:'unpaid';index" json:"status"`
	PaidAt    *time.Time        `json:"paid_at,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func (i *Installment) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

func (Installment) TableName() string {
	return "installments"
}

// DTO Pengajuan Cicilan
type CreateInstallmentRequestPayload struct {
	RequestedTerms int    `json:"requested_terms" binding:"required,min=2,max=4"`
	Reason         string `json:"reason" binding:"required"`
}

// DTO Peninjauan Cicilan oleh Keuangan
type ReviewInstallmentRequestPayload struct {
	Action     string `json:"action" binding:"required,oneof=approve reject"`
	ReviewNote string `json:"review_note"`
}

// DTO Laporan Ringkasan Keuangan
type FinancialReportSummaryResponse struct {
	TotalInvoices          int64                    `json:"total_invoices"`
	TotalExpectedAmount    int64                    `json:"total_expected_amount"`
	TotalPaidAmount        int64                    `json:"total_paid_amount"`
	TotalOutstandingAmount int64                    `json:"total_outstanding_amount"`
	ByFeeType              []FeeTypeSummaryItem     `json:"by_fee_type"`
	RecentTransactions     []TransactionSummaryItem `json:"recent_transactions"`
}

type FeeTypeSummaryItem struct {
	FeeType           string `json:"fee_type"`
	Count             int64  `json:"count"`
	TotalAmount       int64  `json:"total_amount"`
	PaidAmount        int64  `json:"paid_amount"`
	OutstandingAmount int64  `json:"outstanding_amount"`
}

type TransactionSummaryItem struct {
	PaymentID     uuid.UUID `json:"payment_id"`
	InvoiceNumber string    `json:"invoice_number"`
	FeeType       string    `json:"fee_type"`
	Amount        int64     `json:"amount"`
	Method        string    `json:"method"`
	Status        string    `json:"status"`
	PaidAt        time.Time `json:"paid_at"`
}
