package model

import (
	"time"

	"github.com/google/uuid"
)

// ScoreComponent mendefinisikan komponen nilai dan bobotnya per jalur seleksi
type ScoreComponent struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PathID        uuid.UUID `gorm:"type:uuid;not null;index:idx_score_components_path" json:"path_id"`
	Code          string    `gorm:"type:varchar(50);not null" json:"code"`
	Name          string    `gorm:"type:varchar(100);not null" json:"name"`
	WeightPercent float64   `gorm:"type:numeric(5,2);not null" json:"weight_percent"`
	MaxScore      float64   `gorm:"type:numeric(5,2);not null;default:100.00" json:"max_score"`
	IsActive      bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName menentukan nama tabel database
func (ScoreComponent) TableName() string {
	return "score_components"
}

// Score mencatat nilai komponen per pendaftar yang diinput penguji
type Score struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ApplicationID uuid.UUID `gorm:"type:uuid;not null;index:idx_scores_app" json:"application_id"`
	ComponentCode string    `gorm:"type:varchar(50);not null" json:"component_code"`
	Score         float64   `gorm:"type:numeric(5,2);not null" json:"score"`
	InputBy       uuid.UUID `gorm:"type:uuid;not null" json:"input_by"`
	InputAt       time.Time `gorm:"not null;default:now()" json:"input_at"`
	Note          string    `gorm:"type:text" json:"note"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName menentukan nama tabel database
func (Score) TableName() string {
	return "scores"
}

// ScoreSummary menyimpan hasil perhitungan nilai akhir tertimbang dan ranking prodi
type ScoreSummary struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ApplicationID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"application_id"`
	PeriodID         uuid.UUID `gorm:"type:uuid;not null" json:"period_id"`
	PathID           uuid.UUID `gorm:"type:uuid;not null" json:"path_id"`
	StudyProgramCode string    `gorm:"type:varchar(50);not null" json:"study_program_code"`
	FinalScore       float64   `gorm:"type:numeric(5,2);not null;default:0.00" json:"final_score"`
	RankInProgram    int       `gorm:"not null;default:0" json:"rank_in_program"`
	ComputedAt       time.Time `gorm:"not null;default:now()" json:"computed_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName menentukan nama tabel database
func (ScoreSummary) TableName() string {
	return "score_summaries"
}

// ProgramLock mengunci pengubahan nilai pada program studi dan periode tertentu
type ProgramLock struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PeriodID         uuid.UUID  `gorm:"type:uuid;not null" json:"period_id"`
	PathID           uuid.UUID  `gorm:"type:uuid;not null" json:"path_id"`
	StudyProgramCode string     `gorm:"type:varchar(50);not null" json:"study_program_code"`
	IsLocked         bool       `gorm:"not null;default:false" json:"is_locked"`
	LockedBy         *uuid.UUID `gorm:"type:uuid" json:"locked_by,omitempty"`
	LockedAt         *time.Time `json:"locked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName menentukan nama tabel database
func (ProgramLock) TableName() string {
	return "program_locks"
}

// DTO untuk representasi nilai komponen pendaftar
type ComponentScoreDetail struct {
	ComponentCode string  `json:"component_code"`
	ComponentName string  `json:"component_name"`
	WeightPercent float64 `json:"weight_percent"`
	MaxScore      float64 `json:"max_score"`
	Score         float64 `json:"score"`
}

// DTO baris data rekapitulasi pendaftar
type ParticipantRecapItem struct {
	ApplicationID      uuid.UUID              `json:"application_id"`
	RegistrationNumber string                 `json:"registration_number"`
	FullName           string                 `json:"full_name"`
	StudyProgramCode   string                 `json:"study_program_code"`
	ComponentScores    []ComponentScoreDetail `json:"component_scores"`
	FinalScore         float64                `json:"final_score"`
	RankInProgram      int                    `json:"rank_in_program"`
}

// DTO respons rekapitulasi nilai prodi
type ProgramRecapResponse struct {
	PeriodID         uuid.UUID              `json:"period_id"`
	PathID           uuid.UUID              `json:"path_id"`
	StudyProgramCode string                 `json:"study_program_code"`
	IsLocked         bool                   `json:"is_locked"`
	TotalApplicants  int                    `json:"total_applicants"`
	Participants     []ParticipantRecapItem `json:"participants"`
}
