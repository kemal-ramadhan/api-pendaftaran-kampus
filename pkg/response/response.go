package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
)

// Standard Error Codes sesuai PRD Bagian 7
const (
	ErrCodeValidationError     = "VALIDATION_ERROR"
	ErrCodeEmailTaken          = "EMAIL_TAKEN"
	ErrCodeInvalidCredentials  = "INVALID_CREDENTIALS"
	ErrCodeEmailNotVerified    = "EMAIL_NOT_VERIFIED"
	ErrCodeAccountInactive     = "ACCOUNT_INACTIVE"
	ErrCodeTooManyAttempts     = "TOO_MANY_ATTEMPTS"
	ErrCodeOtpInvalid          = "OTP_INVALID"
	ErrCodeOtpExpired          = "OTP_EXPIRED"
	ErrCodeOtpLocked           = "OTP_LOCKED"
	ErrCodeOtpResendCooldown   = "OTP_RESEND_COOLDOWN"
	ErrCodeResetTokenInvalid   = "RESET_TOKEN_INVALID"
	ErrCodeTokenMissing        = "TOKEN_MISSING"
	ErrCodeTokenInvalid        = "TOKEN_INVALID"
	ErrCodeTokenExpired        = "TOKEN_EXPIRED"
	ErrCodeTokenRevoked        = "TOKEN_REVOKED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeRateLimited         = "RATE_LIMITED"
	ErrCodeUpstreamUnavailable = "UPSTREAM_UNAVAILABLE"
	ErrCodeUpstreamTimeout     = "UPSTREAM_TIMEOUT"
	ErrCodeInternalError       = "INTERNAL_ERROR"
)

// Meta memuat data paginasi
type Meta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

// SuccessResponse struktur standar respons berhasil
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorResponse struktur standar respons gagal
type ErrorResponse struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message"`
	ErrorCode string              `json:"error_code"`
	Errors    map[string][]string `json:"errors,omitempty"`
	RequestID string              `json:"request_id"`
}

// Success mengirimkan respons JSON sukses standar (status code default http.StatusOK)
func Success(c *gin.Context, httpStatus int, message string, data interface{}) {
	c.JSON(httpStatus, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta mengirimkan respons JSON sukses dengan metadata paginasi
func SuccessWithMeta(c *gin.Context, httpStatus int, message string, data interface{}, meta *Meta) {
	c.JSON(httpStatus, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Error mengirimkan respons JSON error standar
func Error(c *gin.Context, httpStatus int, errorCode, message string, validationErrors map[string][]string) {
	c.JSON(httpStatus, ErrorResponse{
		Success:   false,
		Message:   message,
		ErrorCode: errorCode,
		Errors:    validationErrors,
		RequestID: requestid.Get(c),
	})
}

// BadRequest mengirimkan respons 400
func BadRequest(c *gin.Context, errorCode, message string) {
	Error(c, http.StatusBadRequest, errorCode, message, nil)
}

// Unauthorized mengirimkan respons 401
func Unauthorized(c *gin.Context, errorCode, message string) {
	Error(c, http.StatusUnauthorized, errorCode, message, nil)
}

// Forbidden mengirimkan respons 403
func Forbidden(c *gin.Context, errorCode, message string) {
	Error(c, http.StatusForbidden, errorCode, message, nil)
}

// NotFound mengirimkan respons 404
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, ErrCodeNotFound, message, nil)
}

// UnprocessableEntity mengirimkan respons 422 untuk kesalahan validasi form/payload
func UnprocessableEntity(c *gin.Context, message string, validationErrors map[string][]string) {
	Error(c, http.StatusUnprocessableEntity, ErrCodeValidationError, message, validationErrors)
}

// TooManyRequests mengirimkan respons 429
func TooManyRequests(c *gin.Context, errorCode, message string) {
	Error(c, http.StatusTooManyRequests, errorCode, message, nil)
}

// InternalServerError mengirimkan respons 500 generik tanpa membocorkan error internal
func InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "Terjadi kesalahan internal pada server"
	}
	Error(c, http.StatusInternalServerError, ErrCodeInternalError, message, nil)
}

// WriteJSON menuliskan objek response ke standard http.ResponseWriter
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func toSnakeCase(str string) string {
	var b strings.Builder
	for i, r := range str {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteRune('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func validationErrorMessage(field, tag, param string) string {
	switch tag {
	case "required":
		return fmt.Sprintf("Field %s wajib diisi", field)
	case "email":
		return "Format email tidak valid"
	case "min":
		return fmt.Sprintf("Panjang minimal %s karakter", param)
	case "max":
		return fmt.Sprintf("Panjang maksimal %s karakter", param)
	case "eqfield":
		return fmt.Sprintf("Nilai harus sama dengan %s", toSnakeCase(param))
	default:
		return fmt.Sprintf("Field %s tidak valid", field)
	}
}

// FormatBindingError menerjemahkan error binding JSON atau validasi Go/Gin menjadi pesan Bahasa Indonesia yang ramah pengguna
func FormatBindingError(err error) (string, map[string][]string) {
	if err == nil {
		return "", nil
	}

	// 1. Kasus body kosong (EOF)
	if errors.Is(err, io.EOF) || strings.TrimSpace(err.Error()) == "EOF" {
		return "Data permintaan (body) tidak boleh kosong", map[string][]string{
			"body": {"Request body tidak boleh kosong"},
		}
	}

	// 2. Kasus format JSON terpotong / sintaks rusak
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || errors.Is(err, io.ErrUnexpectedEOF) || strings.Contains(err.Error(), "unexpected EOF") {
		return "Format data JSON tidak valid atau rusak", map[string][]string{
			"json": {"Sintaks JSON tidak valid atau terpotong"},
		}
	}

	// 3. Kasus tipe data field JSON tidak sesuai
	var unmarshalTypeErr *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeErr) {
		field := unmarshalTypeErr.Field
		if field == "" {
			field = "payload"
		} else {
			field = toSnakeCase(field)
		}
		expectedType := unmarshalTypeErr.Type.String()
		return "Tipe data pada payload tidak sesuai", map[string][]string{
			field: {fmt.Sprintf("Tipe data tidak sesuai (diharapkan %s)", expectedType)},
		}
	}

	// 4. Kasus validasi go-playground validator (tag required, email, min, max, dll.)
	var valErrors validator.ValidationErrors
	if errors.As(err, &valErrors) {
		errMap := make(map[string][]string)
		for _, fe := range valErrors {
			fieldName := toSnakeCase(fe.Field())
			msg := validationErrorMessage(fieldName, fe.Tag(), fe.Param())
			errMap[fieldName] = append(errMap[fieldName], msg)
		}
		return "Data masukan tidak memenuhi validasi form", errMap
	}

	// 5. Fallback generik
	errMsg := err.Error()
	if strings.Contains(strings.ToLower(errMsg), "eof") {
		return "Data permintaan (body) tidak boleh kosong", map[string][]string{
			"body": {"Request body tidak boleh kosong"},
		}
	}

	return "Data yang dikirim tidak valid", map[string][]string{
		"form": {"Format masukan tidak valid"},
	}
}

// BindingError mengirimkan respons HTTP 422 Unprocessable Entity dengan pesan Bahasa Indonesia terstandarisasi untuk binding error
func BindingError(c *gin.Context, err error) {
	msg, errMap := FormatBindingError(err)
	UnprocessableEntity(c, msg, errMap)
}

// BindJSON melakukan parsing dan validasi request body JSON.
// Jika terjadi error, otomatis mengirimkan respons 422 Unprocessable Entity
// dengan pesan Bahasa Indonesia yang ramah pengguna, dan mengembalikan false.
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		BindingError(c, err)
		return false
	}
	return true
}


