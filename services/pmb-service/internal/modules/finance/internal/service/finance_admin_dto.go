package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
)

// ReconciliationKPIs DTO untuk kartu KPI Rekonsiliasi (PRD 2.3)
type ReconciliationKPIs struct {
	PaidFormsCount      int64              `json:"paid_forms_count"`
	TotalFormsCount     int64              `json:"total_forms_count"`
	PaidFormsPercentage float64            `json:"paid_forms_percentage"`
	PendingManualCount  int64              `json:"pending_manual_count"`
	PaidTuitionCount    int64              `json:"paid_tuition_count"`
	TotalTuitionCount   int64              `json:"total_tuition_count"`
	TotalTuitionAmount  int64              `json:"total_tuition_amount"`
	ChannelPercentages  map[string]float64 `json:"channel_percentages"`
}

type ChannelDistributionItem struct {
	Channel    string  `json:"channel"`
	Count      int64   `json:"count"`
	Total      int64   `json:"total"`
	Percentage float64 `json:"percentage"`
}

type FundProportions struct {
	FormAmount    int64 `json:"form_amount"`
	TuitionAmount int64 `json:"tuition_amount"`
}

type ReconciliationInvoiceItem struct {
	ID            uuid.UUID           `json:"id"`
	InvoiceNumber string              `json:"invoice_number"`
	FullName      string              `json:"full_name"`
	ProgramName   string              `json:"program_name"`
	FeeType       string              `json:"fee_type"`
	Amount        int64               `json:"amount"`
	PaidAmount    int64               `json:"paid_amount"`
	Channel       string              `json:"channel"`
	Status        model.InvoiceStatus `json:"status"`
	StatusLabel   string              `json:"status_label"`
	DueDate       time.Time           `json:"due_date"`
	PaidAt        *time.Time          `json:"paid_at,omitempty"`
}

type ReconciliationResponse struct {
	PeriodID             *uint                       `json:"period_id,omitempty"`
	KPIs                 ReconciliationKPIs          `json:"kpis"`
	FundProportions      FundProportions             `json:"fund_proportions"`
	ChannelDistributions []ChannelDistributionItem   `json:"channel_distributions"`
	Items                []ReconciliationInvoiceItem `json:"items"`
	Meta                 struct {
		Total int64 `json:"total"`
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
	} `json:"meta"`
}

// ManualConfirmRequest DTO untuk konfirmasi lunas manual (PRD 2.3)
type ManualConfirmRequest struct {
	Amount          int64     `json:"amount" binding:"required"`
	TransactionTime time.Time `json:"transaction_time"`
	Channel         string    `json:"channel" binding:"required"`
	BankReference   string    `json:"bank_reference" binding:"required"`
	ProofURL        string    `json:"proof_url" binding:"required"`
	Reason          string    `json:"reason" binding:"required"`
}

// LedgerItem DTO baris buku besar kas masuk (PRD 2.7)
type LedgerItem struct {
	TransactionTime time.Time `json:"transaction_time"`
	InvoiceNumber   string    `json:"invoice_number"`
	PayerName       string    `json:"payer_name"`
	FeeType         string    `json:"fee_type"`
	Channel         string    `json:"channel"`
	ReferenceNumber string    `json:"reference_number"`
	CreditAmount    int64     `json:"credit_amount"`
	RunningBalance  int64     `json:"running_balance"`
	SourceLabel     string    `json:"source_label"`
}

type LedgerSummaryCard struct {
	Channel      string `json:"channel"`
	TotalAmount  int64  `json:"total_amount"`
	TotalLabel   string `json:"total_label"`
	CountRecords int64  `json:"count_records"`
}

type LedgerResponse struct {
	PeriodID     *uint               `json:"period_id,omitempty"`
	SummaryCards []LedgerSummaryCard `json:"summary_cards"`
	Items        []LedgerItem        `json:"items"`
	TotalCredit  int64               `json:"total_credit"`
	Meta         struct {
		Total int64 `json:"total"`
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
	} `json:"meta"`
}

// PaymentMethodTabItem DTO metode bayar per tab (PRD 3.5)
type PaymentMethodTabItem struct {
	Group         string `json:"group"` // virtual_account | bank_transfer | qris
	Code          string `json:"code"`
	Name          string `json:"name"`
	BankName      string `json:"bank_name"`
	BadgeLabel    string `json:"badge_label"`
	Fee           int64  `json:"fee"` // 0 jika ditanggung kampus
	FeeLabel      string `json:"fee_label"`
	AccountNumber string `json:"account_number,omitempty"`
	AccountName   string `json:"account_name,omitempty"`
	PaymentURL    string `json:"payment_url,omitempty"`
	QRString      string `json:"qr_string,omitempty"`
}

type PaymentMethodsGroupedResponse struct {
	VirtualAccount []PaymentMethodTabItem `json:"virtual_account"`
	BankTransfer   []PaymentMethodTabItem `json:"bank_transfer"`
	QRIS           []PaymentMethodTabItem `json:"qris"`
	Notes          []string               `json:"payment_info_notes"`
}

// ReceiptViewResponse DTO status pelunasan resmi camaba (PRD 3.8)
type ReceiptViewResponse struct {
	Title              string             `json:"title"`
	InvoiceNumber      string             `json:"invoice_number"`
	FullName           string             `json:"full_name"`
	RegistrationNumber string             `json:"registration_number"`
	ProgramName        string             `json:"program_name"`
	PaymentMethodLabel string             `json:"payment_method_label"`
	VerifiedAt         *time.Time         `json:"verified_at,omitempty"`
	TotalAmount        int64              `json:"total_amount"`
	TotalAmountLabel   string             `json:"total_amount_label"`
	LockNoticeLabel    string             `json:"lock_notice_label"`
	NextAction         *ReceiptNextAction `json:"next_action,omitempty"`
}

type ReceiptNextAction struct {
	Title       string `json:"title"`
	ActionURL   string `json:"action_url"`
	ButtonLabel string `json:"button_label"`
}
