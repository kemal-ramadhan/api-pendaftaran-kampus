package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
)

type ApplicationController struct {
	svc service.AdmissionService
}

func NewApplicationController(svc service.AdmissionService) *ApplicationController {
	return &ApplicationController{svc: svc}
}

type CreateApplicationInput struct {
	PeriodID        uint       `json:"period_id" binding:"required"`
	AdmissionPathID uint       `json:"admission_path_id" binding:"required"`
	Choice1ID       uuid.UUID  `json:"choice_1_id" binding:"required"`
	Choice2ID       *uuid.UUID `json:"choice_2_id"`
}

type ChangeChoicesInput struct {
	Choice1ID uuid.UUID  `json:"choice_1_id" binding:"required"`
	Choice2ID *uuid.UUID `json:"choice_2_id"`
}

func (ctrl *ApplicationController) resolveApplicationID(c *gin.Context, userID uuid.UUID) (uuid.UUID, error) {
	idStr := c.Param("id")
	if idStr != "" && idStr != "me" {
		return uuid.Parse(idStr)
	}
	if qID := c.Query("application_id"); qID != "" {
		return uuid.Parse(qID)
	}
	apps, err := ctrl.svc.GetMyApplications(c.Request.Context(), userID)
	if err != nil || len(apps) == 0 {
		return uuid.Nil, errors.New("tidak ada berkas pendaftaran aktif yang ditemukan")
	}
	return apps[0].ID, nil
}

func (ctrl *ApplicationController) CreateApplication(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	var input CreateApplicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data permohonan pendaftaran tidak lengkap: "+err.Error())
		return
	}

	email := gatewayauth.GetUserEmail(c)
	name := c.GetHeader("X-User-Name")

	req := &service.CreateApplicationRequest{
		UserID:          userID,
		PeriodID:        input.PeriodID,
		AdmissionPathID: input.AdmissionPathID,
		Choice1ID:       input.Choice1ID,
		Choice2ID:       input.Choice2ID,
		UserEmail:       email,
		UserName:        name,
	}

	app, err := ctrl.svc.CreateApplication(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrPeriodClosed) {
			response.BadRequest(c, "PERIOD_CLOSED", err.Error())
			return
		}
		if errors.Is(err, service.ErrDuplicateApplication) {
			response.Conflict(c, "DUPLICATE_APPLICATION", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal membuat formulir pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Pendaftaran berhasil dibuat, silakan lakukan pembayaran", app)
}

func (ctrl *ApplicationController) GetMyApplications(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	apps, err := ctrl.svc.GetMyApplications(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar pendaftaran saya: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar pendaftaran berhasil diambil", apps)
}

func (ctrl *ApplicationController) GetApplicationDetail(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	roles := gatewayauth.GetUserRoles(c)
	role := "user"
	if len(roles) > 0 {
		role = roles[0]
	}

	app, err := ctrl.svc.GetApplicationDetail(c.Request.Context(), appID, userID, role, nil)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak diizinkan mengakses data pendaftaran ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil detail pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Detail pendaftaran berhasil diambil", app)
}

func (ctrl *ApplicationController) ChangeChoices(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	var input ChangeChoicesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data pilihan program studi tidak lengkap: "+err.Error())
		return
	}

	req := &service.ChangeChoicesRequest{
		ApplicationID: appID,
		UserID:        userID,
		Choice1ID:     input.Choice1ID,
		Choice2ID:     input.Choice2ID,
	}

	err = ctrl.svc.ChangeChoices(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak diizinkan mengubah pendaftaran ini")
			return
		}
		if errors.Is(err, service.ErrInvalidState) {
			response.BadRequest(c, "INVALID_STATE", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengubah pilihan program studi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pilihan program studi berhasil diperbarui", nil)
}

// UploadDocument menangani upload berkas persyaratan (multipart: file, document_type)
func (ctrl *ApplicationController) UploadDocument(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	appID, err := ctrl.resolveApplicationID(c, userID)
	if err != nil {
		response.BadRequest(c, "INVALID_APPLICATION_ID", err.Error())
		return
	}

	docType := c.PostForm("document_type")
	if docType == "" {
		docType = c.Query("document_type")
	}
	if docType == "" {
		response.BadRequest(c, "MISSING_DOCUMENT_TYPE", "Parameter document_type wajib diisi")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "MISSING_FILE", "File dokumen persyaratan tidak ditemukan pada payload form: "+err.Error())
		return
	}

	srcFile, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "Gagal membaca berkas dokumen yang diunggah")
		return
	}
	defer func() { _ = srcFile.Close() }()

	declaredMime := fileHeader.Header.Get("Content-Type")
	doc, err := ctrl.svc.UploadDocument(
		c.Request.Context(),
		appID,
		userID,
		docType,
		srcFile,
		fileHeader.Size,
		fileHeader.Filename,
		declaredMime,
	)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		if errors.Is(err, service.ErrDocUploadNotAllowed) {
			response.BadRequest(c, "INVALID_APPLICATION_STATUS", err.Error())
			return
		}
		if errors.Is(err, service.ErrDocTypeNotSupported) {
			response.BadRequest(c, "INVALID_DOCUMENT_TYPE", err.Error())
			return
		}
		if errors.Is(err, service.ErrDocTooLarge) {
			response.BadRequest(c, "FILE_TOO_LARGE", err.Error())
			return
		}
		if errors.Is(err, storage.ErrFileTypeNotAllowed) || strings.Contains(err.Error(), "tipe file tidak diizinkan") {
			response.Error(c, http.StatusUnsupportedMediaType, "FILE_TYPE_NOT_ALLOWED", "Tipe berkas tidak diizinkan atau header file tidak sesuai ekstensi: "+err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mengunggah dokumen persyaratan: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Dokumen persyaratan berhasil diunggah", doc)
}

// DeleteDocument menghapus dokumen yang telah diunggah
func (ctrl *ApplicationController) DeleteDocument(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	appID, err := ctrl.resolveApplicationID(c, userID)
	if err != nil {
		response.BadRequest(c, "INVALID_APPLICATION_ID", err.Error())
		return
	}

	reqCode := c.Param("code")
	if reqCode == "" {
		response.BadRequest(c, "MISSING_DOC_CODE", "Kode dokumen tidak boleh kosong")
		return
	}

	err = ctrl.svc.DeleteDocument(c.Request.Context(), appID, userID, reqCode)
	if err != nil {
		if errors.Is(err, repository.ErrDocumentNotFound) {
			response.NotFound(c, "Dokumen persyaratan tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrDocUploadNotAllowed) {
			response.BadRequest(c, "INVALID_APPLICATION_STATUS", err.Error())
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal menghapus dokumen: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Dokumen persyaratan berhasil dihapus", nil)
}

// GetDocumentURL mengambil presigned URL dokumen milik calon mahasiswa
func (ctrl *ApplicationController) GetDocumentURL(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	appID, err := ctrl.resolveApplicationID(c, userID)
	if err != nil {
		response.BadRequest(c, "INVALID_APPLICATION_ID", err.Error())
		return
	}

	reqCode := c.Param("code")
	roles := gatewayauth.GetUserRoles(c)
	role := "calon_mahasiswa"
	if len(roles) > 0 {
		role = roles[0]
	}

	url, err := ctrl.svc.GetDocumentPresignedURL(c.Request.Context(), appID, reqCode, userID, role)
	if err != nil {
		if errors.Is(err, repository.ErrDocumentNotFound) {
			response.NotFound(c, "Dokumen tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil URL dokumen: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "URL dokumen berhasil dibuat", gin.H{"url": url})
}

// SubmitApplication memvalidasi dan mengirimkan berkas pendaftaran untuk diverifikasi
func (ctrl *ApplicationController) SubmitApplication(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	appID, err := ctrl.resolveApplicationID(c, userID)
	if err != nil {
		response.BadRequest(c, "INVALID_APPLICATION_ID", err.Error())
		return
	}

	app, err := ctrl.svc.SubmitApplication(c.Request.Context(), appID, userID)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		if errors.Is(err, service.ErrSubmitNotAllowed) {
			response.BadRequest(c, "INVALID_APPLICATION_STATUS", err.Error())
			return
		}
		if errors.Is(err, service.ErrBiodataIncomplete) || strings.Contains(err.Error(), "BIODATA_INCOMPLETE") {
			response.Error(c, http.StatusBadRequest, "BIODATA_INCOMPLETE", err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrDocumentRequired) || strings.Contains(err.Error(), "DOCUMENT_REQUIRED") {
			response.Error(c, http.StatusBadRequest, "DOCUMENT_REQUIRED", err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mensubmit pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pendaftaran berhasil disubmit dan menunggu verifikasi panitia PMB", app)
}
