package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OfficialDocumentRepository interface {
	Create(ctx context.Context, doc *model.OfficialDocument) error
	FindByNumber(ctx context.Context, docNumber string) (*model.OfficialDocument, error)
	GenerateOfficialNumber(ctx context.Context, docType string, periodID *uint, createdBy uuid.UUID, createdByName string, institutionCode string, title string) (*model.OfficialDocument, error)
}

type officialDocumentRepository struct {
	db *gorm.DB
	mu sync.Mutex
}

func NewOfficialDocumentRepository(db *gorm.DB) OfficialDocumentRepository {
	return &officialDocumentRepository{db: db}
}

func (r *officialDocumentRepository) Create(ctx context.Context, doc *model.OfficialDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *officialDocumentRepository) FindByNumber(ctx context.Context, docNumber string) (*model.OfficialDocument, error) {
	var doc model.OfficialDocument
	err := r.db.WithContext(ctx).Where("doc_number = ?", docNumber).First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func romanMonth(m time.Month) string {
	romans := [...]string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}
	if m >= 1 && m <= 12 {
		return romans[m-1]
	}
	return "I"
}

func (r *officialDocumentRepository) GenerateOfficialNumber(
	ctx context.Context,
	docType string,
	periodID *uint,
	createdBy uuid.UUID,
	createdByName string,
	institutionCode string,
	title string,
) (*model.OfficialDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	year := now.Year()
	monthRoman := romanMonth(now.Month())
	if institutionCode == "" {
		institutionCode = "UBTH"
	}

	var doc model.OfficialDocument

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lastDoc model.OfficialDocument
		// Lock baris terakhir pada tahun ini
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("year = ?", year).
			Order("sequence_number DESC").
			First(&lastDoc).Error

		nextSeq := 1
		if err == nil {
			nextSeq = lastDoc.SequenceNumber + 1
		}

		docNumber := fmt.Sprintf("%03d/BA-PMB/%s/%s/%d", nextSeq, institutionCode, monthRoman, year)

		doc = model.OfficialDocument{
			ID:             uuid.New(),
			PeriodID:       periodID,
			DocType:        docType,
			DocNumber:      docNumber,
			SequenceNumber: nextSeq,
			Year:           year,
			Title:          title,
			CreatedBy:      createdBy,
			CreatedByName:  createdByName,
			CreatedAt:      now,
		}

		return tx.Create(&doc).Error
	})

	if err != nil {
		return nil, err
	}

	return &doc, nil
}
