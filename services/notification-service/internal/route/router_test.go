package route

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/service"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/template"
	"github.com/stretchr/testify/assert"
)

type dummyRepo struct {
	data map[string]*model.NotificationLog
}

func (d *dummyRepo) Create(ctx context.Context, log *model.NotificationLog) error {
	d.data[log.IdempotencyKey] = log
	return nil
}
func (d *dummyRepo) FindByIdempotencyKey(ctx context.Context, key string) (*model.NotificationLog, error) {
	if l, ok := d.data[key]; ok {
		return l, nil
	}
	return nil, nil
}
func (d *dummyRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.NotificationLog, error) {
	return nil, nil
}
func (d *dummyRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, retries int, errorMsg string) error {
	return nil
}
func (d *dummyRepo) GetPendingQueued(ctx context.Context, limit int) ([]model.NotificationLog, error) {
	return nil, nil
}

type dummyEmailSender struct{}

func (d *dummyEmailSender) Send(ctx context.Context, to, subject, htmlBody string) error {
	return nil
}

type dummyWASender struct{}

func (d *dummyWASender) Send(ctx context.Context, phone, message string) error {
	return nil
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := RouterConfig{
		InternalToken: "valid-secret-token",
	}

	repo := &dummyRepo{data: make(map[string]*model.NotificationLog)}
	notifSvc := service.NewNotificationService(
		repo,
		&dummyEmailSender{},
		&dummyWASender{},
		template.NewTemplateEngine(),
	)

	healthCtrl := controller.NewHealthController()
	notifCtrl := controller.NewNotificationController(notifSvc)

	return SetupRouter(cfg, healthCtrl, notifCtrl)
}

func TestRouter_HealthCheck(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_InternalNotifications_Auth(t *testing.T) {
	r := setupTestRouter()

	t.Run("unauthorized without internal token returns 403", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/internal/notifications", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("authorized with valid internal token", func(t *testing.T) {
		payload := map[string]interface{}{
			"channel":  "email",
			"email":    "test@pmb.ac.id",
			"template": "email_verification",
			"data": map[string]interface{}{
				"name": "Testing",
				"code": "112233",
			},
		}
		bodyBytes, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/internal/notifications", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Token", "valid-secret-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
	})
}
