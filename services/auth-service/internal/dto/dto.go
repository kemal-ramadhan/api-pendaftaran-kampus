package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
)

type OTPCode string

func (o *OTPCode) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	*o = OTPCode(s)
	return nil
}

func (o OTPCode) String() string {
	return string(o)
}

type RegisterRequest struct {
	FullName             string `json:"full_name" binding:"required,min=2,max=255"`
	Email                string `json:"email" binding:"required,email"`
	Phone                string `json:"phone" binding:"required"`
	Password             string `json:"password" binding:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required,eqfield=Password"`
}

type RegisterResponse struct {
	Email             string    `json:"email"`
	OTPExpiresAt      time.Time `json:"otp_expires_at"`
	ResendAvailableAt time.Time `json:"resend_available_at"`
}

type VerifyOTPRequest struct {
	Email string  `json:"email" binding:"required,email"`
	OTP   OTPCode `json:"otp" binding:"required,len=6,numeric"`
}

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserResponseData struct {
	ID       uuid.UUID          `json:"id"`
	Email    string             `json:"email"`
	FullName string             `json:"full_name"`
	Phone    string             `json:"phone"`
	Roles    []string           `json:"roles"`
	Scopes   []token.ScopeClaim `json:"scopes"`
	Status   string             `json:"status"`
}

type AuthResponseData struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	TokenType    string           `json:"token_type"`
	ExpiresIn    int64            `json:"expires_in"`
	User         UserResponseData `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token                string `json:"token" binding:"required"`
	Password             string `json:"password" binding:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required,eqfield=Password"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=255"`
	Phone    string `json:"phone" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword      string `json:"current_password" binding:"required"`
	Password             string `json:"password" binding:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required,eqfield=Password"`
}
