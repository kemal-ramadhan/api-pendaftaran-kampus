package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	AttendancePresent = "present"
	AttendanceAbsent  = "absent"
)

type Participant struct {
	ID                uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID         uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:uq_session_application" json:"session_id"`
	ApplicationID     uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:uq_session_application;index" json:"application_id"`
	UserID            uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	ParticipantNumber string       `gorm:"type:varchar(50);not null;uniqueIndex" json:"participant_number"`
	Attendance        *string      `gorm:"type:varchar(20)" json:"attendance"`
	CreatedAt         time.Time    `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt         time.Time    `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`

	Session           *ExamSession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	ExamCard          *ExamCard    `gorm:"foreignKey:ParticipantID" json:"exam_card,omitempty"`

	// Non-persistent fields enriched from external services
	FullName         string  `gorm:"-" json:"full_name,omitempty"`
	Email            string  `gorm:"-" json:"email,omitempty"`
	StudyProgramName string  `gorm:"-" json:"study_program_name,omitempty"`
	PathName         string  `gorm:"-" json:"path_name,omitempty"`
	PeriodName       string  `gorm:"-" json:"period_name,omitempty"`
	PhotoURL         *string `gorm:"-" json:"photo_url,omitempty"`
}

type ExamCard struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParticipantID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"participant_id"`
	PdfObjectKey  string    `gorm:"type:varchar(255);not null" json:"pdf_object_key"`
	GeneratedAt   time.Time `gorm:"type:timestamptz;not null;default:current_timestamp" json:"generated_at"`
	CreatedAt     time.Time `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`

	Participant   *Participant `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
}
