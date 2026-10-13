package nim

import (
	"context"

	"github.com/google/uuid"
)

type IssueNIMRequest struct {
	ApplicationID    uuid.UUID `json:"application_id"`
	UserID           uuid.UUID `json:"user_id"`
	StudyProgramCode string    `json:"study_program_code"`
	EntryYear        int       `json:"entry_year"`
	SelectionPath    string    `json:"selection_path"`
}

type IssueNIMResponse struct {
	NIM              string `json:"nim"`
	StudyProgramCode string `json:"study_program_code"`
}

type API interface {
	IssueNIM(ctx context.Context, req IssueNIMRequest) (*IssueNIMResponse, error)
}
