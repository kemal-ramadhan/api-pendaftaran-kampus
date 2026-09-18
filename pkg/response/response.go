package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
