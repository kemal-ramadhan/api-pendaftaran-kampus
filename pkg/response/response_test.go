package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestid.Middleware())
	return r
}

func TestSuccessResponse(t *testing.T) {
	r := setupTestRouter()
	r.GET("/success", func(c *gin.Context) {
		Success(c, http.StatusOK, "Berhasil mengambil data", map[string]string{"foo": "bar"})
	})

	req := httptest.NewRequest(http.MethodGet, "/success", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}
	if resp.Message != "Berhasil mengambil data" {
		t.Errorf("expected message 'Berhasil mengambil data', got '%s'", resp.Message)
	}
}

func TestSuccessWithMetaResponse(t *testing.T) {
	r := setupTestRouter()
	r.GET("/meta", func(c *gin.Context) {
		SuccessWithMeta(c, http.StatusOK, "List user", []string{"user1", "user2"}, &Meta{
			Page:    1,
			PerPage: 10,
			Total:   2,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/meta", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Meta == nil || resp.Meta.Total != 2 {
		t.Errorf("expected meta with total 2")
	}
}

func TestErrorResponse(t *testing.T) {
	r := setupTestRouter()
	r.POST("/login", func(c *gin.Context) {
		Unauthorized(c, ErrCodeInvalidCredentials, "Email atau kata sandi salah")
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set(requestid.HeaderKey, "test-req-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Errorf("expected success to be false")
	}
	if resp.ErrorCode != ErrCodeInvalidCredentials {
		t.Errorf("expected error_code %s, got %s", ErrCodeInvalidCredentials, resp.ErrorCode)
	}
	if resp.RequestID != "test-req-id" {
		t.Errorf("expected request_id test-req-id, got %s", resp.RequestID)
	}
}

func TestValidationErrorResponse(t *testing.T) {
	r := setupTestRouter()
	r.POST("/validate", func(c *gin.Context) {
		errs := map[string][]string{
			"email": {"Format email tidak valid"},
		}
		UnprocessableEntity(c, "Validasi gagal", errs)
	})

	req := httptest.NewRequest(http.MethodPost, "/validate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ErrorCode != ErrCodeValidationError {
		t.Errorf("expected %s, got %s", ErrCodeValidationError, resp.ErrorCode)
	}
	if len(resp.Errors["email"]) == 0 {
		t.Errorf("expected errors on email field")
	}
}
