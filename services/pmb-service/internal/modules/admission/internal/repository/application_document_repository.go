package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"gorm.io/gorm"
)

var (
	ErrDocumentNotFound = errors.New("dokumen pendaftaran tidak ditemukan")
)

type ApplicationDocumentRepository interface {
	Save(ctx context.Context, doc *model.ApplicationDocument) error
	FindByApplicationAndType(ctx context.Context, appID uuid.UUID, docType string) (*model.ApplicationDocument, error)
	FindByApplicationID(ctx context.Context, appID uuid.UUID) ([]model.ApplicationDocument, error)
	DeleteByApplicationAndType(ctx context.Context, appID uuid.UUID, docType string) error
	UpdateReviewStatus(ctx context.Context, id uuid.UUID, status string, notes string, reviewerID *uuid.UUID) error
}

type applicationDocumentRepository struct {
	db *gorm.DB
}

func NewApplicationDocumentRepository(db *gorm.DB) ApplicationDocumentRepository {
	return &applicationDocumentRepository{db: db}
}

func (r *applicationDocumentRepository) Save(ctx context.Context, doc *model.ApplicationDocument) error {
	var existing model.ApplicationDocument
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND document_type = ?", doc.ApplicationID, doc.DocumentType).
		First(&existing).Error

	if err == nil {
		// Update existing
		return r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"file_name":   doc.FileName,
			"file_url":    doc.FileURL,
			"object_key":  doc.ObjectKey,
			"mime_type":   doc.MimeType,
			"file_size":   doc.FileSize,
			"status":      model.DocStatusUploaded,
			"notes":       "",
			"verified_by": nil,
			"verified_at": nil,
			"updated_at":  time.Now(),
		}).Error
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if doc.ID == uuid.Nil {
			doc.ID = uuid.New()
		}
		doc.Status = model.DocStatusUploaded
		return r.db.WithContext(ctx).Create(doc).Error
	}

	return err
}

func (r *applicationDocumentRepository) FindByApplicationAndType(ctx context.Context, appID uuid.UUID, docType string) (*model.ApplicationDocument, error) {
	var doc model.ApplicationDocument
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND LOWER(document_type) = LOWER(?)", appID, docType).
		First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	return &doc, nil
}

func (r *applicationDocumentRepository) FindByApplicationID(ctx context.Context, appID uuid.UUID) ([]model.ApplicationDocument, error) {
	var docs []model.ApplicationDocument
	err := r.db.WithContext(ctx).
		Where("application_id = ?", appID).
		Find(&docs).Error
	return docs, err
}

func (r *applicationDocumentRepository) DeleteByApplicationAndType(ctx context.Context, appID uuid.UUID, docType string) error {
	res := r.db.WithContext(ctx).
		Where("application_id = ? AND LOWER(document_type) = LOWER(?)", appID, docType).
		Delete(&model.ApplicationDocument{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

func (r *applicationDocumentRepository) UpdateReviewStatus(ctx context.Context, id uuid.UUID, status string, notes string, reviewerID *uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.ApplicationDocument{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      status,
		"notes":       notes,
		"verified_by": reviewerID,
		"verified_at": &now,
		"updated_at":  now,
	}).Error
}
