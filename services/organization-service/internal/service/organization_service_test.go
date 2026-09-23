package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/service"
)

type mockFacultyRepo struct {
	faculties []model.Faculty
}

func (m *mockFacultyRepo) FindAll(ctx context.Context, isActive *bool) ([]model.Faculty, error) {
	var res []model.Faculty
	for _, f := range m.faculties {
		if isActive == nil || f.IsActive == *isActive {
			res = append(res, f)
		}
	}
	return res, nil
}

func (m *mockFacultyRepo) FindByID(ctx context.Context, id uint) (*model.Faculty, error) {
	for _, f := range m.faculties {
		if f.ID == id {
			return &f, nil
		}
	}
	return nil, nil
}

func (m *mockFacultyRepo) FindByCode(ctx context.Context, code string) (*model.Faculty, error) {
	for _, f := range m.faculties {
		if f.Code == code {
			return &f, nil
		}
	}
	return nil, nil
}

func (m *mockFacultyRepo) Create(ctx context.Context, f *model.Faculty) error {
	f.ID = uint(len(m.faculties) + 1)
	m.faculties = append(m.faculties, *f)
	return nil
}

func (m *mockFacultyRepo) Update(ctx context.Context, f *model.Faculty) error {
	for i, existing := range m.faculties {
		if existing.ID == f.ID {
			m.faculties[i] = *f
			return nil
		}
	}
	return nil
}

func (m *mockFacultyRepo) Delete(ctx context.Context, id uint) error {
	for i, existing := range m.faculties {
		if existing.ID == id {
			m.faculties = append(m.faculties[:i], m.faculties[i+1:]...)
			return nil
		}
	}
	return nil
}

type mockStudyProgramRepo struct {
	programs []model.StudyProgram
}

func (m *mockStudyProgramRepo) FindAll(ctx context.Context, facultyCode string, isActive *bool) ([]model.StudyProgram, error) {
	var res []model.StudyProgram
	for _, p := range m.programs {
		if facultyCode != "" && p.Faculty != nil && p.Faculty.Code != facultyCode {
			continue
		}
		if isActive != nil && p.IsActive != *isActive {
			continue
		}
		res = append(res, p)
	}
	return res, nil
}

func (m *mockStudyProgramRepo) FindByID(ctx context.Context, id uint) (*model.StudyProgram, error) {
	for _, p := range m.programs {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *mockStudyProgramRepo) FindByCode(ctx context.Context, code string) (*model.StudyProgram, error) {
	for _, p := range m.programs {
		if p.Code == code {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *mockStudyProgramRepo) FindByCodes(ctx context.Context, codes []string) ([]model.StudyProgram, error) {
	codeMap := make(map[string]bool)
	for _, c := range codes {
		codeMap[c] = true
	}
	var res []model.StudyProgram
	for _, p := range m.programs {
		if codeMap[p.Code] {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockStudyProgramRepo) Create(ctx context.Context, sp *model.StudyProgram) error {
	sp.ID = uint(len(m.programs) + 1)
	m.programs = append(m.programs, *sp)
	return nil
}

func (m *mockStudyProgramRepo) Update(ctx context.Context, sp *model.StudyProgram) error {
	for i, existing := range m.programs {
		if existing.ID == sp.ID {
			m.programs[i] = *sp
			return nil
		}
	}
	return nil
}

func (m *mockStudyProgramRepo) Delete(ctx context.Context, id uint) error {
	for i, existing := range m.programs {
		if existing.ID == id {
			m.programs = append(m.programs[:i], m.programs[i+1:]...)
			return nil
		}
	}
	return nil
}

func setupTestService() (service.OrganizationService, *mockFacultyRepo, *mockStudyProgramRepo) {
	now := time.Now()
	f1 := model.Faculty{ID: 1, Code: "FT", Name: "Fakultas Teknik", IsActive: true, CreatedAt: now, UpdatedAt: now}
	f2 := model.Faculty{ID: 2, Code: "FEB", Name: "Fakultas Ekonomi dan Bisnis", IsActive: true, CreatedAt: now, UpdatedAt: now}

	p1 := model.StudyProgram{ID: 1, FacultyID: 1, Faculty: &f1, Code: "TI", NIMCode: "55", Name: "Teknik Informatika", Degree: "S1", Accreditation: "Unggul", IsActive: true}
	p2 := model.StudyProgram{ID: 2, FacultyID: 1, Faculty: &f1, Code: "SI", NIMCode: "57", Name: "Sistem Informasi", Degree: "S1", Accreditation: "Baik Sekali", IsActive: true}

	facultyRepo := &mockFacultyRepo{faculties: []model.Faculty{f1, f2}}
	spRepo := &mockStudyProgramRepo{programs: []model.StudyProgram{p1, p2}}

	svc := service.NewOrganizationService(facultyRepo, spRepo)
	return svc, facultyRepo, spRepo
}

func TestListFaculties(t *testing.T) {
	svc, _, _ := setupTestService()
	res, err := svc.ListFaculties(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 faculties, got %d", len(res))
	}
}

func TestGetStudyProgramByCode(t *testing.T) {
	svc, _, _ := setupTestService()

	// Kasus Berhasil
	sp, err := svc.GetStudyProgramByCode(context.Background(), "TI")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sp.Code != "TI" || sp.NIMCode != "55" {
		t.Errorf("expected TI/55, got %s/%s", sp.Code, sp.NIMCode)
	}

	// Kasus Tidak Ditemukan
	_, err = svc.GetStudyProgramByCode(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateStudyProgram(t *testing.T) {
	svc, _, _ := setupTestService()

	req := dto.CreateStudyProgramRequest{
		FacultyID:     1,
		Code:          "TE",
		NIMCode:       "58",
		Name:          "Teknik Elektro",
		Degree:        "S1",
		Accreditation: "Unggul",
	}

	created, err := svc.CreateStudyProgram(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Code != "TE" || created.ID == 0 {
		t.Errorf("expected created TE, got %+v", created)
	}

	// Duplikat kode prodi
	_, err = svc.CreateStudyProgram(context.Background(), req)
	if err == nil {
		t.Fatal("expected duplicate code error, got nil")
	}
}

func TestGetStudyProgramsByCodes_Batch(t *testing.T) {
	svc, _, _ := setupTestService()

	results, err := svc.GetStudyProgramsByCodes(context.Background(), []string{"TI", "SI", "NOTFOUND"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 matching programs, got %d", len(results))
	}
}
