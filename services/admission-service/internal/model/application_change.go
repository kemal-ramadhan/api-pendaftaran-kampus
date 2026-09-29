package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChangeType string

const (
	ChangeTypeCreated             ChangeType = "created"
	ChangeTypeChoicesUpdated      ChangeType = "choices_updated"
	ChangeTypePathChangeRequested ChangeType = "path_change_requested"
	ChangeTypePathChangeApproved  ChangeType = "path_change_approved"
	ChangeTypePathChangeRejected  ChangeType = "path_change_rejected"
	ChangeTypePathChangeCancelled ChangeType = "path_change_cancelled"
	ChangeTypeAdminMoved          ChangeType = "admin_moved"
	ChangeTypeCancelled           ChangeType = "cancelled"
	ChangeTypeReapplied           ChangeType = "reapplied"
	ChangeTypeWithdrawalRequested ChangeType = "withdrawal_requested"
	ChangeTypeWithdrawalApproved  ChangeType = "withdrawal_approved"
	ChangeTypeWithdrawalRejected  ChangeType = "withdrawal_rejected"
	ChangeTypeWithdrawalCancelled ChangeType = "withdrawal_cancelled"
)

type ChangeSource string

const (
	ChangeSourceApplicant ChangeSource = "applicant"
	ChangeSourceAdmin     ChangeSource = "admin"
	ChangeSourceSystem    ChangeSource = "system"
)

// ApplicationChange merepresentasikan baris riwayat perubahan pendaftaran yang immutable (append-only)
type ApplicationChange struct {
	ID               uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	ApplicationID    uuid.UUID    `gorm:"type:uuid;not null;index" json:"application_id"`
	UserID           uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	ChangeType       ChangeType   `gorm:"type:varchar(50);not null" json:"change_type"`
	FromValues       string       `gorm:"type:jsonb;not null;default:'{}'" json:"from_values"`
	ToValues         string       `gorm:"type:jsonb;not null;default:'{}'" json:"to_values"`
	Reason           string       `gorm:"type:text" json:"reason,omitempty"`
	FeeDifference    int64        `gorm:"type:bigint;not null;default:0" json:"fee_difference"`
	ActorID          *uuid.UUID   `gorm:"type:uuid" json:"actor_id,omitempty"`
	ActorRole        string       `gorm:"type:varchar(50);not null;default:'applicant'" json:"actor_role"`
	Source           ChangeSource `gorm:"type:varchar(20);not null;default:'applicant'" json:"source"`
	RelatedRequestID *uuid.UUID   `gorm:"type:uuid" json:"related_request_id,omitempty"`
	CreatedAt        time.Time    `gorm:"index" json:"created_at"`

	// Field pendukung tampilan frontend (computed)
	TitleLabel        string `gorm:"-" json:"title_label,omitempty"`
	Description       string `gorm:"-" json:"description,omitempty"`
	ActorLabel        string `gorm:"-" json:"actor_label,omitempty"`
	CreatedAtLabel    string `gorm:"-" json:"created_at_label,omitempty"`
	RegistrationNo    string `gorm:"-" json:"application_registration_number,omitempty"`
}

func (c *ApplicationChange) TableName() string {
	return "application_changes"
}

func (c *ApplicationChange) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.FromValues == "" {
		c.FromValues = "{}"
	}
	if c.ToValues == "" {
		c.ToValues = "{}"
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	return nil
}

// SetFromValuesHelper memudahkan marshalling JSON untuk from_values
func (c *ApplicationChange) SetFromValues(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.FromValues = string(b)
	return nil
}

// SetToValuesHelper memudahkan marshalling JSON untuk to_values
func (c *ApplicationChange) SetToValues(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.ToValues = string(b)
	return nil
}
