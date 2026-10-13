package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/repository"
)

var (
	ErrFacultyNotFound        = errors.New("fakultas tidak ditemukan")
	ErrFacultyCodeTaken       = errors.New("kode fakultas sudah digunakan")
	ErrStudyProgramNotFound   = errors.New("program studi tidak ditemukan")
	ErrStudyProgramCodeTaken  = errors.New("kode program studi sudah digunakan")
	ErrValidation             = errors.New("validasi gagal")
)

type OrganizationService interface {
	ListFaculties(ctx context.Context, isActive *bool) ([]dto.FacultyResponse, error)
	GetFaculty(ctx context.Context, id uint) (*dto.FacultyResponse, error)
	GetFacultyByCode(ctx context.Context, code string) (*dto.FacultyResponse, error)
	CreateFaculty(ctx context.Context, req dto.CreateFacultyRequest) (*dto.FacultyResponse, error)
	UpdateFaculty(ctx context.Context, id uint, req dto.UpdateFacultyRequest) (*dto.FacultyResponse, error)
	PatchFaculty(ctx context.Context, id uint, req dto.PatchFacultyRequest) (*dto.FacultyResponse, error)

	ListStudyPrograms(ctx context.Context, facultyCode string, isActive *bool) ([]dto.StudyProgramResponse, error)
	GetStudyProgramByCode(ctx context.Context, code string) (*dto.StudyProgramResponse, error)
	GetStudyProgramByID(ctx context.Context, id uint) (*dto.StudyProgramResponse, error)
	GetStudyProgramsByCodes(ctx context.Context, codes []string) ([]dto.StudyProgramResponse, error)
	CreateStudyProgram(ctx context.Context, req dto.CreateStudyProgramRequest) (*dto.StudyProgramResponse, error)
	UpdateStudyProgram(ctx context.Context, id uint, req dto.UpdateStudyProgramRequest) (*dto.StudyProgramResponse, error)
	PatchStudyProgram(ctx context.Context, id uint, req dto.PatchStudyProgramRequest) (*dto.StudyProgramResponse, error)
	PatchAdmissionOpen(ctx context.Context, code string, admissionOpen bool) (*dto.StudyProgramResponse, error)

	GetInstitutionProfile(ctx context.Context) (*dto.InstitutionProfileResponse, error)
	UpdateInstitutionProfile(ctx context.Context, req dto.UpdateInstitutionProfileRequest) (*dto.InstitutionProfileResponse, error)
	GetProgramsOverview(ctx context.Context, facultyFilter, query string, scopeProdi, scopeFaculty string) (*dto.ProgramsOverviewResponse, error)
}

type organizationService struct {
	facultyRepo            repository.FacultyRepository
	studyProgramRepo       repository.StudyProgramRepository
	institutionProfileRepo repository.InstitutionProfileRepository
}

func NewOrganizationService(
	facultyRepo repository.FacultyRepository,
	studyProgramRepo repository.StudyProgramRepository,
	institutionProfileRepo repository.InstitutionProfileRepository,
) OrganizationService {
	return &organizationService{
		facultyRepo:            facultyRepo,
		studyProgramRepo:       studyProgramRepo,
		institutionProfileRepo: institutionProfileRepo,
	}
}

func (s *organizationService) ListFaculties(ctx context.Context, isActive *bool) ([]dto.FacultyResponse, error) {
	faculties, err := s.facultyRepo.FindAll(ctx, isActive)
	if err != nil {
		return nil, err
	}
	res := make([]dto.FacultyResponse, len(faculties))
	for i, f := range faculties {
		res[i] = dto.FacultyResponse{
			ID:        f.ID,
			Code:      f.Code,
			Name:      f.Name,
			IsActive:  f.IsActive,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		}
	}
	return res, nil
}

func (s *organizationService) GetFaculty(ctx context.Context, id uint) (*dto.FacultyResponse, error) {
	f, err := s.facultyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrFacultyNotFound
	}
	return &dto.FacultyResponse{
		ID:        f.ID,
		Code:      f.Code,
		Name:      f.Name,
		IsActive:  f.IsActive,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}, nil
}

func (s *organizationService) CreateFaculty(ctx context.Context, req dto.CreateFacultyRequest) (*dto.FacultyResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	existing, err := s.facultyRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: kode '%s'", ErrFacultyCodeTaken, code)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	now := time.Now()
	f := &model.Faculty{
		Code:      code,
		Name:      strings.TrimSpace(req.Name),
		IsActive:  isActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.facultyRepo.Create(ctx, f); err != nil {
		return nil, err
	}

	return &dto.FacultyResponse{
		ID:        f.ID,
		Code:      f.Code,
		Name:      f.Name,
		IsActive:  f.IsActive,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}, nil
}

func (s *organizationService) UpdateFaculty(ctx context.Context, id uint, req dto.UpdateFacultyRequest) (*dto.FacultyResponse, error) {
	f, err := s.facultyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrFacultyNotFound
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code != f.Code {
		existing, err := s.facultyRepo.FindByCode(ctx, code)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, fmt.Errorf("%w: kode '%s'", ErrFacultyCodeTaken, code)
		}
	}

	f.Code = code
	f.Name = strings.TrimSpace(req.Name)
	f.IsActive = req.IsActive
	f.UpdatedAt = time.Now()

	if err := s.facultyRepo.Update(ctx, f); err != nil {
		return nil, err
	}

	return &dto.FacultyResponse{
		ID:        f.ID,
		Code:      f.Code,
		Name:      f.Name,
		IsActive:  f.IsActive,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}, nil
}

func (s *organizationService) PatchFaculty(ctx context.Context, id uint, req dto.PatchFacultyRequest) (*dto.FacultyResponse, error) {
	f, err := s.facultyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrFacultyNotFound
	}

	if req.Code != nil {
		code := strings.ToUpper(strings.TrimSpace(*req.Code))
		if code != f.Code {
			existing, err := s.facultyRepo.FindByCode(ctx, code)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return nil, fmt.Errorf("%w: kode '%s'", ErrFacultyCodeTaken, code)
			}
			f.Code = code
		}
	}
	if req.Name != nil {
		f.Name = strings.TrimSpace(*req.Name)
	}
	if req.IsActive != nil {
		f.IsActive = *req.IsActive
	}
	f.UpdatedAt = time.Now()

	if err := s.facultyRepo.Update(ctx, f); err != nil {
		return nil, err
	}

	return &dto.FacultyResponse{
		ID:        f.ID,
		Code:      f.Code,
		Name:      f.Name,
		IsActive:  f.IsActive,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}, nil
}

func (s *organizationService) ListStudyPrograms(ctx context.Context, facultyCode string, isActive *bool) ([]dto.StudyProgramResponse, error) {
	programs, err := s.studyProgramRepo.FindAll(ctx, facultyCode, isActive)
	if err != nil {
		return nil, err
	}
	res := make([]dto.StudyProgramResponse, len(programs))
	for i, sp := range programs {
		res[i] = toStudyProgramResponse(&sp)
	}
	return res, nil
}

func (s *organizationService) GetStudyProgramByCode(ctx context.Context, code string) (*dto.StudyProgramResponse, error) {
	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	sp, err := s.studyProgramRepo.FindByCode(ctx, cleanCode)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, ErrStudyProgramNotFound
	}
	res := toStudyProgramResponse(sp)
	return &res, nil
}

func (s *organizationService) GetStudyProgramByID(ctx context.Context, id uint) (*dto.StudyProgramResponse, error) {
	sp, err := s.studyProgramRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, ErrStudyProgramNotFound
	}
	res := toStudyProgramResponse(sp)
	return &res, nil
}

func (s *organizationService) GetStudyProgramsByCodes(ctx context.Context, codes []string) ([]dto.StudyProgramResponse, error) {
	cleanCodes := make([]string, 0, len(codes))
	for _, c := range codes {
		trimmed := strings.ToUpper(strings.TrimSpace(c))
		if trimmed != "" {
			cleanCodes = append(cleanCodes, trimmed)
		}
	}
	programs, err := s.studyProgramRepo.FindByCodes(ctx, cleanCodes)
	if err != nil {
		return nil, err
	}
	res := make([]dto.StudyProgramResponse, len(programs))
	for i, sp := range programs {
		res[i] = toStudyProgramResponse(&sp)
	}
	return res, nil
}

func (s *organizationService) CreateStudyProgram(ctx context.Context, req dto.CreateStudyProgramRequest) (*dto.StudyProgramResponse, error) {
	faculty, err := s.facultyRepo.FindByID(ctx, req.FacultyID)
	if err != nil {
		return nil, err
	}
	if faculty == nil {
		return nil, fmt.Errorf("%w: fakultas ID %d tidak ditemukan", ErrFacultyNotFound, req.FacultyID)
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	existing, err := s.studyProgramRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: kode '%s'", ErrStudyProgramCodeTaken, code)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	now := time.Now()
	sp := &model.StudyProgram{
		FacultyID:     req.FacultyID,
		Code:          code,
		NIMCode:       strings.TrimSpace(req.NIMCode),
		Name:          strings.TrimSpace(req.Name),
		Degree:        strings.TrimSpace(req.Degree),
		Accreditation: strings.TrimSpace(req.Accreditation),
		IsActive:      isActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.studyProgramRepo.Create(ctx, sp); err != nil {
		return nil, err
	}

	sp.Faculty = faculty
	res := toStudyProgramResponse(sp)
	return &res, nil
}

func (s *organizationService) UpdateStudyProgram(ctx context.Context, id uint, req dto.UpdateStudyProgramRequest) (*dto.StudyProgramResponse, error) {
	sp, err := s.studyProgramRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, ErrStudyProgramNotFound
	}

	faculty, err := s.facultyRepo.FindByID(ctx, req.FacultyID)
	if err != nil {
		return nil, err
	}
	if faculty == nil {
		return nil, fmt.Errorf("%w: fakultas ID %d tidak ditemukan", ErrFacultyNotFound, req.FacultyID)
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code != sp.Code {
		existing, err := s.studyProgramRepo.FindByCode(ctx, code)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, fmt.Errorf("%w: kode '%s'", ErrStudyProgramCodeTaken, code)
		}
	}

	sp.FacultyID = req.FacultyID
	sp.Code = code
	sp.NIMCode = strings.TrimSpace(req.NIMCode)
	sp.Name = strings.TrimSpace(req.Name)
	sp.Degree = strings.TrimSpace(req.Degree)
	sp.Accreditation = strings.TrimSpace(req.Accreditation)
	sp.IsActive = req.IsActive
	sp.UpdatedAt = time.Now()

	if err := s.studyProgramRepo.Update(ctx, sp); err != nil {
		return nil, err
	}

	sp.Faculty = faculty
	res := toStudyProgramResponse(sp)
	return &res, nil
}

func (s *organizationService) PatchStudyProgram(ctx context.Context, id uint, req dto.PatchStudyProgramRequest) (*dto.StudyProgramResponse, error) {
	sp, err := s.studyProgramRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, ErrStudyProgramNotFound
	}

	if req.FacultyID != nil {
		faculty, err := s.facultyRepo.FindByID(ctx, *req.FacultyID)
		if err != nil {
			return nil, err
		}
		if faculty == nil {
			return nil, fmt.Errorf("%w: fakultas ID %d tidak ditemukan", ErrFacultyNotFound, *req.FacultyID)
		}
		sp.FacultyID = *req.FacultyID
		sp.Faculty = faculty
	}

	if req.Code != nil {
		code := strings.ToUpper(strings.TrimSpace(*req.Code))
		if code != sp.Code {
			existing, err := s.studyProgramRepo.FindByCode(ctx, code)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return nil, fmt.Errorf("%w: kode '%s'", ErrStudyProgramCodeTaken, code)
			}
			sp.Code = code
		}
	}
	if req.NIMCode != nil {
		sp.NIMCode = strings.TrimSpace(*req.NIMCode)
	}
	if req.Name != nil {
		sp.Name = strings.TrimSpace(*req.Name)
	}
	if req.Degree != nil {
		sp.Degree = strings.TrimSpace(*req.Degree)
	}
	if req.Accreditation != nil {
		sp.Accreditation = strings.TrimSpace(*req.Accreditation)
	}
	if req.IsActive != nil {
		sp.IsActive = *req.IsActive
	}
	sp.UpdatedAt = time.Now()

	if err := s.studyProgramRepo.Update(ctx, sp); err != nil {
		return nil, err
	}

	res := toStudyProgramResponse(sp)
	return &res, nil
}

func (s *organizationService) GetFacultyByCode(ctx context.Context, code string) (*dto.FacultyResponse, error) {
	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	f, err := s.facultyRepo.FindByCode(ctx, cleanCode)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrFacultyNotFound
	}
	return &dto.FacultyResponse{
		ID:        f.ID,
		Code:      f.Code,
		Name:      f.Name,
		IsActive:  f.IsActive,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}, nil
}

func (s *organizationService) PatchAdmissionOpen(ctx context.Context, code string, admissionOpen bool) (*dto.StudyProgramResponse, error) {
	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	sp, err := s.studyProgramRepo.FindByCode(ctx, cleanCode)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, ErrStudyProgramNotFound
	}

	if err := s.studyProgramRepo.UpdateAdmissionOpen(ctx, cleanCode, admissionOpen); err != nil {
		return nil, err
	}

	sp.AdmissionOpen = admissionOpen
	res := toStudyProgramResponse(sp)
	return &res, nil
}

func (s *organizationService) GetInstitutionProfile(ctx context.Context) (*dto.InstitutionProfileResponse, error) {
	if s.institutionProfileRepo == nil {
		return &dto.InstitutionProfileResponse{
			FoundationName:  "Yayasan Pendidikan Bakti Nusantara",
			InstitutionName: "Universitas Bakti Teknologi Nusantara",
			DocumentCode:    "UBTH",
			City:            "Bandung",
		}, nil
	}
	profile, err := s.institutionProfileRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return &dto.InstitutionProfileResponse{
			FoundationName:  "Yayasan Pendidikan Bakti Nusantara",
			InstitutionName: "Universitas Bakti Teknologi Nusantara",
			DocumentCode:    "UBTH",
			City:            "Bandung",
		}, nil
	}

	bankDTOs := make([]dto.BankAccountDTO, len(profile.BankAccounts))
	for i, b := range profile.BankAccounts {
		bankDTOs[i] = dto.BankAccountDTO{
			BankName:      b.BankName,
			AccountNumber: b.AccountNumber,
			AccountName:   b.AccountName,
			IsPrimary:     b.IsPrimary,
		}
	}

	return &dto.InstitutionProfileResponse{
		ID:                  profile.ID,
		FoundationName:      profile.FoundationName,
		InstitutionName:     profile.InstitutionName,
		DocumentCode:        profile.DocumentCode,
		Address:             profile.Address,
		City:                profile.City,
		Email:               profile.Email,
		Phone:               profile.Phone,
		Hotline:             profile.Hotline,
		ServiceHours:        profile.ServiceHours,
		LogoURL:             profile.LogoURL,
		CommitteeChairName:  profile.CommitteeChairName,
		CommitteeChairTitle: profile.CommitteeChairTitle,
		CommitteeChairNIP:   profile.CommitteeChairNIP,
		BankAccounts:        bankDTOs,
		UpdatedAt:           profile.UpdatedAt,
	}, nil
}

func (s *organizationService) UpdateInstitutionProfile(ctx context.Context, req dto.UpdateInstitutionProfileRequest) (*dto.InstitutionProfileResponse, error) {
	if s.institutionProfileRepo == nil {
		return nil, errors.New("repository institution profile belum diinisialisasi")
	}

	banks := make(model.BankAccounts, len(req.BankAccounts))
	for i, b := range req.BankAccounts {
		banks[i] = model.BankAccount{
			BankName:      b.BankName,
			AccountNumber: b.AccountNumber,
			AccountName:   b.AccountName,
			IsPrimary:     b.IsPrimary,
		}
	}

	profile := &model.InstitutionProfile{
		FoundationName:      strings.TrimSpace(req.FoundationName),
		InstitutionName:     strings.TrimSpace(req.InstitutionName),
		DocumentCode:        strings.TrimSpace(req.DocumentCode),
		Address:             strings.TrimSpace(req.Address),
		City:                strings.TrimSpace(req.City),
		Email:               strings.TrimSpace(req.Email),
		Phone:               strings.TrimSpace(req.Phone),
		Hotline:             strings.TrimSpace(req.Hotline),
		ServiceHours:        strings.TrimSpace(req.ServiceHours),
		LogoURL:             strings.TrimSpace(req.LogoURL),
		CommitteeChairName:  strings.TrimSpace(req.CommitteeChairName),
		CommitteeChairTitle: strings.TrimSpace(req.CommitteeChairTitle),
		CommitteeChairNIP:   strings.TrimSpace(req.CommitteeChairNIP),
		BankAccounts:        banks,
		UpdatedAt:           time.Now(),
	}

	if err := s.institutionProfileRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return s.GetInstitutionProfile(ctx)
}

func (s *organizationService) GetProgramsOverview(ctx context.Context, facultyFilter, query string, scopeProdi, scopeFaculty string) (*dto.ProgramsOverviewResponse, error) {
	allPrograms, err := s.studyProgramRepo.FindAll(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	allFaculties, err := s.facultyRepo.FindAll(ctx, nil)
	if err != nil {
		return nil, err
	}

	var activeFacultiesCount int64
	for _, f := range allFaculties {
		if f.IsActive {
			activeFacultiesCount++
		}
	}

	cleanFilter := strings.ToUpper(strings.TrimSpace(facultyFilter))
	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	cleanScopeProdi := strings.ToUpper(strings.TrimSpace(scopeProdi))
	cleanScopeFaculty := strings.ToUpper(strings.TrimSpace(scopeFaculty))

	var totalProgramsCount int64
	var openProgramsCount int64
	var items []dto.ProgramOverviewItem

	for _, p := range allPrograms {
		facCode := ""
		facName := ""
		if p.Faculty != nil {
			facCode = p.Faculty.Code
			facName = p.Faculty.Name
		}

		if p.IsActive {
			totalProgramsCount++
			if p.AdmissionOpen {
				openProgramsCount++
			}
		}

		// Scope filtering
		if cleanScopeProdi != "" && p.Code != cleanScopeProdi {
			continue
		}
		if cleanScopeFaculty != "" && facCode != cleanScopeFaculty {
			continue
		}

		// Parameter filtering
		if cleanFilter != "" && facCode != cleanFilter {
			continue
		}
		if cleanQuery != "" {
			matchCode := strings.Contains(strings.ToLower(p.Code), cleanQuery)
			matchName := strings.Contains(strings.ToLower(p.Name), cleanQuery)
			if !matchCode && !matchName {
				continue
			}
		}

		statusLabel := "Tutup"
		if p.AdmissionOpen {
			statusLabel = "Buka"
		}

		// Estimasi UKT per jenjang/prodi
		var estFee int64 = 5000000
		if p.Degree == "S2" {
			estFee = 7500000
		} else if strings.HasPrefix(p.Code, "T") || p.Degree == "S1" && (p.Code == "TI" || p.Code == "SI") {
			estFee = 6500000
		}

		items = append(items, dto.ProgramOverviewItem{
			ID:                       p.ID,
			Code:                     p.Code,
			NIMCode:                  p.NIMCode,
			Name:                     p.Name,
			FacultyCode:              facCode,
			FacultyName:              facName,
			Degree:                   p.Degree,
			Accreditation:            p.Accreditation,
			AdmissionOpen:            p.AdmissionOpen,
			AdmissionStatusLabel:     statusLabel,
			EstimatedTuitionFee:      estFee,
			EstimatedTuitionFeeLabel: fmt.Sprintf("Rp %s / semester", formatRupiahNumber(estFee)),
		})
	}

	return &dto.ProgramsOverviewResponse{
		ActiveFacultiesCount: activeFacultiesCount,
		TotalProgramsCount:   totalProgramsCount,
		OpenProgramsCount:    openProgramsCount,
		NIMFormat:            "TAHUN-FAKULTAS-PRODI-URUT",
		Programs:             items,
	}, nil
}

func formatRupiahNumber(n int64) string {
	in := fmt.Sprintf("%d", n)
	var out []byte
	l := len(in)
	for i, c := range in {
		if i > 0 && (l-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, byte(c))
	}
	return string(out)
}

func toStudyProgramResponse(sp *model.StudyProgram) dto.StudyProgramResponse {
	admissionLabel := "Tutup"
	if sp.AdmissionOpen {
		admissionLabel = "Buka"
	}
	res := dto.StudyProgramResponse{
		ID:                   sp.ID,
		FacultyID:            sp.FacultyID,
		Code:                 sp.Code,
		NIMCode:              sp.NIMCode,
		Name:                 sp.Name,
		Degree:               sp.Degree,
		Accreditation:        sp.Accreditation,
		IsActive:             sp.IsActive,
		AdmissionOpen:        sp.AdmissionOpen,
		AdmissionStatusLabel: admissionLabel,
		CreatedAt:            sp.CreatedAt,
		UpdatedAt:            sp.UpdatedAt,
	}
	if sp.Faculty != nil {
		res.FacultyCode = sp.Faculty.Code
		res.FacultyName = sp.Faculty.Name
	}
	return res
}
