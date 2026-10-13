package scoring

import (
	"context"

	"github.com/google/uuid"
)

type FinalScoreDTO struct {
	ID            uuid.UUID `json:"id"`
	ApplicationID uuid.UUID `json:"application_id"`
	TotalScore    float64   `json:"total_score"`
	Passed        bool      `json:"passed"`
}

type API interface {
	GetFinalScore(ctx context.Context, appID uuid.UUID) (*FinalScoreDTO, error)
}
