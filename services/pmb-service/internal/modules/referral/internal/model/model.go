package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReferralCode struct {
	ID                       uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Code                     string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	PaymentCode              string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"payment_code"`
	InstitutionName          string         `gorm:"type:varchar(255);not null" json:"institution_name"`
	InstitutionType          string         `gorm:"type:varchar(50);not null;default:'sekolah'" json:"institution_type"` // sekolah / program / komunitas
	NPSN                     string         `gorm:"type:varchar(20)" json:"npsn"`
	Description              string         `gorm:"type:text" json:"description"`
	Status                   string         `gorm:"type:varchar(20);not null;default:'active'" json:"status"` // active / inactive
	ValidFrom                *time.Time     `gorm:"type:timestamptz" json:"valid_from"`
	ValidUntil               *time.Time     `gorm:"type:timestamptz" json:"valid_until"`
	PeriodID                 *uuid.UUID     `gorm:"type:uuid;index" json:"period_id"`
	CommissionType           string         `gorm:"type:varchar(20);not null;default:'persen'" json:"commission_type"` // persen / nominal / tidak_ada
	CommissionValue          float64        `gorm:"type:numeric(12,2);not null;default:0" json:"commission_value"`
	CommissionSettlementMode string         `gorm:"type:varchar(30);default:''" json:"commission_settlement_mode"` // terpisah / potong_setoran / "" (ikut global)
	CreatedBy                *uuid.UUID     `gorm:"type:uuid" json:"created_by"`
	CreatedAt                time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt                time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"-"`

	PICs []ReferralCodePIC `gorm:"foreignKey:CodeID" json:"pics,omitempty"`
}

func (r *ReferralCode) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type ReferralCodePIC struct {
	CodeID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"code_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	IsPrimary bool      `gorm:"default:true" json:"is_primary"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

type ReferralUsage struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CodeID        uuid.UUID  `gorm:"type:uuid;index;not null" json:"code_id"`
	ApplicationID uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"application_id"`
	UserID        uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"` // Camaba
	ConsentAt     *time.Time `gorm:"type:timestamptz" json:"consent_at"`
	AttachedAt    time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"attached_at"`
	LockedAt      *time.Time `gorm:"type:timestamptz" json:"locked_at"`
	ChangedBy     *uuid.UUID `gorm:"type:uuid" json:"changed_by,omitempty"`
	ChangeReason  string     `gorm:"type:text" json:"change_reason,omitempty"`
	CreatedAt     time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	ReferralCode ReferralCode `gorm:"foreignKey:CodeID" json:"referral_code,omitempty"`
}

func (u *ReferralUsage) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type ReferralTransaction struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CodeID                 uuid.UUID  `gorm:"type:uuid;index;not null" json:"code_id"`
	UsageID                uuid.UUID  `gorm:"type:uuid;index;not null" json:"usage_id"`
	InvoiceID              uuid.UUID  `gorm:"type:uuid;index;not null" json:"invoice_id"`
	Type                   string     `gorm:"type:varchar(30);not null;default:'REGISTRATION'" json:"type"` // REGISTRATION, REREGISTRATION
	Amount                 float64    `gorm:"type:numeric(12,2);not null;default:0" json:"amount"`
	ApproveStatus          string     `gorm:"type:varchar(20);not null;default:'PENDING'" json:"approve_status"` // PENDING, SUCCESS, REJECTED
	CampusStatus           string     `gorm:"type:varchar(30);not null;default:'TAGIHAN'" json:"campus_status"`  // TAGIHAN, MENUNGGU_VERIFIKASI, DISETOR
	ApprovedBy             *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt             *time.Time `gorm:"type:timestamptz" json:"approved_at,omitempty"`
	RejectReason           string     `gorm:"type:text" json:"reject_reason,omitempty"`
	SettlementID           *uuid.UUID `gorm:"type:uuid;index" json:"settlement_id,omitempty"`
	CommissionTypeSnapshot string     `gorm:"type:varchar(20)" json:"commission_type_snapshot"`
	CommissionValueSnapshot float64   `gorm:"type:numeric(12,2);default:0" json:"commission_value_snapshot"`
	CommissionAmount       float64    `gorm:"type:numeric(12,2);default:0" json:"commission_amount"`
	CommissionStatus       string     `gorm:"type:varchar(20);not null;default:'BELUM_DIBAYAR'" json:"commission_status"` // TIDAK_ADA, BELUM_DIBAYAR, DIBAYAR, DIPOTONG
	CommissionPayoutID     *uuid.UUID `gorm:"type:uuid" json:"commission_payout_id,omitempty"`
	CreatedAt              time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	ReferralCode  ReferralCode  `gorm:"foreignKey:CodeID" json:"referral_code,omitempty"`
	ReferralUsage ReferralUsage `gorm:"foreignKey:UsageID" json:"referral_usage,omitempty"`
}

func (t *ReferralTransaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type ReferralSettlement struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CodeID             uuid.UUID  `gorm:"type:uuid;index;not null" json:"code_id"`
	GrossAmount        float64    `gorm:"type:numeric(12,2);not null;default:0" json:"gross_amount"`
	CommissionDeducted float64    `gorm:"type:numeric(12,2);not null;default:0" json:"commission_deducted"`
	Amount             float64    `gorm:"type:numeric(12,2);not null;default:0" json:"amount"`
	TransferDate       time.Time  `gorm:"type:date;not null" json:"transfer_date"`
	BankName           string     `gorm:"type:varchar(100);not null" json:"bank_name"`
	ReferenceNo        string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"reference_no"`
	ProofObjectKey     string     `gorm:"type:varchar(255)" json:"proof_object_key"`
	Notes              string     `gorm:"type:text" json:"notes"`
	Status             string     `gorm:"type:varchar(20);not null;default:'SUBMITTED'" json:"status"` // SUBMITTED, VERIFIED, REJECTED
	SubmittedBy        uuid.UUID  `gorm:"type:uuid;not null" json:"submitted_by"`
	VerifiedBy         *uuid.UUID `gorm:"type:uuid" json:"verified_by,omitempty"`
	VerifiedAt         *time.Time `gorm:"type:timestamptz" json:"verified_at,omitempty"`
	RejectReason       string     `gorm:"type:text" json:"reject_reason,omitempty"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	ReferralCode ReferralCode `gorm:"foreignKey:CodeID" json:"referral_code,omitempty"`
}

func (s *ReferralSettlement) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type ReferralCommissionPayout struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	CodeID         uuid.UUID `gorm:"type:uuid;index;not null" json:"code_id"`
	Amount         float64   `gorm:"type:numeric(12,2);not null;default:0" json:"amount"`
	TransactionIDs string    `gorm:"type:text" json:"transaction_ids"`
	PaidAt         time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"paid_at"`
	Method         string    `gorm:"type:varchar(50)" json:"method"`
	ReferenceNo    string    `gorm:"type:varchar(100)" json:"reference_no"`
	ProofObjectKey string    `gorm:"type:varchar(255)" json:"proof_object_key"`
	PaidBy         uuid.UUID `gorm:"type:uuid;not null" json:"paid_by"`
	CreatedAt      time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (p *ReferralCommissionPayout) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type ReferralSetting struct {
	ID                       uint    `gorm:"primaryKey" json:"id"`
	CommissionSettlementMode string  `gorm:"type:varchar(30);not null;default:'terpisah'" json:"commission_settlement_mode"` // terpisah / potong_setoran
	DefaultCommissionType    string  `gorm:"type:varchar(20);not null;default:'persen'" json:"default_commission_type"`      // persen / nominal / tidak_ada
	DefaultCommissionValue   float64 `gorm:"type:numeric(12,2);not null;default:10" json:"default_commission_value"`
	CreatedAt                time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt                time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}
