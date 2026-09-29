package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PathChangeStatus string

const (
	PathChangePending   PathChangeStatus = "pending"
	PathChangeApproved  PathChangeStatus = "approved"
	PathChangeRejected  PathChangeStatus = "rejected"
	PathChangeCancelled PathChangeStatus = "cancelled"
)

// PathChangeRequest merepresentasikan permohonan pindah jalur dari calon mahasiswa setelah pembayaran
type PathChangeRequest struct {
	ID            uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	ApplicationID uuid.UUID        `gorm:"type:uuid;not null;index" json:"application_id"`
	UserID        uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	FromPeriodID  uint             `gorm:"not null" json:"from_period_id"`
	FromPathID    uint             `gorm:"not null" json:"from_path_id"`
	FromChoice1ID uuid.UUID        `gorm:"type:uuid;not null" json:"from_choice_1_id"`
	FromChoice2ID *uuid.UUID       `gorm:"type:uuid" json:"from_choice_2_id,omitempty"`
	ToPeriodID    uint             `gorm:"not null" json:"to_period_id"`
	ToPathID      uint             `gorm:"not null" json:"to_path_id"`
	ToChoice1ID   uuid.UUID        `gorm:"type:uuid;not null" json:"to_choice_1_id"`
	ToChoice2ID   *uuid.UUID       `gorm:"type:uuid" json:"to_choice_2_id,omitempty"`
	Reason        string           `gorm:"type:text" json:"reason,omitempty"`
	Status        PathChangeStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	FeeDifference int64            `gorm:"type:bigint;not null;default:0" json:"fee_difference"`
	ReviewedBy    *uuid.UUID       `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewNote    string           `gorm:"type:text" json:"review_note,omitempty"`
	ReviewedAt    *time.Time       `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time        `gorm:"index" json:"created_at"`

	// Relasi & Komputasi
	Application *Application   `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	FromPath    *AdmissionPath `gorm:"foreignKey:FromPathID" json:"from_path,omitempty"`
	ToPath      *AdmissionPath `gorm:"foreignKey:ToPathID" json:"to_path,omitempty"`
	FromPeriod  *Period        `gorm:"foreignKey:FromPeriodID" json:"from_period,omitempty"`
	ToPeriod    *Period        `gorm:"foreignKey:ToPeriodID" json:"to_period,omitempty"`
}

func (r *PathChangeRequest) TableName() string {
	return "path_change_requests"
}

func (r *PathChangeRequest) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.Status == "" {
		r.Status = PathChangePending
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	return nil
}
