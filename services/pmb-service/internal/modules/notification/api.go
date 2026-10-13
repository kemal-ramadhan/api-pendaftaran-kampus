package notification

import (
	"context"

	"github.com/google/uuid"
)

type SendNotificationRequest struct {
	UserID    uuid.UUID              `json:"user_id"`
	Type      string                 `json:"type"` // email, wa, system
	Subject   string                 `json:"subject"`
	Message   string                 `json:"message"`
	Recipient string                 `json:"recipient,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type API interface {
	SendNotification(ctx context.Context, req SendNotificationRequest) error
}
