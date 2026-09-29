package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type BankAccount struct {
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	IsPrimary     bool   `json:"is_primary"`
}

type BankAccounts []BankAccount

func (b BankAccounts) Value() (driver.Value, error) {
	return json.Marshal(b)
}

func (b *BankAccounts) Scan(value interface{}) error {
	if value == nil {
		*b = BankAccounts{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, b)
	case string:
		return json.Unmarshal([]byte(v), b)
	default:
		return errors.New("type assertion failed for BankAccounts")
	}
}

// InstitutionProfile merepresentasikan entitas profil institusi universitas
type InstitutionProfile struct {
	ID                  uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	FoundationName      string         `gorm:"type:varchar(150);not null" json:"foundation_name"`
	InstitutionName     string         `gorm:"type:varchar(150);not null" json:"institution_name"`
	DocumentCode        string         `gorm:"type:varchar(20);not null" json:"document_code"`
	Address             string         `gorm:"type:text" json:"address"`
	City                string         `gorm:"type:varchar(100);not null" json:"city"`
	Email               string         `gorm:"type:varchar(100)" json:"email"`
	Phone               string         `gorm:"type:varchar(50)" json:"phone"`
	Hotline             string         `gorm:"type:varchar(50)" json:"hotline"`
	ServiceHours        string         `gorm:"type:varchar(255)" json:"service_hours"`
	LogoURL             string         `gorm:"type:text" json:"logo_url"`
	CommitteeChairName  string         `gorm:"type:varchar(150)" json:"committee_chair_name"`
	CommitteeChairTitle string         `gorm:"type:varchar(150)" json:"committee_chair_title"`
	CommitteeChairNIP   string         `gorm:"type:varchar(50)" json:"committee_chair_nip"`
	BankAccounts        BankAccounts   `gorm:"type:jsonb" json:"bank_accounts"`
	CreatedAt           time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (InstitutionProfile) TableName() string {
	return "institution_profiles"
}
