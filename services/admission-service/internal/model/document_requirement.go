package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentRequirement struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	AdmissionPathID uint      `gorm:"not null;index" json:"admission_path_id"`
	DocumentType    string    `gorm:"type:varchar(50);not null" json:"document_type"` // e.g. "ijazah", "rapor", "ktp", "kk", "prestasi"
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	IsRequired      bool      `gorm:"not null;default:true" json:"is_required"`
	MaxSizeMB       int       `gorm:"not null;default:5" json:"max_size_mb"`
	AllowedFormats  string    `gorm:"type:varchar(100);not null;default:'pdf,jpg,jpeg,png'" json:"allowed_formats"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (d *DocumentRequirement) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
