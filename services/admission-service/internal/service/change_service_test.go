package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Mock Change Repo
type mockChangeRepo struct {
	changes  map[uuid.UUID]*model.ApplicationChange
	requests map[uuid.UUID]*model.PathChangeRequest
}

func newMockChangeRepo() *mockChangeRepo {
	return &mockChangeRepo{
		changes:  make(map[uuid.UUID]*model.ApplicationChange),
		requests: make(map[uuid.UUID]*model.PathChangeRequest),
	}
}

func (m *mockChangeRepo) CreateChange(ctx context.Context, tx *gorm.DB, change *model.ApplicationChange) error {
	if change.ID == uuid.Nil {
		change.ID = uuid.New()
	}
	change.CreatedAt = time.Now()
	m.changes[change.ID] = change
	return nil
}

func (m *mockChangeRepo) GetChangesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error) {
	var list []model.ApplicationChange
	for _, c := range m.changes {
		if c.UserID == userID {
			list = append(list, *c)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockChangeRepo) GetChangesByApplicationID(ctx context.Context, appID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error) {
	var list []model.ApplicationChange
	for _, c := range m.changes {
		if c.ApplicationID == appID {
			list = append(list, *c)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockChangeRepo) CreatePathChangeRequest(ctx context.Context, tx *gorm.DB, req *model.PathChangeRequest) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	req.CreatedAt = time.Now()
	m.requests[req.ID] = req
	return nil
}

func (m *mockChangeRepo) GetPathChangeRequestByID(ctx context.Context, id uuid.UUID) (*model.PathChangeRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return nil, nil
	}
	return req, nil
}

func (m *mockChangeRepo) GetPendingPathChangeRequest(ctx context.Context, appID uuid.UUID) (*model.PathChangeRequest, error) {
	for _, req := range m.requests {
		if req.ApplicationID == appID && req.Status == model.PathChangePending {
			return req, nil
		}
	}
	return nil, nil
}

func (m *mockChangeRepo) UpdatePathChangeRequest(ctx context.Context, tx *gorm.DB, req *model.PathChangeRequest) error {
	m.requests[req.ID] = req
	return nil
}

func (m *mockChangeRepo) ListPathChangeRequests(ctx context.Context, status string, limit, offset int) ([]model.PathChangeRequest, int64, error) {
	var list []model.PathChangeRequest
	for _, r := range m.requests {
		if status == "" || string(r.Status) == status {
			list = append(list, *r)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockChangeRepo) InitApplicationChangesTriggersAndBackfill(db *gorm.DB) error {
	return nil
}

// Mock Path Repo
type mockAdmissionPathRepo struct {
	paths map[uint]*model.AdmissionPath
}

func newMockAdmissionPathRepo() *mockAdmissionPathRepo {
	return &mockAdmissionPathRepo{
		paths: map[uint]*model.AdmissionPath{
			1: {ID: 1, Code: "REGULER", Name: "Jalur Reguler", RegistrationFee: 250000, IsActive: true},
			2: {ID: 2, Code: "PRESTASI", Name: "Jalur Prestasi", RegistrationFee: 300000, IsActive: true},
			3: {ID: 3, Code: "BEASISWA", Name: "Jalur Beasiswa", RegistrationFee: 200000, IsActive: true},
		},
	}
}

func (m *mockAdmissionPathRepo) FindAllActive(ctx context.Context) ([]model.AdmissionPath, error) {
	var list []model.AdmissionPath
	for _, p := range m.paths {
		if p.IsActive {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockAdmissionPathRepo) FindByID(ctx context.Context, id uint) (*model.AdmissionPath, error) {
	p, ok := m.paths[id]
	if !ok {
		return nil, repository.ErrPathNotFound
	}
	return p, nil
}

func (m *mockAdmissionPathRepo) FindAll(ctx context.Context) ([]model.AdmissionPath, error) {
	var list []model.AdmissionPath
	for _, p := range m.paths {
		list = append(list, *p)
	}
	return list, nil
}

func (m *mockAdmissionPathRepo) Create(ctx context.Context, p *model.AdmissionPath) error { return nil }
func (m *mockAdmissionPathRepo) Update(ctx context.Context, p *model.AdmissionPath) error { return nil }

// Mock App Repo for Change Test
type mockAppRepoForChange struct {
	apps map[uuid.UUID]*model.Application
}

func newMockAppRepoForChange() *mockAppRepoForChange {
	return &mockAppRepoForChange{
		apps: make(map[uuid.UUID]*model.Application),
	}
}

func (m *mockAppRepoForChange) Create(ctx context.Context, a *model.Application) error {
	m.apps[a.ID] = a
	return nil
}

func (m *mockAppRepoForChange) FindByID(ctx context.Context, id uuid.UUID) (*model.Application, error) {
	app, ok := m.apps[id]
	if !ok {
		return nil, repository.ErrApplicationNotFound
	}
	return app, nil
}

func (m *mockAppRepoForChange) FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*model.Application, error) {
	for _, app := range m.apps {
		if app.UserID == userID && app.IsActive() {
			return app, nil
		}
	}
	return nil, repository.ErrApplicationNotFound
}

func (m *mockAppRepoForChange) Update(ctx context.Context, a *model.Application) error {
	m.apps[a.ID] = a
	return nil
}

func (m *mockAppRepoForChange) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	var list []model.Application
	for _, app := range m.apps {
		if app.UserID == userID {
			list = append(list, *app)
		}
	}
	return list, nil
}

func (m *mockAppRepoForChange) UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus model.ApplicationStatus, notes string, changedBy *uuid.UUID) error {
	app, ok := m.apps[id]
	if !ok {
		return repository.ErrApplicationNotFound
	}
	app.Status = toStatus
	return nil
}

func (m *mockAppRepoForChange) List(ctx context.Context, f repository.ApplicationFilter) ([]model.Application, int64, error) {
	var list []model.Application
	for _, app := range m.apps {
		list = append(list, *app)
	}
	return list, int64(len(list)), nil
}

func (m *mockAppRepoForChange) DB() *gorm.DB { return nil }
func (m *mockAppRepoForChange) CreateInTx(tx *gorm.DB, app *model.Application) error {
	m.apps[app.ID] = app
	return nil
}
func (m *mockAppRepoForChange) FindByUserIDAndPeriod(ctx context.Context, userID uuid.UUID, periodID uint) (*model.Application, error) {
	for _, a := range m.apps {
		if a.UserID == userID && a.PeriodID == periodID {
			return a, nil
		}
	}
	return nil, repository.ErrApplicationNotFound
}
func (m *mockAppRepoForChange) FindByRegistrationNo(ctx context.Context, regNo string) (*model.Application, error) {
	for _, a := range m.apps {
		if a.RegistrationNo == regNo {
			return a, nil
		}
	}
	return nil, repository.ErrApplicationNotFound
}
func (m *mockAppRepoForChange) UpdateChoices(ctx context.Context, id uuid.UUID, choice1ID uuid.UUID, choice2ID *uuid.UUID) error {
	if a, ok := m.apps[id]; ok {
		a.Choice1ID = choice1ID
		a.Choice2ID = choice2ID
		return nil
	}
	return repository.ErrApplicationNotFound
}
func (m *mockAppRepoForChange) GenerateRegistrationNo(ctx context.Context, tx *gorm.DB, periodID uint, academicYear string) (string, error) {
	return "PMB-2026-1-00001", nil
}
func (m *mockAppRepoForChange) ListApplications(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, int64, error) {
	var list []model.Application
	for _, a := range m.apps {
		list = append(list, *a)
	}
	return list, int64(len(list)), nil
}
func (m *mockAppRepoForChange) ListAllForExport(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, error) {
	var list []model.Application
	for _, a := range m.apps {
		list = append(list, *a)
	}
	return list, nil
}
func (m *mockAppRepoForChange) GetStats(ctx context.Context) (*model.AdmissionStats, error) {
	return &model.AdmissionStats{}, nil
}
func (m *mockAppRepoForChange) CreateStatusLog(tx *gorm.DB, log *model.ApplicationStatusLog) error {
	return nil
}
func (m *mockAppRepoForChange) GetNextSequence(ctx context.Context, periodID uint) (int64, error) {
	return 1, nil
}

func TestChangeService_Flow(t *testing.T) {
	ctx := context.Background()
	changeRepo := newMockChangeRepo()
	appRepo := newMockAppRepoForChange()
	pathRepo := newMockAdmissionPathRepo()

	svc := service.NewChangeService(
		changeRepo,
		appRepo,
		pathRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	userID := uuid.New()
	appID := uuid.New()
	choice1 := uuid.New()
	choice2 := uuid.New()

	app := &model.Application{
		ID:                          appID,
		RegistrationNo:              "PMB-2026-1-00001",
		UserID:                      userID,
		PeriodID:                    1,
		AdmissionPathID:             1,
		Choice1ID:                   choice1,
		Status:                      model.StatusAwaitingPayment,
		PathChangeCountAfterPayment: 0,
	}
	_ = appRepo.Create(ctx, app)

	// 1. Pilihan path change options
	opts, err := svc.GetPathChangeOptions(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, opts.Allowed)
	assert.Len(t, opts.Options, 3)

	// 2. Pindah langsung saat awaiting_payment
	targetPathID := uint(2)
	newChoice1 := uuid.New()
	result, err := svc.RequestOrApplyPathChange(ctx, userID, &service.PathChangeInput{
		PathID:        targetPathID,
		FirstChoiceID: newChoice1,
	})
	assert.NoError(t, err)
	assert.True(t, result.AppliedDirectly)

	// Verifikasi path berubah langsung
	updatedApp, _ := appRepo.FindByID(ctx, appID)
	assert.Equal(t, uint(2), updatedApp.AdmissionPathID)
	assert.Equal(t, newChoice1, updatedApp.Choice1ID)

	// 3. Simulasikan pendaftar sudah bayar (form_open)
	updatedApp.Status = model.StatusFormOpen
	_ = appRepo.Update(ctx, updatedApp)

	// Ajukan pindah jalur ke Beasiswa (fee difference: 200000 - 300000 = -100000)
	reqResult, err := svc.RequestOrApplyPathChange(ctx, userID, &service.PathChangeInput{
		PathID:        3,
		FirstChoiceID: choice2,
		Reason:        "Mengajukan pindah ke jalur beasiswa karena keterbatasan ekonomi",
	})
	assert.NoError(t, err)
	assert.False(t, reqResult.AppliedDirectly)
	assert.Equal(t, "pending", reqResult.Status)

	// Ajukan lagi saat pending -> tolak PATH_CHANGE_PENDING
	_, errPending := svc.RequestOrApplyPathChange(ctx, userID, &service.PathChangeInput{
		PathID:        1,
		FirstChoiceID: choice1,
		Reason:        "Ingin pindah lagi",
	})
	assert.ErrorIs(t, errPending, service.ErrPathChangePending)

	// 4. Admin melihat list request
	list, total, err := svc.AdminListPathChangeRequests(ctx, "pending", 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)

	// 5. Admin Approve request
	adminID := uuid.New()
	err = svc.AdminApprovePathChangeRequest(ctx, *reqResult.RequestID, adminID, "Disetujui panitia")
	assert.NoError(t, err)

	approvedApp, _ := appRepo.FindByID(ctx, appID)
	assert.Equal(t, uint(3), approvedApp.AdmissionPathID)
	assert.Equal(t, 1, approvedApp.PathChangeCountAfterPayment)

	// 6. Ajukan pindah kedua kali setelah bayar -> tolak PATH_CHANGE_LIMIT_REACHED
	_, errLimit := svc.RequestOrApplyPathChange(ctx, userID, &service.PathChangeInput{
		PathID:        1,
		FirstChoiceID: choice1,
		Reason:        "Ingin pindah kedua kali",
	})
	assert.ErrorIs(t, errLimit, service.ErrPathChangeLimitReached)

	// 7. Cek riwayat linimasa
	changes, countChanges, err := svc.GetApplicationChanges(ctx, userID, 10, 0)
	assert.NoError(t, err)
	assert.True(t, countChanges >= 3)
	assert.NotEmpty(t, changes[0].TitleLabel)
	assert.NotEmpty(t, changes[0].ActorLabel)
}
