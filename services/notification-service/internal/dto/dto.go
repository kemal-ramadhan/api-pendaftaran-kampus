package dto

import (
	"time"

	"github.com/google/uuid"
)

// RecordActivityRequest payload dari service lain lewat outbox internal (PRD 2.9)
type RecordActivityRequest struct {
	PeriodID    string    `json:"period_id"`
	ActorID     uuid.UUID `json:"actor_id"`
	ActorName   string    `json:"actor_name"`
	ActorRole   string    `json:"actor_role"`
	Action      string    `json:"action" binding:"required"`
	ActionLabel string    `json:"action_label" binding:"required"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	TargetLabel string    `json:"target_label"`
	Service     string    `json:"service" binding:"required"`
}

// ActivityEventResponse item log aktivitas panitia untuk dashboard (PRD 2.1, 2.9)
type ActivityEventResponse struct {
	ID          uuid.UUID `json:"id"`
	PeriodID    string    `json:"period_id"`
	ActorName   string    `json:"actor_name"`
	ActorRole   string    `json:"actor_role"`
	Action      string    `json:"action"`
	ActionLabel string    `json:"action_label"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	TargetLabel string    `json:"target_label"`
	Service     string    `json:"service"`
	CreatedAt   time.Time `json:"created_at"`
}

// Broadcast Template DTOs
type CreateBroadcastTemplateRequest struct {
	Code         string `json:"code" binding:"required,min=2,max=50"`
	Name         string `json:"name" binding:"required,min=3,max=150"`
	Channel      string `json:"channel" binding:"required,oneof=email whatsapp in_app"`
	Subject      string `json:"subject"`
	BodyTemplate string `json:"body_template" binding:"required"`
}

type UpdateBroadcastTemplateRequest struct {
	Name         string `json:"name" binding:"required,min=3,max=150"`
	Channel      string `json:"channel" binding:"required,oneof=email whatsapp in_app"`
	Subject      string `json:"subject"`
	BodyTemplate string `json:"body_template" binding:"required"`
}

// Broadcast Preview DTOs (PRD 2.7)
type BroadcastPreviewRequest struct {
	TemplateID    *uint  `json:"template_id"`
	BodyTemplate  string `json:"body_template"`
	Subject       string `json:"subject"`
	TargetSegment string `json:"target_segment" binding:"required"`
	FilterValue   string `json:"filter_value"`
	PeriodID      string `json:"period_id"`
}

type BroadcastPreviewResponse struct {
	TargetSegment      string   `json:"target_segment"`
	RecipientsCount    int      `json:"recipients_count"`
	SampleMessage      string   `json:"sample_message"`
	SampleSubject      string   `json:"sample_subject"`
	SupportedVariables []string `json:"supported_variables"`
}

// Send Broadcast DTOs
type SendBroadcastRequest struct {
	TemplateID         *uint      `json:"template_id"`
	Subject            string     `json:"subject"`
	Message            string     `json:"message" binding:"required"`
	Channel            string     `json:"channel" binding:"required,oneof=email whatsapp in_app"`
	TargetSegment      string     `json:"target_segment" binding:"required"`
	FilterValue        string     `json:"filter_value"`
	PeriodID           string     `json:"period_id"`
	ExpectedRecipients int        `json:"expected_recipients"`
	ScheduledAt        *time.Time `json:"scheduled_at"`
}
