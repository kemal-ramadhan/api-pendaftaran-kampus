package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// mockConcurrentRepo mengemulasikan SELECT ... FOR UPDATE database menggunakan Go Mutex
type mockConcurrentRepo struct {
	mu        sync.Mutex
	sequences map[string]int
	issuances map[uuid.UUID]*model.NIMIssuance
	formats   map[string]*model.NIMFormat
}

func newMockConcurrentRepo() *mockConcurrentRepo {
	return &mockConcurrentRepo{
		sequences: make(map[string]int),
		issuances: make(map[uuid.UUID]*model.NIMIssuance),
		formats:   make(map[string]*model.NIMFormat),
	}
}

func (m *mockConcurrentRepo) DB() *gorm.DB {
	// DB nyata tidak dipanggil karena di-mock di level method repositori
	return &gorm.DB{}
}

func (m *mockConcurrentRepo) GetIssuanceByApplicationID(_ context.Context, appID uuid.UUID) (*model.NIMIssuance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if item, ok := m.issuances[appID]; ok {
		return item, nil
	}
	return nil, nil
}

func (m *mockConcurrentRepo) GetIssuanceByNIM(_ context.Context, nim string) (*model.NIMIssuance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range m.issuances {
		if item.NIM == nim {
			return item, nil
		}
	}
	return nil, nil
}

func (m *mockConcurrentRepo) CreateIssuanceInTx(_ context.Context, _ *gorm.DB, issuance *model.NIMIssuance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.issuances[issuance.ApplicationID] = issuance
	return nil
}

func (m *mockConcurrentRepo) ListIssuances(_ context.Context, _ model.IssuanceFilter) ([]model.NIMIssuance, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []model.NIMIssuance
	for _, item := range m.issuances {
		list = append(list, *item)
	}
	return list, int64(len(list)), nil
}

func (m *mockConcurrentRepo) GetNextSequenceWithLock(_ context.Context, _ *gorm.DB, entryYear int, studyProgramCode string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%d_%s", entryYear, studyProgramCode)
	current := m.sequences[key]
	next := current + 1
	m.sequences[key] = next
	return next, nil
}

func (m *mockConcurrentRepo) GetFormatForProgram(_ context.Context, studyProgramCode string) (*model.NIMFormat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fmtItem, ok := m.formats[studyProgramCode]; ok {
		return fmtItem, nil
	}
	return &model.NIMFormat{
		Pattern:  "{YY}{nim_code_prodi}{jalur_digit}{urut:4}",
		IsActive: true,
	}, nil
}

func (m *mockConcurrentRepo) ListFormats(_ context.Context) ([]model.NIMFormat, error) {
	return nil, nil
}

func (m *mockConcurrentRepo) ListActiveFormats(_ context.Context) ([]model.NIMFormat, error) {
	return nil, nil
}

func (m *mockConcurrentRepo) GetFormatByID(_ context.Context, _ uuid.UUID) (*model.NIMFormat, error) {
	return nil, nil
}

func (m *mockConcurrentRepo) CreateFormat(_ context.Context, _ *model.NIMFormat) error {
	return nil
}

func (m *mockConcurrentRepo) UpdateFormat(_ context.Context, _ *model.NIMFormat) error {
	return nil
}

func (m *mockConcurrentRepo) VoidIssuance(_ context.Context, appID uuid.UUID) (*model.NIMIssuance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if item, ok := m.issuances[appID]; ok {
		item.Status = model.NIMIssuanceStatusVoided
		return item, nil
	}
	return nil, nil
}


// -----------------------------------------------------------------------------
// Test Suites
// -----------------------------------------------------------------------------

func TestNIMService_ConcurrencyAndIdempotency(t *testing.T) {
	cfg := &config.Config{
		AppEnv:        "test",
		Port:          "8092",
		IsDevelopment: true,
	}

	repo := newMockConcurrentRepo()
	svc := NewNIMService(cfg, repo, nil, nil, nil, nil, nil)

	// 1. Pengujian Idempotensi
	t.Run("Penerbitan NIM bersifat idempoten untuk application_id yang sama", func(t *testing.T) {
		appID := uuid.New()
		userID := uuid.New()

		req := &model.IssueNIMRequest{
			ApplicationID:    appID,
			UserID:           userID,
			StudyProgramCode: "TI",
			EntryYear:        2026,
			SelectionPath:    "reguler",
		}

		res1, err1 := svc.IssueNIM(context.Background(), req)
		assert.NoError(t, err1)
		assert.NotNil(t, res1)
		assert.False(t, res1.IsIdempotent)
		assert.Equal(t, "265510001", res1.NIM)
		assert.Equal(t, 1, res1.SequenceNumber)

		// Request kedua dengan application_id yang sama persis
		res2, err2 := svc.IssueNIM(context.Background(), req)
		assert.NoError(t, err2)
		assert.NotNil(t, res2)
		assert.True(t, res2.IsIdempotent)
		assert.Equal(t, res1.NIM, res2.NIM)
		assert.Equal(t, res1.SequenceNumber, res2.SequenceNumber)
	})

	// 2. Pengujian Konkurensi: 50 Permintaan NIM Bersamaan Tanpa NIM Ganda
	t.Run("50 permintaan NIM bersamaan menghasilkan nomor urut unik tanpa duplikasi", func(t *testing.T) {
		const totalRequests = 50
		var wg sync.WaitGroup
		wg.Add(totalRequests)

		startBarrier := make(chan struct{})
		results := make(chan string, totalRequests)
		errorsChan := make(chan error, totalRequests)

		for i := 0; i < totalRequests; i++ {
			go func(idx int) {
				defer wg.Done()
				// Tunggu sinyal mulai serentak
				<-startBarrier

				req := &model.IssueNIMRequest{
					ApplicationID:    uuid.New(),
					UserID:           uuid.New(),
					StudyProgramCode: "SI",
					EntryYear:        2026,
					SelectionPath:    "prestasi",
				}

				res, err := svc.IssueNIM(context.Background(), req)
				if err != nil {
					errorsChan <- err
					return
				}
				results <- res.NIM
			}(i)
		}

		// Lepaskan barrier untuk memicu 50 goroutine secara serentak
		close(startBarrier)
		wg.Wait()
		close(results)
		close(errorsChan)

		// Pastikan tidak ada error selama konkurensi
		for err := range errorsChan {
			assert.NoError(t, err)
		}

		// Kumpulkan seluruh NIM dan verifikasi keunikan
		seenNIMs := make(map[string]bool)
		for nim := range results {
			assert.False(t, seenNIMs[nim], "Ditemukan duplikasi NIM: %s", nim)
			seenNIMs[nim] = true
			// Verifikasi format: tahun 26, prodi SI 57, jalur prestasi 2, 4 digit urut
			assert.Contains(t, nim, "26572")
		}

		// Pastikan jumlah NIM unik tepat 50
		assert.Equal(t, totalRequests, len(seenNIMs), "Jumlah NIM unik harus tepat 50")
	})

	t.Run("void issued NIM sets status to voided", func(t *testing.T) {
		appID := uuid.New()
		req := &model.IssueNIMRequest{
			ApplicationID:    appID,
			UserID:           uuid.New(),
			StudyProgramCode: "TI",
			EntryYear:        2026,
			SelectionPath:    "reguler",
		}

		res, err := svc.IssueNIM(context.Background(), req)
		assert.NoError(t, err)
		assert.NotEmpty(t, res.NIM)

		voided, err := svc.VoidNIM(context.Background(), appID)
		assert.NoError(t, err)
		assert.Equal(t, model.NIMIssuanceStatusVoided, voided.Status)
	})
}

func TestRenderNIMSK065_AllCases(t *testing.T) {
	tests := []struct {
		name         string
		fakultasCode string
		prodiCode    string
		kkniCode     string
		jalurCode    string
		year         int
		semesterCode string
		urut         int
		expected     string
	}{
		{
			name:         "1. S1 Farmasi Reguler 2026 Ganjil Urut 1",
			fakultasCode: "01",
			prodiCode:    "01",
			kkniCode:     "7",
			jalurCode:    "1",
			year:         2026,
			semesterCode: "1",
			urut:         1,
			expected:     "010171261001",
		},
		{
			name:         "2. S1 Farmasi RPL 2026 Ganjil Urut 1",
			fakultasCode: "01",
			prodiCode:    "01",
			kkniCode:     "7",
			jalurCode:    "2",
			year:         2026,
			semesterCode: "1",
			urut:         1,
			expected:     "010172261001",
		},
		{
			name:         "3. D3 Farmasi Reguler 2026 Ganjil Urut 1",
			fakultasCode: "01",
			prodiCode:    "02",
			kkniCode:     "5",
			jalurCode:    "1",
			year:         2026,
			semesterCode: "1",
			urut:         1,
			expected:     "010251261001",
		},
		{
			name:         "4. D3 TLM Reguler 2026 Ganjil Urut 1",
			fakultasCode: "01",
			prodiCode:    "03",
			kkniCode:     "5",
			jalurCode:    "1",
			year:         2026,
			semesterCode: "1",
			urut:         1,
			expected:     "010351261001",
		},
		{
			name:         "5. D3 RMIK Reguler 2026 Ganjil Urut 1",
			fakultasCode: "01",
			prodiCode:    "04",
			kkniCode:     "5",
			jalurCode:    "1",
			year:         2026,
			semesterCode: "1",
			urut:         1,
			expected:     "010451261001",
		},
		{
			name:         "6. S1 Manajemen Reguler 2026 Ganjil Urut 1",
			fakultasCode: "02",
			prodiCode:    "01",
			kkniCode:     "7",
			jalurCode:    "1",
			year:         2026,
			semesterCode: "1",
			urut:         1,
			expected:     "020171261001",
		},
		{
			name:         "7. S1 Bisnis Digital Reguler 2026 Ganjil Urut 1",
			fakultasCode: "02",
			prodiCode:    "02",
			kkniCode:     "7",
			jalurCode:    "1",
			year:         2026,
			semesterCode: "1",
			urut:         1,
			expected:     "020271261001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := RenderNIMSK065(DefaultNIMPattern, tt.fakultasCode, tt.prodiCode, tt.kkniCode, tt.jalurCode, tt.year, tt.semesterCode, tt.urut)
			assert.Equal(t, tt.expected, res)
		})
	}
}

