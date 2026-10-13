package exam

import (
	"context"

	"github.com/google/uuid"
)

type ParticipantDTO struct {
	ID            uuid.UUID `json:"id"`
	ApplicationID uuid.UUID `json:"application_id"`
	CardNumber    string    `json:"card_number"`
	ScheduleID    uuid.UUID `json:"schedule_id"`
	Status        string    `json:"status"`
}

type API interface {
	GetParticipant(ctx context.Context, appID uuid.UUID) (*ParticipantDTO, error)
}
