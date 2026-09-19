package response

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

type dummyPayload struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required,min=3"`
}

func TestFormatBindingError_EmptyBody(t *testing.T) {
	msg, errMap := FormatBindingError(io.EOF)
	if msg != "Data permintaan (body) tidak boleh kosong" {
		t.Errorf("expected clean indonesian message for EOF, got '%s'", msg)
	}
	if len(errMap["body"]) == 0 || errMap["body"][0] == "EOF" {
		t.Errorf("errors.body must not be EOF, got: %v", errMap["body"])
	}
}

func TestFormatBindingError_InvalidJSON(t *testing.T) {
	msg, errMap := FormatBindingError(io.ErrUnexpectedEOF)
	if msg != "Format data JSON tidak valid atau rusak" {
		t.Errorf("expected clean indonesian message for syntax error, got '%s'", msg)
	}
	if len(errMap["json"]) == 0 {
		t.Errorf("expected errors.json to be populated")
	}
}

func TestBindJSON_EmptyBodyHTTP(t *testing.T) {
	r := setupTestRouter()
	r.POST("/test-bind", func(c *gin.Context) {
		var req dummyPayload
		if !BindJSON(c, &req) {
			return
		}
		Success(c, http.StatusOK, "OK", nil)
	})

	// Request dengan body kosong
	req := httptest.NewRequest(http.MethodPost, "/test-bind", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for empty body, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ErrorCode != ErrCodeValidationError {
		t.Errorf("expected error_code %s, got %s", ErrCodeValidationError, resp.ErrorCode)
	}
	if resp.Message != "Data permintaan (body) tidak boleh kosong" {
		t.Errorf("expected indonesian empty body message, got '%s'", resp.Message)
	}
	// Pastikan TIDAK ADA "EOF" yang membocorkan error internal Go
	for field, msgs := range resp.Errors {
		for _, m := range msgs {
			if m == "EOF" {
				t.Errorf("internal Go error 'EOF' must not be exposed in field %s", field)
			}
		}
	}
}

func TestBindJSON_InvalidJSONHTTP(t *testing.T) {
	r := setupTestRouter()
	r.POST("/test-bind", func(c *gin.Context) {
		var req dummyPayload
		if !BindJSON(c, &req) {
			return
		}
		Success(c, http.StatusOK, "OK", nil)
	})

	// Request dengan format JSON rusak
	req := httptest.NewRequest(http.MethodPost, "/test-bind", strings.NewReader("{ broken json "))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for malformed json, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Message != "Format data JSON tidak valid atau rusak" {
		t.Errorf("expected malformed JSON message, got '%s'", resp.Message)
	}
	if len(resp.Errors["json"]) == 0 {
		t.Errorf("expected errors['json'] to be populated")
	}
}

func TestBindJSON_ValidationFailedHTTP(t *testing.T) {
	r := setupTestRouter()
	r.POST("/test-bind", func(c *gin.Context) {
		var req dummyPayload
		if !BindJSON(c, &req) {
			return
		}
		Success(c, http.StatusOK, "OK", nil)
	})

	// Request dengan valid JSON tetapi field tidak valid
	req := httptest.NewRequest(http.MethodPost, "/test-bind", strings.NewReader(`{"email":"bukan-email","name":"a"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for validation error, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Errors["email"]) == 0 {
		t.Errorf("expected validation error on email")
	}
	if len(resp.Errors["name"]) == 0 {
		t.Errorf("expected validation error on name")
	}
}

