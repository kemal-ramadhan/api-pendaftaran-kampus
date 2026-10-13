package model

import (
	"time"

	"gorm.io/gorm"
)

// Faculty merepresentasikan entitas fakultas di universitas
type Faculty struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Code          string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"code"`
	Name          string         `gorm:"type:varchar(100);not null" json:"name"`
	IsActive      bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	StudyPrograms []StudyProgram `gorm:"foreignKey:FacultyID" json:"study_programs,omitempty"`
}

func (Faculty) TableName() string {
	return "faculties"
}
