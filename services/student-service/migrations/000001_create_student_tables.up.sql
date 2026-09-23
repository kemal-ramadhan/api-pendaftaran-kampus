-- Migrasi 000001: Membuat tabel reregistrations, students, dan outbox_events

CREATE TABLE IF NOT EXISTS reregistrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL UNIQUE,
    user_id UUID NOT NULL,
    accepted_program VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'waiting_payment',
    jacket_size VARCHAR(10),
    emergency_contact_name VARCHAR(150),
    emergency_contact_phone VARCHAR(30),
    statement_agreed BOOLEAN NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reregistrations_user_id ON reregistrations(user_id);
CREATE INDEX IF NOT EXISTS idx_reregistrations_status ON reregistrations(status);

CREATE TABLE IF NOT EXISTS students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    application_id UUID NOT NULL UNIQUE,
    nim VARCHAR(30),
    study_program_code VARCHAR(30) NOT NULL,
    entry_year INT NOT NULL,
    entry_path VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'reregistered',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_students_prodi ON students(study_program_code);
CREATE INDEX IF NOT EXISTS idx_students_status ON students(status);
CREATE INDEX IF NOT EXISTS idx_students_entry_year ON students(entry_year);

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_status ON outbox_events(status);
