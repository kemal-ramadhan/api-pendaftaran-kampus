package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
)

type InstallmentController struct {
	installmentSvc service.InstallmentService
}

func NewInstallmentController(installmentSvc service.InstallmentService) *InstallmentController {
	return &InstallmentController{installmentSvc: installmentSvc}
}

// RequestInstallment menangani pengajuan cicilan pembayaran daftar ulang oleh calon mahasiswa
func (ctrl *InstallmentController) RequestInstallment(c *gin.Context) {
	invoiceIDStr := c.Param("id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format id invoice tidak valid")
		return
	}

	var payload model.CreateInstallmentRequestPayload
	if !response.BindJSON(c, &payload) {
		return
	}

	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}

	terms := payload.RequestedTerms
	if terms == 0 {
		terms = payload.NumTerms
	}
	if terms < 2 || terms > 4 {
		response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeValidationError, "Jumlah termin harus antara 2 sampai 4", nil)
		return
	}

	req, err := ctrl.installmentSvc.RequestInstallment(c.Request.Context(), invoiceID, userID, terms, payload.Reason)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		if errors.Is(err, service.ErrInstallmentNotAllowed) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeInstallmentNotAllowed, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrInvalidInstallmentTerms) || errors.Is(err, service.ErrInstallmentRequestAlreadyExist) || errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeValidationError, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mengajukan permohonan cicilan: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Pengajuan cicilan pembayaran berhasil dikirim dan menunggu persetujuan", req)
}

// ListInstallmentRequests menampilkan daftar pengajuan cicilan untuk staf keuangan
func (ctrl *InstallmentController) ListInstallmentRequests(c *gin.Context) {
	status := c.Query("status")
	list, err := ctrl.installmentSvc.ListInstallmentRequests(c.Request.Context(), status)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar pengajuan cicilan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar pengajuan cicilan berhasil diambil", list)
}

// ReviewInstallmentRequest memproses persetujuan atau penolakan cicilan oleh staf keuangan
func (ctrl *InstallmentController) ReviewInstallmentRequest(c *gin.Context) {
	reqIDStr := c.Param("id")
	reqID, err := uuid.Parse(reqIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format id pengajuan tidak valid")
		return
	}

	var payload model.ReviewInstallmentRequestPayload
	if !response.BindJSON(c, &payload) {
		return
	}

	reviewerID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}

	notes := payload.ReviewNote
	if notes == "" {
		notes = payload.Notes
	}

	req, err := ctrl.installmentSvc.ReviewInstallmentRequest(c.Request.Context(), reqID, reviewerID, payload.Action, notes)
	if err != nil {
		if errors.Is(err, service.ErrInstallmentAlreadyProcessed) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeInvalidState, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal meninjau permohonan cicilan: "+err.Error())
		return
	}

	msg := "Pengajuan cicilan berhasil disetujui dan jadwal termin pembayaran telah dibuat"
	if payload.Action == "reject" {
		msg = "Pengajuan cicilan telah ditolak"
	}

	response.Success(c, http.StatusOK, msg, req)
}

// GetInstallmentSchedule mengambil jadwal termin pembayaran cicilan untuk suatu tagihan
func (ctrl *InstallmentController) GetInstallmentSchedule(c *gin.Context) {
	invoiceIDStr := c.Param("id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format id invoice tidak valid")
		return
	}

	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}

	roles := gatewayauth.GetUserRoles(c)
	userRole := ""
	if len(roles) > 0 {
		userRole = roles[0]
	}

	installments, err := ctrl.installmentSvc.GetInstallmentsByInvoiceID(c.Request.Context(), invoiceID, userID, userRole)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil jadwal termin cicilan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Jadwal termin cicilan berhasil diambil", installments)
}
