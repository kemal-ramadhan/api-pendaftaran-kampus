package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdmissionOnboarding struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	StepKey     string     `gorm:"type:varchar(50);not null;index" json:"step_key"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (o *AdmissionOnboarding) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

func (AdmissionOnboarding) TableName() string {
	return "admission_onboardings"
}
