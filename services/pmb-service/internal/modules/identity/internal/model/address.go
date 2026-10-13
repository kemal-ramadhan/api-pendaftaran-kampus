package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Address menyimpan alamat KTP maupun domisili calon mahasiswa.
type Address struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID      `gorm:"type:uuid;uniqueIndex:idx_addresses_user_id_type;not null" json:"user_id"`
	Type       string         `gorm:"type:varchar(20);uniqueIndex:idx_addresses_user_id_type;not null" json:"type"` // "ktp" / "domisili"
	Street     string         `gorm:"type:text" json:"street"`
	RT         string         `gorm:"type:varchar(10)" json:"rt"`
	RW         string         `gorm:"type:varchar(10)" json:"rw"`
	Village    string         `gorm:"type:varchar(100)" json:"village"`       // Kelurahan / Desa
	District   string         `gorm:"type:varchar(100)" json:"district"`      // Kecamatan
	City       string         `gorm:"type:varchar(100)" json:"city"`          // Kota / Kabupaten
	Province   string         `gorm:"type:varchar(100)" json:"province"`      // Provinsi
	PostalCode string         `gorm:"type:varchar(10)" json:"postal_code"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *Address) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
