package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	UserStatusPending  = "pending"
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
)

type User struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email             string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Phone             string         `gorm:"type:varchar(20)" json:"phone"`
	PasswordHash      string         `gorm:"type:varchar(255);not null" json:"-"`
	FullName          string         `gorm:"type:varchar(255);not null" json:"full_name"`
	Status            string         `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	EmailVerifiedAt   *time.Time     `gorm:"type:timestamptz" json:"email_verified_at,omitempty"`
	LastLoginAt       *time.Time     `gorm:"type:timestamptz" json:"last_login_at,omitempty"`
	PasswordChangedAt *time.Time     `gorm:"type:timestamptz" json:"password_changed_at,omitempty"`
	CreatedAt         time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	UserRoles []UserRole `gorm:"foreignKey:UserID" json:"user_roles,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
