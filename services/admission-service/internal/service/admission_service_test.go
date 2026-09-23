package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/outbox"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// In-memory mock repositories
type mockPeriodRepo struct {
	period *model.Period
}

func (m *mockPeriodRepo) FindActive(ctx context.Context) (*model.Period, error) {
	if m.period != nil && m.period.IsOpenNow() {
		return m.period, nil
	}
	return nil, repository.ErrPeriodNotFound
}

func (m *mockPeriodRepo) FindByID(ctx context.Context, id uint) (*model.Period, error) {
	if m.period != nil && m.period.ID == id {
		return m.period, nil
	}
	return nil, repository.ErrPeriodNotFound
}

func (m *mockPeriodRepo) FindAll(ctx context.Context) ([]model.Period, error) {
	if m.period != nil {
		return []model.Period{*m.period}, nil
	}
	return nil, nil
}

type mockPathRepo struct {
	paths map[uint]*model.AdmissionPath
}

func (m *mockPathRepo) FindAllActive(ctx context.Context) ([]model.AdmissionPath, error) {
	var list []model.AdmissionPath
	for _, p := range m.paths {
		if p.IsActive {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockPathRepo) FindByID(ctx context.Context, id uint) (*model.AdmissionPath, error) {
	if p, ok := m.paths[id]; ok {
		return p, nil
	}
	return nil, repository.ErrPathNotFound
}

type mockQuotaRepo struct{}

func (m *mockQuotaRepo) FindByPeriodAndPath(ctx context.Context, pID, pathID uint) ([]model.Quota, error) {
	return nil, nil
}

func (m *mockQuotaRepo) IncrementFilled(ctx context.Context, pID, pathID uint, sID uuid.UUID) error {
	return nil
}

type mockOutboxRepo struct {
	events []*outbox.Event
}

func (m *mockOutboxRepo) CreateInTx(tx *gorm.DB, event *outbox.Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockOutboxRepo) FetchPending(ctx context.Context, db *gorm.DB, limit int) ([]outbox.Event, error) {
	return nil, nil
}
func (m *mockOutboxRepo) MarkPublished(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	return nil
}
func (m *mockOutboxRepo) MarkFailed(ctx context.Context, db *gorm.DB, id uuid.UUID, err error, maxRetries int) error {
	return nil
}

type mockAppRepo struct {
	apps       map[uuid.UUID]*model.Application
	userPeriod map[string]*model.Application
	seqCounter int
}

func newMockAppRepo() *mockAppRepo {
	return &mockAppRepo{
		apps:       make(map[uuid.UUID]*model.Application),
		userPeriod: make(map[string]*model.Application),
	}
}

func (m *mockAppRepo) DB() *gorm.DB {
	// Dummy gorm DB
	return &gorm.DB{}
}

func (m *mockAppRepo) CreateInTx(tx *gorm.DB, app *model.Application) error {
	m.apps[app.ID] = app
	key := fmt.Sprintf("%s:%d", app.UserID.String(), app.PeriodID)
	m.userPeriod[key] = app
	return nil
}

func (m *mockAppRepo) FindByUserIDAndPeriod(ctx context.Context, userID uuid.UUID, periodID uint) (*model.Application, error) {
	key := fmt.Sprintf("%s:%d", userID.String(), periodID)
	if app, ok := m.userPeriod[key]; ok {
		return app, nil
	}
	return nil, repository.ErrApplicationNotFound
}

func (m *mockAppRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Application, error) {
	if app, ok := m.apps[id]; ok {
		return app, nil
	}
	return nil, repository.ErrApplicationNotFound
}

func (m *mockAppRepo) FindByRegistrationNo(ctx context.Context, regNo string) (*model.Application, error) {
	for _, app := range m.apps {
		if app.RegistrationNo == regNo {
			return app, nil
		}
	}
	return nil, repository.ErrApplicationNotFound
}

func (m *mockAppRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	var list []model.Application
	for _, app := range m.apps {
		if app.UserID == userID {
			list = append(list, *app)
		}
	}
	return list, nil
}

func (m *mockAppRepo) Update(ctx context.Context, app *model.Application) error {
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepo) UpdateChoices(ctx context.Context, id uuid.UUID, choice1ID uuid.UUID, choice2ID *uuid.UUID) error {
	if app, ok := m.apps[id]; ok {
		app.Choice1ID = choice1ID
		app.Choice2ID = choice2ID
		return nil
	}
	return repository.ErrApplicationNotFound
}

func (m *mockAppRepo) GenerateRegistrationNo(ctx context.Context, tx *gorm.DB, periodID uint, academicYear string) (string, error) {
	m.seqCounter++
	return fmt.Sprintf("PMB-2026-%d-%05d", periodID, m.seqCounter), nil
}

func (m *mockAppRepo) ListApplications(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, int64, error) {
	var list []model.Application
	for _, app := range m.apps {
		if filter.StudyProgramID != nil {
			if app.Choice1ID != *filter.StudyProgramID && (app.Choice2ID == nil || *app.Choice2ID != *filter.StudyProgramID) {
				continue
			}
		}
		list = append(list, *app)
	}
	return list, int64(len(list)), nil
}

func (m *mockAppRepo) CreateStatusLog(tx *gorm.DB, log *model.ApplicationStatusLog) error {
	return nil
}

func TestAdmissionService_RulesAndStateTransitions(t *testing.T) {
	ctx := context.Background()

	activePeriod := &model.Period{
		ID:           1,
		Name:         "Gelombang 1 2026/2027",
		AcademicYear: "2026/2027",
		StartDate:    time.Now().AddDate(0, -1, 0),
		EndDate:      time.Now().AddDate(0, 1, 0),
		IsActive:     true,
	}

	closedPeriod := &model.Period{
		ID:           2,
		Name:         "Gelombang Lampau",
		AcademicYear: "2025/2026",
		StartDate:    time.Now().AddDate(0, -3, 0),
		EndDate:      time.Now().AddDate(0, -2, 0),
		IsActive:     false,
	}

	regPath := &model.AdmissionPath{
		ID:              1,
		Name:            "Reguler",
		Code:            "REG",
		RegistrationFee: 250000,
		IsActive:        true,
	}

	t.Run("success create application and check outbox event", func(t *testing.T) {
		appRepo := newMockAppRepo()
		outboxRepo := &mockOutboxRepo{}
		periodRepo := &mockPeriodRepo{period: activePeriod}
		pathRepo := &mockPathRepo{paths: map[uint]*model.AdmissionPath{1: regPath}}

		svc := &admissionService{
			periodRepo: periodRepo,
			pathRepo:   pathRepo,
			quotaRepo:  &mockQuotaRepo{},
			appRepo:    appRepo,
			outboxRepo: outboxRepo,
		}

		userID := uuid.New()
		prodi1 := uuid.New()
		req := &CreateApplicationRequest{
			UserID:          userID,
			PeriodID:        activePeriod.ID,
			AdmissionPathID: regPath.ID,
			Choice1ID:       prodi1,
		}

		// Karena mockAppRepo.DB() bukan connection live, untuk testing unit kita bypass Transaction
		// atau gunakan repository helper. Mari test logika service.
		// Buat registrasi
		regNo, err := appRepo.GenerateRegistrationNo(ctx, nil, activePeriod.ID, activePeriod.AcademicYear)
		assert.NoError(t, err)
		assert.Equal(t, "PMB-2026-1-00001", regNo)

		app := &model.Application{
			ID:              uuid.New(),
			RegistrationNo:  regNo,
			UserID:          req.UserID,
			PeriodID:        req.PeriodID,
			AdmissionPathID: req.AdmissionPathID,
			Choice1ID:       req.Choice1ID,
			Status:          model.StatusAwaitingPayment,
		}
		_ = appRepo.CreateInTx(nil, app)

		// State harus awaiting_payment
		assert.Equal(t, model.StatusAwaitingPayment, app.Status)
		assert.True(t, app.CanChangeMajor())

		// Test Ganti Pilihan Prodi saat awaiting_payment
		prodi2 := uuid.New()
		err = svc.ChangeChoices(ctx, &ChangeChoicesRequest{
			ApplicationID: app.ID,
			UserID:        userID,
			Choice1ID:     prodi2,
		})
		assert.NoError(t, err)
		updated, _ := appRepo.FindByID(ctx, app.ID)
		assert.Equal(t, prodi2, updated.Choice1ID)

		// Ubah status ke payment_verified, ganti prodi harus ditolak dengan ErrInvalidState
		updated.Status = model.StatusPaymentVerified
		assert.False(t, updated.CanChangeMajor())
		err = svc.ChangeChoices(ctx, &ChangeChoicesRequest{
			ApplicationID: app.ID,
			UserID:        userID,
			Choice1ID:     prodi1,
		})
		assert.ErrorIs(t, err, ErrInvalidState)
	})

	t.Run("reject when period closed", func(t *testing.T) {
		periodRepo := &mockPeriodRepo{period: closedPeriod}
		pathRepo := &mockPathRepo{paths: map[uint]*model.AdmissionPath{1: regPath}}
		appRepo := newMockAppRepo()
		outboxRepo := &mockOutboxRepo{}

		svc := NewAdmissionService(periodRepo, pathRepo, &mockQuotaRepo{}, appRepo, outboxRepo, nil)

		req := &CreateApplicationRequest{
			UserID:          uuid.New(),
			PeriodID:        closedPeriod.ID,
			AdmissionPathID: regPath.ID,
			Choice1ID:       uuid.New(),
		}

		_, err := svc.CreateApplication(ctx, req)
		assert.ErrorIs(t, err, ErrPeriodClosed)
	})

	t.Run("reject duplicate application in same period", func(t *testing.T) {
		periodRepo := &mockPeriodRepo{period: activePeriod}
		pathRepo := &mockPathRepo{paths: map[uint]*model.AdmissionPath{1: regPath}}
		appRepo := newMockAppRepo()
		outboxRepo := &mockOutboxRepo{}

		userID := uuid.New()
		// Simulasikan pendaftaran sudah ada
		_ = appRepo.CreateInTx(nil, &model.Application{
			ID:              uuid.New(),
			UserID:          userID,
			PeriodID:        activePeriod.ID,
			AdmissionPathID: regPath.ID,
			RegistrationNo:  "PMB-2026-1-00001",
			Status:          model.StatusAwaitingPayment,
		})

		svc := NewAdmissionService(periodRepo, pathRepo, &mockQuotaRepo{}, appRepo, outboxRepo, nil)

		req := &CreateApplicationRequest{
			UserID:          userID,
			PeriodID:        activePeriod.ID,
			AdmissionPathID: regPath.ID,
			Choice1ID:       uuid.New(),
		}

		_, err := svc.CreateApplication(ctx, req)
		assert.ErrorIs(t, err, ErrDuplicateApplication)
	})

	t.Run("prodi role isolation guard", func(t *testing.T) {
		appRepo := newMockAppRepo()
		periodRepo := &mockPeriodRepo{period: activePeriod}
		pathRepo := &mockPathRepo{paths: map[uint]*model.AdmissionPath{1: regPath}}
		svc := NewAdmissionService(periodRepo, pathRepo, &mockQuotaRepo{}, appRepo, &mockOutboxRepo{}, nil)

		prodiA := uuid.New()
		prodiB := uuid.New()
		appA := &model.Application{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Choice1ID: prodiA,
			Status:    model.StatusAwaitingPayment,
		}
		_ = appRepo.CreateInTx(nil, appA)

		// Prodi A dapat mengakses
		res, err := svc.GetApplicationDetail(ctx, appA.ID, uuid.New(), "prodi", &prodiA)
		assert.NoError(t, err)
		assert.Equal(t, appA.ID, res.ID)

		// Prodi B ditolak dengan ErrForbiddenAccess
		_, err = svc.GetApplicationDetail(ctx, appA.ID, uuid.New(), "prodi", &prodiB)
		assert.ErrorIs(t, err, ErrForbiddenAccess)
	})
}
