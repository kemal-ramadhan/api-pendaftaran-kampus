package route

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/service"
	"github.com/stretchr/testify/assert"
)

type dummySessionRepo struct{}

func (d *dummySessionRepo) Create(ctx context.Context, session *model.ExamSession) error { return nil }
func (d *dummySessionRepo) Update(ctx context.Context, session *model.ExamSession) error { return nil }
func (d *dummySessionRepo) Delete(ctx context.Context, id uuid.UUID) error               { return nil }
func (d *dummySessionRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.ExamSession, error) {
	return &model.ExamSession{ID: id, Title: "Test Session"}, nil
}
func (d *dummySessionRepo) List(ctx context.Context, periodID, pathID *uuid.UUID, sessionType *string, isActive *bool) ([]model.ExamSession, error) {
	return []model.ExamSession{}, nil
}
func (d *dummySessionRepo) AddExaminer(ctx context.Context, examiner *model.SessionExaminer) error {
	return nil
}
func (d *dummySessionRepo) RemoveExaminer(ctx context.Context, sessionID, examinerUserID uuid.UUID) error {
	return nil
}
func (d *dummySessionRepo) GetExaminersBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.SessionExaminer, error) {
	return []model.SessionExaminer{}, nil
}
func (d *dummySessionRepo) GetSessionsByExaminerID(ctx context.Context, examinerUserID uuid.UUID) ([]model.ExamSession, error) {
	return []model.ExamSession{}, nil
}
func (d *dummySessionRepo) IsExaminerAssigned(ctx context.Context, sessionID, examinerUserID uuid.UUID) (bool, error) {
	return true, nil
}

type dummyParticipantRepo struct{}

func (d *dummyParticipantRepo) AddParticipant(ctx context.Context, p *model.Participant) error {
	return nil
}
func (d *dummyParticipantRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Participant, error) {
	return &model.Participant{ID: id}, nil
}
func (d *dummyParticipantRepo) FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Participant, error) {
	return nil, nil
}
func (d *dummyParticipantRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Participant, error) {
	return []model.Participant{}, nil
}
func (d *dummyParticipantRepo) FindBySessionAndUser(ctx context.Context, sessionID, userID uuid.UUID) (*model.Participant, error) {
	return &model.Participant{SessionID: sessionID, UserID: userID}, nil
}
func (d *dummyParticipantRepo) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.Participant, error) {
	return []model.Participant{}, nil
}
func (d *dummyParticipantRepo) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	return 0, nil
}
func (d *dummyParticipantRepo) UpdateAttendance(ctx context.Context, participantID uuid.UUID, attendance string) error {
	return nil
}
func (d *dummyParticipantRepo) GetNextParticipantSequence(ctx context.Context, year int) (int64, error) {
	return 1, nil
}
func (d *dummyParticipantRepo) SaveExamCard(ctx context.Context, card *model.ExamCard) error {
	return nil
}
func (d *dummyParticipantRepo) FindCardByParticipantID(ctx context.Context, participantID uuid.UUID) (*model.ExamCard, error) {
	return nil, nil
}
func (d *dummyParticipantRepo) RemoveByApplicationID(ctx context.Context, appID uuid.UUID) error {
	return nil
}


func setGatewayAuthHeaders(req *http.Request, secret, reqID, userID, role string) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := gatewayauth.GenerateSignature(secret, reqID, userID, timestamp)

	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, timestamp)
	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderUserID, userID)
	req.Header.Set(gatewayauth.HeaderUserRoles, role)
	req.Header.Set("X-User-Role", role)
}

func TestExamRouter_AuthAndPublicEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretKey := "test-gateway-secret-minimum-32-bytes-long!"
	internalToken := "test-internal-token"

	cfg := RouterConfig{
		GatewaySecret: secretKey,
		InternalToken: internalToken,
		IsDevelopment: true,
	}

	appCfg := &config.Config{ExamCardBucket: "pmb-exam-cards"}
	examSvc := service.NewExamService(appCfg, &dummySessionRepo{}, &dummyParticipantRepo{}, nil, nil, nil, nil)

	healthCtrl := controller.NewHealthController()
	sessionCtrl := controller.NewSessionController(examSvc)
	partCtrl := controller.NewParticipantController(examSvc)
	studentCtrl := controller.NewStudentController(examSvc)
	examinerCtrl := controller.NewExaminerController(examSvc)
	internalCtrl := controller.NewInternalController(examSvc)
	cbtCtrl := controller.NewCBTController(nil)

	router := SetupRouter(cfg, healthCtrl, sessionCtrl, partCtrl, studentCtrl, examinerCtrl, internalCtrl, cbtCtrl)

	// 1. Health check tanpa auth -> HTTP 200
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 1b. Time check publik tanpa auth (CR-003) -> HTTP 200
	req, _ = http.NewRequest(http.MethodGet, "/time", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. Akses endpoint mahasiswa tanpa gateway signature -> HTTP 401
	req, _ = http.NewRequest(http.MethodGet, "/exam/me/schedule", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 3. Akses endpoint mahasiswa dengan gateway signature valid -> HTTP 200
	req, _ = http.NewRequest(http.MethodGet, "/exam/me/schedule", nil)
	userID := uuid.New().String()
	reqID := uuid.New().String()
	setGatewayAuthHeaders(req, secretKey, reqID, userID, "calon_mahasiswa")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Role calon_mahasiswa mengakses endpoint admin -> HTTP 403 Forbidden
	req, _ = http.NewRequest(http.MethodGet, "/exam/admin/sessions", nil)
	setGatewayAuthHeaders(req, secretKey, reqID, userID, "calon_mahasiswa")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 5. Endpoint internal tanpa X-Internal-Token -> HTTP 403 Forbidden
	req, _ = http.NewRequest(http.MethodGet, "/internal/sessions/"+uuid.New().String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 6. Endpoint internal dengan X-Internal-Token valid -> HTTP 200
	req, _ = http.NewRequest(http.MethodGet, "/internal/sessions/"+uuid.New().String(), nil)
	req.Header.Set(internalauth.HeaderInternalToken, internalToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
