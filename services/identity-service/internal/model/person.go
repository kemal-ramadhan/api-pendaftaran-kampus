package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Person menyimpan data personal pengguna yang terpisah dari akun Auth.
type Person struct {
	UserID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"user_id"`
	FullName       string         `gorm:"type:varchar(255);not null" json:"full_name"`
	NIK            string         `gorm:"type:text;not null" json:"-"` // Terenkripsi AES-GCM, jangan expose via JSON default
	NIKHash        string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"` // SHA-256 hexdigest untuk pencarian unik
	BirthPlace     string         `gorm:"type:varchar(100)" json:"birth_place"`
	BirthDate      *time.Time     `gorm:"type:date" json:"birth_date"`
	Gender         string         `gorm:"type:varchar(10)" json:"gender"` // L / P
	Religion       string         `gorm:"type:varchar(50)" json:"religion"`
	Nationality    string         `gorm:"type:varchar(50);default:'WNI'" json:"nationality"`
	Phone          string         `gorm:"type:varchar(30)" json:"phone"`
	PhotoObjectKey string         `gorm:"type:varchar(255)" json:"photo_object_key"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	Addresses          []Address          `gorm:"foreignKey:UserID;references:UserID" json:"addresses,omitempty"`
	Guardians          []Guardian         `gorm:"foreignKey:UserID;references:UserID" json:"guardians,omitempty"`
	EducationHistories []EducationHistory `gorm:"foreignKey:UserID;references:UserID" json:"education_histories,omitempty"`
}
