package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Mock Withdrawal Repository
type mockWithdrawalRepo struct {
	items       map[uuid.UUID]*model.Withdrawal
	sequenceSeq int
}

func newMockWithdrawalRepo() *mockWithdrawalRepo {
	return &mockWithdrawalRepo{
		items:       make(map[uuid.UUID]*model.Withdrawal),
		sequenceSeq: 1,
	}
}

func (m *mockWithdrawalRepo) DB() *gorm.DB {
	return nil
}

func (m *mockWithdrawalRepo) Create(ctx context.Context, w *model.Withdrawal) error {
	m.items[w.ID] = w
	return nil
}

func (m *mockWithdrawalRepo) Update(ctx context.Context, w *model.Withdrawal) error {
	m.items[w.ID] = w
	return nil
}

func (m *mockWithdrawalRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Withdrawal, error) {
	w, exists := m.items[id]
	if !exists {
		return nil, service.ErrWithdrawalNotFound
	}
	return w, nil
}

func (m *mockWithdrawalRepo) FindPendingByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Withdrawal, error) {
	for _, w := range m.items {
		if w.ApplicationID == appID && w.Status == model.WithdrawalStatusPending {
			return w, nil
		}
	}
	return nil, nil
}

func (m *mockWithdrawalRepo) FindLatestByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Withdrawal, error) {
	var latest *model.Withdrawal
	for _, w := range m.items {
		if w.ApplicationID == appID {
			if latest == nil || w.RequestedAt.After(latest.RequestedAt) {
				latest = w
			}
		}
	}
	return latest, nil
}

func (m *mockWithdrawalRepo) FindByVerifyCodeHash(ctx context.Context, hash string) (*model.Withdrawal, error) {
	for _, w := range m.items {
		if w.VerifyCodeHash == hash {
			return w, nil
		}
	}
	return nil, nil
}

func (m *mockWithdrawalRepo) FindAll(ctx context.Context, status string, page, limit int) ([]model.Withdrawal, int64, error) {
	var res []model.Withdrawal
	for _, w := range m.items {
		if status == "" || string(w.Status) == status {
			res = append(res, *w)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockWithdrawalRepo) GetNextLetterSequence(ctx context.Context, year int) (int, error) {
	seq := m.sequenceSeq
	m.sequenceSeq++
	return seq, nil
}

func (m *mockWithdrawalRepo) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total": len(m.items),
	}, nil
}

// Mock Change Repository
type mockChangeRepoForWithdrawal struct {
	changes []model.ApplicationChange
}

func (m *mockChangeRepoForWithdrawal) CreateChange(ctx context.Context, tx *gorm.DB, change *model.ApplicationChange) error {
	m.changes = append(m.changes, *change)
	return nil
}

func (m *mockChangeRepoForWithdrawal) GetChangesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error) {
	return m.changes, int64(len(m.changes)), nil
}

func (m *mockChangeRepoForWithdrawal) GetChangesByApplicationID(ctx context.Context, appID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error) {
	return m.changes, int64(len(m.changes)), nil
}

func (m *mockChangeRepoForWithdrawal) CreatePathChangeRequest(ctx context.Context, tx *gorm.DB, req *model.PathChangeRequest) error {
	return nil
}

func (m *mockChangeRepoForWithdrawal) GetPathChangeRequestByID(ctx context.Context, id uuid.UUID) (*model.PathChangeRequest, error) {
	return nil, nil
}

func (m *mockChangeRepoForWithdrawal) GetPendingPathChangeRequest(ctx context.Context, appID uuid.UUID) (*model.PathChangeRequest, error) {
	return nil, nil
}

func (m *mockChangeRepoForWithdrawal) UpdatePathChangeRequest(ctx context.Context, tx *gorm.DB, req *model.PathChangeRequest) error {
	return nil
}

func (m *mockChangeRepoForWithdrawal) ListPathChangeRequests(ctx context.Context, status string, limit, offset int) ([]model.PathChangeRequest, int64, error) {
	return nil, 0, nil
}

func (m *mockChangeRepoForWithdrawal) InitApplicationChangesTriggersAndBackfill(db *gorm.DB) error {
	return nil
}

// Mock App Repository
type mockAppRepoForWithdrawal struct {
	apps map[uuid.UUID]*model.Application
}

func newMockAppRepoForWithdrawal() *mockAppRepoForWithdrawal {
	return &mockAppRepoForWithdrawal{
		apps: make(map[uuid.UUID]*model.Application),
	}
}

func (m *mockAppRepoForWithdrawal) DB() *gorm.DB { return nil }
func (m *mockAppRepoForWithdrawal) CreateInTx(tx *gorm.DB, app *model.Application) error {
	m.apps[app.ID] = app
	return nil
}
func (m *mockAppRepoForWithdrawal) FindByUserIDAndPeriod(ctx context.Context, userID uuid.UUID, periodID uint) (*model.Application, error) {
	for _, a := range m.apps {
		if a.UserID == userID && a.PeriodID == periodID {
			return a, nil
		}
	}
	return nil, nil
}
func (m *mockAppRepoForWithdrawal) FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*model.Application, error) {
	for _, a := range m.apps {
		if a.UserID == userID && a.Status != model.StatusCancelled && a.Status != model.StatusWithdrawn {
			return a, nil
		}
	}
	return nil, nil
}
func (m *mockAppRepoForWithdrawal) FindByID(ctx context.Context, id uuid.UUID) (*model.Application, error) {
	a, ok := m.apps[id]
	if !ok {
		return nil, repository.ErrApplicationNotFound
	}
	return a, nil
}
func (m *mockAppRepoForWithdrawal) FindByRegistrationNo(ctx context.Context, regNo string) (*model.Application, error) {
	for _, a := range m.apps {
		if a.RegistrationNo == regNo {
			return a, nil
		}
	}
	return nil, nil
}
func (m *mockAppRepoForWithdrawal) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	var res []model.Application
	for _, a := range m.apps {
		if a.UserID == userID {
			res = append(res, *a)
		}
	}
	return res, nil
}
func (m *mockAppRepoForWithdrawal) Update(ctx context.Context, app *model.Application) error {
	m.apps[app.ID] = app
	return nil
}
func (m *mockAppRepoForWithdrawal) UpdateChoices(ctx context.Context, id uuid.UUID, choice1ID uuid.UUID, choice2ID *uuid.UUID) error {
	return nil
}
func (m *mockAppRepoForWithdrawal) UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus model.ApplicationStatus, notes string, changedBy *uuid.UUID) error {
	if a, ok := m.apps[id]; ok {
		a.Status = toStatus
	}
	return nil
}
func (m *mockAppRepoForWithdrawal) GenerateRegistrationNo(ctx context.Context, tx *gorm.DB, periodID uint, academicYear string) (string, error) {
	return "REG-2026-TEST", nil
}
func (m *mockAppRepoForWithdrawal) ListApplications(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, int64, error) {
	return nil, 0, nil
}
func (m *mockAppRepoForWithdrawal) ListAllForExport(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, error) {
	return nil, nil
}
func (m *mockAppRepoForWithdrawal) GetStats(ctx context.Context) (*model.AdmissionStats, error) {
	return nil, nil
}
func (m *mockAppRepoForWithdrawal) CreateStatusLog(tx *gorm.DB, log *model.ApplicationStatusLog) error {
	return nil
}

func TestWithdrawalService_EligibilityAndFlow(t *testing.T) {
	ctx := context.Background()
	withdrawalRepo := newMockWithdrawalRepo()
	appRepo := newMockAppRepoForWithdrawal()
	changeRepo := &mockChangeRepoForWithdrawal{}

	svc := service.NewWithdrawalService(
		withdrawalRepo,
		appRepo,
		changeRepo,
		nil,
		nil, // authCli (nil berarti bypass validasi password dalam mock)
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

	// 1. Kasus Belum Ada Pendaftaran
	elig, err := svc.GetWithdrawalEligibility(ctx, userID)
	require.NoError(t, err)
	assert.False(t, elig.Allowed)

	// 2. Kasus Status Draft atau Awaiting Payment -> Tidak Diizinkan (Arahkan ke Batalkan)
	app := &model.Application{
		ID:             appID,
		UserID:         userID,
		RegistrationNo: "REG-2026-001",
		Status:         model.StatusAwaitingPayment,
		Choice1ID:      uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	appRepo.apps[appID] = app

	elig, err = svc.GetWithdrawalEligibility(ctx, userID)
	require.NoError(t, err)
	assert.False(t, elig.Allowed)
	assert.Contains(t, elig.BlockReasonLabel, "sebelum melakukan pembayaran")

	// 3. Kasus Status Verified (Sudah Bayar, Belum Lulus) -> Diizinkan Langsung (RequiresApproval: false)
	app.Status = model.StatusVerified
	elig, err = svc.GetWithdrawalEligibility(ctx, userID)
	require.NoError(t, err)
	assert.True(t, elig.Allowed)
	assert.False(t, elig.RequiresApproval)

	// 4. Request Withdrawal pada Status Verified -> Langsung Efektif
	res, err := svc.RequestWithdrawal(ctx, userID, &service.WithdrawalRequestInput{
		ReasonCategory: "financial",
		ReasonText:     "Kendala biaya pendidikan dan memilih bekerja terlebih dahulu",
		AgreeStatement: true,
		Password:       "ValidPassword123!",
	})
	require.NoError(t, err)
	assert.NotNil(t, res.Withdrawal)
	assert.Equal(t, model.WithdrawalStatusEffective, res.Withdrawal.Status)
	assert.NotEmpty(t, res.Withdrawal.LetterNumber)
	assert.Contains(t, res.Withdrawal.LetterNumber, "PMB/UND/")

	// Cek status app berubah menjadi Withdrawn
	assert.Equal(t, model.StatusWithdrawn, app.Status)

	// Cek riwayat tercatat di changeRepo
	assert.NotEmpty(t, changeRepo.changes)
	assert.Equal(t, model.ChangeTypeWithdrawalApproved, changeRepo.changes[len(changeRepo.changes)-1].ChangeType)

	// 5. Cek Verifikasi Publik Bukti Surat
	verifyRes, err := svc.VerifyWithdrawalPublic(ctx, res.Withdrawal.VerifyCodeHash)
	require.NoError(t, err)
	assert.True(t, verifyRes.Valid)
	assert.Equal(t, res.Withdrawal.LetterNumber, verifyRes.LetterNumber)

	// 6. Kasus Status Passed -> Memerlukan Persetujuan Admin (RequiresApproval: true)
	user2ID := uuid.New()
	app2ID := uuid.New()
	app2 := &model.Application{
		ID:             app2ID,
		UserID:         user2ID,
		RegistrationNo: "REG-2026-002",
		Status:         model.StatusPassed,
		Choice1ID:      uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	appRepo.apps[app2ID] = app2

	elig2, err := svc.GetWithdrawalEligibility(ctx, user2ID)
	require.NoError(t, err)
	assert.True(t, elig2.Allowed)
	assert.True(t, elig2.RequiresApproval)

	// Request Withdrawal pada Status Passed -> Pending
	res2, err := svc.RequestWithdrawal(ctx, user2ID, &service.WithdrawalRequestInput{
		ReasonCategory: "accepted_elsewhere",
		ReasonText:     "Diterima di perguruan tinggi negeri pilihan pertama",
		AgreeStatement: true,
		Password:       "ValidPassword123!",
	})
	require.NoError(t, err)
	assert.Equal(t, model.WithdrawalStatusPending, res2.Withdrawal.Status)

	// Coba ajukan ulang saat masih pending -> harus ditolak
	_, errDuplicate := svc.RequestWithdrawal(ctx, user2ID, &service.WithdrawalRequestInput{
		ReasonCategory: "accepted_elsewhere",
		ReasonText:     "Pengajuan ulang saat masih pending",
		AgreeStatement: true,
		Password:       "ValidPassword123!",
	})
	assert.Error(t, errDuplicate)

	// Admin Approve
	adminID := uuid.New()
	errApprove := svc.AdminApproveWithdrawal(ctx, res2.Withdrawal.ID, adminID, "Disetujui, semoga sukses di kampus tujuan")
	require.NoError(t, errApprove)

	wApproved, err := svc.GetMyWithdrawal(ctx, user2ID)
	require.NoError(t, err)
	assert.Equal(t, model.WithdrawalStatusApproved, wApproved.Status)
	assert.Equal(t, model.StatusWithdrawn, app2.Status)

	// 7. Pengujian Pembatalan Permohonan Pending oleh Mahasiswa
	user3ID := uuid.New()
	app3ID := uuid.New()
	app3 := &model.Application{
		ID:             app3ID,
		UserID:         user3ID,
		RegistrationNo: "REG-2026-003",
		Status:         model.StatusPassed,
		Choice1ID:      uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	appRepo.apps[app3ID] = app3

	res3, err := svc.RequestWithdrawal(ctx, user3ID, &service.WithdrawalRequestInput{
		ReasonCategory: "relocation",
		ReasonText:     "Pindah domisili mengikuti keluarga keluar daerah",
		AgreeStatement: true,
		Password:       "ValidPassword123!",
	})
	require.NoError(t, err)
	assert.Equal(t, model.WithdrawalStatusPending, res3.Withdrawal.Status)

	// Calon mahasiswa membatalkan permohonannya sendiri
	errCancel := svc.CancelPendingWithdrawal(ctx, user3ID)
	require.NoError(t, errCancel)

	wCancelled, _ := withdrawalRepo.FindByID(ctx, res3.Withdrawal.ID)
	assert.Equal(t, model.WithdrawalStatusCancelled, wCancelled.Status)
	assert.Equal(t, model.StatusPassed, app3.Status) // Status app tidak berubah ke Withdrawn
}
