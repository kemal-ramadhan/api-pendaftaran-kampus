package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/model"
	"gorm.io/gorm"
)

type StudentRepository interface {
	DB() *gorm.DB

	// Reregistration methods
	GetReregistrationByUserID(ctx context.Context, userID uuid.UUID) (*model.Reregistration, error)
	GetReregistrationByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Reregistration, error)
	CreateReregistration(ctx context.Context, r *model.Reregistration) error
	UpdateReregistration(ctx context.Context, r *model.Reregistration) error
	UpdateReregistrationInTx(tx *gorm.DB, r *model.Reregistration) error

	// Student methods
	CreateStudentInTx(tx *gorm.DB, s *model.Student) error
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*model.Student, error)
	GetStudentByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Student, error)
	ListStudents(ctx context.Context, filter model.StudentFilter) ([]model.Student, int64, error)
	UpdateStudentNIMInTx(tx *gorm.DB, appID uuid.UUID, nim string) error

	// Outbox methods
	CreateOutboxEventInTx(tx *gorm.DB, event *model.OutboxEvent) error
}

type studentRepository struct {
	db *gorm.DB
}

func NewStudentRepository(db *gorm.DB) StudentRepository {
	return &studentRepository{db: db}
}

func (r *studentRepository) DB() *gorm.DB {
	return r.db
}

func (r *studentRepository) GetReregistrationByUserID(ctx context.Context, userID uuid.UUID) (*model.Reregistration, error) {
	var item model.Reregistration
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *studentRepository) GetReregistrationByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Reregistration, error) {
	var item model.Reregistration
	err := r.db.WithContext(ctx).Where("application_id = ?", appID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *studentRepository) CreateReregistration(ctx context.Context, reg *model.Reregistration) error {
	return r.db.WithContext(ctx).Create(reg).Error
}

func (r *studentRepository) UpdateReregistration(ctx context.Context, reg *model.Reregistration) error {
	return r.db.WithContext(ctx).Save(reg).Error
}

func (r *studentRepository) UpdateReregistrationInTx(tx *gorm.DB, reg *model.Reregistration) error {
	return tx.Save(reg).Error
}

func (r *studentRepository) CreateStudentInTx(tx *gorm.DB, s *model.Student) error {
	return tx.Create(s).Error
}

func (r *studentRepository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*model.Student, error) {
	var s model.Student
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *studentRepository) GetStudentByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Student, error) {
	var s model.Student
	err := r.db.WithContext(ctx).Where("application_id = ?", appID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *studentRepository) ListStudents(ctx context.Context, filter model.StudentFilter) ([]model.Student, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Student{})

	if filter.StudyProgramCode != "" {
		query = query.Where("study_program_code = ?", filter.StudyProgramCode)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.EntryYear > 0 {
		query = query.Where("entry_year = ?", filter.EntryYear)
	}
	if filter.Search != "" {
		s := "%" + filter.Search + "%"
		query = query.Where("nim ILIKE ? OR user_id::text ILIKE ? OR application_id::text ILIKE ?", s, s, s)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var items []model.Student
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *studentRepository) UpdateStudentNIMInTx(tx *gorm.DB, appID uuid.UUID, nim string) error {
	return tx.Model(&model.Student{}).
		Where("application_id = ?", appID).
		Updates(map[string]interface{}{
			"nim":    nim,
			"status": model.StudentStatusActive,
		}).Error
}

func (r *studentRepository) CreateOutboxEventInTx(tx *gorm.DB, event *model.OutboxEvent) error {
	return tx.Create(event).Error
}
