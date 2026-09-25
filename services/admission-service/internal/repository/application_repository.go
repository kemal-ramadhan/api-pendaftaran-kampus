package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrApplicationNotFound = errors.New("data pendaftaran tidak ditemukan")
)

type ApplicationFilter struct {
	PeriodID        *uint
	AdmissionPathID *uint
	Status          *string
	StudyProgramID  *uuid.UUID // Filter untuk role prodi (bisa choice_1 atau choice_2)
	Search          string
	Limit           int
	Offset          int
}

type ApplicationRepository interface {
	DB() *gorm.DB
	CreateInTx(tx *gorm.DB, app *model.Application) error
	FindByUserIDAndPeriod(ctx context.Context, userID uuid.UUID, periodID uint) (*model.Application, error)
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*model.Application, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Application, error)
	FindByRegistrationNo(ctx context.Context, regNo string) (*model.Application, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Application, error)
	Update(ctx context.Context, app *model.Application) error
	UpdateChoices(ctx context.Context, id uuid.UUID, choice1ID uuid.UUID, choice2ID *uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus model.ApplicationStatus, notes string, changedBy *uuid.UUID) error
	GenerateRegistrationNo(ctx context.Context, tx *gorm.DB, periodID uint, academicYear string) (string, error)
	ListApplications(ctx context.Context, filter ApplicationFilter) ([]model.Application, int64, error)
	ListAllForExport(ctx context.Context, filter ApplicationFilter) ([]model.Application, error)
	GetStats(ctx context.Context) (*model.AdmissionStats, error)
	CreateStatusLog(tx *gorm.DB, log *model.ApplicationStatusLog) error
}

type applicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) ApplicationRepository {
	return &applicationRepository{db: db}
}

func (r *applicationRepository) DB() *gorm.DB {
	return r.db
}

func (r *applicationRepository) CreateInTx(tx *gorm.DB, app *model.Application) error {
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	return tx.Create(app).Error
}

func (r *applicationRepository) FindByUserIDAndPeriod(ctx context.Context, userID uuid.UUID, periodID uint) (*model.Application, error) {
	var app model.Application
	err := r.db.WithContext(ctx).
		Preload("Period").
		Preload("AdmissionPath").
		Where("user_id = ? AND period_id = ? AND status NOT IN (?, ?, ?)", userID, periodID, model.StatusCancelled, model.StatusFailed, model.StatusExpired).
		First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *applicationRepository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*model.Application, error) {
	var app model.Application
	err := r.db.WithContext(ctx).
		Preload("Period").
		Preload("AdmissionPath").
		Preload("Documents").
		Preload("StatusLogs").
		Where("user_id = ? AND status NOT IN (?, ?, ?)", userID, model.StatusCancelled, model.StatusFailed, model.StatusExpired).
		First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *applicationRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Application, error) {
	var app model.Application
	err := r.db.WithContext(ctx).
		Preload("Period").
		Preload("AdmissionPath").
		Preload("Documents").
		Preload("StatusLogs").
		First(&app, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *applicationRepository) FindByRegistrationNo(ctx context.Context, regNo string) (*model.Application, error) {
	var app model.Application
	err := r.db.WithContext(ctx).
		Preload("Period").
		Preload("AdmissionPath").
		Preload("Documents").
		Preload("StatusLogs").
		First(&app, "registration_no = ?", regNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *applicationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	var apps []model.Application
	err := r.db.WithContext(ctx).
		Preload("Period").
		Preload("AdmissionPath").
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&apps).Error
	return apps, err
}

func (r *applicationRepository) Update(ctx context.Context, app *model.Application) error {
	return r.db.WithContext(ctx).Save(app).Error
}

func (r *applicationRepository) UpdateChoices(ctx context.Context, id uuid.UUID, choice1ID uuid.UUID, choice2ID *uuid.UUID) error {
	updates := map[string]interface{}{
		"choice_1_id": choice1ID,
		"choice_2_id": choice2ID,
		"updated_at":  time.Now(),
	}
	return r.db.WithContext(ctx).Model(&model.Application{}).Where("id = ?", id).Updates(updates).Error
}

func (r *applicationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus model.ApplicationStatus, notes string, changedBy *uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		updates := map[string]interface{}{
			"status":     toStatus,
			"updated_at": now,
		}
		if toStatus == model.StatusSubmitted {
			updates["submitted_at"] = &now
		} else if toStatus == model.StatusVerified {
			updates["verified_at"] = &now
			updates["verified_by"] = changedBy
		}

		if err := tx.Model(&model.Application{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}

		statusLog := &model.ApplicationStatusLog{
			ID:            uuid.New(),
			ApplicationID: id,
			FromStatus:    fromStatus,
			ToStatus:      toStatus,
			ChangedBy:     changedBy,
			Notes:         notes,
			CreatedAt:     now,
		}
		return tx.Create(statusLog).Error
	})
}

func (r *applicationRepository) GenerateRegistrationNo(ctx context.Context, tx *gorm.DB, periodID uint, academicYear string) (string, error) {
	var count int64
	dbRunner := r.db
	if tx != nil {
		dbRunner = tx
	}

	err := dbRunner.WithContext(ctx).Model(&model.Application{}).
		Where("period_id = ?", periodID).
		Count(&count).Error
	if err != nil {
		return "", err
	}

	yearStr := fmt.Sprintf("%d", time.Now().Year())
	if academicYear != "" {
		parts := strings.Split(academicYear, "/")
		if len(parts) > 0 && len(strings.TrimSpace(parts[0])) == 4 {
			yearStr = strings.TrimSpace(parts[0])
		}
	}

	sequence := count + 1
	regNo := fmt.Sprintf("PMB-%s-%d-%05d", yearStr, periodID, sequence)
	return regNo, nil
}

func (r *applicationRepository) ListApplications(ctx context.Context, filter ApplicationFilter) ([]model.Application, int64, error) {
	var apps []model.Application
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Application{})

	if filter.PeriodID != nil {
		q = q.Where("period_id = ?", *filter.PeriodID)
	}
	if filter.AdmissionPathID != nil {
		q = q.Where("admission_path_id = ?", *filter.AdmissionPathID)
	}
	if filter.Status != nil && *filter.Status != "" {
		q = q.Where("status = ?", *filter.Status)
	}
	if filter.StudyProgramID != nil {
		// Filter untuk staf prodi: harus menjadi choice_1 atau choice_2
		q = q.Where("(choice_1_id = ? OR choice_2_id = ?)", *filter.StudyProgramID, *filter.StudyProgramID)
	}
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		q = q.Where("registration_no ILIKE ?", searchPattern)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	err := q.Preload("Period").
		Preload("AdmissionPath").
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&apps).Error

	return apps, total, err
}

func (r *applicationRepository) ListAllForExport(ctx context.Context, filter ApplicationFilter) ([]model.Application, error) {
	var apps []model.Application
	q := r.db.WithContext(ctx).Model(&model.Application{})

	if filter.PeriodID != nil {
		q = q.Where("period_id = ?", *filter.PeriodID)
	}
	if filter.AdmissionPathID != nil {
		q = q.Where("admission_path_id = ?", *filter.AdmissionPathID)
	}
	if filter.Status != nil && *filter.Status != "" {
		q = q.Where("status = ?", *filter.Status)
	}
	if filter.StudyProgramID != nil {
		q = q.Where("(choice_1_id = ? OR choice_2_id = ?)", *filter.StudyProgramID, *filter.StudyProgramID)
	}
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		q = q.Where("registration_no ILIKE ?", searchPattern)
	}

	err := q.Preload("Period").
		Preload("AdmissionPath").
		Preload("Documents").
		Order("created_at desc").
		Find(&apps).Error

	return apps, err
}

func (r *applicationRepository) GetStats(ctx context.Context) (*model.AdmissionStats, error) {
	stats := &model.AdmissionStats{
		ByStatus:        make(map[string]int64),
		ByStudyProgram:  make(map[string]int64),
		ByAdmissionPath: make(map[string]int64),
		ByPeriod:        make(map[string]int64),
	}

	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Application{}).Count(&total).Error; err != nil {
		return nil, err
	}
	stats.TotalApplications = total

	type statusCount struct {
		Status string
		Count  int64
	}
	var sc []statusCount
	_ = r.db.WithContext(ctx).Model(&model.Application{}).
		Select("status, count(*) as count").
		Group("status").Scan(&sc).Error
	for _, s := range sc {
		stats.ByStatus[s.Status] = s.Count
	}

	type pathCount struct {
		AdmissionPathID uint
		Count           int64
	}
	var pc []pathCount
	_ = r.db.WithContext(ctx).Model(&model.Application{}).
		Select("admission_path_id, count(*) as count").
		Group("admission_path_id").Scan(&pc).Error
	for _, p := range pc {
		stats.ByAdmissionPath[fmt.Sprintf("%d", p.AdmissionPathID)] = p.Count
	}

	type periodCount struct {
		PeriodID uint
		Count    int64
	}
	var perc []periodCount
	_ = r.db.WithContext(ctx).Model(&model.Application{}).
		Select("period_id, count(*) as count").
		Group("period_id").Scan(&perc).Error
	for _, p := range perc {
		stats.ByPeriod[fmt.Sprintf("%d", p.PeriodID)] = p.Count
	}

	type prodiCount struct {
		Choice1ID uuid.UUID
		Count     int64
	}
	var proc []prodiCount
	_ = r.db.WithContext(ctx).Model(&model.Application{}).
		Select("choice_1_id, count(*) as count").
		Group("choice_1_id").Scan(&proc).Error
	for _, p := range proc {
		stats.ByStudyProgram[p.Choice1ID.String()] = p.Count
	}

	return stats, nil
}

func (r *applicationRepository) CreateStatusLog(tx *gorm.DB, log *model.ApplicationStatusLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return tx.Create(log).Error
}
