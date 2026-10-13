ALTER TABLE nim_issuances ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
CREATE INDEX IF NOT EXISTS idx_nim_issuances_status ON nim_issuances (status);
