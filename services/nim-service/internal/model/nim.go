package model

import (
	"time"

	"github.com/google/uuid"
)

// NIMFormat menyimpan template format penomoran NIM (global atau spesifik prodi)
type NIMFormat struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudyProgramCode *string   `gorm:"type:varchar(30);index" json:"study_program_code"`
	Pattern          string    `gorm:"type:varchar(100);not null" json:"pattern"`
	Description      string    `gorm:"type:varchar(255)" json:"description"`
	IsActive         bool      `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (NIMFormat) TableName() string {
	return "nim_formats"
}

// NIMSequence menyimpan nomor urut terakhir per tahun masuk dan kode program studi
type NIMSequence struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EntryYear        int       `gorm:"not null" json:"entry_year"`
	StudyProgramCode string    `gorm:"type:varchar(30);not null" json:"study_program_code"`
	LastNumber       int       `gorm:"not null;default:0" json:"last_number"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (NIMSequence) TableName() string {
	return "nim_sequences"
}

// NIMIssuance mencatat riwayat resmi penerbitan NIM (idempoten per application_id)
type NIMIssuance struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ApplicationID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"application_id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	NIM              string    `gorm:"type:varchar(30);not null;uniqueIndex" json:"nim"`
	EntryYear        int       `gorm:"not null;index" json:"entry_year"`
	StudyProgramCode string    `gorm:"type:varchar(30);not null;index" json:"study_program_code"`
	SelectionPath    string    `gorm:"type:varchar(50);not null" json:"selection_path"`
	SequenceNumber   int       `gorm:"not null" json:"sequence_number"`
	IssuedAt         time.Time `json:"issued_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (NIMIssuance) TableName() string {
	return "nim_issuances"
}

// DTOs

type IssueNIMRequest struct {
	ApplicationID    uuid.UUID `json:"application_id" binding:"required"`
	UserID           uuid.UUID `json:"user_id" binding:"required"`
	StudyProgramCode string    `json:"study_program_code" binding:"required"`
	EntryYear        int       `json:"entry_year" binding:"required"`
	SelectionPath    string    `json:"selection_path"`
}

type IssueNIMResponse struct {
	NIM              string    `json:"nim"`
	ApplicationID    uuid.UUID `json:"application_id"`
	UserID           uuid.UUID `json:"user_id"`
	StudyProgramCode string    `json:"study_program_code"`
	EntryYear        int       `json:"entry_year"`
	SelectionPath    string    `json:"selection_path"`
	SequenceNumber   int       `json:"sequence_number"`
	IssuedAt         time.Time `json:"issued_at"`
	IsIdempotent     bool      `json:"is_idempotent"`
}

type CreateNIMFormatRequest struct {
	StudyProgramCode *string `json:"study_program_code"`
	Pattern          string  `json:"pattern" binding:"required"`
	Description      string  `json:"description"`
	IsActive         *bool   `json:"is_active"`
}

type UpdateNIMFormatRequest struct {
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type IssuanceFilter struct {
	StudyProgramCode string
	EntryYear        int
	Search           string
	Limit            int
	Offset           int
}
