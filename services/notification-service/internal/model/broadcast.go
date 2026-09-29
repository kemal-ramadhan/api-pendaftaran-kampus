package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BroadcastTemplate merepresentasikan template siap pakai untuk broadcast notifikasi massal (PRD 2.7)
type BroadcastTemplate struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code         string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name         string    `gorm:"type:varchar(150);not null" json:"name"`
	Channel      string    `gorm:"type:varchar(20);not null" json:"channel"` // email, whatsapp, in_app
	Subject      string    `gorm:"type:varchar(255)" json:"subject"`
	BodyTemplate string    `gorm:"type:text;not null" json:"body_template"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BroadcastLog merepresentasikan riwayat pelaksanaan pengiriman broadcast massal (PRD 2.7)
type BroadcastLog struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	PeriodID           string     `gorm:"type:varchar(50);index" json:"period_id"`
	TemplateID         *uint      `gorm:"index" json:"template_id,omitempty"`
	ActorID            uuid.UUID  `gorm:"type:uuid" json:"actor_id"`
	ActorName          string     `gorm:"type:varchar(100)" json:"actor_name"`
	Channel            string     `gorm:"type:varchar(20);not null" json:"channel"`
	TargetSegment      string     `gorm:"type:varchar(50);not null" json:"target_segment"`
	FilterValue        string     `gorm:"type:varchar(100)" json:"filter_value,omitempty"`
	Subject            string     `gorm:"type:varchar(255)" json:"subject"`
	Message            string     `gorm:"type:text;not null" json:"message"`
	ExpectedRecipients int        `gorm:"not null" json:"expected_recipients"`
	SentCount          int        `gorm:"not null;default:0" json:"sent_count"`
	FailedCount        int        `gorm:"not null;default:0" json:"failed_count"`
	Status             string     `gorm:"type:varchar(20);not null;default:'completed'" json:"status"`
	ScheduledAt        *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt          time.Time  `gorm:"index" json:"created_at"`
}

func (b *BroadcastLog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	return nil
}
