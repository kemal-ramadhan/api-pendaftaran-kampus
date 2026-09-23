package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationStatus string

const (
	StatusQueued NotificationStatus = "queued"
	StatusSent   NotificationStatus = "sent"
	StatusFailed NotificationStatus = "failed"
)

type NotificationLog struct {
	ID             uuid.UUID          `gorm:"type:uuid;primaryKey" json:"id"`
	IdempotencyKey string             `gorm:"type:varchar(100);uniqueIndex;not null" json:"idempotency_key"`
	UserID         *uuid.UUID         `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Email          string             `gorm:"type:varchar(255)" json:"email,omitempty"`
	Phone          string             `gorm:"type:varchar(30)" json:"phone,omitempty"`
	Channel        string             `gorm:"type:varchar(20);not null" json:"channel"` // email, whatsapp
	Template       string             `gorm:"type:varchar(100);not null" json:"template"`
	Data           string             `gorm:"type:text" json:"data"`
	Status         NotificationStatus `gorm:"type:varchar(20);not null;default:'queued'" json:"status"`
	Retries        int                `gorm:"not null;default:0" json:"retries"`
	ErrorMessage   string             `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

func (n *NotificationLog) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
