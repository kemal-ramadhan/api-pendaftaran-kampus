package model

import (
	"time"

	"github.com/google/uuid"
)

// KTPScan mencatat riwayat pemindaian dan konfirmasi data KTP calon mahasiswa.
type KTPScan struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID             uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	ObjectKey          string     `gorm:"type:varchar(255);not null" json:"object_key"` // Lokasi di MinIO bucket pmb-identity-ktp
	JobID              string     `gorm:"type:varchar(100)" json:"job_id"`              // ID job di ocr-service
	Status             string     `gorm:"type:varchar(30);not null" json:"status"`      // "processing", "completed", "retake_required", "failed"
	QualityScore       float64    `json:"quality_score"`
	Enhanced           bool       `json:"enhanced"`
	Reason             string     `gorm:"type:varchar(50)" json:"reason,omitempty"`
	ReasonMessage      string     `gorm:"type:text" json:"reason_message,omitempty"`
	FieldConfidences   string     `gorm:"type:jsonb" json:"field_confidences,omitempty"` // json key -> float, tanpa NIK
	OCRDataRaw         string     `gorm:"type:text" json:"-"`                             // Saran OCR (diserialisasi untuk komparasi saat confirm)
	ConfirmedAt        *time.Time `json:"confirmed_at,omitempty"`
	FieldsEditedByUser string     `gorm:"type:jsonb" json:"fields_edited_by_user,omitempty"` // array string nama field
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
