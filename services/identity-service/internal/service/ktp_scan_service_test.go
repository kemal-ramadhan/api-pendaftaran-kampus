package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/crypto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/repository"
)

// Mock KTP Scan Repository
type mockKTPScanRepo struct {
	scans map[uuid.UUID]*model.KTPScan
}

func newMockKTPScanRepo() *mockKTPScanRepo {
	return &mockKTPScanRepo{scans: make(map[uuid.UUID]*model.KTPScan)}
}

func (m *mockKTPScanRepo) Create(ctx context.Context, scan *model.KTPScan) error {
	m.scans[scan.ID] = scan
	return nil
}

func (m *mockKTPScanRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.KTPScan, error) {
	s, ok := m.scans[id]
	if !ok {
		return nil, repository.ErrKTPScanNotFound
	}
	return s, nil
}

func (m *mockKTPScanRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.KTPScan, error) {
	var list []model.KTPScan
	for _, s := range m.scans {
		if s.UserID == userID {
			list = append(list, *s)
		}
	}
	return list, nil
}

func (m *mockKTPScanRepo) Update(ctx context.Context, scan *model.KTPScan) error {
	m.scans[scan.ID] = scan
	return nil
}

func (m *mockKTPScanRepo) CountScansSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	for _, s := range m.scans {
		if s.UserID == userID && s.CreatedAt.After(since) {
			count++
		}
	}
	return count, nil
}

// Mock Internal OCR Client
type mockOCRClient struct {
	jobResult map[string]interface{}
}

func (m *mockOCRClient) Get(ctx context.Context, path string, query map[string]string, dest interface{}) error {
	return nil
}

func (m *mockOCRClient) Post(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return nil
}

func (m *mockOCRClient) PostMultipart(ctx context.Context, path string, fieldName, fileName string, fileReader io.Reader, extraFields map[string]string, dest interface{}) error {
	if res, ok := dest.(*struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}); ok {
		res.JobID = "job-mock-ocr-123"
		res.Status = "processing"
	}
	return nil
}

func (m *mockOCRClient) Put(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return nil
}

func (m *mockOCRClient) Patch(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return nil
}

func (m *mockOCRClient) Delete(ctx context.Context, path string, dest interface{}) error {
	return nil
}

func TestKTPScanService_Flow(t *testing.T) {
	key := crypto.DeriveKey("test-secret-key-32-bytes-abcdef0123")
	personRepo := newMockPersonRepo()
	addressRepo := newMockAddressRepo()
	guardianRepo := newMockGuardianRepo()
	eduRepo := newMockEducationRepo()
	ktpScanRepo := newMockKTPScanRepo()

	identitySvc := NewIdentityService(personRepo, addressRepo, guardianRepo, eduRepo, nil, key, "pmb-identity-photos")
	ocrMock := &mockOCRClient{}

	scanSvc := NewKTPScanService(ktpScanRepo, identitySvc, nil, ocrMock, "pmb-identity-ktp")
	ctx := context.Background()

	user := uuid.New()

	// 1. Inisiasi scan KTP
	imgContent := []byte("fake-ktp-image-binary-data")
	initResp, err := scanSvc.InitiateKTPScan(ctx, user, bytes.NewReader(imgContent), int64(len(imgContent)), "ktp.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("InitiateKTPScan gagal: %v", err)
	}

	if initResp.ScanID == "" || initResp.Status != "processing" {
		t.Fatalf("Inisiasi scan tidak sesuai: %+v", initResp)
	}

	scanUUID, _ := uuid.Parse(initResp.ScanID)

	// Simulasikan hasil OCR tersimpan di record
	scanRecord, _ := ktpScanRepo.FindByID(ctx, scanUUID)
	scanRecord.Status = "completed"
	scanRecord.QualityScore = 92.5
	scanRecord.OCRDataRaw = `{"nik":{"value":"3201015505050002","confidence":0.98,"needs_review":false},"nama":{"value":"AHMAD SYAFIQ","confidence":0.95,"needs_review":false},"tanggal_lahir":{"value":"2005-05-15","confidence":0.99,"needs_review":false}}`
	_ = ktpScanRepo.Update(ctx, scanRecord)

	// 2. Ambil status scan
	statusResp, err := scanSvc.GetKTPScanStatus(ctx, scanUUID, user)
	if err != nil {
		t.Fatalf("GetKTPScanStatus gagal: %v", err)
	}

	if statusResp.Status != "completed" || statusResp.Fields == nil || statusResp.Fields.NIK.Value != "3201015505050002" {
		t.Fatalf("Status response tidak sesuai: %+v", statusResp)
	}

	// 3. User lain coba akses scan -> 403 Forbidden / ErrUnauthorizedAccess
	otherUser := uuid.New()
	_, err = scanSvc.GetKTPScanStatus(ctx, scanUUID, otherUser)
	if !errors.Is(err, ErrUnauthorizedAccess) {
		t.Fatalf("User lain tidak boleh akses scan: %v", err)
	}

	// 4. Konfirmasi scan (user mengedit nama sedikit: "AHMAD SYAFIQ M")
	confirmReq := dto.KTPConfirmRequest{
		FullName:   "AHMAD SYAFIQ M",
		NIK:        "3201015505050002",
		BirthPlace: "Bandung",
		BirthDate:  "2005-05-15",
		Gender:     "L",
		Religion:   "Islam",
		Phone:      "081234567890",
		Address: dto.AddressRequest{
			Type:       "ktp",
			Street:     "Jl. Sukajadi No. 12",
			RT:         "02",
			RW:         "03",
			Village:    "Sukagalih",
			District:   "Sukajadi",
			City:       "Bandung",
			Province:   "Jawa Barat",
			PostalCode: "40163",
		},
	}

	profileResp, err := scanSvc.ConfirmKTPScan(ctx, scanUUID, user, confirmReq)
	if err != nil {
		t.Fatalf("ConfirmKTPScan gagal: %v", err)
	}

	if profileResp.FullName != "AHMAD SYAFIQ M" || profileResp.NIK != "3201015505050002" {
		t.Fatalf("Data profil hasil konfirmasi tidak cocok: %+v", profileResp)
	}

	// Pastikan confirmed_at terisi dan fields_edited_by_user mencatat 'nama'
	updatedScan, _ := ktpScanRepo.FindByID(ctx, scanUUID)
	if updatedScan.ConfirmedAt == nil {
		t.Fatal("ConfirmedAt harus terisi")
	}
	if updatedScan.FieldsEditedByUser == "" {
		t.Fatal("FieldsEditedByUser harus mencatat perbedaan")
	}

	// 5. Coba konfirmasi ulang -> harus ditolak ErrScanAlreadyConfirmed
	_, err = scanSvc.ConfirmKTPScan(ctx, scanUUID, user, confirmReq)
	if !errors.Is(err, ErrScanAlreadyConfirmed) {
		t.Fatalf("Konfirmasi ulang harus ditolak: %v", err)
	}
}
