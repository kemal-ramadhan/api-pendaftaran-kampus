package model

import (
	"time"
)

type AdmissionPath struct {
	ID              uint                  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string                `gorm:"type:varchar(100);not null" json:"name"`
	Code            string                `gorm:"type:varchar(20);uniqueIndex;not null" json:"code"`
	Description     string                `gorm:"type:text" json:"description"`
	RegistrationFee float64               `gorm:"type:numeric(12,2);not null;default:0" json:"registration_fee"`
	RequiresExam              bool                  `gorm:"not null;default:false" json:"requires_exam"`
	IsActive                  bool                  `gorm:"not null;default:true" json:"is_active"`
	ReferralCommissionEnabled bool                  `gorm:"not null;default:true" json:"referral_commission_enabled"`
	WAGroupName               string                `gorm:"type:varchar(100)" json:"wa_group_name,omitempty"`
	WAGroupURL                string                `gorm:"type:varchar(255)" json:"wa_group_url,omitempty"`
	NIMCategoryCode           string                `gorm:"type:varchar(10);default:'1'" json:"nim_category_code"`
	Requirements              []DocumentRequirement `gorm:"foreignKey:AdmissionPathID" json:"requirements,omitempty"`
	CreatedAt                 time.Time             `json:"created_at"`
	UpdatedAt                 time.Time             `json:"updated_at"`
}
