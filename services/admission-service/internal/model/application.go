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
	StatusPaymentVerified    ApplicationStatus = "payment_verified"
	StatusDocumentSubmitted  ApplicationStatus = "document_submitted"
	StatusUnderReview        ApplicationStatus = "under_review"
	StatusExamScheduled      ApplicationStatus = "exam_scheduled"
	StatusInterviewScheduled ApplicationStatus = "interview_scheduled"
	StatusDirectDecision     ApplicationStatus = "direct_decision"
	StatusSelectionPassed    ApplicationStatus = "selection_passed"
	StatusSelectionFailed    ApplicationStatus = "selection_failed"
	StatusRegistered         ApplicationStatus = "registered"
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
	Period          *Period                `gorm:"foreignKey:PeriodID" json:"period,omitempty"`
	AdmissionPath   *AdmissionPath         `gorm:"foreignKey:AdmissionPathID" json:"admission_path,omitempty"`
	Documents       []ApplicationDocument  `gorm:"foreignKey:ApplicationID" json:"documents,omitempty"`
	StatusLogs      []ApplicationStatusLog `gorm:"foreignKey:ApplicationID" json:"status_logs,omitempty"`
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
