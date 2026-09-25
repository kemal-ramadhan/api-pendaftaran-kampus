package repository

import (
	"context"
	"errors"

	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrPathNotFound = errors.New("jalur pendaftaran tidak ditemukan")
)

type AdmissionPathRepository interface {
	FindAllActive(ctx context.Context) ([]model.AdmissionPath, error)
	FindByID(ctx context.Context, id uint) (*model.AdmissionPath, error)
	FindAll(ctx context.Context) ([]model.AdmissionPath, error)
	Create(ctx context.Context, p *model.AdmissionPath) error
	Update(ctx context.Context, p *model.AdmissionPath) error
}

type admissionPathRepository struct {
	db *gorm.DB
}

func NewAdmissionPathRepository(db *gorm.DB) AdmissionPathRepository {
	return &admissionPathRepository{db: db}
}

func (r *admissionPathRepository) FindAllActive(ctx context.Context) ([]model.AdmissionPath, error) {
	var paths []model.AdmissionPath
	err := r.db.WithContext(ctx).
		Preload("Requirements").
		Where("is_active = ?", true).
		Order("id asc").
		Find(&paths).Error
	return paths, err
}

func (r *admissionPathRepository) FindByID(ctx context.Context, id uint) (*model.AdmissionPath, error) {
	var path model.AdmissionPath
	err := r.db.WithContext(ctx).
		Preload("Requirements").
		First(&path, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPathNotFound
		}
		return nil, err
	}
	return &path, nil
}

func (r *admissionPathRepository) FindAll(ctx context.Context) ([]model.AdmissionPath, error) {
	var paths []model.AdmissionPath
	err := r.db.WithContext(ctx).
		Preload("Requirements").
		Order("id asc").
		Find(&paths).Error
	return paths, err
}

func (r *admissionPathRepository) Create(ctx context.Context, p *model.AdmissionPath) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *admissionPathRepository) Update(ctx context.Context, p *model.AdmissionPath) error {
	return r.db.WithContext(ctx).Save(p).Error
}
