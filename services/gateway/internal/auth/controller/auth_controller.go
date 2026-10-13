package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/dto"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/middleware"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/service"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/token"
)

type AuthController struct {
	authService  service.AuthService
	tokenManager token.TokenManager
}

func NewAuthController(authService service.AuthService, tokenManager token.TokenManager) *AuthController {
	return &AuthController{
		authService:  authService,
		tokenManager: tokenManager,
	}
}

// Register godoc
// @Summary Registrasi Calon Mahasiswa
// @Description Pendaftaran mandiri calon mahasiswa baru. Otomatis mengirim kode OTP ke email.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Data Pendaftaran"
// @Success 201 {object} response.SuccessResponse{data=dto.RegisterResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /auth/register [post]
func (ctl *AuthController) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.authService.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			response.Error(c, http.StatusConflict, response.ErrCodeEmailTaken, "Email sudah terdaftar dan terverifikasi", nil)
			return
		}
		if errors.Is(err, service.ErrValidation) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal memproses pendaftaran")
		return
	}

	response.Success(c, http.StatusCreated, "Pendaftaran berhasil, silakan periksa email Anda untuk kode OTP verifikasi", res)
}

// VerifyOTP godoc
// @Summary Verifikasi OTP Email (Auto-login)
// @Description Verifikasi kode OTP pendaftaran. Jika sukses, akun menjadi aktif dan langsung menerbitkan token login.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.VerifyOTPRequest true "Data Verifikasi OTP"
// @Success 200 {object} response.SuccessResponse{data=dto.AuthResponseData}
// @Failure 400 {object} response.ErrorResponse
// @Failure 429 {object} response.ErrorResponse
// @Router /auth/verify [post]
func (ctl *AuthController) VerifyOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.authService.VerifyOTP(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, service.ErrAccountAlreadyActive) {
			response.Error(c, http.StatusBadRequest, "ACCOUNT_ALREADY_ACTIVE", "Akun Anda sudah aktif dan terverifikasi. Silakan masuk (login) ke sistem.", nil)
			return
		}
		if errors.Is(err, service.ErrOTPExpired) {
			response.Error(c, http.StatusBadRequest, response.ErrCodeOtpExpired, "Kode OTP telah kedaluwarsa, silakan kirim ulang", nil)
			return
		}
		if errors.Is(err, service.ErrOTPLocked) {
			response.Error(c, http.StatusTooManyRequests, response.ErrCodeOtpLocked, "Percobaan OTP melebihi batas maksimal, silakan kirim ulang OTP baru", nil)
			return
		}
		if errors.Is(err, service.ErrOTPInvalid) {
			response.Error(c, http.StatusBadRequest, response.ErrCodeOtpInvalid, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal memverifikasi OTP")
		return
	}

	response.Success(c, http.StatusOK, "Verifikasi email berhasil, Anda telah masuk ke sistem", res)
}

// ResendOTP godoc
// @Summary Kirim Ulang OTP
// @Description Mengirimkan ulang kode OTP verifikasi dengan cooldown 60 detik.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.ResendOTPRequest true "Email Tujuan"
// @Success 200 {object} response.SuccessResponse
// @Failure 429 {object} response.ErrorResponse
// @Router /auth/resend-otp [post]
func (ctl *AuthController) ResendOTP(c *gin.Context) {
	var req dto.ResendOTPRequest
	if !response.BindJSON(c, &req) {
		return
	}

	if err := ctl.authService.ResendOTP(c.Request.Context(), req); err != nil {
		if errors.Is(err, service.ErrOTPResendCooldown) {
			response.Error(c, http.StatusTooManyRequests, response.ErrCodeOtpResendCooldown, "Mohon tunggu 60 detik sebelum meminta OTP kembali", nil)
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			response.Error(c, http.StatusNotFound, response.ErrCodeNotFound, "Email belum terdaftar. Silakan lakukan pendaftaran terlebih dahulu.", nil)
			return
		}
		if errors.Is(err, service.ErrAccountAlreadyActive) {
			response.Error(c, http.StatusBadRequest, "ACCOUNT_ALREADY_ACTIVE", "Akun Anda sudah aktif dan terverifikasi. Silakan masuk (login) ke portal PMB.", nil)
			return
		}
		response.InternalServerError(c, "Gagal mengirim ulang OTP")
		return
	}

	response.Success(c, http.StatusOK, "Kode OTP baru telah berhasil dikirimkan ke email Anda", nil)
}

// Login godoc
// @Summary Login Pengguna
// @Description Masuk ke akun menggunakan email dan kata sandi. Mengembalikan access token dan refresh token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Kredensial Login"
// @Success 200 {object} response.SuccessResponse{data=dto.AuthResponseData}
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 429 {object} response.ErrorResponse
// @Router /auth/login [post]
func (ctl *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.authService.Login(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, service.ErrTooManyAttempts) {
			response.Error(c, http.StatusTooManyRequests, response.ErrCodeTooManyAttempts, "Terlalu banyak percobaan gagal. Akun dikunci sementara selama 15 menit", nil)
			return
		}
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, response.ErrCodeInvalidCredentials, "Email atau kata sandi yang Anda masukkan salah", nil)
			return
		}
		if errors.Is(err, service.ErrEmailNotVerified) {
			response.Error(c, http.StatusForbidden, response.ErrCodeEmailNotVerified, "Email Anda belum diverifikasi. Silakan masukkan kode OTP yang telah dikirim", map[string][]string{
				"email": {req.Email},
			})
			return
		}
		if errors.Is(err, service.ErrAccountInactive) {
			response.Error(c, http.StatusForbidden, response.ErrCodeAccountInactive, "Akun Anda dinonaktifkan oleh administrator", nil)
			return
		}
		if errors.Is(err, service.ErrServiceAccessDenied) {
			response.Error(c, http.StatusForbidden, "SERVICE_ACCESS_DENIED", "Anda tidak memiliki akses ke layanan ini", nil)
			return
		}
		response.InternalServerError(c, "Terjadi kesalahan saat proses login")
		return
	}

	response.Success(c, http.StatusOK, "Login berhasil", res)
}

// Refresh godoc
// @Summary Refresh Token (Rotasi Pasangan Token)
// @Description Menukarkan refresh token aktif dengan pasangan access token dan refresh token baru.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshRequest true "Refresh Token"
// @Success 200 {object} response.SuccessResponse{data=dto.AuthResponseData}
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/refresh [post]
func (ctl *AuthController) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.authService.Refresh(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrTokenRevoked) {
			response.Error(c, http.StatusUnauthorized, response.ErrCodeTokenRevoked, "Sesi Anda telah kedaluwarsa atau digunakan di perangkat lain. Silakan login kembali", nil)
			return
		}
		response.Error(c, http.StatusUnauthorized, response.ErrCodeTokenInvalid, "Refresh token tidak valid", nil)
		return
	}

	response.Success(c, http.StatusOK, "Token berhasil diperbarui", res)
}

// Logout godoc
// @Summary Logout Sesi Saat Ini
// @Description Mencabut sesi aktif saat ini dan memasukkan access token ke denylist.
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/logout [post]
func (ctl *AuthController) Logout(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Tidak terautentikasi")
		return
	}

	sessionID := middleware.GetSessionID(c)
	accessJTI := middleware.GetTokenJTI(c)
	exp := middleware.GetTokenExp(c)

	_ = ctl.authService.Logout(c.Request.Context(), userID, sessionID, accessJTI, exp)
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

// LogoutAll godoc
// @Summary Logout Semua Sesi
// @Description Mencabut seluruh sesi login user di semua perangkat.
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/logout-all [post]
func (ctl *AuthController) LogoutAll(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Tidak terautentikasi")
		return
	}

	accessJTI := middleware.GetTokenJTI(c)
	exp := middleware.GetTokenExp(c)

	_ = ctl.authService.LogoutAll(c.Request.Context(), userID, accessJTI, exp)
	response.Success(c, http.StatusOK, "Seluruh sesi berhasil dikeluarkan", nil)
}

// GetMe godoc
// @Summary Profil Pengguna Saat Ini
// @Description Mengambil informasi identitas akun, role, dan scope dari token terautentikasi.
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/me [get]
func (ctl *AuthController) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Tidak terautentikasi")
		return
	}

	user, err := ctl.authService.GetMe(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "Pengguna tidak ditemukan")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil profil pengguna", user)
}

// GetMePermissions godoc
// @Summary Daftar Permissions dan Modul Pengguna Saat Ini
// @Description Mengambil hak akses aktif pengguna saat ini, daftar permission, dan status 8 modul admin portal.
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=dto.UserPermissionsResponse}
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/me/permissions [get]
func (ctl *AuthController) GetMePermissions(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Tidak terautentikasi")
		return
	}

	res, err := ctl.authService.GetMePermissions(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil permissions pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil permissions pengguna", res)
}

// UpdateProfile godoc
// @Summary Perbarui Profil Akun Sendiri
// @Description Memperbarui data nama lengkap dan nomor telepon akun sendiri.
// @Tags Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.UpdateProfileRequest true "Data Profil Baru"
// @Success 200 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 401 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /auth/profile [put]
func (ctl *AuthController) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Tidak terautentikasi")
		return
	}

	var req dto.UpdateProfileRequest
	if !response.BindJSON(c, &req) {
		return
	}

	user, err := ctl.authService.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalServerError(c, "Gagal memperbarui profil")
		return
	}

	response.Success(c, http.StatusOK, "Profil berhasil diperbarui", user)
}

// ChangePassword godoc
// @Summary Ganti Kata Sandi Akun
// @Description Mengganti kata sandi akun sendiri (wajib memasukkan kata sandi saat ini). Otomatis mencabut sesi di perangkat lain.
// @Tags Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.ChangePasswordRequest true "Data Kata Sandi Baru"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /auth/password [put]
func (ctl *AuthController) ChangePassword(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Tidak terautentikasi")
		return
	}

	var req dto.ChangePasswordRequest
	if !response.BindJSON(c, &req) {
		return
	}

	if err := ctl.authService.ChangePassword(c.Request.Context(), userID, req); err != nil {
		if errors.Is(err, service.ErrOldPasswordMismatch) {
			response.BadRequest(c, response.ErrCodeInvalidCredentials, "Kata sandi saat ini salah")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mengganti kata sandi")
		return
	}

	response.Success(c, http.StatusOK, "Kata sandi berhasil diubah. Sesi di perangkat lain telah dinonaktifkan", nil)
}

// ForgotPassword godoc
// @Summary Lupa Kata Sandi (Kirim Tautan Reset)
// @Description Mengirimkan tautan reset kata sandi ke email pengguna. Selalu mengembalikan respon sukses generik untuk keamanan.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Email Terdaftar"
// @Success 200 {object} response.SuccessResponse
// @Failure 429 {object} response.ErrorResponse
// @Router /auth/forgot-password [post]
func (ctl *AuthController) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if !response.BindJSON(c, &req) {
		return
	}

	if err := ctl.authService.ForgotPassword(c.Request.Context(), req); err != nil {
		if errors.Is(err, service.ErrOTPResendCooldown) {
			response.Error(c, http.StatusTooManyRequests, response.ErrCodeOtpResendCooldown, "Permintaan reset kata sandi terlalu sering, mohon tunggu 60 detik", nil)
			return
		}
		response.InternalServerError(c, "Gagal memproses permintaan lupa kata sandi")
		return
	}

	response.Success(c, http.StatusOK, "Jika email terdaftar, tautan reset kata sandi telah dikirimkan ke email Anda", nil)
}

// ResetPassword godoc
// @Summary Reset Kata Sandi Baru
// @Description Mengubah kata sandi menggunakan token reset yang dikirim ke email.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Data Reset Kata Sandi"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /auth/reset-password [post]
func (ctl *AuthController) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if !response.BindJSON(c, &req) {
		return
	}

	if err := ctl.authService.ResetPassword(c.Request.Context(), req); err != nil {
		if errors.Is(err, service.ErrResetTokenInvalid) {
			response.Error(c, http.StatusBadRequest, response.ErrCodeResetTokenInvalid, "Token reset kata sandi tidak valid atau telah kedaluwarsa", nil)
			return
		}
		if errors.Is(err, service.ErrValidation) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mengatur ulang kata sandi")
		return
	}

	response.Success(c, http.StatusOK, "Kata sandi Anda berhasil diatur ulang. Silakan login dengan kata sandi baru", nil)
}

// GetJWKS godoc
// @Summary Public Key JWKS
// @Description Mengembalikan JSON Web Key Set (JWKS) public key RS256 untuk verifikasi token oleh API Gateway dan service lain.
// @Tags Auth
// @Produce json
// @Success 200 {object} token.JWKSResponse
// @Router /.well-known/jwks.json [get]
func (ctl *AuthController) GetJWKS(c *gin.Context) {
	c.JSON(http.StatusOK, ctl.tokenManager.GetJWKS())
}
