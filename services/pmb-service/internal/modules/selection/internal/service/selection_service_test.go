package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Recommendation Repository
type mockRecRepo struct {
	recs map[uuid.UUID]*model.Recommendation
}

func newMockRecRepo() *mockRecRepo {
	return &mockRecRepo{recs: make(map[uuid.UUID]*model.Recommendation)}
}

func (m *mockRecRepo) SaveRecommendation(ctx context.Context, rec *model.Recommendation, items []model.RecommendationItem) error {
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	rec.Items = items
	m.recs[rec.ID] = rec
	return nil
}

func (m *mockRecRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Recommendation, error) {
	if r, ok := m.recs[id]; ok {
		return r, nil
	}
	return nil, nil
}

func (m *mockRecRepo) FindByPeriodAndProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (*model.Recommendation, error) {
	for _, r := range m.recs {
		if r.PeriodID == periodID && r.PathID == pathID && r.StudyProgramCode == programCode {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockRecRepo) ListByPeriod(ctx context.Context, periodID uuid.UUID, status *model.RecommendationStatus) ([]model.Recommendation, error) {
	var list []model.Recommendation
	for _, r := range m.recs {
		if r.PeriodID == periodID {
			if status == nil || *status == "" || r.Status == *status {
				list = append(list, *r)
			}
		}
	}
	return list, nil
}

func (m *mockRecRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.RecommendationStatus, actorID *uuid.UUID, note string) error {
	if r, ok := m.recs[id]; ok {
		r.Status = status
		if note != "" {
			r.Note = note
		}
		if status == model.StatusSubmitted {
			r.SubmittedBy = actorID
			now := time.Now()
			r.SubmittedAt = &now
		} else if status == model.StatusApproved {
			r.ApprovedBy = actorID
			now := time.Now()
			r.ApprovedAt = &now
		}
		return nil
	}
	return service.ErrRecommendationNotFound
}

// Mock Announcement Repository
type mockAnnRepo struct {
	anns map[uuid.UUID]*model.Announcement
}

func newMockAnnRepo() *mockAnnRepo {
	return &mockAnnRepo{anns: make(map[uuid.UUID]*model.Announcement)}
}

func (m *mockAnnRepo) Create(ctx context.Context, ann *model.Announcement) error {
	if ann.ID == uuid.Nil {
		ann.ID = uuid.New()
	}
	m.anns[ann.ID] = ann
	return nil
}

func (m *mockAnnRepo) FindByPeriodID(ctx context.Context, periodID uuid.UUID) (*model.Announcement, error) {
	for _, a := range m.anns {
		if a.PeriodID == periodID {
			return a, nil
		}
	}
	return nil, nil
}

// Mock Result Repository
type mockResultRepo struct {
	results map[uuid.UUID]*model.Result
}

func newMockResultRepo() *mockResultRepo {
	return &mockResultRepo{results: make(map[uuid.UUID]*model.Result)}
}

func (m *mockResultRepo) BatchCreate(ctx context.Context, results []model.Result) error {
	for _, res := range results {
		r := res
		if r.ID == uuid.Nil {
			r.ID = uuid.New()
		}
		m.results[r.ApplicationID] = &r
	}
	return nil
}

func (m *mockResultRepo) FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Result, error) {
	if res, ok := m.results[appID]; ok {
		return res, nil
	}
	return nil, nil
}

func (m *mockResultRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Result, error) {
	for _, res := range m.results {
		if res.UserID == userID {
			return res, nil
		}
	}
	return nil, nil
}

func (m *mockResultRepo) WithdrawApplication(ctx context.Context, appID uuid.UUID) (*model.Result, error) {
	if res, ok := m.results[appID]; ok {
		res.Decision = model.DecisionWithdrawn
		return res, nil
	}
	return nil, nil
}


// -------------------------------------------------------------
// Test Suite
// -------------------------------------------------------------

func TestRecommendationQuotaValidationAndSubmission(t *testing.T) {
	recRepo := newMockRecRepo()
	annRepo := newMockAnnRepo()
	resultRepo := newMockResultRepo()
	cfg := &config.Config{}

	svc := service.NewSelectionService(cfg, recRepo, annRepo, resultRepo, nil, nil, nil, nil, nil)
	periodID := uuid.New()
	pathID := uuid.New()
	prodiUserID := uuid.New()

	// 1. Rekomendasi melebihi kuota default (default 5, kita usulkan 6 peserta 'pass') -> Tolak QUOTA_EXCEEDED
	payloadExcess := &model.SaveRecommendationPayload{
		PeriodID:         periodID,
		PathID:           pathID,
		StudyProgramCode: "TIF",
		Items: []model.SaveRecommendationItemPayload{
			{ApplicationID: uuid.New(), Decision: model.DecisionPass, FinalScore: 95.0, Rank: 1},
			{ApplicationID: uuid.New(), Decision: model.DecisionPass, FinalScore: 90.0, Rank: 2},
			{ApplicationID: uuid.New(), Decision: model.DecisionPass, FinalScore: 88.0, Rank: 3},
			{ApplicationID: uuid.New(), Decision: model.DecisionPass, FinalScore: 85.0, Rank: 4},
			{ApplicationID: uuid.New(), Decision: model.DecisionPass, FinalScore: 82.0, Rank: 5},
			{ApplicationID: uuid.New(), Decision: model.DecisionPass, FinalScore: 80.0, Rank: 6}, // Melebihi kuota 5
		},
	}

	_, err := svc.SaveProgramRecommendation(context.Background(), payloadExcess, prodiUserID, []string{"prodi"}, []string{"TIF"})
	assert.ErrorIs(t, err, service.ErrQuotaExceeded)

	// 2. Rekomendasi sesuai kuota (2 pass, 1 waitlist, 1 fail) -> Sukses
	app1 := uuid.New()
	app2 := uuid.New()
	app3 := uuid.New()
	app4 := uuid.New()

	payloadValid := &model.SaveRecommendationPayload{
		PeriodID:         periodID,
		PathID:           pathID,
		StudyProgramCode: "TIF",
		Items: []model.SaveRecommendationItemPayload{
			{ApplicationID: app1, Decision: model.DecisionPass, FinalScore: 90.0, Rank: 1},
			{ApplicationID: app2, Decision: model.DecisionPass, FinalScore: 85.0, Rank: 2},
			{ApplicationID: app3, Decision: model.DecisionWaitlist, FinalScore: 78.0, Rank: 3},
			{ApplicationID: app4, Decision: model.DecisionFail, FinalScore: 60.0, Rank: 4},
		},
	}

	rec, err := svc.SaveProgramRecommendation(context.Background(), payloadValid, prodiUserID, []string{"prodi"}, []string{"TIF"})
	require.NoError(t, err)
	assert.Equal(t, model.StatusDraft, rec.Status)
	assert.Len(t, rec.Items, 4)

	// 3. Submit rekomendasi oleh prodi -> Status bertransisi ke submitted
	submittedRec, err := svc.SubmitProgramRecommendation(context.Background(), rec.ID, prodiUserID, []string{"prodi"}, []string{"TIF"})
	require.NoError(t, err)
	assert.Equal(t, model.StatusSubmitted, submittedRec.Status)
	assert.Equal(t, prodiUserID, *submittedRec.SubmittedBy)

	// 4. Upaya edit kembali saat sudah submitted -> Ditolak INVALID_STATE
	_, err = svc.SaveProgramRecommendation(context.Background(), payloadValid, prodiUserID, []string{"prodi"}, []string{"TIF"})
	assert.ErrorIs(t, err, service.ErrInvalidRecommendationState)
}

func TestLeadershipApprovalFlowAndAnnouncements(t *testing.T) {
	recRepo := newMockRecRepo()
	annRepo := newMockAnnRepo()
	resultRepo := newMockResultRepo()
	cfg := &config.Config{}

	svc := service.NewSelectionService(cfg, recRepo, annRepo, resultRepo, nil, nil, nil, nil, nil)
	periodID := uuid.New()
	pathID := uuid.New()
	prodiUserID := uuid.New()
	pimpinanUserID := uuid.New()
	adminUserID := uuid.New()
	studentUserID := uuid.New()
	appStudent := uuid.New()

	// 1. Simpan & Submit rekomendasi
	payload := &model.SaveRecommendationPayload{
		PeriodID:         periodID,
		PathID:           pathID,
		StudyProgramCode: "TIF",
		Items: []model.SaveRecommendationItemPayload{
			{ApplicationID: appStudent, Decision: model.DecisionPass, FinalScore: 88.0, Rank: 1},
		},
	}
	rec, err := svc.SaveProgramRecommendation(context.Background(), payload, prodiUserID, []string{"prodi"}, []string{"TIF"})
	require.NoError(t, err)
	rec, err = svc.SubmitProgramRecommendation(context.Background(), rec.ID, prodiUserID, []string{"prodi"}, []string{"TIF"})
	require.NoError(t, err)

	// 2. Pimpinan melakukan 'return' disertai catatan
	returnedRec, err := svc.ReturnRecommendation(context.Background(), rec.ID, "Perbaiki pertimbangan pendaftar", pimpinanUserID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusReturned, returnedRec.Status)
	assert.Equal(t, "Perbaiki pertimbangan pendaftar", returnedRec.Note)

	// Prodi dapat mengedit dan men-submit ulang
	rec, err = svc.SubmitProgramRecommendation(context.Background(), rec.ID, prodiUserID, []string{"prodi"}, []string{"TIF"})
	require.NoError(t, err)

	// 3. Pimpinan menyetujui rekomendasi ('approve')
	approvedRec, err := svc.ApproveRecommendation(context.Background(), rec.ID, pimpinanUserID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusApproved, approvedRec.Status)

	// 4. Calon mahasiswa mencoba cek hasil sebelum pengumuman dipublikasikan -> Ditolak RESULT_NOT_PUBLISHED
	_, err = svc.GetMyResult(context.Background(), studentUserID)
	assert.ErrorIs(t, err, service.ErrResultNotPublished)

	// 5. Admin PMB mempublikasikan pengumuman
	annPayload := &model.PublishAnnouncementPayload{
		PeriodID: periodID,
		Title:    "Pengumuman Kelulusan Seleksi PMB Jalur Reguler 2026/2027",
		Content:  "Selamat kepada seluruh calon mahasiswa yang dinyatakan diterima.",
	}
	ann, err := svc.PublishAnnouncement(context.Background(), annPayload, adminUserID)
	require.NoError(t, err)
	assert.NotEmpty(t, ann.ID)

	// Simulasikan hasil calon mahasiswa tersimpan di resultRepo
	err = resultRepo.BatchCreate(context.Background(), []model.Result{
		{
			ApplicationID:   appStudent,
			UserID:          studentUserID,
			Decision:        model.DecisionPass,
			AcceptedProgram: "TIF",
			AnnouncedAt:     time.Now(),
		},
	})
	require.NoError(t, err)

	// 6. Calon mahasiswa melihat hasil seleksi setelah publikasi -> Berhasil
	myResult, err := svc.GetMyResult(context.Background(), studentUserID)
	require.NoError(t, err)
	assert.Equal(t, model.DecisionPass, myResult.Decision)
	assert.Equal(t, "TIF", myResult.AcceptedProgram)

	// 7. Pengunduran diri pendaftar (CR-007)
	err = svc.WithdrawApplication(context.Background(), appStudent)
	require.NoError(t, err)

	withdrawnResult, err := svc.GetMyResult(context.Background(), studentUserID)
	require.NoError(t, err)
	assert.Equal(t, model.DecisionWithdrawn, withdrawnResult.Decision)

	// 8. Pengecekan SK Rektor & Scoreboard (CR-009 / K11 & K14)
	assert.NotEmpty(t, ann.DecreeNumber)

	sb, errSb := svc.GetScoreboard(context.Background(), periodID, "all", "", []string{"superadmin"}, nil)
	require.NoError(t, errSb)
	assert.Equal(t, 70.0, sb.PassingThreshold)
	assert.NotNil(t, sb.Counts)
}
