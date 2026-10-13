package controller

import (
	"errors"
	"net/http"
	"strings"

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

func (ctrl *UserController) GetReferenceData(c *gin.Context) {
	jenis := strings.ToLower(c.Param("jenis"))
	switch jenis {
	case "agama":
		response.Success(c, http.StatusOK, "Data referensi agama berhasil diambil", []gin.H{
			{"id": "1", "name": "Islam"},
			{"id": "2", "name": "Kristen"},
			{"id": "3", "name": "Katolik"},
			{"id": "4", "name": "Hindu"},
			{"id": "5", "name": "Buddha"},
			{"id": "6", "name": "Konghucu"},
		})
	case "negara":
		response.Success(c, http.StatusOK, "Data referensi negara berhasil diambil", []gin.H{
			{"id": "IDN", "name": "Indonesia"},
			{"id": "MYS", "name": "Malaysia"},
			{"id": "SGP", "name": "Singapura"},
		})
	case "penghasilan":
		response.Success(c, http.StatusOK, "Data referensi penghasilan berhasil diambil", []gin.H{
			{"id": "1", "name": "Kurang dari/ Sama dengan 1.000.000"},
			{"id": "2", "name": "1.000.001 - 2.000.000"},
			{"id": "3", "name": "2.000.001 - 4.000.000"},
			{"id": "4", "name": "4.000.001 - 6.000.000"},
			{"id": "5", "name": "Lebih dari 6.000.000"},
		})
	case "pendidikan":
		response.Success(c, http.StatusOK, "Data referensi pendidikan berhasil diambil", []gin.H{
			{"id": "SD", "name": "SD - Sekolah Dasar"},
			{"id": "SMP", "name": "SMP/ Sederajat"},
			{"id": "SMA", "name": "SMA/SMK Sederajat"},
			{"id": "D1", "name": "Diploma 1"},
			{"id": "D3", "name": "Diploma 3"},
			{"id": "S1", "name": "Sarjana (S1)"},
			{"id": "S2", "name": "Magister (S2)"},
			{"id": "S3", "name": "Doktor (S3)"},
		})
	case "pekerjaan":
		response.Success(c, http.StatusOK, "Data referensi pekerjaan berhasil diambil", []gin.H{
			{"id": "0", "name": "Tidak Bekerja"},
			{"id": "1", "name": "PNS / ASN"},
			{"id": "2", "name": "Pegawai Swasta"},
			{"id": "3", "name": "Wiraswasta / Pengusaha"},
			{"id": "4", "name": "TNI / Polri"},
			{"id": "5", "name": "Petani / Nelayan"},
			{"id": "6", "name": "Buruh"},
			{"id": "7", "name": "Ibu Rumah Tangga"},
		})
	case "status_hidup":
		response.Success(c, http.StatusOK, "Data referensi status hidup berhasil diambil", []gin.H{
			{"id": "1", "name": "Hidup"},
			{"id": "2", "name": "Meninggal"},
		})
	case "ukuran_almamater", "almamater":
		response.Success(c, http.StatusOK, "Data referensi ukuran almamater berhasil diambil", []gin.H{
			{"id": "XS", "name": "XS"},
			{"id": "S", "name": "S"},
			{"id": "M", "name": "M"},
			{"id": "L", "name": "L"},
			{"id": "XL", "name": "XL"},
			{"id": "XXL", "name": "XXL"},
		})
	case "jenis_institusi":
		response.Success(c, http.StatusOK, "Data referensi jenis institusi berhasil diambil", []gin.H{
			{"id": "1", "name": "SMA", "pendidikan_id": "SMA"},
			{"id": "2", "name": "SMK", "pendidikan_id": "SMA"},
			{"id": "3", "name": "MA", "pendidikan_id": "SMA"},
			{"id": "4", "name": "MAK", "pendidikan_id": "SMA"},
		})
	case "provinsi":
		response.Success(c, http.StatusOK, "Data referensi provinsi berhasil diambil", []gin.H{
			{"id": "11", "name": "ACEH"},
			{"id": "12", "name": "SUMATERA UTARA"},
			{"id": "13", "name": "SUMATERA BARAT"},
			{"id": "31", "name": "DKI JAKARTA"},
			{"id": "32", "name": "JAWA BARAT"},
			{"id": "33", "name": "JAWA TENGAH"},
			{"id": "34", "name": "DI YOGYAKARTA"},
			{"id": "35", "name": "JAWA TIMUR"},
			{"id": "36", "name": "BANTEN"},
		})
	case "jenis_program":
		response.Success(c, http.StatusOK, "Data referensi jenis program berhasil diambil", []gin.H{
			{"id": "IPA", "name": "IPA"},
			{"id": "IPS", "name": "IPS"},
			{"id": "REG", "name": "Reguler"},
		})
	default:
		response.NotFound(c, "Jenis referensi tidak dikenali")
	}
}

