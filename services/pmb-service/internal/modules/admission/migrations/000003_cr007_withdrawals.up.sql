-- CR-007: Tabel Pengunduran Diri (withdrawals)

CREATE TABLE IF NOT EXISTS withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL,
    user_id UUID NOT NULL,
    stage_at_request VARCHAR(50) NOT NULL,
    reason_category VARCHAR(50) NOT NULL,
    reason_text TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_by UUID,
    review_note TEXT,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    effective_at TIMESTAMP WITH TIME ZONE,
    source VARCHAR(20) NOT NULL DEFAULT 'applicant',
    attachment_object_key VARCHAR(255),
    letter_number VARCHAR(100),
    letter_object_key VARCHAR(255),
    verify_code_hash VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_app_id ON withdrawals (application_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals (user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_status ON withdrawals (status);
CREATE INDEX IF NOT EXISTS idx_withdrawals_verify_code ON withdrawals (verify_code_hash);
