package identity

import (
	"context"

	"github.com/google/uuid"
)

type PersonDTO struct {
	UserID     uuid.UUID `json:"user_id"`
	FullName   string    `json:"full_name"`
	NIK        string    `json:"nik"`
	BirthPlace string    `json:"birth_place"`
	Gender     string    `json:"gender"`
	Phone      string    `json:"phone"`
	PhotoURL   string    `json:"photo_url"`
}

type ProfileDTO struct {
	UserID   uuid.UUID  `json:"user_id"`
	FullName string     `json:"full_name"`
	NIK      string     `json:"nik"`
	Phone    string     `json:"phone"`
	Person   *PersonDTO `json:"person,omitempty"`
}

type API interface {
	GetPerson(ctx context.Context, userID uuid.UUID) (*PersonDTO, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileDTO, error)
}
