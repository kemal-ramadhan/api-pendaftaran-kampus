package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentWebhookLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Provider     string    `gorm:"type:varchar(50);not null" json:"provider"`
	ProviderRef  string    `gorm:"type:varchar(100);index;not null" json:"provider_ref"`
	Payload      string    `gorm:"type:text;not null" json:"payload"`
	Signature    string    `gorm:"type:varchar(255)" json:"signature"`
	Status       string    `gorm:"type:varchar(30);not null;default:'processed'" json:"status"` // processed, rejected, duplicate
	ErrorMessage string    `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (w *PaymentWebhookLog) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}
