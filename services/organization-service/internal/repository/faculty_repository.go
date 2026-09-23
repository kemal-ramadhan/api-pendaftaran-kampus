package repository

import (
	"context"
	"errors"

	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/model"
	"gorm.io/gorm"
)

type FacultyRepository interface {
	FindAll(ctx context.Context, isActive *bool) ([]model.Faculty, error)
	FindByID(ctx context.Context, id uint) (*model.Faculty, error)
	FindByCode(ctx context.Context, code string) (*model.Faculty, error)
	Create(ctx context.Context, faculty *model.Faculty) error
	Update(ctx context.Context, faculty *model.Faculty) error
	Delete(ctx context.Context, id uint) error
}

type facultyRepository struct {
	db *gorm.DB
}

func NewFacultyRepository(db *gorm.DB) FacultyRepository {
	return &facultyRepository{db: db}
}

func (r *facultyRepository) FindAll(ctx context.Context, isActive *bool) ([]model.Faculty, error) {
	var faculties []model.Faculty
	query := r.db.WithContext(ctx).Order("code ASC")
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}
	if err := query.Find(&faculties).Error; err != nil {
		return nil, err
	}
	return faculties, nil
}

func (r *facultyRepository) FindByID(ctx context.Context, id uint) (*model.Faculty, error) {
	var faculty model.Faculty
	if err := r.db.WithContext(ctx).First(&faculty, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &faculty, nil
}

func (r *facultyRepository) FindByCode(ctx context.Context, code string) (*model.Faculty, error) {
	var faculty model.Faculty
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&faculty).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &faculty, nil
}

func (r *facultyRepository) Create(ctx context.Context, faculty *model.Faculty) error {
	return r.db.WithContext(ctx).Create(faculty).Error
}

func (r *facultyRepository) Update(ctx context.Context, faculty *model.Faculty) error {
	return r.db.WithContext(ctx).Save(faculty).Error
}

func (r *facultyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Faculty{}, id).Error
}
