package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicationStatus string

const (
	StatusDraft              ApplicationStatus = "draft"
	StatusAwaitingPayment    ApplicationStatus = "awaiting_payment"
	StatusFormOpen           ApplicationStatus = "form_open"
	StatusSubmitted          ApplicationStatus = "submitted"
	StatusRevisionRequired   ApplicationStatus = "revision_required"
	StatusVerified           ApplicationStatus = "verified"
	StatusExamScheduled      ApplicationStatus = "exam_scheduled"
	StatusInterviewScheduled ApplicationStatus = "interview_scheduled"
	StatusDirectDecision     ApplicationStatus = "direct_decision"
	StatusPassed             ApplicationStatus = "passed"
	StatusFailed             ApplicationStatus = "failed"
	StatusReregistered       ApplicationStatus = "reregistered"
	StatusEnrolled           ApplicationStatus = "enrolled"
	StatusCancelled          ApplicationStatus = "cancelled"
	StatusExpired            ApplicationStatus = "expired"
)

type Application struct {
	ID              uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	RegistrationNo  string                 `gorm:"type:varchar(50);uniqueIndex;not null" json:"registration_no"`
	UserID          uuid.UUID              `gorm:"type:uuid;index;not null" json:"user_id"`
	PeriodID        uint                   `gorm:"not null;index" json:"period_id"`
	AdmissionPathID uint                   `gorm:"not null;index" json:"admission_path_id"`
	Choice1ID       uuid.UUID              `gorm:"type:uuid;not null;index" json:"choice_1_id"`
	Choice2ID       *uuid.UUID             `gorm:"type:uuid;index" json:"choice_2_id,omitempty"`
	Status          ApplicationStatus      `gorm:"type:varchar(30);not null;default:'awaiting_payment';index" json:"status"`
	InvoiceID       *uuid.UUID             `gorm:"type:uuid" json:"invoice_id,omitempty"`
	FinalChoiceID   *uuid.UUID             `gorm:"type:uuid" json:"final_choice_id,omitempty"`
	SubmittedAt     *time.Time             `json:"submitted_at,omitempty"`
	VerifiedAt      *time.Time             `json:"verified_at,omitempty"`
	VerifiedBy      *uuid.UUID             `gorm:"type:uuid" json:"verified_by,omitempty"`
	Notes           string                 `gorm:"type:text" json:"notes,omitempty"`
	Period          *Period                `gorm:"foreignKey:PeriodID" json:"period,omitempty"`
	AdmissionPath   *AdmissionPath         `gorm:"foreignKey:AdmissionPathID" json:"admission_path,omitempty"`
	Documents       []ApplicationDocument  `gorm:"foreignKey:ApplicationID" json:"documents,omitempty"`
	StatusLogs      []ApplicationStatusLog `gorm:"foreignKey:ApplicationID" json:"status_logs,omitempty"`
	PersonData      interface{}            `gorm:"-" json:"person_data,omitempty"`
	InvoiceData     interface{}            `gorm:"-" json:"invoice_data,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func (a *Application) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// CanChangeMajor memeriksa apakah pendaftar diizinkan mengubah pilihan prodi
func (a *Application) CanChangeMajor() bool {
	return a.Status == StatusDraft || a.Status == StatusAwaitingPayment
}

// CanUploadDocument memeriksa apakah pendaftar diizinkan mengunggah dokumen
func (a *Application) CanUploadDocument() bool {
	return a.Status == StatusFormOpen || a.Status == StatusRevisionRequired
}

// CanSubmit memeriksa apakah pendaftar diizinkan men-submit formulir
func (a *Application) CanSubmit() bool {
	return a.Status == StatusFormOpen || a.Status == StatusRevisionRequired
}

type AdmissionStats struct {
	TotalApplications int64            `json:"total_applications"`
	ByStatus          map[string]int64 `json:"by_status"`
	ByStudyProgram    map[string]int64 `json:"by_study_program"`
	ByAdmissionPath   map[string]int64 `json:"by_admission_path"`
	ByPeriod          map[string]int64 `json:"by_period"`
}
