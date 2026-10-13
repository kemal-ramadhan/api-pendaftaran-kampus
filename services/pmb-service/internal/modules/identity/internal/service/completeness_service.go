package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/repository"
)

type CompletenessService interface {
	CheckCompleteness(ctx context.Context, userID uuid.UUID) (*dto.CompletenessResponse, error)
}

type completenessService struct {
	personRepo repository.PersonRepository
}

func NewCompletenessService(personRepo repository.PersonRepository) CompletenessService {
	return &completenessService{personRepo: personRepo}
}

func (s *completenessService) CheckCompleteness(ctx context.Context, userID uuid.UUID) (*dto.CompletenessResponse, error) {
	person, err := s.personRepo.GetDetailWithRelations(ctx, userID)
	if err != nil {
		if err == repository.ErrPersonNotFound {
			return &dto.CompletenessResponse{
				Complete: false,
				Missing: []string{
					"profile.full_name",
					"profile.nik",
					"profile.birth_place",
					"profile.birth_date",
					"profile.gender",
					"profile.religion",
					"profile.phone",
					"addresses.ktp",
					"guardians",
					"education_histories",
				},
			}, nil
		}
		return nil, err
	}

	var missing []string

	// 1. Validasi Profil
	if strings.TrimSpace(person.FullName) == "" {
		missing = append(missing, "profile.full_name")
	}
	if strings.TrimSpace(person.NIK) == "" {
		missing = append(missing, "profile.nik")
	}
	if strings.TrimSpace(person.BirthPlace) == "" {
		missing = append(missing, "profile.birth_place")
	}
	if person.BirthDate == nil {
		missing = append(missing, "profile.birth_date")
	}
	if strings.TrimSpace(person.Gender) == "" {
		missing = append(missing, "profile.gender")
	}
	if strings.TrimSpace(person.Religion) == "" {
		missing = append(missing, "profile.religion")
	}
	if strings.TrimSpace(person.Phone) == "" {
		missing = append(missing, "profile.phone")
	}

	// 2. Validasi Alamat KTP
	hasValidKTPAddress := false
	for _, addr := range person.Addresses {
		if strings.ToLower(addr.Type) == "ktp" {
			if strings.TrimSpace(addr.Street) != "" &&
				strings.TrimSpace(addr.Village) != "" &&
				strings.TrimSpace(addr.District) != "" &&
				strings.TrimSpace(addr.City) != "" &&
				strings.TrimSpace(addr.Province) != "" {
				hasValidKTPAddress = true
				break
			}
		}
	}
	if !hasValidKTPAddress {
		missing = append(missing, "addresses.ktp")
	}

	// 3. Validasi Orang Tua / Wali
	hasValidGuardian := false
	for _, g := range person.Guardians {
		if strings.TrimSpace(g.Name) != "" && strings.TrimSpace(g.Phone) != "" {
			hasValidGuardian = true
			break
		}
	}
	if !hasValidGuardian {
		missing = append(missing, "guardians")
	}

	// 4. Validasi Riwayat Pendidikan Terakhir
	hasValidEducation := false
	for _, e := range person.EducationHistories {
		if strings.TrimSpace(e.SchoolName) != "" && strings.TrimSpace(e.Level) != "" && e.GraduationYear > 0 {
			hasValidEducation = true
			break
		}
	}
	if !hasValidEducation {
		missing = append(missing, "education_histories")
	}

	return &dto.CompletenessResponse{
		Complete: len(missing) == 0,
		Missing:  missing,
	}, nil
}
