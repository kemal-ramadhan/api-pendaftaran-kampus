DROP INDEX IF EXISTS idx_nim_issuances_status;
ALTER TABLE nim_issuances DROP COLUMN IF EXISTS status;
