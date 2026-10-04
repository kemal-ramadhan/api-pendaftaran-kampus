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
	StatusWithdrawn          ApplicationStatus = "withdrawn"
	StatusExpired            ApplicationStatus = "expired"
)

type Application struct {
	ID                          uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	RegistrationNo              string                 `gorm:"type:varchar(50);uniqueIndex;not null" json:"registration_no"`
	UserID                      uuid.UUID              `gorm:"type:uuid;index;not null" json:"user_id"`
	PeriodID                    uint                   `gorm:"not null;index" json:"period_id"`
	AdmissionPathID             uint                   `gorm:"not null;index" json:"admission_path_id"`
	Choice1ID                   uuid.UUID              `gorm:"type:uuid;not null;index" json:"choice_1_id"`
	Choice2ID                   *uuid.UUID             `gorm:"type:uuid;index" json:"choice_2_id,omitempty"`
	Status                      ApplicationStatus      `gorm:"type:varchar(30);not null;default:'awaiting_payment';index" json:"status"`
	InvoiceID                   *uuid.UUID             `gorm:"type:uuid" json:"invoice_id,omitempty"`
	FinalChoiceID               *uuid.UUID             `gorm:"type:uuid" json:"final_choice_id,omitempty"`
	PathChangeCountAfterPayment int                    `gorm:"not null;default:0" json:"path_change_count_after_payment"`
	PreviousApplicationID       *uuid.UUID             `gorm:"type:uuid;index" json:"previous_application_id,omitempty"`
	SubmittedAt                 *time.Time             `json:"submitted_at,omitempty"`
	VerifiedAt                  *time.Time             `json:"verified_at,omitempty"`
	VerifiedBy                  *uuid.UUID             `gorm:"type:uuid" json:"verified_by,omitempty"`
	Notes                       string                 `gorm:"type:text" json:"notes,omitempty"`
	Period                      *Period                `gorm:"foreignKey:PeriodID" json:"period,omitempty"`
	AdmissionPath               *AdmissionPath         `gorm:"foreignKey:AdmissionPathID" json:"admission_path,omitempty"`
	PreviousApplication         *Application           `gorm:"foreignKey:PreviousApplicationID" json:"previous_application,omitempty"`
	Documents                   []ApplicationDocument  `gorm:"foreignKey:ApplicationID" json:"documents,omitempty"`
	StatusLogs                  []ApplicationStatusLog `gorm:"foreignKey:ApplicationID" json:"status_logs,omitempty"`
	Changes                     []ApplicationChange    `gorm:"foreignKey:ApplicationID" json:"changes,omitempty"`
	PathChangeRequests          []PathChangeRequest    `gorm:"foreignKey:ApplicationID" json:"path_change_requests,omitempty"`
	PersonData                  interface{}            `gorm:"-" json:"person_data,omitempty"`
	InvoiceData                 interface{}            `gorm:"-" json:"invoice_data,omitempty"`
	CanCancel                   bool                   `gorm:"-" json:"can_cancel"`
	CreatedAt                   time.Time              `json:"created_at"`
	UpdatedAt                   time.Time              `json:"updated_at"`
}

func (a *Application) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// IsCancellable memeriksa apakah pendaftaran diizinkan untuk dibatalkan (CR-002, Bagian 26)
func (a *Application) IsCancellable() bool {
	if a.Status == StatusDraft || a.Status == StatusAwaitingPayment {
		return true
	}
	// Pendaftaran jalur gratis (registration_fee = 0) diperlakukan seperti belum membayar (boleh dibatalkan pada tahap form_open)
	if a.Status == StatusFormOpen && a.AdmissionPath != nil && a.AdmissionPath.RegistrationFee == 0 {
		return true
	}
	return false
}

// IsActive memeriksa apakah pendaftaran masih aktif
func (a *Application) IsActive() bool {
	return a.Status != StatusCancelled && a.Status != StatusFailed && a.Status != StatusExpired && a.Status != StatusWithdrawn
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
