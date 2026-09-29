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
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type dummyNIMRepo struct{}

func (d *dummyNIMRepo) DB() *gorm.DB                                                                 { return nil }
func (d *dummyNIMRepo) GetIssuanceByApplicationID(_ context.Context, _ uuid.UUID) (*model.NIMIssuance, error) {
	return nil, nil
}
func (d *dummyNIMRepo) GetIssuanceByNIM(_ context.Context, _ string) (*model.NIMIssuance, error) {
	return nil, nil
}
func (d *dummyNIMRepo) CreateIssuanceInTx(_ context.Context, _ *gorm.DB, _ *model.NIMIssuance) error {
	return nil
}
func (d *dummyNIMRepo) ListIssuances(_ context.Context, _ model.IssuanceFilter) ([]model.NIMIssuance, int64, error) {
	return nil, 0, nil
}
func (d *dummyNIMRepo) GetNextSequenceWithLock(_ context.Context, _ *gorm.DB, _ int, _ string) (int, error) {
	return 1, nil
}
func (d *dummyNIMRepo) GetFormatForProgram(_ context.Context, _ string) (*model.NIMFormat, error) {
	return nil, nil
}
func (d *dummyNIMRepo) ListFormats(_ context.Context) ([]model.NIMFormat, error) {
	return nil, nil
}
func (d *dummyNIMRepo) GetFormatByID(_ context.Context, _ uuid.UUID) (*model.NIMFormat, error) {
	return nil, nil
}
func (d *dummyNIMRepo) CreateFormat(_ context.Context, _ *model.NIMFormat) error {
	return nil
}
func (d *dummyNIMRepo) UpdateFormat(_ context.Context, _ *model.NIMFormat) error {
	return nil
}
func (d *dummyNIMRepo) VoidIssuance(_ context.Context, _ uuid.UUID) (*model.NIMIssuance, error) {
	return nil, nil
}

func setGatewayAuthHeaders(req *http.Request, secret, reqID, userID, role string) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := gatewayauth.GenerateSignature(secret, reqID, userID, timestamp)

	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, timestamp)
	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderUserID, userID)
	req.Header.Set(gatewayauth.HeaderUserRoles, role)
}

func TestNIMRouter(t *testing.T) {
	cfg := &config.Config{
		AppEnv:        "test",
		Port:          "8092",
		IsDevelopment: true,
		GatewaySecret: "test-secret-gateway",
		InternalToken: "test-secret-internal",
	}

	repo := &dummyNIMRepo{}
	svc := service.NewNIMService(cfg, repo, nil, nil, nil, nil, nil)
	ctrl := controller.NewNIMController(svc)
	router := SetupRouter(cfg, ctrl)

	t.Run("GET /health adalah publik dan mengembalikan status 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "pmb-nim-service")
	})

	t.Run("GET /nim/admin/issuances tanpa gateway signature ditolak 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/nim/admin/issuances", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("GET /nim/admin/issuances dengan signature valid diterima gatewayauth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/nim/admin/issuances", nil)
		setGatewayAuthHeaders(req, cfg.GatewaySecret, "req-1", uuid.New().String(), "admin_pmb")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
