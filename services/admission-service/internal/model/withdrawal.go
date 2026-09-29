package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WithdrawalStatus string

const (
	WithdrawalStatusPending   WithdrawalStatus = "pending"
	WithdrawalStatusApproved  WithdrawalStatus = "approved"
	WithdrawalStatusRejected  WithdrawalStatus = "rejected"
	WithdrawalStatusCancelled WithdrawalStatus = "cancelled"
	WithdrawalStatusEffective WithdrawalStatus = "effective"
)

type WithdrawalSource string

const (
	WithdrawalSourceApplicant WithdrawalSource = "applicant"
	WithdrawalSourceAdmin     WithdrawalSource = "admin"
)

type ReasonCategoryItem struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

var DefaultReasonCategories = []ReasonCategoryItem{
	{Code: "accepted_elsewhere", Label: "Diterima di Perguruan Tinggi Lain"},
	{Code: "financial", Label: "Kendala Finansial / Biaya"},
	{Code: "family", Label: "Alasan Keluarga / Pribadi"},
	{Code: "health", Label: "Kondisi Kesehatan"},
	{Code: "relocation", Label: "Pindah Domisili / Kota"},
	{Code: "work", Label: "Memilih Bekerja"},
	{Code: "other", Label: "Alasan Lainnya"},
}

type Withdrawal struct {
	ID                  uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	ApplicationID       uuid.UUID        `gorm:"type:uuid;not null;index" json:"application_id"`
	Application         *Application     `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	UserID              uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	StageAtRequest      string           `gorm:"type:varchar(50);not null" json:"stage_at_request"`
	ReasonCategory      string           `gorm:"type:varchar(50);not null" json:"reason_category"`
	ReasonText          string           `gorm:"type:text;not null" json:"reason_text"`
	Status              WithdrawalStatus `gorm:"type:varchar(30);not null;default:'pending';index" json:"status"`
	RequiresApproval    bool             `gorm:"not null;default:false" json:"requires_approval"`
	RequestedAt         time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"requested_at"`
	ReviewedBy          *uuid.UUID       `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewNote          string           `gorm:"type:text" json:"review_note,omitempty"`
	ReviewedAt          *time.Time       `json:"reviewed_at,omitempty"`
	EffectiveAt         *time.Time       `json:"effective_at,omitempty"`
	Source              WithdrawalSource `gorm:"type:varchar(20);not null;default:'applicant'" json:"source"`
	AttachmentObjectKey string           `gorm:"type:varchar(255)" json:"attachment_object_key,omitempty"`
	LetterNumber        string           `gorm:"type:varchar(100)" json:"letter_number,omitempty"`
	LetterObjectKey     string           `gorm:"type:varchar(255)" json:"letter_object_key,omitempty"`
	VerifyCodeHash      string           `gorm:"type:varchar(64);index" json:"verify_code_hash,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

func (w *Withdrawal) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

func (Withdrawal) TableName() string {
	return "withdrawals"
}
