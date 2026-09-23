package controller

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
)

type InternalController struct {
	svc service.InvoiceService
}

func NewInternalController(svc service.InvoiceService) *InternalController {
	return &InternalController{svc: svc}
}

type CreateInvoiceInternalInput struct {
	UserID        string      `json:"user_id"`
	ApplicationID string      `json:"application_id"`
	FeeType       string      `json:"fee_type"`
	Amount        interface{} `json:"amount"` // bisa float atau int atau string dari payload
	Description   string      `json:"description"`
	DueDateDays   int         `json:"due_date_days"`
}

func (ctrl *InternalController) CreateInvoice(c *gin.Context) {
	var input CreateInvoiceInternalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "INVALID_PAYLOAD", "Payload invoice internal tidak valid: "+err.Error())
		return
	}

	userUUID, err := uuid.Parse(input.UserID)
	if err != nil {
		response.BadRequest(c, "INVALID_USER_ID", "ID user tidak valid")
		return
	}

	var appUUID *uuid.UUID
	if input.ApplicationID != "" {
		if parsed, err := uuid.Parse(input.ApplicationID); err == nil {
			appUUID = &parsed
		}
	}

	var intAmount int64
	switch v := input.Amount.(type) {
	case float64:
		intAmount = int64(v)
	case int64:
		intAmount = v
	case int:
		intAmount = int64(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			intAmount = i
		}
	}

	req := &service.CreateInvoiceRequest{
		UserID:        userUUID,
		ApplicationID: appUUID,
		FeeType:       input.FeeType,
		Amount:        intAmount,
		Description:   input.Description,
		DueDateDays:   input.DueDateDays,
	}

	inv, err := ctrl.svc.CreateInvoice(c.Request.Context(), req)
	if err != nil {
		response.InternalServerError(c, "Gagal membuat invoice tagihan: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Invoice berhasil dibuat", inv)
}

func (ctrl *InternalController) GetInvoices(c *gin.Context) {
	appIDStr := c.Query("application_id")
	if appIDStr == "" {
		response.BadRequest(c, "MISSING_PARAM", "Parameter application_id wajib diisi")
		return
	}

	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format application_id tidak valid")
		return
	}

	inv, err := ctrl.svc.GetInvoiceByApplicationID(c.Request.Context(), appID)
	if err != nil {
		response.NotFound(c, "Invoice untuk pendaftaran tersebut tidak ditemukan")
		return
	}

	response.Success(c, http.StatusOK, "Invoice berhasil ditemukan", inv)
}
