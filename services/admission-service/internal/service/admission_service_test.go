package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/outbox"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/config"
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

func (m *mockAppRepo) UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus model.ApplicationStatus, notes string, changedBy *uuid.UUID) error {
	if app, ok := m.apps[id]; ok {
		app.Status = toStatus
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

func (m *mockAppRepo) ListAllForExport(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, error) {
	var list []model.Application
	for _, app := range m.apps {
		list = append(list, *app)
	}
	return list, nil
}

func (m *mockAppRepo) GetStats(ctx context.Context) (*model.AdmissionStats, error) {
	return &model.AdmissionStats{
		TotalApplications: int64(len(m.apps)),
		ByStatus:          map[string]int64{"submitted": int64(len(m.apps))},
	}, nil
}

func (m *mockAppRepo) CreateStatusLog(tx *gorm.DB, log *model.ApplicationStatusLog) error {
	return nil
}

type mockDocRepo struct {
	docs map[string]*model.ApplicationDocument
}

func newMockDocRepo() *mockDocRepo {
	return &mockDocRepo{
		docs: make(map[string]*model.ApplicationDocument),
	}
}

func (m *mockDocRepo) Save(ctx context.Context, doc *model.ApplicationDocument) error {
	key := fmt.Sprintf("%s:%s", doc.ApplicationID.String(), doc.DocumentType)
	m.docs[key] = doc
	return nil
}

func (m *mockDocRepo) FindByApplicationAndType(ctx context.Context, appID uuid.UUID, docType string) (*model.ApplicationDocument, error) {
	key := fmt.Sprintf("%s:%s", appID.String(), docType)
	if doc, ok := m.docs[key]; ok {
		return doc, nil
	}
	return nil, repository.ErrDocumentNotFound
}

func (m *mockDocRepo) FindByApplicationID(ctx context.Context, appID uuid.UUID) ([]model.ApplicationDocument, error) {
	var list []model.ApplicationDocument
	for _, d := range m.docs {
		if d.ApplicationID == appID {
			list = append(list, *d)
		}
	}
	return list, nil
}

func (m *mockDocRepo) DeleteByApplicationAndType(ctx context.Context, appID uuid.UUID, docType string) error {
	key := fmt.Sprintf("%s:%s", appID.String(), docType)
	if _, ok := m.docs[key]; ok {
		delete(m.docs, key)
		return nil
	}
	return repository.ErrDocumentNotFound
}

func (m *mockDocRepo) UpdateReviewStatus(ctx context.Context, id uuid.UUID, status string, notes string, reviewerID *uuid.UUID) error {
	for _, d := range m.docs {
		if d.ID == id {
			d.Status = status
			d.Notes = notes
			d.VerifiedBy = reviewerID
			return nil
		}
	}
	return repository.ErrDocumentNotFound
}

type mockStorage struct {
	files       map[string][]byte
	lastExpiry  time.Duration
	rejectMagic bool
}

func (s *mockStorage) EnsureBucket(ctx context.Context, bucketName string) error { return nil }
func (s *mockStorage) Upload(ctx context.Context, bucketName, objectKey string, reader io.Reader, size int64, contentType string) error {
	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, reader)
	if s.files == nil {
		s.files = make(map[string][]byte)
	}
	s.files[objectKey] = buf.Bytes()
	return nil
}
func (s *mockStorage) Download(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error) {
	return nil, nil
}
func (s *mockStorage) PresignedGetObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error) {
	s.lastExpiry = expiry
	return fmt.Sprintf("https://minio.pmb.ac.id/%s/%s?exp=%d", bucketName, objectKey, int(expiry.Seconds())), nil
}
func (s *mockStorage) PresignedPutObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error) {
	return "", nil
}
func (s *mockStorage) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	delete(s.files, objectKey)
	return nil
}
func (s *mockStorage) SniffAndValidate(reader io.Reader, allowedTypes []string) (io.Reader, string, error) {
	if s.rejectMagic {
		return nil, "", storage.ErrFileTypeNotAllowed
	}
	buf := make([]byte, 512)
	n, _ := reader.Read(buf)
	fullReader := io.MultiReader(bytes.NewReader(buf[:n]), reader)
	return fullReader, "application/pdf", nil
}

type mockInternalCli struct {
	completenessResult bool
	missingFields      []string
}

func (c *mockInternalCli) Get(ctx context.Context, path string, query map[string]string, dest interface{}) error {
	if strings.Contains(path, "/completeness") {
		// Populate dest
		target, ok := dest.(*struct {
			Data struct {
				Complete bool     `json:"complete"`
				Missing  []string `json:"missing"`
			} `json:"data"`
		})
		if ok {
			target.Data.Complete = c.completenessResult
			target.Data.Missing = c.missingFields
			return nil
		}
	}
	return nil
}

func (c *mockInternalCli) Post(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return nil
}
func (c *mockInternalCli) Put(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return nil
}
func (c *mockInternalCli) Patch(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return nil
}
func (c *mockInternalCli) Delete(ctx context.Context, path string, dest interface{}) error {
	return nil
}
func (c *mockInternalCli) PostMultipart(ctx context.Context, path, fileField, filename string, fileReader io.Reader, extraFields map[string]string, dest interface{}) error {
	return nil
}

func TestAdmissionService_DocumentsAndSubmitFlow(t *testing.T) {
	ctx := context.Background()

	activePeriod := &model.Period{
		ID:           1,
		AcademicYear: "2026/2027",
		StartDate:    time.Now().Add(-24 * time.Hour),
		EndDate:      time.Now().Add(24 * time.Hour),
		IsActive:     true,
	}

	reqIjazah := model.DocumentRequirement{
		ID:              uuid.New(),
		AdmissionPathID: 1,
		DocumentType:    "ijazah",
		Name:            "Ijazah / SKL",
		IsRequired:      true,
		MaxSizeMB:       5,
		AllowedFormats:  "pdf,jpg,png",
	}

	reqKtp := model.DocumentRequirement{
		ID:              uuid.New(),
		AdmissionPathID: 1,
		DocumentType:    "ktp",
		Name:            "KTP Asli",
		IsRequired:      true,
		MaxSizeMB:       3,
		AllowedFormats:  "pdf,jpg,png",
	}

	regPath := &model.AdmissionPath{
		ID:              1,
		Name:            "Reguler",
		Code:            "REG",
		RegistrationFee: 250000,
		IsActive:        true,
		Requirements:    []model.DocumentRequirement{reqIjazah, reqKtp},
	}

	periodRepo := &mockPeriodRepo{period: activePeriod}
	pathRepo := &mockPathRepo{paths: map[uint]*model.AdmissionPath{1: regPath}}
	appRepo := newMockAppRepo()
	docRepo := newMockDocRepo()
	outboxRepo := &mockOutboxRepo{}
	storageCli := &mockStorage{files: make(map[string][]byte)}
	identityCli := &mockInternalCli{completenessResult: true}
	notifCli := &mockInternalCli{}
	financeCli := &mockInternalCli{}

	cfg := config.Config{
		DocumentBucket: "pmb-admission-documents",
	}

	svc := NewAdmissionService(
		periodRepo,
		pathRepo,
		&mockQuotaRepo{},
		appRepo,
		docRepo,
		outboxRepo,
		notifCli,
		identityCli,
		financeCli,
		storageCli,
		cfg,
	)

	userID := uuid.New()
	appID := uuid.New()
	app := &model.Application{
		ID:              appID,
		RegistrationNo:  "PMB-2026-1-00001",
		UserID:          userID,
		PeriodID:        1,
		AdmissionPathID: 1,
		Status:          model.StatusFormOpen,
	}
	_ = appRepo.CreateInTx(&gorm.DB{}, app)

	// 1. Upload dokumen legal (PDF)
	fileData := []byte("%PDF-1.4 dummy pdf content")
	doc, err := svc.UploadDocument(ctx, appID, userID, "ijazah", bytes.NewReader(fileData), int64(len(fileData)), "ijazah.pdf", "application/pdf")
	assert.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "ijazah", doc.DocumentType)
	assert.Equal(t, model.DocStatusUploaded, doc.Status)

	// 2. Upload ditolak jika magic bytes tidak sesuai (misal file .exe berkedok .pdf)
	storageCli.rejectMagic = true
	_, errExe := svc.UploadDocument(ctx, appID, userID, "ktp", bytes.NewReader([]byte("MZ executable header")), 20, "virus.pdf", "application/pdf")
	assert.Error(t, errExe)
	assert.True(t, errors.Is(errExe, storage.ErrFileTypeNotAllowed))
	storageCli.rejectMagic = false

	// 3. Submit ditolak jika biodata belum lengkap (BIODATA_INCOMPLETE)
	identityCli.completenessResult = false
	identityCli.missingFields = []string{"alamat ktp", "nama ibu"}
	_, errSubmitIncomplete := svc.SubmitApplication(ctx, appID, userID)
	assert.Error(t, errSubmitIncomplete)
	assert.Contains(t, errSubmitIncomplete.Error(), "BIODATA_INCOMPLETE")

	// 4. Submit ditolak jika dokumen wajib belum lengkap (DOCUMENT_REQUIRED)
	identityCli.completenessResult = true
	_, errSubmitMissingDoc := svc.SubmitApplication(ctx, appID, userID)
	assert.Error(t, errSubmitMissingDoc)
	assert.Contains(t, errSubmitMissingDoc.Error(), "DOCUMENT_REQUIRED")
	assert.Contains(t, errSubmitMissingDoc.Error(), "KTP Asli")

	// 5. Upload dokumen KTP agar lengkap
	docKtp, err := svc.UploadDocument(ctx, appID, userID, "ktp", bytes.NewReader(fileData), int64(len(fileData)), "ktp.pdf", "application/pdf")
	assert.NoError(t, err)
	assert.NotNil(t, docKtp)

	// 6. Submit sukses -> status bertransisi ke submitted
	submittedApp, err := svc.SubmitApplication(ctx, appID, userID)
	assert.NoError(t, err)
	assert.Equal(t, model.StatusSubmitted, submittedApp.Status)

	// 7. Admin PMB Verifikasi Berkas: Minta Revisi (salah satu invalid)
	adminID := uuid.New()
	reviewsRevision := []DocumentReviewItem{
		{DocumentType: "ijazah", Status: "valid", Notes: "Ijazah jelas"},
		{DocumentType: "ktp", Status: "invalid", Notes: "Foto KTP buram dan terpotong"},
	}
	appRev, err := svc.VerifyApplicationDocuments(ctx, appID, adminID, reviewsRevision)
	assert.NoError(t, err)
	assert.Equal(t, model.StatusRevisionRequired, appRev.Status)

	// 8. Mahasiswa mengunggah ulang dokumen revisi (re-upload KTP) dan re-submit
	docKtpNew, err := svc.UploadDocument(ctx, appID, userID, "ktp", bytes.NewReader(fileData), int64(len(fileData)), "ktp_jelas.pdf", "application/pdf")
	assert.NoError(t, err)
	assert.Equal(t, model.DocStatusUploaded, docKtpNew.Status)

	resubmittedApp, err := svc.SubmitApplication(ctx, appID, userID)
	assert.NoError(t, err)
	assert.Equal(t, model.StatusSubmitted, resubmittedApp.Status)

	// 9. Admin PMB Verifikasi Berkas: Lolos Verifikasi (seluruh wajib valid)
	reviewsPass := []DocumentReviewItem{
		{DocumentType: "ktp", Status: "valid", Notes: "KTP perbaikan telah jelas"},
	}
	appVerified, err := svc.VerifyApplicationDocuments(ctx, appID, adminID, reviewsPass)
	assert.NoError(t, err)
	assert.Equal(t, model.StatusVerified, appVerified.Status)

	// 10. Admin membaca Presigned URL dokumen: kedaluwarsa tepat 300 detik (5 menit)
	url, err := svc.GetDocumentPresignedURL(ctx, appID, "ktp", adminID, "admin_pmb")
	assert.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Equal(t, 300*time.Second, storageCli.lastExpiry)

	// 11. Ekspor data ke XLSX
	xlsxBytes, err := svc.ExportApplicationsXLSX(ctx, repository.ApplicationFilter{})
	assert.NoError(t, err)
	assert.NotEmpty(t, xlsxBytes)

	// 12. Statistik Pendaftaran
	stats, err := svc.GetStats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalApplications, int64(1))
}
