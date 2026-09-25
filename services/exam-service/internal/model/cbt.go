package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuestionType string

const (
	QuestionTypeSingleChoice   QuestionType = "single_choice"
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
	QuestionTypeTrueFalse      QuestionType = "true_false"
	QuestionTypeEssay          QuestionType = "essay"
)

type CBTExamStatus string

const (
	CBTExamStatusDraft     CBTExamStatus = "draft"
	CBTExamStatusPublished CBTExamStatus = "published"
	CBTExamStatusClosed    CBTExamStatus = "closed"
)

type CBTAttemptStatus string

const (
	CBTAttemptStatusNotStarted   CBTAttemptStatus = "not_started"
	CBTAttemptStatusInProgress   CBTAttemptStatus = "in_progress"
	CBTAttemptStatusSubmitted    CBTAttemptStatus = "submitted"
	CBTAttemptStatusAutoSubmitted CBTAttemptStatus = "auto_submitted"
	CBTAttemptStatusDisqualified CBTAttemptStatus = "disqualified"
)

// QuestionBank merepresentasikan bank soal per subjek ujian
type QuestionBank struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name             string         `gorm:"type:varchar(255);not null" json:"name"`
	Subject          string         `gorm:"type:varchar(100);not null" json:"subject"` // Matematika, Bahasa Inggris, TPA, dll.
	PathID           *uuid.UUID     `gorm:"type:uuid" json:"path_id,omitempty"`
	StudyProgramCode *string        `gorm:"type:varchar(50)" json:"study_program_code,omitempty"`
	IsActive         bool           `gorm:"type:boolean;not null;default:true" json:"is_active"`
	Questions        []Question     `gorm:"foreignKey:BankID" json:"questions,omitempty"`
	CreatedAt        time.Time      `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// Question merepresentasikan satu butir pertanyaan
type Question struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BankID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"bank_id"`
	Type           QuestionType     `gorm:"type:varchar(30);not null;index" json:"type"`
	Content        string           `gorm:"type:text;not null" json:"content"` // Markdown / teks tersanitasi
	ImageObjectKey *string          `gorm:"type:varchar(255)" json:"image_object_key,omitempty"`
	ImageURL       string           `gorm:"-" json:"image_url,omitempty"`
	Points         int              `gorm:"type:int;not null;default:10" json:"points"`
	Difficulty     string           `gorm:"type:varchar(20);not null;default:'medium'" json:"difficulty"` // easy, medium, hard
	Explanation    *string          `gorm:"type:text" json:"explanation,omitempty"`                     // Hanya untuk admin
	IsActive       bool             `gorm:"type:boolean;not null;default:true" json:"is_active"`
	Options        []QuestionOption `gorm:"foreignKey:QuestionID" json:"options,omitempty"`
	CreatedAt      time.Time        `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time        `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`
	DeletedAt      gorm.DeletedAt   `gorm:"index" json:"-"`
}

// QuestionOption merepresentasikan pilihan jawaban (A-E)
type QuestionOption struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	QuestionID     uuid.UUID `gorm:"type:uuid;not null;index" json:"question_id"`
	Label          string    `gorm:"type:varchar(5);not null" json:"label"` // A, B, C, D, E
	Content        string    `gorm:"type:text;not null" json:"content"`
	ImageObjectKey *string   `gorm:"type:varchar(255)" json:"image_object_key,omitempty"`
	ImageURL       string    `gorm:"-" json:"image_url,omitempty"`
	IsCorrect      bool      `gorm:"type:boolean;not null;default:false" json:"is_correct,omitempty"`
	CreatedAt      time.Time `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`
}

// CBTExam merepresentasikan satu paket ujian CBT yang terikat pada ExamSession
type CBTExam struct {
	ID                       uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID                uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"session_id"`
	Title                    string         `gorm:"type:varchar(255);not null" json:"title"`
	Instructions             string         `gorm:"type:text" json:"instructions"`
	DurationMinutes          int            `gorm:"type:int;not null;default:90" json:"duration_minutes"`
	QuestionCount            int            `gorm:"type:int;not null;default:50" json:"question_count"`
	RandomizeQuestions       bool           `gorm:"type:boolean;not null;default:true" json:"randomize_questions"`
	RandomizeOptions         bool           `gorm:"type:boolean;not null;default:true" json:"randomize_options"`
	ShowResultToParticipant  bool           `gorm:"type:boolean;not null;default:false" json:"show_result_to_participant"`
	RequiresAccessToken      bool           `gorm:"type:boolean;not null;default:true" json:"requires_access_token"`
	AccessTokenHash          string         `gorm:"type:varchar(255)" json:"-"`
	AccessTokenPlain         string         `gorm:"-" json:"access_token,omitempty"`
	Status                   CBTExamStatus  `gorm:"type:varchar(30);not null;default:'draft';index" json:"status"`
	Rules                    []CBTExamRule  `gorm:"foreignKey:CBTExamID" json:"rules,omitempty"`
	Session                  *ExamSession   `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	CreatedAt                time.Time      `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt                time.Time      `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"-"`
}

// CBTExamRule merepresentasikan aturan komposisi soal dari bank soal
type CBTExamRule struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CBTExamID  uuid.UUID `gorm:"type:uuid;not null;index" json:"cbt_exam_id"`
	BankID     uuid.UUID `gorm:"type:uuid;not null" json:"bank_id"`
	Difficulty *string   `gorm:"type:varchar(20)" json:"difficulty,omitempty"`
	Count      int       `gorm:"type:int;not null;default:10" json:"count"`
}

// CBTAttempt merepresentasikan satu sesi pengerjaan ujian oleh peserta
type CBTAttempt struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CBTExamID      uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:uq_exam_participant" json:"cbt_exam_id"`
	ParticipantID  uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:uq_exam_participant" json:"participant_id"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	Status         CBTAttemptStatus `gorm:"type:varchar(30);not null;default:'not_started';index" json:"status"`
	StartedAt      *time.Time       `gorm:"type:timestamptz" json:"started_at,omitempty"`
	DeadlineAt     *time.Time       `gorm:"type:timestamptz" json:"deadline_at,omitempty"`
	SubmittedAt    *time.Time       `gorm:"type:timestamptz" json:"submitted_at,omitempty"`
	QuestionOrder  string           `gorm:"type:text" json:"-"` // JSON snapshot ID soal terurut
	ObjectiveScore float64          `gorm:"type:decimal(5,2);not null;default:0" json:"objective_score"`
	EssayScore     float64          `gorm:"type:decimal(5,2);not null;default:0" json:"essay_score"`
	FinalScore     float64          `gorm:"type:decimal(5,2);not null;default:0" json:"final_score"`
	GradedAt       *time.Time       `gorm:"type:timestamptz" json:"graded_at,omitempty"`
	Answers        []CBTAnswer      `gorm:"foreignKey:AttemptID" json:"answers,omitempty"`
	CBTExam        *CBTExam         `gorm:"foreignKey:CBTExamID" json:"cbt_exam,omitempty"`
	Participant    *Participant     `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
	CreatedAt      time.Time        `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time        `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`
}

// CBTAnswer merepresentasikan respon jawaban peserta untuk satu soal
type CBTAnswer struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AttemptID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_attempt_question" json:"attempt_id"`
	QuestionID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_attempt_question" json:"question_id"`
	SelectedOptionIDs string    `gorm:"type:text" json:"selected_option_ids,omitempty"` // JSON string array uuid
	EssayText         *string   `gorm:"type:text" json:"essay_text,omitempty"`
	IsFlagged         bool      `gorm:"type:boolean;not null;default:false" json:"is_flagged"`
	IsCorrect         *bool     `gorm:"type:boolean" json:"is_correct,omitempty"`
	PointsAwarded     float64   `gorm:"type:decimal(5,2);not null;default:0" json:"points_awarded"`
	AnsweredAt        *time.Time `gorm:"type:timestamptz" json:"answered_at,omitempty"`
	GradedBy          *uuid.UUID `gorm:"type:uuid" json:"graded_by,omitempty"`
	Question          *Question `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
	CreatedAt         time.Time `gorm:"type:timestamptz;not null;default:current_timestamp" json:"created_at"`
	UpdatedAt         time.Time `gorm:"type:timestamptz;not null;default:current_timestamp" json:"updated_at"`
}

// CBTEvent merepresentasikan log aktivitas pengawasan ujian
type CBTEvent struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AttemptID  uuid.UUID `gorm:"type:uuid;not null;index" json:"attempt_id"`
	Type       string    `gorm:"type:varchar(50);not null" json:"type"` // start, resume, focus_lost, focus_back, fullscreen_exit, submit, auto_submit, reset
	ClientTime *time.Time `gorm:"type:timestamptz" json:"client_time,omitempty"`
	ServerTime time.Time `gorm:"type:timestamptz;not null;default:current_timestamp" json:"server_time"`
	Metadata   string    `gorm:"type:text" json:"metadata,omitempty"` // JSON string
}
