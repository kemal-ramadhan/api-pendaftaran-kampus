package student

import (
	"context"

	"github.com/google/uuid"
)

type StudentDTO struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	ApplicationID      uuid.UUID `json:"application_id"`
	NIM                string    `json:"nim"`
	StudyProgramID     uuid.UUID `json:"study_program_id"`
	Status             string    `json:"status"`
}

type ReregistrationDTO struct {
	ID            uuid.UUID `json:"id"`
	ApplicationID uuid.UUID `json:"application_id"`
	Status        string    `json:"status"` // pending, verified, completed
}

type API interface {
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*StudentDTO, error)
	GetReregistration(ctx context.Context, appID uuid.UUID) (*ReregistrationDTO, error)
}
