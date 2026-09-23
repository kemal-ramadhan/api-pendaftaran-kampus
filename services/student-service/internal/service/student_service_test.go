package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Mock repository in-memory berbasis map / mock untuk pengujian
type mockStudentRepository struct {
	reregistrations map[uuid.UUID]*model.Reregistration
	students        map[uuid.UUID]*model.Student
	outboxEvents    []*model.OutboxEvent
	db              *gorm.DB
}

func newMockRepository() *mockStudentRepository {
	// Buat koneksi dummy gorm jika diperlukan atau gorm nil dengan custom mock
	return &mockStudentRepository{
		reregistrations: make(map[uuid.UUID]*model.Reregistration),
		students:        make(map[uuid.UUID]*model.Student),
		outboxEvents:    make([]*model.OutboxEvent, 0),
	}
}

func (m *mockStudentRepository) DB() *gorm.DB {
	// Fallback DB jika tidak ada koneksi postgres nyata
	if m.db == nil {
		dialector := postgres.New(postgres.Config{
			DriverName: "postgres",
			DSN:        "user=pmb_user password=pmb_password dbname=pmb_student sslmode=disable",
		})
		db, _ := gorm.Open(dialector, &gorm.Config{})
		m.db = db
	}
	return m.db
}

func (m *mockStudentRepository) GetReregistrationByUserID(ctx context.Context, userID uuid.UUID) (*model.Reregistration, error) {
	for _, r := range m.reregistrations {
		if r.UserID == userID {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockStudentRepository) GetReregistrationByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Reregistration, error) {
	if r, ok := m.reregistrations[appID]; ok {
		return r, nil
	}
	return nil, nil
}

func (m *mockStudentRepository) CreateReregistration(ctx context.Context, r *model.Reregistration) error {
	m.reregistrations[r.ApplicationID] = r
	return nil
}

func (m *mockStudentRepository) UpdateReregistration(ctx context.Context, r *model.Reregistration) error {
	m.reregistrations[r.ApplicationID] = r
	return nil
}

func (m *mockStudentRepository) UpdateReregistrationInTx(tx *gorm.DB, r *model.Reregistration) error {
	m.reregistrations[r.ApplicationID] = r
	return nil
}

func (m *mockStudentRepository) CreateStudentInTx(tx *gorm.DB, s *model.Student) error {
	m.students[s.ApplicationID] = s
	return nil
}

func (m *mockStudentRepository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*model.Student, error) {
	for _, s := range m.students {
		if s.UserID == userID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockStudentRepository) GetStudentByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Student, error) {
	if s, ok := m.students[appID]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockStudentRepository) ListStudents(ctx context.Context, filter model.StudentFilter) ([]model.Student, int64, error) {
	var list []model.Student
	for _, s := range m.students {
		if filter.StudyProgramCode != "" && s.StudyProgramCode != filter.StudyProgramCode {
			continue
		}
		list = append(list, *s)
	}
	return list, int64(len(list)), nil
}

func (m *mockStudentRepository) UpdateStudentNIMInTx(tx *gorm.DB, appID uuid.UUID, nim string) error {
	if s, ok := m.students[appID]; ok {
		s.NIM = nim
		s.Status = model.StudentStatusActive
	}
	return nil
}

func (m *mockStudentRepository) CreateOutboxEventInTx(tx *gorm.DB, event *model.OutboxEvent) error {
	m.outboxEvents = append(m.outboxEvents, event)
	return nil
}

// Unit Test Logika Bisnis & Validasi
func TestStudentService_BusinessValidation(t *testing.T) {
	mockRepo := newMockRepository()
	exp := exporter.NewStudentExporter()
	svc := NewStudentService(mockRepo, exp, nil, nil, nil)
	ctx := context.Background()

	userID := uuid.New()
	appID := uuid.New()

	t.Run("calon mahasiswa belum lulus seleksi -> tolak dengan NOT_PASSED", func(t *testing.T) {
		// Ketika klien eksternal mengembalikan nil (belum ada pendaftaran / belum lulus)
		_, err := svc.GetMyReregistration(ctx, userID)
		assert.ErrorIs(t, err, ErrNotPassed)
	})

	t.Run("calon mahasiswa lulus seleksi -> pendaftaran ulang dibuat otomatis", func(t *testing.T) {
		// Simulasikan pendaftaran ulang tersimpan di repo
		reg := &model.Reregistration{
			ID:              uuid.New(),
			ApplicationID:   appID,
			UserID:          userID,
			AcceptedProgram: "TI",
			Status:          model.ReregistrationStatusWaitingPayment,
			StatementAgreed: false,
		}
		err := mockRepo.CreateReregistration(ctx, reg)
		require.NoError(t, err)

		detail, err := svc.GetMyReregistration(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "TI", detail.Reregistration.AcceptedProgram)
		assert.False(t, detail.CanComplete) // Belum lengkap dan belum bayar
	})

	t.Run("update data tambahan daftar ulang", func(t *testing.T) {
		agreed := true
		req := &model.UpdateReregistrationRequest{
			JacketSize:            "XL",
			EmergencyContactName:  "Bapak Sudirman",
			EmergencyContactPhone: "081234567890",
			StatementAgreed:       &agreed,
		}
		detail, err := svc.UpdateMyReregistration(ctx, userID, req)
		require.NoError(t, err)
		assert.Equal(t, "XL", detail.Reregistration.JacketSize)
		assert.Equal(t, "Bapak Sudirman", detail.Reregistration.EmergencyContactName)
		assert.True(t, detail.Reregistration.StatementAgreed)
		assert.Equal(t, model.ReregistrationStatusInProgress, detail.Reregistration.Status)
	})

	t.Run("ukuran jaket tidak valid ditolak", func(t *testing.T) {
		agreed := true
		req := &model.UpdateReregistrationRequest{
			JacketSize:            "INVALID_SIZE",
			EmergencyContactName:  "Bapak Sudirman",
			EmergencyContactPhone: "081234567890",
			StatementAgreed:       &agreed,
		}
		_, err := svc.UpdateMyReregistration(ctx, userID, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ukuran jas almamater tidak valid")
	})

	t.Run("isolasi role program studi", func(t *testing.T) {
		// Prodi TI mengakses prodi SI -> harus ditolak
		filter := model.StudentFilter{StudyProgramCode: "SI"}
		_, _, err := svc.ListStudents(ctx, filter, []string{"prodi"}, []string{"TI"})
		assert.ErrorIs(t, err, ErrForbiddenProdiAccess)

		// Prodi TI mengakses prodi TI -> diizinkan
		filterOK := model.StudentFilter{StudyProgramCode: "TI"}
		_, _, errOK := svc.ListStudents(ctx, filterOK, []string{"prodi"}, []string{"TI"})
		assert.NoError(t, errOK)

		// Superadmin / Pimpinan bebas mengakses prodi mana pun
		_, _, errAdmin := svc.ListStudents(ctx, filter, []string{"pimpinan"}, nil)
		assert.NoError(t, errAdmin)
	})

	t.Run("ekspor spreadsheet Excel XLSX", func(t *testing.T) {
		// Tambahkan satu mahasiswa baru contoh
		mockRepo.students[appID] = &model.Student{
			ID:               uuid.New(),
			UserID:           userID,
			ApplicationID:    appID,
			NIM:              "265510001",
			StudyProgramCode: "TI",
			EntryYear:        2026,
			EntryPath:        "Reguler",
			Status:           model.StudentStatusActive,
			CreatedAt:        time.Now(),
		}

		xlsxBytes, err := svc.ExportStudentsXLSX(ctx, model.StudentFilter{}, []string{"admin_pmb"}, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, xlsxBytes)
		// Verifikasi header OpenXML / ZIP (PK)
		assert.Equal(t, byte('P'), xlsxBytes[0])
		assert.Equal(t, byte('K'), xlsxBytes[1])
	})
}
