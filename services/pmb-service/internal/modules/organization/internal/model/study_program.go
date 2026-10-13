package model

import (
	"time"

	"gorm.io/gorm"
)

// StudyProgram merepresentasikan entitas program studi di universitas
type StudyProgram struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	FacultyID     uint           `gorm:"index;not null" json:"faculty_id"`
	Faculty       *Faculty       `gorm:"foreignKey:FacultyID" json:"faculty,omitempty"`
	Code          string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"code"`
	NIMCode       string         `gorm:"type:varchar(5);not null" json:"nim_code"`
	Name          string         `gorm:"type:varchar(100);not null" json:"name"`
	Degree        string         `gorm:"type:varchar(10);not null" json:"degree"`
	Accreditation string         `gorm:"type:varchar(30);not null" json:"accreditation"`
	IsActive      bool           `gorm:"not null;default:true" json:"is_active"`
	AdmissionOpen bool           `gorm:"not null;default:true" json:"admission_open"`
	CreatedAt     time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (StudyProgram) TableName() string {
	return "study_programs"
}
