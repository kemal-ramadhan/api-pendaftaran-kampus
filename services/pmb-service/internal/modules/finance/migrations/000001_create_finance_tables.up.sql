-- Migration Up: Create Finance Tables
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS fee_schemes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    period_id INT NOT NULL,
    admission_path_id INT NOT NULL,
    study_program_id UUID,
    fee_type VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    installment_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    max_installments INT NOT NULL DEFAULT 1,
    min_first_payment_percent INT NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    user_id UUID NOT NULL,
    application_id UUID,
    fee_type VARCHAR(50) NOT NULL,
    description TEXT,
    amount BIGINT NOT NULL,
    paid_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'unpaid',
    due_date TIMESTAMP WITH TIME ZONE NOT NULL,
    payment_scheme VARCHAR(20) NOT NULL DEFAULT 'full',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    method VARCHAR(30) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'mock',
    provider_ref VARCHAR(100) UNIQUE NOT NULL,
    va_number VARCHAR(50),
    amount BIGINT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    proof_object_key VARCHAR(255),
    proof_url VARCHAR(500),
    notes TEXT,
    verified_by UUID,
    verified_at TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payment_id UUID UNIQUE NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    receipt_number VARCHAR(50) UNIQUE NOT NULL,
    pdf_object_key VARCHAR(255) NOT NULL,
    pdf_url VARCHAR(500),
    amount BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payment_webhook_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(50) NOT NULL,
    provider_ref VARCHAR(100) NOT NULL,
    payload TEXT NOT NULL,
    signature VARCHAR(255),
    status VARCHAR(30) NOT NULL DEFAULT 'processed',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_invoices_user_id ON invoices(user_id);
CREATE INDEX IF NOT EXISTS idx_invoices_application_id ON invoices(application_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id ON payments(invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_provider_ref ON payments(provider_ref);
