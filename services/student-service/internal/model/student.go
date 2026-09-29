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
	SiakadExportedAt *time.Time    `json:"siakad_exported_at,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

func (Student) TableName() string {
	return "students"
}

// PKKMBGroup merepresentasikan data kelompok gugus PKKMB
type PKKMBGroup struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PeriodID      *uuid.UUID `gorm:"type:uuid;index" json:"period_id,omitempty"`
	Name          string     `gorm:"type:varchar(100);not null" json:"name"`
	Capacity      int        `gorm:"not null;default:50" json:"capacity"`
	LeaderName    string     `gorm:"type:varchar(150)" json:"leader_name"`
	ScheduleNote  string     `gorm:"type:text" json:"schedule_note"`
	AssignedCount int        `gorm:"-" json:"assigned_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (PKKMBGroup) TableName() string {
	return "pkkmb_groups"
}

// PKKMBAssignment merepresentasikan penempatan mahasiswa ke suatu gugus PKKMB
type PKKMBAssignment struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StudentID  uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"student_id"`
	GroupID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"group_id"`
	AssignedBy *uuid.UUID `gorm:"type:uuid" json:"assigned_by,omitempty"`
	Method     string     `gorm:"type:varchar(30);not null;default:'auto'" json:"method"` // 'auto' | 'manual'
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (PKKMBAssignment) TableName() string {
	return "pkkmb_assignments"
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

type PKKMBGroupInfo struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	LeaderName   string    `json:"leader_name"`
	ScheduleNote string    `json:"schedule_note"`
}

type ReregistrationDetailResponse struct {
	Reregistration   *Reregistration `json:"reregistration"`
	IsPaymentCleared bool            `json:"is_payment_cleared"`
	PaymentStatus    string          `json:"payment_status"` // unpaid, partially_paid, paid
	InvoiceID        *uuid.UUID      `json:"invoice_id,omitempty"`
	CanComplete      bool            `json:"can_complete"`
	PKKMBGroup       *PKKMBGroupInfo `json:"pkkmb_group,omitempty"`
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

// DTOs baru Butir h

type EnrolmentItem struct {
	StudentID          *uuid.UUID `json:"student_id,omitempty"`
	ApplicationID      uuid.UUID  `json:"application_id"`
	UserID             uuid.UUID  `json:"user_id"`
	RegistrationNumber string     `json:"registration_number"`
	Name               string     `json:"name"`
	StudyProgramCode   string     `json:"study_program_code"`
	SelectionStatus    string     `json:"selection_status"`
	TuitionStatus      string     `json:"tuition_status"`
	NIM                string     `json:"nim"`
	PKKMBGroup         string     `json:"pkkmb_group"`
	PKKMBGroupID       *uuid.UUID `json:"pkkmb_group_id,omitempty"`
	CanIssueNIM        bool       `json:"can_issue_nim"`
	Actions            []string   `json:"actions"`
}

type EnrolmentFilter struct {
	PeriodID         string
	Filter           string // 'all', 'has_nim', 'ready_for_nim', 'waiting_tuition'
	Search           string
	StudyProgramCode string
	Limit            int
	Offset           int
}

type BatchNIMRequest struct {
	PeriodID         *uuid.UUID `json:"period_id"`
	StudyProgramCode string     `json:"study_program_code"`
}

type BatchNIMDetailItem struct {
	StudentID     *uuid.UUID `json:"student_id,omitempty"`
	ApplicationID uuid.UUID  `json:"application_id"`
	NIM           string     `json:"nim,omitempty"`
	Status        string     `json:"status"` // 'success', 'skipped', 'failed'
	Reason        string     `json:"reason,omitempty"`
}

type BatchNIMResponse struct {
	Processed int                  `json:"processed"`
	Success   int                  `json:"success"`
	Skipped   int                  `json:"skipped"`
	Failed    int                  `json:"failed"`
	Details   []BatchNIMDetailItem `json:"details"`
}

type CreatePKKMBGroupRequest struct {
	PeriodID     *uuid.UUID `json:"period_id"`
	Name         string     `json:"name" binding:"required"`
	Capacity     int        `json:"capacity" binding:"required,min=1"`
	LeaderName   string     `json:"leader_name"`
	ScheduleNote string     `json:"schedule_note"`
}

type UpdatePKKMBGroupRequest struct {
	Name         string `json:"name"`
	Capacity     int    `json:"capacity" binding:"omitempty,min=1"`
	LeaderName   string `json:"leader_name"`
	ScheduleNote string `json:"schedule_note"`
}

type AutoAssignPKKMBRequest struct {
	PeriodID *uuid.UUID `json:"period_id"`
}

type AutoAssignPKKMBResponse struct {
	AssignedCount   int `json:"assigned_count"`
	UnassignedCount int `json:"unassigned_count"`
	TotalGroups     int `json:"total_groups"`
}

type AssignPKKMBStudentRequest struct {
	GroupID uuid.UUID `json:"group_id" binding:"required"`
}

type SIAKADStudentItem struct {
	NIM              string `json:"nim"`
	Name             string `json:"name"`
	StudyProgramCode string `json:"study_program_code"`
	Email            string `json:"email"`
	EntryPath        string `json:"entry_path"`
	EntryYear        int    `json:"entry_year"`
}
