package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EducationHistory menyimpan riwayat pendidikan formal calon mahasiswa.
type EducationHistory struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;index;not null" json:"user_id"`
	Level          string         `gorm:"type:varchar(20);not null" json:"level"` // "SMA", "SMK", "MA", "D3", dll
	SchoolName     string         `gorm:"type:varchar(255);not null" json:"school_name"`
	NPSN           string         `gorm:"type:varchar(20)" json:"npsn"`
	Major          string         `gorm:"type:varchar(100)" json:"major"` // mis. "IPA", "IPS", "Teknik Komputer"
	GraduationYear int            `gorm:"not null" json:"graduation_year"`
	ReportAverage  *float64       `gorm:"type:decimal(5,2)" json:"report_average"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (e *EducationHistory) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
