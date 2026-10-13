package selection

import (
	"context"

	"github.com/google/uuid"
)

type SelectionResultDTO struct {
	ID               uuid.UUID `json:"id"`
	ApplicationID    uuid.UUID `json:"application_id"`
	Status           string    `json:"status"` // passed, rejected, waitlist
	AcceptedProgramID *uuid.UUID `json:"accepted_program_id,omitempty"`
}

type API interface {
	GetSelectionResult(ctx context.Context, appID uuid.UUID) (*SelectionResultDTO, error)
}
