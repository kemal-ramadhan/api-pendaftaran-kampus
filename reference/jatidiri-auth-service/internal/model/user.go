package model

import "time"

type User struct {
	ID                     uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Email                  string     `gorm:"uniqueIndex;not null" json:"email"`
	Phone                  string     `gorm:"index;null" json:"phone"`
	Password               string     `json:"-"`
	PhotoProfile           string     `gorm:"gorm:"type:text" json:"photo_profile,omitempty"`
	Role                   string     `gorm:"type:varchar(50);not null;default:'customer'" json:"role"`
	GoogleID               *string    `gorm:"uniqueIndex;null" json:"google_id,omitempty"`
	GoogleAccessToken      *string    `gorm:"type:text;null" json:"google_access_token,omitempty"`
	EmailVerifiedAt        *time.Time `json:"email_verified_at,omitempty"`
	TwoFactorSecret        *string    `gorm:"type:text;null" json:"two_factor_secret,omitempty"`
	TwoFactorRecoveryCodes *string    `gorm:"type:text;null" json:"two_factor_recovery_codes,omitempty"`
	TwoFactorConfirmedAt   *time.Time `json:"two_factor_confirmed_at,omitempty"`
	RememberToken          *string    `gorm:"type:varchar(255);null" json:"remember_token,omitempty"`
	EmailOtp               *int       `json:"email_otp,omitempty"`
	EmailOtpExpiresAt      *time.Time `json:"email_otp_expires_at,omitempty"`
	EmailOtpAttempt        *int16     `json:"email_otp_attempt,omitempty"`
	ResetPasswordToken     *string    `gorm:"type:text;null" json:"-"`
	ResetPasswordExpiresAt *time.Time `json:"-"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`

	Customer Customer `gorm:"foreignKey:UserID" json:"customer,omitempty"`
}
