package referral

import (
	"context"

	"github.com/google/uuid"
)

type CheckCodeResult struct {
	Valid           bool   `json:"valid"`
	InstitutionName string `json:"institution_name,omitempty"`
	PICName         string `json:"pic_name,omitempty"`
}

type API interface {
	CheckCode(ctx context.Context, code string) (*CheckCodeResult, error)
	GetApplicationReferral(ctx context.Context, applicationID uuid.UUID) (string, error)
	MarkInvoicePaidByPartner(ctx context.Context, invoiceID uuid.UUID, partnerUserID uuid.UUID) error
}
