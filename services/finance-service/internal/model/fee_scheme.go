package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FeeScheme struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	PeriodID               uint       `gorm:"not null;index" json:"period_id"`
	AdmissionPathID        uint       `gorm:"not null;index" json:"admission_path_id"`
	StudyProgramID         *uuid.UUID `gorm:"type:uuid;index" json:"study_program_id,omitempty"`
	FeeType                string     `gorm:"type:varchar(50);not null" json:"fee_type"` // registration, reregistration
	Amount                 int64      `gorm:"type:bigint;not null" json:"amount"`        // Nominal Rupiah selalu integer
	InstallmentAllowed     bool       `gorm:"not null;default:false" json:"installment_allowed"`
	MaxInstallments        int        `gorm:"not null;default:1" json:"max_installments"`
	MinFirstPaymentPercent int        `gorm:"not null;default:100" json:"min_first_payment_percent"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (f *FeeScheme) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
