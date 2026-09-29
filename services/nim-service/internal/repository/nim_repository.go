package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NIMRepository interface {
	DB() *gorm.DB

	// Issuance
	GetIssuanceByApplicationID(ctx context.Context, appID uuid.UUID) (*model.NIMIssuance, error)
	GetIssuanceByNIM(ctx context.Context, nim string) (*model.NIMIssuance, error)
	CreateIssuanceInTx(ctx context.Context, tx *gorm.DB, issuance *model.NIMIssuance) error
	ListIssuances(ctx context.Context, filter model.IssuanceFilter) ([]model.NIMIssuance, int64, error)
	VoidIssuance(ctx context.Context, appID uuid.UUID) (*model.NIMIssuance, error)

	// Concurrency & Sequence
	GetNextSequenceWithLock(ctx context.Context, tx *gorm.DB, entryYear int, studyProgramCode string) (int, error)

	// Formats
	GetFormatForProgram(ctx context.Context, studyProgramCode string) (*model.NIMFormat, error)
	ListFormats(ctx context.Context) ([]model.NIMFormat, error)
	ListActiveFormats(ctx context.Context) ([]model.NIMFormat, error)
	GetFormatByID(ctx context.Context, id uuid.UUID) (*model.NIMFormat, error)
	CreateFormat(ctx context.Context, format *model.NIMFormat) error
	UpdateFormat(ctx context.Context, format *model.NIMFormat) error
}

type nimRepository struct {
	db *gorm.DB
}

func NewNIMRepository(db *gorm.DB) NIMRepository {
	return &nimRepository{db: db}
}

func (r *nimRepository) DB() *gorm.DB {
	return r.db
}

func (r *nimRepository) GetIssuanceByApplicationID(ctx context.Context, appID uuid.UUID) (*model.NIMIssuance, error) {
	var issuance model.NIMIssuance
	err := r.db.WithContext(ctx).Where("application_id = ?", appID).First(&issuance).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &issuance, nil
}

func (r *nimRepository) GetIssuanceByNIM(ctx context.Context, nim string) (*model.NIMIssuance, error) {
	var issuance model.NIMIssuance
	err := r.db.WithContext(ctx).Where("nim = ?", nim).First(&issuance).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &issuance, nil
}

func (r *nimRepository) CreateIssuanceInTx(ctx context.Context, tx *gorm.DB, issuance *model.NIMIssuance) error {
	return tx.WithContext(ctx).Create(issuance).Error
}

func (r *nimRepository) ListIssuances(ctx context.Context, filter model.IssuanceFilter) ([]model.NIMIssuance, int64, error) {
	var issuances []model.NIMIssuance
	var total int64

	query := r.db.WithContext(ctx).Model(&model.NIMIssuance{})

	if filter.StudyProgramCode != "" {
		query = query.Where("study_program_code = ?", filter.StudyProgramCode)
	}
	if filter.EntryYear > 0 {
		query = query.Where("entry_year = ?", filter.EntryYear)
	}
	if filter.Search != "" {
		s := "%" + filter.Search + "%"
		query = query.Where("nim ILIKE ? OR selection_path ILIKE ?", s, s)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	err := query.Order("issued_at DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&issuances).Error

	return issuances, total, err
}

func (r *nimRepository) VoidIssuance(ctx context.Context, appID uuid.UUID) (*model.NIMIssuance, error) {
	var issuance model.NIMIssuance
	err := r.db.WithContext(ctx).Where("application_id = ?", appID).First(&issuance).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	issuance.Status = model.NIMIssuanceStatusVoided
	issuance.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Save(&issuance).Error; err != nil {
		return nil, err
	}
	return &issuance, nil
}

func (r *nimRepository) GetNextSequenceWithLock(ctx context.Context, tx *gorm.DB, entryYear int, studyProgramCode string) (int, error) {
	var seq model.NIMSequence

	// SELECT ... FOR UPDATE untuk mengunci baris sequence
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("entry_year = ? AND study_program_code = ?", entryYear, studyProgramCode).
		First(&seq).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Jika belum ada record sequence untuk tahun & prodi ini, buat record baru dengan nomor urut 1
			newSeq := model.NIMSequence{
				ID:               uuid.New(),
				EntryYear:        entryYear,
				StudyProgramCode: studyProgramCode,
				LastNumber:       1,
			}
			if errInsert := tx.WithContext(ctx).Create(&newSeq).Error; errInsert != nil {
				return 0, errInsert
			}
			return 1, nil
		}
		return 0, err
	}

	// Increment sequence number
	nextNumber := seq.LastNumber + 1
	if errUpdate := tx.WithContext(ctx).
		Model(&seq).
		Update("last_number", nextNumber).Error; errUpdate != nil {
		return 0, errUpdate
	}

	return nextNumber, nil
}

func (r *nimRepository) GetFormatForProgram(ctx context.Context, studyProgramCode string) (*model.NIMFormat, error) {
	var fmt model.NIMFormat

	// 1. Coba cari format spesifik program studi yang aktif
	if studyProgramCode != "" {
		err := r.db.WithContext(ctx).
			Where("study_program_code = ? AND is_active = ?", studyProgramCode, true).
			First(&fmt).Error
		if err == nil {
			return &fmt, nil
		}
	}

	// 2. Fallback ke format global (study_program_code IS NULL) yang aktif
	err := r.db.WithContext(ctx).
		Where("study_program_code IS NULL AND is_active = ?", true).
		First(&fmt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &fmt, nil
}

func (r *nimRepository) ListFormats(ctx context.Context) ([]model.NIMFormat, error) {
	var formats []model.NIMFormat
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&formats).Error
	return formats, err
}

func (r *nimRepository) ListActiveFormats(ctx context.Context) ([]model.NIMFormat, error) {
	var formats []model.NIMFormat
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("study_program_code ASC NULLS FIRST, created_at DESC").Find(&formats).Error
	return formats, err
}

func (r *nimRepository) GetFormatByID(ctx context.Context, id uuid.UUID) (*model.NIMFormat, error) {
	var fmt model.NIMFormat
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&fmt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &fmt, nil
}

func (r *nimRepository) CreateFormat(ctx context.Context, format *model.NIMFormat) error {
	return r.db.WithContext(ctx).Create(format).Error
}

func (r *nimRepository) UpdateFormat(ctx context.Context, format *model.NIMFormat) error {
	return r.db.WithContext(ctx).Save(format).Error
}
