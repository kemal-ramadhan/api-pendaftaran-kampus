-- Migrasi Up: Pembuatan Tabel Layanan NIM (pmb_nim)

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Tabel Format Konfigurasi Template NIM
CREATE TABLE IF NOT EXISTS nim_formats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_program_code VARCHAR(30) NULL, -- NULL berlaku sebagai format default global
    pattern VARCHAR(100) NOT NULL DEFAULT '{YY}{nim_code_prodi}{jalur_digit}{urut:4}',
    description VARCHAR(255) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nim_formats_prodi ON nim_formats (study_program_code);
CREATE INDEX IF NOT EXISTS idx_nim_formats_active ON nim_formats (is_active);

-- Seed default global template
INSERT INTO nim_formats (id, study_program_code, pattern, description, is_active)
VALUES (
    gen_random_uuid(),
    NULL,
    '{YY}{nim_code_prodi}{jalur_digit}{urut:4}',
    'Format NIM Standar Universitas ({YY}=2 digit tahun, {nim_code_prodi}=2 digit prodi, {jalur_digit}=1 digit jalur, {urut:4}=4 digit urutan)',
    TRUE
) ON CONFLICT DO NOTHING;

-- 2. Tabel Sequence Generator (Kunci Mutex Antrian per Tahun & Program Studi)
CREATE TABLE IF NOT EXISTS nim_sequences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_year INT NOT NULL,
    study_program_code VARCHAR(30) NOT NULL,
    last_number INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_nim_sequences_year_prodi UNIQUE (entry_year, study_program_code)
);

CREATE INDEX IF NOT EXISTS idx_nim_sequences_lookup ON nim_sequences (entry_year, study_program_code);

-- 3. Tabel Riwayat Penerbitan NIM (Idempoten per application_id)
CREATE TABLE IF NOT EXISTS nim_issuances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL UNIQUE,
    user_id UUID NOT NULL,
    nim VARCHAR(30) NOT NULL UNIQUE,
    entry_year INT NOT NULL,
    study_program_code VARCHAR(30) NOT NULL,
    selection_path VARCHAR(50) NOT NULL,
    sequence_number INT NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nim_issuances_user ON nim_issuances (user_id);
CREATE INDEX IF NOT EXISTS idx_nim_issuances_prodi ON nim_issuances (study_program_code);
CREATE INDEX IF NOT EXISTS idx_nim_issuances_year ON nim_issuances (entry_year);
