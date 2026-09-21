package dto

import "github.com/google/uuid"

type ItemResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
