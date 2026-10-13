package model

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	ActorID   *uuid.UUID `gorm:"type:uuid;index" json:"actor_id,omitempty"`
	Action    string     `gorm:"type:varchar(100);not null;index" json:"action"`
	IP        string     `gorm:"type:varchar(45)" json:"ip"`
	UserAgent string     `gorm:"type:text" json:"user_agent"`
	Metadata  string     `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}
