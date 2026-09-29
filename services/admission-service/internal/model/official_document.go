package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OfficialDocument struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PeriodID       *uint     `gorm:"index" json:"period_id,omitempty"`
	DocType        string    `gorm:"type:varchar(50);not null;index" json:"doc_type"` // registrant, selection, reconciliation
	DocNumber      string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"doc_number"`
	SequenceNumber int       `gorm:"not null" json:"sequence_number"`
	Year           int       `gorm:"not null;index" json:"year"`
	Title          string    `gorm:"type:varchar(200);not null" json:"title"`
	CreatedBy      uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	CreatedByName  string    `gorm:"type:varchar(100)" json:"created_by_name"`
	MetadataJSON   string    `gorm:"type:text" json:"metadata_json,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (d *OfficialDocument) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
