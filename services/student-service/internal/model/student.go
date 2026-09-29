package model

import (
	"time"

	"github.com/google/uuid"
)

type ReregistrationStatus string

const (
	ReregistrationStatusWaitingPayment ReregistrationStatus = "waiting_payment"
	ReregistrationStatusInProgress     ReregistrationStatus = "in_progress"
	ReregistrationStatusCompleted      ReregistrationStatus = "completed"
	ReregistrationStatusCancelled      ReregistrationStatus = "cancelled"
)

type StudentStatus string

const (
	StudentStatusReregistered StudentStatus = "reregistered"
	StudentStatusActive       StudentStatus = "active"
	StudentStatusWithdrawn    StudentStatus = "withdrawn"
)

// Reregistration merepresentasikan data pendaftaran ulang calon mahasiswa baru
type Reregistration struct {
	ID                    uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ApplicationID         uuid.UUID            `gorm:"type:uuid;not null;uniqueIndex" json:"application_id"`
	UserID                uuid.UUID            `gorm:"type:uuid;not null;index" json:"user_id"`
	AcceptedProgram       string               `gorm:"type:varchar(30);not null" json:"accepted_program"`
	Status                ReregistrationStatus `gorm:"type:varchar(30);not null;default:'waiting_payment'" json:"status"`
	JacketSize            string               `gorm:"type:varchar(10)" json:"jacket_size"`
	EmergencyContactName  string               `gorm:"type:varchar(150)" json:"emergency_contact_name"`
	EmergencyContactPhone string               `gorm:"type:varchar(30)" json:"emergency_contact_phone"`
	StatementAgreed       bool                 `gorm:"not null;default:false" json:"statement_agreed"`
	CompletedAt           *time.Time           `json:"completed_at"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

func (Reregistration) TableName() string {
	return "reregistrations"
}

// Student merepresentasikan data induk mahasiswa yang telah menyelesaikan daftar ulang
type Student struct {
	ID               uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID           uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	ApplicationID    uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex" json:"application_id"`
	NIM              string        `gorm:"type:varchar(30)" json:"nim"`
	StudyProgramCode string        `gorm:"type:varchar(30);not null;index" json:"study_program_code"`
	EntryYear        int           `gorm:"not null;index" json:"entry_year"`
	EntryPath        string        `gorm:"type:varchar(50);not null" json:"entry_path"`
	Status           StudentStatus `gorm:"type:varchar(30);not null;default:'reregistered';index" json:"status"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

func (Student) TableName() string {
	return "students"
}

// OutboxEvent menyimpan pesan kejadian antar-layanan (mis. penerbitan NIM di Tahap 11)
type OutboxEvent struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EventType   string     `gorm:"type:varchar(100);not null" json:"event_type"`
	Payload     string     `gorm:"type:jsonb;not null" json:"payload"`
	Status      string     `gorm:"type:varchar(30);not null;default:'pending';index" json:"status"`
	RetryCount  int        `gorm:"not null;default:0" json:"retry_count"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}

// DTOs

type UpdateReregistrationRequest struct {
	JacketSize            string `json:"jacket_size" binding:"required"`
	EmergencyContactName  string `json:"emergency_contact_name" binding:"required"`
	EmergencyContactPhone string `json:"emergency_contact_phone" binding:"required"`
	StatementAgreed       *bool  `json:"statement_agreed" binding:"required"`
}

type ReregistrationDetailResponse struct {
	Reregistration   *Reregistration `json:"reregistration"`
	IsPaymentCleared bool            `json:"is_payment_cleared"`
	PaymentStatus    string          `json:"payment_status"` // unpaid, partially_paid, paid
	InvoiceID        *uuid.UUID      `json:"invoice_id,omitempty"`
	CanComplete      bool            `json:"can_complete"`
}

type StudentFilter struct {
	StudyProgramCode string
	Status           string
	EntryYear        int
	Search           string
	Limit            int
	Offset           int
}

type NIMIssuancePayload struct {
	ApplicationID    uuid.UUID `json:"application_id"`
	UserID           uuid.UUID `json:"user_id"`
	StudyProgramCode string    `json:"study_program_code"`
	EntryYear        int       `json:"entry_year"`
	EntryPath        string    `json:"entry_path"`
}
