package repository

import (
	"context"
	"errors"

	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/model"
	"gorm.io/gorm"
)

type StudyProgramRepository interface {
	FindAll(ctx context.Context, facultyCode string, isActive *bool) ([]model.StudyProgram, error)
	FindByID(ctx context.Context, id uint) (*model.StudyProgram, error)
	FindByCode(ctx context.Context, code string) (*model.StudyProgram, error)
	FindByCodes(ctx context.Context, codes []string) ([]model.StudyProgram, error)
	Create(ctx context.Context, sp *model.StudyProgram) error
	Update(ctx context.Context, sp *model.StudyProgram) error
	UpdateAdmissionOpen(ctx context.Context, code string, admissionOpen bool) error
	Delete(ctx context.Context, id uint) error
}

type studyProgramRepository struct {
	db *gorm.DB
}

func NewStudyProgramRepository(db *gorm.DB) StudyProgramRepository {
	return &studyProgramRepository{db: db}
}

func (r *studyProgramRepository) FindAll(ctx context.Context, facultyCode string, isActive *bool) ([]model.StudyProgram, error) {
	var programs []model.StudyProgram
	query := r.db.WithContext(ctx).Preload("Faculty").Order("code ASC")

	if facultyCode != "" {
		query = query.Joins("JOIN faculties ON faculties.id = study_programs.faculty_id").
			Where("faculties.code = ?", facultyCode)
	}
	if isActive != nil {
		query = query.Where("study_programs.is_active = ?", *isActive)
	}

	if err := query.Find(&programs).Error; err != nil {
		return nil, err
	}
	return programs, nil
}

func (r *studyProgramRepository) FindByID(ctx context.Context, id uint) (*model.StudyProgram, error) {
	var sp model.StudyProgram
	if err := r.db.WithContext(ctx).Preload("Faculty").First(&sp, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sp, nil
}

func (r *studyProgramRepository) FindByCode(ctx context.Context, code string) (*model.StudyProgram, error) {
	var sp model.StudyProgram
	if err := r.db.WithContext(ctx).Preload("Faculty").Where("code = ?", code).First(&sp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sp, nil
}

func (r *studyProgramRepository) FindByCodes(ctx context.Context, codes []string) ([]model.StudyProgram, error) {
	if len(codes) == 0 {
		return []model.StudyProgram{}, nil
	}
	var programs []model.StudyProgram
	if err := r.db.WithContext(ctx).Preload("Faculty").Where("code IN ?", codes).Order("code ASC").Find(&programs).Error; err != nil {
		return nil, err
	}
	return programs, nil
}

func (r *studyProgramRepository) Create(ctx context.Context, sp *model.StudyProgram) error {
	return r.db.WithContext(ctx).Create(sp).Error
}

func (r *studyProgramRepository) Update(ctx context.Context, sp *model.StudyProgram) error {
	return r.db.WithContext(ctx).Save(sp).Error
}

func (r *studyProgramRepository) UpdateAdmissionOpen(ctx context.Context, code string, admissionOpen bool) error {
	return r.db.WithContext(ctx).Model(&model.StudyProgram{}).Where("code = ?", code).Update("admission_open", admissionOpen).Error
}

func (r *studyProgramRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.StudyProgram{}, id).Error
}
