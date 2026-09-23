package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Guardian menyimpan data orang tua atau wali calon mahasiswa.
type Guardian struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;index;not null" json:"user_id"`
	Relation    string         `gorm:"type:varchar(20);not null" json:"relation"` // "ayah", "ibu", "wali"
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Occupation  string         `gorm:"type:varchar(100)" json:"occupation"`
	IncomeRange string         `gorm:"type:varchar(100)" json:"income_range"`
	Phone       string         `gorm:"type:varchar(30)" json:"phone"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (g *Guardian) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
