package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Quota struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PeriodID        uint      `gorm:"not null;index" json:"period_id"`
	AdmissionPathID uint      `gorm:"not null;index" json:"admission_path_id"`
	StudyProgramID  uuid.UUID `gorm:"type:uuid;not null;index" json:"study_program_id"`
	Capacity        int       `gorm:"not null;default:0" json:"capacity"`
	FilledCount     int       `gorm:"not null;default:0" json:"filled_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (q *Quota) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}
