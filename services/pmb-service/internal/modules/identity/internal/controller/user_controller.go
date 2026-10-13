package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/service"
)

type UserController struct {
	identitySvc     service.IdentityService
	completenessSvc service.CompletenessService
	ktpScanSvc      service.KTPScanService
}

func NewUserController(
	identitySvc service.IdentityService,
	completenessSvc service.CompletenessService,
	ktpScanSvc service.KTPScanService,
) *UserController {
	return &UserController{
		identitySvc:     identitySvc,
		completenessSvc: completenessSvc,
		ktpScanSvc:      ktpScanSvc,
	}
}

func (ctrl *UserController) GetMyProfile(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	roles := gatewayauth.GetUserRoles(c)
	profile, err := ctrl.identitySvc.GetProfile(c.Request.Context(), userID, userID, roles)
	if err != nil {
		if errors.Is(err, repository.ErrPersonNotFound) {
			response.NotFound(c, "Data profil belum diisi")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data profil")
		return
	}

	response.Success(c, http.StatusOK, "Data profil berhasil diambil", profile)
}

func (ctrl *UserController) UpdateMyProfile(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	var req dto.ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Format data profil tidak valid", nil)
		return
	}

	profile, err := ctrl.identitySvc.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		if errors.Is(err, repository.ErrNIKDuplicate) {
			response.Conflict(c, "DUPLICATE_NIK", "NIK sudah terdaftar pada akun lain")
			return
		}
		response.InternalServerError(c, "Gagal menyimpan data profil")
		return
	}

	response.Success(c, http.StatusOK, "Data profil berhasil diperbarui", profile)
}

func (ctrl *UserController) GetMyAddresses(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	addrs, err := ctrl.identitySvc.GetAddresses(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data alamat")
		return
	}

	response.Success(c, http.StatusOK, "Daftar alamat berhasil diambil", addrs)
}

func (ctrl *UserController) SaveMyAddress(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	var req dto.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Format data alamat tidak valid", nil)
		return
	}

	addr, err := ctrl.identitySvc.SaveAddress(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalServerError(c, "Gagal menyimpan data alamat")
		return
	}

	response.Success(c, http.StatusOK, "Data alamat berhasil disimpan", addr)
}

func (ctrl *UserController) GetMyGuardians(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	guardians, err := ctrl.identitySvc.GetGuardians(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data orang tua/wali")
		return
	}

	response.Success(c, http.StatusOK, "Daftar orang tua/wali berhasil diambil", guardians)
}

func (ctrl *UserController) SaveMyGuardian(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	var req dto.GuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Format data orang tua/wali tidak valid", nil)
		return
	}

	guardian, err := ctrl.identitySvc.SaveGuardian(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalServerError(c, "Gagal menyimpan data orang tua/wali")
		return
	}

	response.Success(c, http.StatusOK, "Data orang tua/wali berhasil disimpan", guardian)
}

func (ctrl *UserController) GetMyEducation(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	edus, err := ctrl.identitySvc.GetEducation(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data riwayat pendidikan")
		return
	}

	response.Success(c, http.StatusOK, "Daftar riwayat pendidikan berhasil diambil", edus)
}

func (ctrl *UserController) SaveMyEducation(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	var req dto.EducationHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Format data riwayat pendidikan tidak valid", nil)
		return
	}

	edu, err := ctrl.identitySvc.SaveEducation(c.Request.Context(), userID, req)
	if err != nil {
		response.InternalServerError(c, "Gagal menyimpan riwayat pendidikan")
		return
	}

	response.Success(c, http.StatusOK, "Riwayat pendidikan berhasil disimpan", edu)
}

func (ctrl *UserController) UpdateMyPhoto(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		file, header, err = c.Request.FormFile("file")
		if err != nil {
			response.BadRequest(c, "MISSING_FILE", "File foto profil wajib diunggah (field 'photo' atau 'file')")
			return
		}
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	photoURL, err := ctrl.identitySvc.UpdatePhoto(c.Request.Context(), userID, file, header.Size, contentType)
	if err != nil {
		if errors.Is(err, service.ErrFileTooLarge) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrInvalidFileType) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mengunggah foto profil")
		return
	}

	response.Success(c, http.StatusOK, "Foto profil berhasil diperbarui", gin.H{
		"photo_url": photoURL,
	})
}

func (ctrl *UserController) GetMyCompleteness(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	comp, err := ctrl.completenessSvc.CheckCompleteness(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal memeriksa kelengkapan biodata")
		return
	}

	response.Success(c, http.StatusOK, "Status kelengkapan biodata berhasil diperiksa", comp)
}

func (ctrl *UserController) InitiateKTPScan(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		file, header, err = c.Request.FormFile("ktp")
		if err != nil {
			response.BadRequest(c, "MISSING_FILE", "File gambar KTP wajib disertakan")
			return
		}
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	initResp, err := ctrl.ktpScanSvc.InitiateKTPScan(c.Request.Context(), userID, file, header.Size, header.Filename, contentType)
	if err != nil {
		if errors.Is(err, service.ErrScanRateLimitExceeded) {
			response.TooManyRequests(c, "RATE_LIMIT_EXCEEDED", err.Error())
			return
		}
		if errors.Is(err, service.ErrScanFileTooLarge) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrScanInvalidFileType) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal memproses unggahan KTP")
		return
	}

	response.Success(c, http.StatusAccepted, "Proses pembacaan KTP sedang berjalan di latar belakang", initResp)
}

func (ctrl *UserController) GetKTPScanStatus(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	scanIDStr := c.Param("scan_id")
	scanID, err := uuid.Parse(scanIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "ID pemindaian tidak valid")
		return
	}

	res, err := ctrl.ktpScanSvc.GetKTPScanStatus(c.Request.Context(), scanID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrKTPScanNotFound) {
			response.NotFound(c, "Data pemindaian KTP tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrUnauthorizedAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak memiliki akses ke pemindaian KTP ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil status pemindaian KTP")
		return
	}

	response.Success(c, http.StatusOK, "Status pemindaian KTP berhasil diambil", res)
}

func (ctrl *UserController) ConfirmKTPScan(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pengguna tidak ditemukan")
		return
	}

	scanIDStr := c.Param("scan_id")
	scanID, err := uuid.Parse(scanIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "ID pemindaian tidak valid")
		return
	}

	var req dto.KTPConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Data konfirmasi KTP tidak valid", nil)
		return
	}

	updatedProfile, err := ctrl.ktpScanSvc.ConfirmKTPScan(c.Request.Context(), scanID, userID, req)
	if err != nil {
		if errors.Is(err, repository.ErrKTPScanNotFound) {
			response.NotFound(c, "Data pemindaian KTP tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrUnauthorizedAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak memiliki akses ke pemindaian KTP ini")
			return
		}
		if errors.Is(err, service.ErrScanAlreadyConfirmed) {
			response.Conflict(c, "ALREADY_CONFIRMED", "Pemindaian KTP ini sudah dikonfirmasi sebelumnya")
			return
		}
		if errors.Is(err, repository.ErrNIKDuplicate) {
			response.Conflict(c, "DUPLICATE_NIK", "NIK sudah terdaftar pada pengguna lain")
			return
		}
		response.InternalServerError(c, "Gagal mengonfirmasi data KTP")
		return
	}

	response.Success(c, http.StatusOK, "Data KTP berhasil dikonfirmasi dan disimpan ke biodata", updatedProfile)
}
