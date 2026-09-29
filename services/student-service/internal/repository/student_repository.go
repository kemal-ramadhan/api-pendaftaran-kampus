package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/model"
	"gorm.io/gorm"
)

type StudentRepository interface {
	DB() *gorm.DB

	// Reregistration methods
	GetReregistrationByUserID(ctx context.Context, userID uuid.UUID) (*model.Reregistration, error)
	GetReregistrationByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Reregistration, error)
	ListReregistrations(ctx context.Context, status string) ([]model.Reregistration, error)
	CreateReregistration(ctx context.Context, r *model.Reregistration) error
	UpdateReregistration(ctx context.Context, r *model.Reregistration) error
	UpdateReregistrationInTx(tx *gorm.DB, r *model.Reregistration) error

	// Student methods
	CreateStudentInTx(tx *gorm.DB, s *model.Student) error
	GetStudentByID(ctx context.Context, id uuid.UUID) (*model.Student, error)
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*model.Student, error)
	GetStudentByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Student, error)
	ListStudents(ctx context.Context, filter model.StudentFilter) ([]model.Student, int64, error)
	ListStudentsReadyForNIM(ctx context.Context, prodi string) ([]model.Student, error)
	ListStudentsWithNIM(ctx context.Context, prodi string) ([]model.Student, error)
	UpdateStudent(ctx context.Context, s *model.Student) error
	UpdateStudentNIMInTx(tx *gorm.DB, appID uuid.UUID, nim string) error
	MarkSiakadExported(ctx context.Context, studentIDs []uuid.UUID) error

	// PKKMB methods
	ListPKKMBGroups(ctx context.Context, periodID *uuid.UUID) ([]model.PKKMBGroup, error)
	GetPKKMBGroupByID(ctx context.Context, id uuid.UUID) (*model.PKKMBGroup, error)
	CreatePKKMBGroup(ctx context.Context, group *model.PKKMBGroup) error
	UpdatePKKMBGroup(ctx context.Context, group *model.PKKMBGroup) error
	DeletePKKMBGroup(ctx context.Context, id uuid.UUID) error
	GetPKKMBAssignmentByStudentID(ctx context.Context, studentID uuid.UUID) (*model.PKKMBAssignment, error)
	CreateOrUpdatePKKMBAssignment(ctx context.Context, assignment *model.PKKMBAssignment) error
	ListPKKMBAssignmentsByGroupID(ctx context.Context, groupID uuid.UUID) ([]model.PKKMBAssignment, error)
	ListAllPKKMBAssignments(ctx context.Context) ([]model.PKKMBAssignment, error)
	CountAssignedPerGroup(ctx context.Context) (map[uuid.UUID]int, error)

	// Outbox methods
	CreateOutboxEventInTx(tx *gorm.DB, event *model.OutboxEvent) error
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	UpdateOutboxEvent(ctx context.Context, event *model.OutboxEvent) error
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

func (r *studentRepository) ListReregistrations(ctx context.Context, status string) ([]model.Reregistration, error) {
	var items []model.Reregistration
	query := r.db.WithContext(ctx).Model(&model.Reregistration{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *studentRepository) GetStudentByID(ctx context.Context, id uuid.UUID) (*model.Student, error) {
	var s model.Student
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *studentRepository) UpdateStudent(ctx context.Context, s *model.Student) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *studentRepository) ListStudentsReadyForNIM(ctx context.Context, prodi string) ([]model.Student, error) {
	var items []model.Student
	query := r.db.WithContext(ctx).Model(&model.Student{}).
		Where("(nim IS NULL OR nim = '') AND status != ?", model.StudentStatusWithdrawn)
	if prodi != "" {
		query = query.Where("study_program_code = ?", prodi)
	}
	err := query.Order("created_at ASC").Find(&items).Error
	return items, err
}

func (r *studentRepository) ListStudentsWithNIM(ctx context.Context, prodi string) ([]model.Student, error) {
	var items []model.Student
	query := r.db.WithContext(ctx).Model(&model.Student{}).
		Where("nim IS NOT NULL AND nim != '' AND status != ?", model.StudentStatusWithdrawn)
	if prodi != "" {
		query = query.Where("study_program_code = ?", prodi)
	}
	err := query.Order("study_program_code ASC, nim ASC").Find(&items).Error
	return items, err
}

func (r *studentRepository) MarkSiakadExported(ctx context.Context, studentIDs []uuid.UUID) error {
	if len(studentIDs) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Student{}).
		Where("id IN ?", studentIDs).
		Update("siakad_exported_at", &now).Error
}

// -----------------------------------------------------------------------------
// PKKMB Repository Methods
// -----------------------------------------------------------------------------

func (r *studentRepository) ListPKKMBGroups(ctx context.Context, periodID *uuid.UUID) ([]model.PKKMBGroup, error) {
	var groups []model.PKKMBGroup
	query := r.db.WithContext(ctx).Model(&model.PKKMBGroup{})
	if periodID != nil && *periodID != uuid.Nil {
		query = query.Where("period_id = ?", *periodID)
	}
	err := query.Order("name ASC").Find(&groups).Error
	return groups, err
}

func (r *studentRepository) GetPKKMBGroupByID(ctx context.Context, id uuid.UUID) (*model.PKKMBGroup, error) {
	var g model.PKKMBGroup
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&g).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func (r *studentRepository) CreatePKKMBGroup(ctx context.Context, group *model.PKKMBGroup) error {
	return r.db.WithContext(ctx).Create(group).Error
}

func (r *studentRepository) UpdatePKKMBGroup(ctx context.Context, group *model.PKKMBGroup) error {
	return r.db.WithContext(ctx).Save(group).Error
}

func (r *studentRepository) DeletePKKMBGroup(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.PKKMBGroup{}).Error
}

func (r *studentRepository) GetPKKMBAssignmentByStudentID(ctx context.Context, studentID uuid.UUID) (*model.PKKMBAssignment, error) {
	var a model.PKKMBAssignment
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *studentRepository) CreateOrUpdatePKKMBAssignment(ctx context.Context, assignment *model.PKKMBAssignment) error {
	var existing model.PKKMBAssignment
	err := r.db.WithContext(ctx).Where("student_id = ?", assignment.StudentID).First(&existing).Error
	if err == nil {
		existing.GroupID = assignment.GroupID
		existing.Method = assignment.Method
		existing.AssignedBy = assignment.AssignedBy
		existing.UpdatedAt = time.Now()
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if assignment.ID == uuid.Nil {
			assignment.ID = uuid.New()
		}
		now := time.Now()
		assignment.CreatedAt = now
		assignment.UpdatedAt = now
		return r.db.WithContext(ctx).Create(assignment).Error
	}
	return err
}

func (r *studentRepository) ListPKKMBAssignmentsByGroupID(ctx context.Context, groupID uuid.UUID) ([]model.PKKMBAssignment, error) {
	var assignments []model.PKKMBAssignment
	err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&assignments).Error
	return assignments, err
}

func (r *studentRepository) ListAllPKKMBAssignments(ctx context.Context) ([]model.PKKMBAssignment, error) {
	var assignments []model.PKKMBAssignment
	err := r.db.WithContext(ctx).Find(&assignments).Error
	return assignments, err
}

func (r *studentRepository) CountAssignedPerGroup(ctx context.Context) (map[uuid.UUID]int, error) {
	type countRow struct {
		GroupID uuid.UUID `gorm:"column:group_id"`
		Count   int       `gorm:"column:count"`
	}
	var rows []countRow
	err := r.db.WithContext(ctx).Model(&model.PKKMBAssignment{}).
		Select("group_id, count(1) as count").
		Group("group_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	res := make(map[uuid.UUID]int)
	for _, row := range rows {
		res[row.GroupID] = row.Count
	}
	return res, nil
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

func (r *studentRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	var events []model.OutboxEvent
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *studentRepository) UpdateOutboxEvent(ctx context.Context, event *model.OutboxEvent) error {
	return r.db.WithContext(ctx).Save(event).Error
}
