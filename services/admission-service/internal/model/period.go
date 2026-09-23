package model

import (
	"time"
)

type Period struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"type:varchar(100);not null" json:"name"`
	AcademicYear string    `gorm:"type:varchar(20);not null" json:"academic_year"` // e.g. "2026/2027"
	StartDate    time.Time `gorm:"not null" json:"start_date"`
	EndDate      time.Time `gorm:"not null" json:"end_date"`
	IsActive     bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (p *Period) IsOpenNow() bool {
	now := time.Now()
	return p.IsActive && !now.Before(p.StartDate) && !now.After(p.EndDate)
}
