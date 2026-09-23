package route

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/crypto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/service"
)

type mockPersonRepo struct{}
func (m *mockPersonRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Person, error) { return nil, repository.ErrPersonNotFound }
func (m *mockPersonRepo) FindByNIKHash(ctx context.Context, nikHash string) (*model.Person, error) { return nil, repository.ErrPersonNotFound }
func (m *mockPersonRepo) Save(ctx context.Context, person *model.Person) error { return nil }
func (m *mockPersonRepo) UpdatePhoto(ctx context.Context, userID uuid.UUID, photoKey string) error { return nil }
func (m *mockPersonRepo) GetDetailWithRelations(ctx context.Context, userID uuid.UUID) (*model.Person, error) { return nil, repository.ErrPersonNotFound }
func (m *mockPersonRepo) FindByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]model.Person, error) { return nil, nil }

type mockAddressRepo struct{}
func (m *mockAddressRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Address, error) { return nil, nil }
func (m *mockAddressRepo) FindByUserIDAndType(ctx context.Context, userID uuid.UUID, addrType string) (*model.Address, error) { return nil, nil }
func (m *mockAddressRepo) Save(ctx context.Context, address *model.Address) error { return nil }

type mockGuardianRepo struct{}
func (m *mockGuardianRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Guardian, error) { return nil, nil }
func (m *mockGuardianRepo) FindByUserIDAndRelation(ctx context.Context, userID uuid.UUID, relation string) (*model.Guardian, error) { return nil, nil }
func (m *mockGuardianRepo) Save(ctx context.Context, guardian *model.Guardian) error { return nil }

type mockEducationRepo struct{}
func (m *mockEducationRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.EducationHistory, error) { return nil, nil }
func (m *mockEducationRepo) Save(ctx context.Context, edu *model.EducationHistory) error { return nil }
func (m *mockEducationRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error { return nil }

func TestRouter_Security(t *testing.T) {
	gatewaySecret := "secret-gw-key"
	internalToken := "secret-internal-key"

	key := crypto.DeriveKey("test-secret-key-32-bytes-abcdef0123")
	personRepo := &mockPersonRepo{}
	addressRepo := &mockAddressRepo{}
	guardianRepo := &mockGuardianRepo{}
	eduRepo := &mockEducationRepo{}

	identitySvc := service.NewIdentityService(personRepo, addressRepo, guardianRepo, eduRepo, nil, key, "pmb-identity-photos")
	compSvc := service.NewCompletenessService(personRepo)

	healthCtrl := controller.NewHealthController(nil)
	userCtrl := controller.NewUserController(identitySvc, compSvc, nil)
	adminCtrl := controller.NewAdminController(identitySvc, compSvc)
	internalCtrl := controller.NewInternalController(identitySvc, compSvc)

	r := SetupRouter(RouterConfig{
		GatewaySecret: gatewaySecret,
		InternalToken: internalToken,
	}, healthCtrl, userCtrl, adminCtrl, internalCtrl)

	// 1. Health check tanpa auth -> 200 OK
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Health check harus 200 OK, didapat: %d", w.Code)
	}

	// 2. Request ke /identity/me/profile tanpa gateway signature -> 401 GATEWAY_SIGNATURE_INVALID
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/identity/me/profile", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Bypass gateway harus 401 Unauthorized, didapat: %d", w.Code)
	}

	// 3. Request ke /identity/me/profile dengan gateway signature valid -> 200 / 404
	userID := uuid.New()
	reqID := "req-test-123"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := gatewayauth.GenerateSignature(gatewaySecret, reqID, userID.String(), timestamp)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/identity/me/profile", nil)
	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, timestamp)
	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderUserID, userID.String())
	req.Header.Set(gatewayauth.HeaderUserRoles, "calon_mahasiswa")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
		t.Fatalf("Signature valid harus lolos middleware (status 404 data kosong / 200), didapat: %d", w.Code)
	}

	// 4. Request ke /identity/admin/persons/:id oleh calon_mahasiswa -> 403 Forbidden
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/identity/admin/persons/"+userID.String(), nil)
	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, timestamp)
	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderUserID, userID.String())
	req.Header.Set(gatewayauth.HeaderUserRoles, "calon_mahasiswa")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Calon mahasiswa akses admin harus 403 Forbidden, didapat: %d", w.Code)
	}

	// 5. Internal endpoint tanpa internal token -> 403 Forbidden
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/internal/persons/"+userID.String(), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Internal endpoint tanpa token harus 403, didapat: %d", w.Code)
	}

	// 6. Internal endpoint dengan internal token valid -> 404 (data not found)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/internal/persons/"+userID.String(), nil)
	req.Header.Set("X-Internal-Token", internalToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Internal endpoint dengan token valid harus 404 data tidak ada, didapat: %d", w.Code)
	}
}
