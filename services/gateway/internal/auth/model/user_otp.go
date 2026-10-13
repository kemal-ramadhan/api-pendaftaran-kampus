package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	OTPPurposeEmailVerification = "email_verification"
	OTPPurposeLogin             = "login"
	OTPPurposePhoneVerification = "phone_verification"

	OTPChannelEmail = "email"
	OTPChannelSMS   = "sms"
	OTPChannelWA    = "whatsapp"
)

type UserOTP struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	Purpose     string     `gorm:"type:varchar(30);not null;default:'email_verification'" json:"purpose"`
	Channel     string     `gorm:"type:varchar(20);not null;default:'email'" json:"channel"`
	CodeHash    string     `gorm:"type:varchar(64);not null" json:"-"`
	ExpiresAt   time.Time  `gorm:"type:timestamptz;not null" json:"expires_at"`
	Attempts    int16      `gorm:"type:smallint;not null;default:0" json:"attempts"`
	MaxAttempts int16      `gorm:"type:smallint;not null;default:5" json:"max_attempts"`
	ConsumedAt  *time.Time `gorm:"type:timestamptz" json:"consumed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (o *UserOTP) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
