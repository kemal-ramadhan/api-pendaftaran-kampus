package repository

import (
	"context"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"gorm.io/gorm"
)

type DocumentRequirementRepository interface {
	FindAll(ctx context.Context) ([]model.DocumentRequirement, error)
	FindByPathID(ctx context.Context, pathID uint) ([]model.DocumentRequirement, error)
	Create(ctx context.Context, req *model.DocumentRequirement) error
}

type documentRequirementRepository struct {
	db *gorm.DB
}

func NewDocumentRequirementRepository(db *gorm.DB) DocumentRequirementRepository {
	return &documentRequirementRepository{db: db}
}

func (r *documentRequirementRepository) FindAll(ctx context.Context) ([]model.DocumentRequirement, error) {
	var list []model.DocumentRequirement
	err := r.db.WithContext(ctx).Order("admission_path_id asc, document_type asc").Find(&list).Error
	return list, err
}

func (r *documentRequirementRepository) FindByPathID(ctx context.Context, pathID uint) ([]model.DocumentRequirement, error) {
	var list []model.DocumentRequirement
	err := r.db.WithContext(ctx).Where("admission_path_id = ?", pathID).Find(&list).Error
	return list, err
}

func (r *documentRequirementRepository) Create(ctx context.Context, req *model.DocumentRequirement) error {
	return r.db.WithContext(ctx).Create(req).Error
}
