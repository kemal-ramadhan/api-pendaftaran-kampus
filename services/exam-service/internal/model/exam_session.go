package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionType string

const (
	SessionTypeWritten   SessionType = "written_test"
	SessionTypeInterview SessionType = "interview"
)

type ExamSession struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PeriodID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"period_id"`
	PathID            uuid.UUID      `gorm:"type:uuid;not null;index" json:"path_id"`
	Type              SessionType    `gorm:"type:varchar(50);not null;index" json:"type"`
	StudyProgramCode  *string        `gorm:"type:varchar(50)" json:"study_program_code,omitempty"`
	Title             string         `gorm:"type:varchar(255);not null" json:"title"`
	StartAt           time.Time      `gorm:"type:timestamptz;not null" json:"start_at"`
	EndAt             time.Time      `gorm:"type:timestamptz;not null" json:"end_at"`
	Location          *string        `gorm:"type:text" json:"location,omitempty"`
	MeetingLink       *string        `gorm:"type:text" json:"meeting_link,omitempty"`
	Capacity          int            `gorm:"type:int;not null;default:30" json:"capacity"`
	IsActive          bool           `gorm:"type:boolean;not null;default:true" json:"is_active"`
	CreatedAt         time.Time      `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships & non-persistent fields
	Examiners         []SessionExaminer `gorm:"foreignKey:SessionID" json:"examiners,omitempty"`
	ParticipantsCount int               `gorm:"-" json:"participants_count"`
}

type SessionExaminer struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID      uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:uq_session_examiner" json:"session_id"`
	ExaminerUserID uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:uq_session_examiner" json:"examiner_user_id"`
	CreatedAt      time.Time    `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time    `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`
	Session        *ExamSession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
}
