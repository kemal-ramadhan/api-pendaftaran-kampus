package dto

import (
	"time"

	"github.com/google/uuid"
)

type CheckCodeResponse struct {
	Valid           bool   `json:"valid"`
	InstitutionName string `json:"institution_name,omitempty"`
	PICName         string `json:"pic_name,omitempty"`
}

type AttachReferralRequest struct {
	Code    string `json:"code" binding:"required"`
	Consent bool   `json:"consent" binding:"required"`
}

type ApproveTransactionsRequest struct {
	IDs []uuid.UUID `json:"ids" binding:"required"`
}

type RejectTransactionRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SubmitSettlementRequest struct {
	TransferDate   string  `json:"transfer_date" binding:"required"` // YYYY-MM-DD
	BankName       string  `json:"bank_name" binding:"required"`
	ReferenceNo    string  `json:"reference_no" binding:"required"`
	Notes          string  `json:"notes"`
	ProofObjectKey string  `json:"proof_object_key"`
}

type CreateCommissionPayoutRequest struct {
	CodeID         uuid.UUID   `json:"code_id" binding:"required"`
	Amount         float64     `json:"amount" binding:"required"`
	TransactionIDs []uuid.UUID `json:"transaction_ids"`
	Method         string      `json:"method" binding:"required"`
	ReferenceNo    string      `json:"reference_no"`
	ProofObjectKey string      `json:"proof_object_key"`
}

type CreatePICRequest struct {
	Name            string  `json:"nama" binding:"required"`
	Email           string  `json:"email" binding:"required,email"`
	Phone           string  `json:"no_hp"`
	InstitutionName string  `json:"nama_institusi" binding:"required"`
	InstitutionType string  `json:"jenis_institusi"`
	NPSN            string  `json:"npsn"`
	Code            string  `json:"kode"` // Opsional: jika kosong di-generate
	CommissionType  string  `json:"tipe_komisi"` // persen / nominal / tidak_ada
	CommissionValue float64 `json:"nilai_komisi"`
	SendActivation  bool    `json:"kirim_aktivasi"`
}

type ReferralSummaryResponse struct {
	TotalUsers               int64   `json:"total_users"`
	AllTransactionsSuccess   float64 `json:"all_transactions_success"`
	MyBillToCampus           float64 `json:"my_bill_to_campus"`
	DepositedToCampus        float64 `json:"deposited_to_campus"`
	PendingVerificationQueue float64 `json:"pending_verification_queue"`
	MyCommissionTotal        float64 `json:"my_commission_total"`
	MyCommissionPaid         float64 `json:"my_commission_paid"`
	MyCommissionPending      float64 `json:"my_commission_pending"`
	SettlementPendingAmount  float64 `json:"settlement_pending_amount"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type ReferralUserItem struct {
	No                 int        `json:"no"`
	ID                 uuid.UUID  `json:"id"`
	ApplicationID      uuid.UUID  `json:"application_id"`
	Name               string     `json:"nama"`
	Email              string     `json:"email"`
	NIK                string     `json:"nik"` // Disamarkan untuk guru BK (misal: 3204********0002)
	NISN               string     `json:"nisn"`
	PathName           string     `json:"jalur"`
	Amount             float64    `json:"nominal"`
	Type               string     `json:"tipe"`
	ApproveStatus      string     `json:"status_approve"`
	ApproveStatusLabel string     `json:"status_approve_label"`
	CampusStatus       string     `json:"keuangan_kampus"`
	CampusStatusLabel  string     `json:"keuangan_kampus_label"`
	CommissionAmount   float64    `json:"komisi"`
	AttachedAt         time.Time  `json:"tanggal_pakai_kode"`
	ProgramName        string     `json:"prodi_pilihan"`
	RegistrationStep   string     `json:"tahap_pendaftaran"`
}

type PartnerDashboardResponse struct {
	Profile struct {
		ID       uuid.UUID `json:"id"`
		FullName string    `json:"full_name"`
		Email    string    `json:"email"`
		Phone    string    `json:"phone"`
	} `json:"profile"`
	TotalCodes            int                     `json:"total_codes"`
	Summary               ReferralSummaryResponse `json:"summary"`
	PendingTransactions   []ReferralUserItem      `json:"pending_transactions"`
	WAGroupGuruBKURL      string                  `json:"wa_group_guru_bk_url"`
	WAGroupGuruBKName     string                  `json:"wa_group_guru_bk_name"`
	CampusContact         string                  `json:"campus_contact"`
}
