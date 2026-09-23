package model

import (
	"time"

	"github.com/google/uuid"
)

type RecommendationStatus string

const (
	StatusDraft     RecommendationStatus = "draft"
	StatusSubmitted RecommendationStatus = "submitted"
	StatusApproved  RecommendationStatus = "approved"
	StatusReturned  RecommendationStatus = "returned"
)

type DecisionType string

const (
	DecisionPass     DecisionType = "pass"
	DecisionFail     DecisionType = "fail"
	DecisionWaitlist DecisionType = "waitlist"
)

// Recommendation merepresentasikan usulan kelulusan program studi
type Recommendation struct {
	ID               uuid.UUID            `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PeriodID         uuid.UUID            `gorm:"type:uuid;not null;index:idx_recommendations_lookup" json:"period_id"`
	PathID           uuid.UUID            `gorm:"type:uuid;not null;index:idx_recommendations_lookup" json:"path_id"`
	StudyProgramCode string               `gorm:"type:varchar(50);not null;index:idx_recommendations_lookup" json:"study_program_code"`
	Status           RecommendationStatus `gorm:"type:varchar(20);not null;default:'draft';index:idx_recommendations_status" json:"status"`
	SubmittedBy      *uuid.UUID           `gorm:"type:uuid" json:"submitted_by,omitempty"`
	SubmittedAt      *time.Time           `json:"submitted_at,omitempty"`
	ApprovedBy       *uuid.UUID           `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt       *time.Time           `json:"approved_at,omitempty"`
	Note             string               `gorm:"type:text" json:"note,omitempty"`
	Items            []RecommendationItem `gorm:"foreignKey:RecommendationID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

func (Recommendation) TableName() string {
	return "recommendations"
}

// RecommendationItem mencatat keputusan per pendaftar dalam usulan kelulusan
type RecommendationItem struct {
	ID               uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RecommendationID uuid.UUID    `gorm:"type:uuid;not null;index:idx_rec_items_rec_id" json:"recommendation_id"`
	ApplicationID    uuid.UUID    `gorm:"type:uuid;not null;index:idx_rec_items_app" json:"application_id"`
	Decision         DecisionType `gorm:"type:varchar(20);not null" json:"decision"`
	FinalScore       float64      `gorm:"type:numeric(5,2);not null;default:0.00" json:"final_score"`
	Rank             int          `gorm:"not null;default:0" json:"rank"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

func (RecommendationItem) TableName() string {
	return "recommendation_items"
}

// Announcement menyimpan rilis pengumuman resmi kelulusan
type Announcement struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PeriodID    uuid.UUID `gorm:"type:uuid;not null;index:idx_announcements_period" json:"period_id"`
	Title       string    `gorm:"type:varchar(200);not null" json:"title"`
	Content     string    `gorm:"type:text" json:"content"`
	PublishedAt time.Time `gorm:"not null;default:now()" json:"published_at"`
	PublishedBy uuid.UUID `gorm:"type:uuid;not null" json:"published_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Announcement) TableName() string {
	return "announcements"
}

// Result mencatat hasil akhir penetapan kelulusan calon mahasiswa
type Result struct {
	ID              uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ApplicationID   uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_results_app" json:"application_id"`
	UserID          uuid.UUID    `gorm:"type:uuid;not null;index:idx_results_user" json:"user_id"`
	Decision        DecisionType `gorm:"type:varchar(20);not null" json:"decision"`
	AcceptedProgram string       `gorm:"type:varchar(50)" json:"accepted_program,omitempty"`
	AnnouncedAt     time.Time    `gorm:"not null" json:"announced_at"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

func (Result) TableName() string {
	return "results"
}

// DTO untuk input rekomendasi
type SaveRecommendationItemPayload struct {
	ApplicationID uuid.UUID    `json:"application_id" binding:"required"`
	Decision      DecisionType `json:"decision" binding:"required"`
	FinalScore    float64      `json:"final_score"`
	Rank          int          `json:"rank"`
}

type SaveRecommendationPayload struct {
	PeriodID         uuid.UUID                       `json:"period_id" binding:"required"`
	PathID           uuid.UUID                       `json:"path_id" binding:"required"`
	StudyProgramCode string                          `json:"study_program_code" binding:"required"`
	Items            []SaveRecommendationItemPayload `json:"items" binding:"required"`
}

type ReviewRecommendationPayload struct {
	Note string `json:"note"`
}

type PublishAnnouncementPayload struct {
	PeriodID uuid.UUID `json:"period_id" binding:"required"`
	Title    string    `json:"title" binding:"required"`
	Content  string    `json:"content"`
}

type MyResultResponse struct {
	ApplicationID      uuid.UUID    `json:"application_id"`
	RegistrationNumber string       `json:"registration_number"`
	FullName           string       `json:"full_name"`
	Decision           DecisionType `json:"decision"`
	AcceptedProgram    string       `json:"accepted_program,omitempty"`
	AnnouncedAt        time.Time    `json:"announced_at"`
	InvoiceID          *uuid.UUID   `json:"invoice_id,omitempty"`
	InvoiceNumber      string       `json:"invoice_number,omitempty"`
	Amount             int64        `json:"amount,omitempty"`
	PaymentStatus      string       `json:"payment_status,omitempty"`
}
