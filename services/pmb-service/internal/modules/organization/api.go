package organization

import (
	"context"

	"github.com/google/uuid"
)

type StudyProgramDTO struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Level        string    `json:"level"`
	FacultyID    uuid.UUID `json:"faculty_id"`
	FacultyName  string    `json:"faculty_name"`
	Quota        int       `json:"quota"`
	IsActive     bool      `json:"is_active"`
	AdmissionOpen bool     `json:"admission_open"`
}

type FacultyDTO struct {
	ID       uuid.UUID `json:"id"`
	Code     string    `json:"code"`
	Name     string    `json:"name"`
	IsActive bool      `json:"is_active"`
}

type InstitutionProfileDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	ShortName string    `json:"short_name"`
	Address   string    `json:"address"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Website   string    `json:"website"`
	LogoURL   string    `json:"logo_url"`
}

type API interface {
	GetStudyProgramByCode(ctx context.Context, code string) (*StudyProgramDTO, error)
	GetFacultyByCode(ctx context.Context, code string) (*FacultyDTO, error)
	GetInstitutionProfile(ctx context.Context) (*InstitutionProfileDTO, error)
	ListStudyPrograms(ctx context.Context) ([]StudyProgramDTO, error)
	GetBatchStudyPrograms(ctx context.Context, codes []string) ([]StudyProgramDTO, error)
}
