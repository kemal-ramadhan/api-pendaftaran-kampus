CREATE TABLE IF NOT EXISTS refund_reviews (
    id UUID PRIMARY KEY,
    payment_id UUID NOT NULL,
    invoice_id UUID NOT NULL,
    application_id UUID NOT NULL,
    amount BIGINT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    notes TEXT,
    reviewed_by UUID,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_refund_reviews_payment UNIQUE (payment_id)
);

CREATE INDEX IF NOT EXISTS idx_refund_reviews_app ON refund_reviews (application_id);
CREATE INDEX IF NOT EXISTS idx_refund_reviews_status ON refund_reviews (status);
