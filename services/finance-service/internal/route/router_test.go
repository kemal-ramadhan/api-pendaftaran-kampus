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
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
	"github.com/stretchr/testify/assert"
)

type dummyInvoiceSvc struct {
	service.InvoiceService
}

func (d *dummyInvoiceSvc) GetMyInvoices(ctx context.Context, userID uuid.UUID) ([]model.Invoice, error) {
	return []model.Invoice{}, nil
}

type dummyPaymentSvc struct {
	service.PaymentService
}

func (d *dummyPaymentSvc) ListAwaitingVerification(ctx context.Context, limit, offset int) ([]model.Payment, int64, error) {
	return []model.Payment{}, 0, nil
}

func (d *dummyPaymentSvc) SimulatePaid(ctx context.Context, paymentID uuid.UUID) (*model.Payment, error) {
	return &model.Payment{
		ID:     paymentID,
		Status: model.PaymentStatusSuccess,
	}, nil
}

func (d *dummyPaymentSvc) ReconcilePendingPayments(ctx context.Context) (int, error) {
	return 0, nil
}

func setupTestFinanceRouter(gatewaySecret string, isDev bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := RouterConfig{
		GatewaySecret: gatewaySecret,
		InternalToken: "test-internal-token",
		IsDevelopment: isDev,
	}

	appCfg := &config.Config{
		AppEnv: "production",
	}
	if isDev {
		appCfg.AppEnv = "development"
	}

	invSvc := &dummyInvoiceSvc{}
	paySvc := &dummyPaymentSvc{}

	healthCtrl := controller.NewHealthController(nil)
	invCtrl := controller.NewInvoiceController(invSvc)
	payCtrl := controller.NewPaymentController(paySvc)
	rcCtrl := controller.NewReceiptController(paySvc)
	webhookCtrl := controller.NewWebhookController(paySvc)
	devCtrl := controller.NewDevController(paySvc, appCfg)
	adminCtrl := controller.NewAdminController(paySvc, nil)
	internalCtrl := controller.NewInternalController(invSvc)
	installmentCtrl := controller.NewInstallmentController(nil)
	reportCtrl := controller.NewReportController(nil, nil)

	return SetupRouter(cfg, healthCtrl, invCtrl, payCtrl, rcCtrl, webhookCtrl, devCtrl, adminCtrl, internalCtrl, installmentCtrl, reportCtrl)
}

func setGatewayAuthHeaders(req *http.Request, secret, reqID, userID, role string) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := gatewayauth.GenerateSignature(secret, reqID, userID, timestamp)

	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, timestamp)
	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderUserID, userID)
	req.Header.Set(gatewayauth.HeaderUserRoles, fmt.Sprintf(`["%s"]`, role))
}

func TestFinanceRouter(t *testing.T) {
	secret := "finance-test-secret"
	r := setupTestFinanceRouter(secret, true)

	t.Run("GET /health is public and returns 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("GET /invoices/me without gateway signature fails 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/invoices/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("GET /invoices/me with valid signature succeeds 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/invoices/me", nil)
		userID := uuid.New().String()
		reqID := uuid.New().String()

		setGatewayAuthHeaders(req, secret, reqID, userID, "applicant")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("dev endpoint available when IsDevelopment is true", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/dev/payments/"+uuid.New().String()+"/simulate-paid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Handler reached (not 404)
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("dev endpoint disabled in non-dev environment", func(t *testing.T) {
		prodRouter := setupTestFinanceRouter(secret, false)
		req, _ := http.NewRequest(http.MethodPost, "/dev/payments/"+uuid.New().String()+"/simulate-paid", nil)
		w := httptest.NewRecorder()
		prodRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
