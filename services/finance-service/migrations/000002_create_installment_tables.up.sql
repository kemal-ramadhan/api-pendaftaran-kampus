-- Migration Up: Create Installment Tables and Update Payments
CREATE TABLE IF NOT EXISTS installment_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    requested_terms INT NOT NULL,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by UUID,
    review_note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS installments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    term_no INT NOT NULL,
    amount BIGINT NOT NULL,
    due_date TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'unpaid',
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE payments ADD COLUMN IF NOT EXISTS installment_id UUID REFERENCES installments(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_installment_requests_invoice_id ON installment_requests(invoice_id);
CREATE INDEX IF NOT EXISTS idx_installment_requests_status ON installment_requests(status);
CREATE INDEX IF NOT EXISTS idx_installments_invoice_id ON installments(invoice_id);
CREATE INDEX IF NOT EXISTS idx_installments_status ON installments(status);
CREATE INDEX IF NOT EXISTS idx_installments_due_date ON installments(due_date);
CREATE INDEX IF NOT EXISTS idx_payments_installment_id ON payments(installment_id);
