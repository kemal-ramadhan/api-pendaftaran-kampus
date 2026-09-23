package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DocStatusUploaded = "uploaded"
	DocStatusValid    = "valid"
	DocStatusInvalid  = "invalid"
	DocStatusPending  = "pending"
)

type ApplicationDocument struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ApplicationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"application_id"`
	DocumentType  string     `gorm:"type:varchar(50);not null" json:"document_type"`
	FileName      string     `gorm:"type:varchar(255);not null" json:"file_name"`
	FileURL       string     `gorm:"type:varchar(500);not null" json:"file_url"`
	ObjectKey     string     `gorm:"type:varchar(255)" json:"object_key,omitempty"`
	MimeType      string     `gorm:"type:varchar(100)" json:"mime_type,omitempty"`
	FileSize      int64      `gorm:"not null;default:0" json:"file_size"`
	Status        string     `gorm:"type:varchar(20);not null;default:'uploaded'" json:"status"` // uploaded, valid, invalid
	Notes         string     `gorm:"type:text" json:"notes,omitempty"`
	VerifiedBy    *uuid.UUID `gorm:"type:uuid" json:"verified_by,omitempty"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (d *ApplicationDocument) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
