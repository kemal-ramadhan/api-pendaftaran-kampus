package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicationStatusLog struct {
	ID            uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	ApplicationID uuid.UUID         `gorm:"type:uuid;not null;index" json:"application_id"`
	FromStatus    ApplicationStatus `gorm:"type:varchar(30);not null" json:"from_status"`
	ToStatus      ApplicationStatus `gorm:"type:varchar(30);not null" json:"to_status"`
	ChangedBy     *uuid.UUID        `gorm:"type:uuid" json:"changed_by,omitempty"`
	Notes         string            `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

func (l *ApplicationStatusLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
