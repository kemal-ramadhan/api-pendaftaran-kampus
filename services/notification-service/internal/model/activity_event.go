package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActivityEvent merepresentasikan catatan audit jejak aktivitas panitia PMB (PRD 2.9)
type ActivityEvent struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PeriodID    string    `gorm:"type:varchar(50);index" json:"period_id"`
	ActorID     uuid.UUID `gorm:"type:uuid;index" json:"actor_id"`
	ActorName   string    `gorm:"type:varchar(100)" json:"actor_name"`
	ActorRole   string    `gorm:"type:varchar(50)" json:"actor_role"`
	Action      string    `gorm:"type:varchar(100);index" json:"action"`
	ActionLabel string    `gorm:"type:varchar(150)" json:"action_label"`
	TargetType  string    `gorm:"type:varchar(50)" json:"target_type"`
	TargetID    string    `gorm:"type:varchar(100)" json:"target_id"`
	TargetLabel string    `gorm:"type:varchar(200)" json:"target_label"`
	Service     string    `gorm:"type:varchar(50);index" json:"service"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

func (a *ActivityEvent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	return nil
}
