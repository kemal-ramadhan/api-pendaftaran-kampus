package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Component Repository
type mockCompRepo struct {
	comps map[uuid.UUID]*model.ScoreComponent
}

func newMockCompRepo() *mockCompRepo {
	return &mockCompRepo{comps: make(map[uuid.UUID]*model.ScoreComponent)}
}

func (m *mockCompRepo) Create(ctx context.Context, comp *model.ScoreComponent) error {
	if comp.ID == uuid.Nil {
		comp.ID = uuid.New()
	}
	m.comps[comp.ID] = comp
	return nil
}

func (m *mockCompRepo) Update(ctx context.Context, comp *model.ScoreComponent) error {
	m.comps[comp.ID] = comp
	return nil
}

func (m *mockCompRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.comps, id)
	return nil
}

func (m *mockCompRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.ScoreComponent, error) {
	if comp, ok := m.comps[id]; ok {
		return comp, nil
	}
	return nil, nil
}

func (m *mockCompRepo) FindByPathID(ctx context.Context, pathID uuid.UUID) ([]model.ScoreComponent, error) {
	var res []model.ScoreComponent
	for _, c := range m.comps {
		if c.PathID == pathID {
			res = append(res, *c)
		}
	}
	return res, nil
}

func (m *mockCompRepo) FindByPathAndCode(ctx context.Context, pathID uuid.UUID, code string) (*model.ScoreComponent, error) {
	for _, c := range m.comps {
		if c.PathID == pathID && c.Code == code {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockCompRepo) GetTotalWeightByPath(ctx context.Context, pathID uuid.UUID, excludeID *uuid.UUID) (float64, error) {
	var total float64
	for _, c := range m.comps {
		if c.PathID == pathID && c.IsActive {
			if excludeID != nil && c.ID == *excludeID {
				continue
			}
			total += c.WeightPercent
		}
	}
	return total, nil
}

// Mock Score Repository
type mockScoreRepo struct {
	scores    map[string]*model.Score
	summaries map[uuid.UUID]*model.ScoreSummary
	locks     map[string]*model.ProgramLock
}

func newMockScoreRepo() *mockScoreRepo {
	return &mockScoreRepo{
		scores:    make(map[string]*model.Score),
		summaries: make(map[uuid.UUID]*model.ScoreSummary),
		locks:     make(map[string]*model.ProgramLock),
	}
}

func (m *mockScoreRepo) UpsertScore(ctx context.Context, score *model.Score) error {
	key := score.ApplicationID.String() + ":" + score.ComponentCode
	m.scores[key] = score
	return nil
}

func (m *mockScoreRepo) FindScoresByApplicationID(ctx context.Context, appID uuid.UUID) ([]model.Score, error) {
	var res []model.Score
	for _, s := range m.scores {
		if s.ApplicationID == appID {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (m *mockScoreRepo) FindScoreByAppAndComponent(ctx context.Context, appID uuid.UUID, componentCode string) (*model.Score, error) {
	key := appID.String() + ":" + componentCode
	if s, ok := m.scores[key]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockScoreRepo) UpsertSummary(ctx context.Context, summary *model.ScoreSummary) error {
	m.summaries[summary.ApplicationID] = summary
	return nil
}

func (m *mockScoreRepo) FindSummaryByApplicationID(ctx context.Context, appID uuid.UUID) (*model.ScoreSummary, error) {
	if sm, ok := m.summaries[appID]; ok {
		return sm, nil
	}
	return nil, nil
}

func (m *mockScoreRepo) ListSummariesByProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) ([]model.ScoreSummary, error) {
	var res []model.ScoreSummary
	for _, sm := range m.summaries {
		if sm.PeriodID == periodID && sm.StudyProgramCode == programCode {
			res = append(res, *sm)
		}
	}
	return res, nil
}

func (m *mockScoreRepo) RecalculateRanksInProgram(ctx context.Context, periodID, pathID uuid.UUID, programCode string) error {
	// Simple mock ranking: assign rank 1 to all
	for _, sm := range m.summaries {
		if sm.PeriodID == periodID && sm.StudyProgramCode == programCode {
			sm.RankInProgram = 1
		}
	}
	return nil
}

func (m *mockScoreRepo) IsProgramLocked(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (bool, error) {
	key := periodID.String() + ":" + pathID.String() + ":" + programCode
	if lk, ok := m.locks[key]; ok {
		return lk.IsLocked, nil
	}
	return false, nil
}

func (m *mockScoreRepo) SetProgramLock(ctx context.Context, lock *model.ProgramLock) error {
	key := lock.PeriodID.String() + ":" + lock.PathID.String() + ":" + lock.StudyProgramCode
	m.locks[key] = lock
	return nil
}

func (m *mockScoreRepo) GetProgramLock(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (*model.ProgramLock, error) {
	key := periodID.String() + ":" + pathID.String() + ":" + programCode
	if lk, ok := m.locks[key]; ok {
		return lk, nil
	}
	return nil, nil
}

func (m *mockScoreRepo) ListSummariesInternal(ctx context.Context, periodID uuid.UUID, programCode *string) ([]model.ScoreSummary, error) {
	var res []model.ScoreSummary
	for _, sm := range m.summaries {
		if sm.PeriodID == periodID {
			if programCode == nil || *programCode == "" || sm.StudyProgramCode == *programCode {
				res = append(res, *sm)
			}
		}
	}
	return res, nil
}

// -------------------------------------------------------------
// Test Suite
// -------------------------------------------------------------

func TestComponentManagementAndWeightValidation(t *testing.T) {
	compRepo := newMockCompRepo()
	scoreRepo := newMockScoreRepo()
	excelExp := exporter.NewExcelExporter()
	cfg := &config.Config{}

	svc := service.NewScoringService(cfg, compRepo, scoreRepo, excelExp, nil, nil, nil)
	pathID := uuid.New()

	// 1. Buat komponen pertama: Tes Tulis (60%)
	comp1, err := svc.CreateComponent(context.Background(), &model.ScoreComponent{
		PathID:        pathID,
		Code:          "written_test",
		Name:          "Ujian Tulis",
		WeightPercent: 60.00,
		MaxScore:      100.00,
		IsActive:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, 60.00, comp1.WeightPercent)

	// 2. Coba buat komponen kedua dengan bobot 50% (total 110%) -> harus gagal!
	_, err = svc.CreateComponent(context.Background(), &model.ScoreComponent{
		PathID:        pathID,
		Code:          "interview",
		Name:          "Wawancara",
		WeightPercent: 50.00,
		MaxScore:      100.00,
		IsActive:      true,
	})
	assert.ErrorIs(t, err, service.ErrWeightExceeds100)

	// 3. Buat komponen kedua dengan bobot 40% (total 100%) -> harus sukses
	comp2, err := svc.CreateComponent(context.Background(), &model.ScoreComponent{
		PathID:        pathID,
		Code:          "interview",
		Name:          "Wawancara",
		WeightPercent: 40.00,
		MaxScore:      100.00,
		IsActive:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, 40.00, comp2.WeightPercent)

	// 4. Verifikasi total bobot jalur tepat 100%
	err = svc.ValidatePathWeightTotal(context.Background(), pathID)
	assert.NoError(t, err)

	// 5. Cek duplikasi kode pada jalur yang sama -> tolak
	_, err = svc.CreateComponent(context.Background(), &model.ScoreComponent{
		PathID:        pathID,
		Code:          "written_test",
		Name:          "Ujian Tulis Duplicate",
		WeightPercent: 10.00,
		MaxScore:      100.00,
		IsActive:      true,
	})
	assert.ErrorIs(t, err, service.ErrDuplicateComponent)
}

func TestScoreInputAndWeightedAverage(t *testing.T) {
	compRepo := newMockCompRepo()
	scoreRepo := newMockScoreRepo()
	excelExp := exporter.NewExcelExporter()
	cfg := &config.Config{}

	svc := service.NewScoringService(cfg, compRepo, scoreRepo, excelExp, nil, nil, nil)
	pathID := uuid.MustParse("00000000-0000-0000-0000-000000000002") // Sesuai mock app fallback
	appID := uuid.New()
	examinerID := uuid.New()

	// Daftarkan 2 komponen: written_test (60%), interview (40%)
	_, err := svc.CreateComponent(context.Background(), &model.ScoreComponent{
		PathID:        pathID,
		Code:          "written_test",
		Name:          "Ujian Tulis",
		WeightPercent: 60.00,
		MaxScore:      100.00,
		IsActive:      true,
	})
	require.NoError(t, err)

	_, err = svc.CreateComponent(context.Background(), &model.ScoreComponent{
		PathID:        pathID,
		Code:          "interview",
		Name:          "Wawancara",
		WeightPercent: 40.00,
		MaxScore:      100.00,
		IsActive:      true,
	})
	require.NoError(t, err)

	// 1. Input nilai written_test = 80.0
	score1, summary1, err := svc.InputScore(
		context.Background(),
		appID,
		"written_test",
		80.0,
		"Nilai ujian tulis memuaskan",
		examinerID,
		[]string{"admin_pmb"},
	)
	require.NoError(t, err)
	assert.Equal(t, 80.0, score1.Score)
	// Hanya 1 nilai: 80 * 0.6 = 48.00
	assert.Equal(t, 48.00, summary1.FinalScore)

	// 2. Input nilai interview = 90.0
	score2, summary2, err := svc.InputScore(
		context.Background(),
		appID,
		"interview",
		90.0,
		"Kemampuan komunikasi sangat baik",
		examinerID,
		[]string{"admin_pmb"},
	)
	require.NoError(t, err)
	assert.Equal(t, 90.0, score2.Score)
	// (80 * 0.6) + (90 * 0.4) = 48 + 36 = 84.00
	assert.Equal(t, 84.00, summary2.FinalScore)

	// 3. Tes proteksi flag locked
	periodID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	_, err = svc.SetProgramLock(context.Background(), periodID, pathID, "TIF", true, &examinerID)
	require.NoError(t, err)

	// Coba update nilai lagi saat terkunci -> harus ditolak ErrScoresLocked
	_, _, err = svc.InputScore(
		context.Background(),
		appID,
		"interview",
		95.0,
		"Mencoba mengubah nilai setelah dikunci",
		examinerID,
		[]string{"admin_pmb"},
	)
	assert.ErrorIs(t, err, service.ErrScoresLocked)
}

func TestProgramRecapAndScopeIsolation(t *testing.T) {
	compRepo := newMockCompRepo()
	scoreRepo := newMockScoreRepo()
	excelExp := exporter.NewExcelExporter()
	cfg := &config.Config{}

	svc := service.NewScoringService(cfg, compRepo, scoreRepo, excelExp, nil, nil, nil)
	periodID := uuid.New()
	pathID := uuid.New()

	// 1. Role prodi untuk program TIF mencoba akses TEI -> DITOLAK
	_, err := svc.GetProgramRecap(context.Background(), periodID, pathID, "TEI", []string{"prodi"}, []string{"TIF"})
	assert.ErrorIs(t, err, service.ErrForbiddenProgramAccess)

	// 2. Role prodi untuk program TIF mengakses TIF -> DIIZINKAN
	recap, err := svc.GetProgramRecap(context.Background(), periodID, pathID, "TIF", []string{"prodi"}, []string{"TIF"})
	assert.NoError(t, err)
	assert.Equal(t, "TIF", recap.StudyProgramCode)

	// 3. Role superadmin/admin_pmb mengakses prodi mana pun -> DIIZINKAN
	recapAdmin, err := svc.GetProgramRecap(context.Background(), periodID, pathID, "TEI", []string{"admin_pmb"}, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "TEI", recapAdmin.StudyProgramCode)

	// 4. Ekspor Excel
	xlsxBytes, err := svc.ExportProgramRecapXLSX(context.Background(), periodID, pathID, "TIF", []string{"admin_pmb"}, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, xlsxBytes)
}
