package admission

import (
	"context"

	"github.com/google/uuid"
)

type ApplicationDTO struct {
	ID                 uuid.UUID `json:"id"`
	RegistrationNumber string    `json:"registration_number"`
	UserID             uuid.UUID `json:"user_id"`
	PeriodID           uuid.UUID `json:"period_id"`
	AdmissionPathID    uuid.UUID `json:"admission_path_id"`
	FirstProgramID     uuid.UUID `json:"first_program_id"`
	FirstProgramCode   string    `json:"first_program_code"`
	SecondProgramID    *uuid.UUID `json:"second_program_id,omitempty"`
	Status             string    `json:"status"`
	PaymentStatus      string    `json:"payment_status"`
	InvoiceID          *uuid.UUID `json:"invoice_id,omitempty"`
}

type API interface {
	GetApplication(ctx context.Context, appID uuid.UUID) (*ApplicationDTO, error)
	GetApplicationByUserID(ctx context.Context, userID uuid.UUID) (*ApplicationDTO, error)
	UpdateApplicationStatus(ctx context.Context, appID uuid.UUID, status string) error
	UpdatePaymentStatus(ctx context.Context, appID uuid.UUID, isPaid bool) error
}
