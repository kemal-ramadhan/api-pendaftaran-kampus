# Class Diagram — Finance Service (Bagian 1: F2-M5)

Dokumen ini memuat diagram kelas dan dependensi antarmuka untuk **Finance Service** (`services/finance-service`).

```mermaid
classDiagram
    direction TB

    class InvoiceController {
        -invSvc InvoiceService
        +GetMyInvoices(c *gin.Context)
        +GetInvoiceDetail(c *gin.Context)
    }

    class PaymentController {
        -paySvc PaymentService
        +PayWithVirtualAccount(c *gin.Context)
        +PayWithManualTransfer(c *gin.Context)
        +GetPaymentDetail(c *gin.Context)
    }

    class ReceiptController {
        -paySvc PaymentService
        +GetReceipt(c *gin.Context)
    }

    class WebhookController {
        -paySvc PaymentService
        +HandleWebhook(c *gin.Context)
    }

    class DevController {
        -svc PaymentService
        -cfg *config.Config
        +SimulatePaid(c *gin.Context)
    }

    class AdminController {
        -paymentSvc PaymentService
        +ListManualTransfers(c *gin.Context)
        +VerifyPayment(c *gin.Context)
    }

    class InternalController {
        -invSvc InvoiceService
        +CreateInvoice(c *gin.Context)
        +GetInvoices(c *gin.Context)
    }

    class InvoiceService {
        <<interface>>
        +CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*Invoice, error)
        +GetMyInvoices(ctx context.Context, userID uuid.UUID) ([]Invoice, error)
        +GetInvoiceDetail(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*Invoice, error)
        +GetInvoiceByApplicationID(ctx context.Context, appID uuid.UUID) (*Invoice, error)
    }

    class PaymentService {
        <<interface>>
        +PayWithVirtualAccount(ctx context.Context, invoiceID, userID uuid.UUID) (*Payment, error)
        +PayWithManualTransfer(ctx context.Context, invoiceID, userID uuid.UUID, proofBytes []byte, fileName string) (*Payment, error)
        +ProcessWebhook(ctx context.Context, providerName string, signature string, payloadBytes []byte) error
        +SimulatePaid(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
        +VerifyManualTransfer(ctx context.Context, paymentID uuid.UUID, action, notes string, verifiedBy uuid.UUID) error
        +GetPaymentByID(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, role string) (*Payment, error)
        +GetReceiptByPaymentID(ctx context.Context, paymentID uuid.UUID, requesterID uuid.UUID, role string) (*Receipt, []byte, error)
        +ListAwaitingVerification(ctx context.Context, limit, offset int) ([]Payment, int64, error)
    }

    class PaymentProvider {
        <<interface>>
        +CreateVA(ctx context.Context, invoiceID uuid.UUID, amount int64) (*VAResult, error)
        +VerifyWebhookSignature(signature string, payload []byte) bool
        +ParseWebhookPayload(payload []byte) (*WebhookPayload, error)
    }

    class MockProvider {
        -secretKey string
    }

    class InvoiceRepository {
        <<interface>>
        +DB() *gorm.DB
        +Create(ctx context.Context, inv *Invoice) error
        +CreateInTx(tx *gorm.DB, inv *Invoice) error
        +FindByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
        +FindByUserID(ctx context.Context, userID uuid.UUID) ([]Invoice, error)
        +FindByApplicationID(ctx context.Context, appID uuid.UUID) (*Invoice, error)
        +FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*Invoice, error)
        +GenerateInvoiceNumber(ctx context.Context, tx *gorm.DB) (string, error)
        +UpdateStatus(ctx context.Context, id uuid.UUID, status InvoiceStatus, paidAmount int64) error
        +UpdateStatusInTx(tx *gorm.DB, id uuid.UUID, status InvoiceStatus, paidAmount int64) error
    }

    class PaymentRepository {
        <<interface>>
        +DB() *gorm.DB
        +CreateInTx(tx *gorm.DB, payment *Payment) error
        +FindByID(ctx context.Context, id uuid.UUID) (*Payment, error)
        +FindByProviderRef(ctx context.Context, providerRef string) (*Payment, error)
        +ListAwaitingVerification(ctx context.Context, limit, offset int) ([]Payment, int64, error)
        +UpdateStatusInTx(tx *gorm.DB, paymentID uuid.UUID, status PaymentStatus, notes string, verifiedBy *uuid.UUID, paidAt *time.Time) error
    }

    class ReceiptRepository {
        <<interface>>
        +CreateInTx(tx *gorm.DB, receipt *Receipt) error
        +FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*Receipt, error)
        +FindByReceiptNumber(ctx context.Context, receiptNo string) (*Receipt, error)
        +GenerateReceiptNumber(ctx context.Context, tx *gorm.DB) (string, error)
    }

    class WebhookLogRepository {
        <<interface>>
        +Create(ctx context.Context, log *PaymentWebhookLog) error
        +FindByProviderRef(ctx context.Context, providerRef string) (*PaymentWebhookLog, error)
    }

    InvoiceController --> InvoiceService
    PaymentController --> PaymentService
    ReceiptController --> PaymentService
    WebhookController --> PaymentService
    DevController --> PaymentService
    AdminController --> PaymentService
    InternalController --> InvoiceService

    PaymentService ..> PaymentProvider
    MockProvider ..|> PaymentProvider

    InvoiceService ..> InvoiceRepository
    PaymentService ..> PaymentRepository
    PaymentService ..> InvoiceRepository
    PaymentService ..> ReceiptRepository
    PaymentService ..> WebhookLogRepository
```
