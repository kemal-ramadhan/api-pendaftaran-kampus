-- Migration Up: CR-006 Data & Riwayat (application_changes, path_change_requests, applications columns, triggers, backfill)

-- 1. Tambah kolom ke applications bila belum ada
ALTER TABLE applications ADD COLUMN IF NOT EXISTS path_change_count_after_payment INT NOT NULL DEFAULT 0;
ALTER TABLE applications ADD COLUMN IF NOT EXISTS previous_application_id UUID REFERENCES applications(id);

-- Lepas constraint uq_user_period bila ada, agar pendaftaran baru setelah batal/gagal/mundur diizinkan
ALTER TABLE applications DROP CONSTRAINT IF EXISTS uq_user_period;

-- 2. Buat tabel application_changes (hanya insert)
CREATE TABLE IF NOT EXISTS application_changes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    change_type VARCHAR(50) NOT NULL,
    from_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    to_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason TEXT,
    fee_difference BIGINT NOT NULL DEFAULT 0,
    actor_id UUID,
    actor_role VARCHAR(50) NOT NULL DEFAULT 'applicant',
    source VARCHAR(20) NOT NULL DEFAULT 'applicant',
    related_request_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_application_changes_app_id ON application_changes(application_id);
CREATE INDEX IF NOT EXISTS idx_application_changes_user_id ON application_changes(user_id);
CREATE INDEX IF NOT EXISTS idx_application_changes_created_at ON application_changes(created_at);

-- 3. Trigger DB untuk mencegah UPDATE dan DELETE pada application_changes (immutable audit trail)
CREATE OR REPLACE FUNCTION prevent_application_changes_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Tabel application_changes bersifat immutable (hanya insert). Perubahan (UPDATE) dan penghapusan (DELETE) ditolak.';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_application_changes_update ON application_changes;
CREATE TRIGGER trg_prevent_application_changes_update
BEFORE UPDATE ON application_changes
FOR EACH ROW EXECUTE FUNCTION prevent_application_changes_modification();

DROP TRIGGER IF EXISTS trg_prevent_application_changes_delete ON application_changes;
CREATE TRIGGER trg_prevent_application_changes_delete
BEFORE DELETE ON application_changes
FOR EACH ROW EXECUTE FUNCTION prevent_application_changes_modification();

-- 4. Buat tabel path_change_requests
CREATE TABLE IF NOT EXISTS path_change_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    from_period_id INT NOT NULL REFERENCES periods(id),
    from_path_id INT NOT NULL REFERENCES admission_paths(id),
    from_choice_1_id UUID NOT NULL,
    from_choice_2_id UUID,
    to_period_id INT NOT NULL REFERENCES periods(id),
    to_path_id INT NOT NULL REFERENCES admission_paths(id),
    to_choice_1_id UUID NOT NULL,
    to_choice_2_id UUID,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    fee_difference BIGINT NOT NULL DEFAULT 0,
    reviewed_by UUID,
    review_note TEXT,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_path_change_requests_app_id ON path_change_requests(application_id);
CREATE INDEX IF NOT EXISTS idx_path_change_requests_user_id ON path_change_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_path_change_requests_status ON path_change_requests(status);

-- 5. Backfill dari application_status_logs & applications yang sudah ada
-- Entri 'created'
INSERT INTO application_changes (
    id, application_id, user_id, change_type, from_values, to_values,
    reason, fee_difference, actor_id, actor_role, source, created_at
)
SELECT 
    uuid_generate_v4(),
    a.id,
    a.user_id,
    'created',
    '{}'::jsonb,
    jsonb_build_object(
        'period_id', a.period_id,
        'path_id', a.admission_path_id,
        'choice_1_id', a.choice_1_id,
        'choice_2_id', a.choice_2_id
    ),
    'Pendaftaran baru dibuat',
    0,
    a.user_id,
    'calon_mahasiswa',
    'applicant',
    a.created_at
FROM applications a
WHERE NOT EXISTS (
    SELECT 1 FROM application_changes ac 
    WHERE ac.application_id = a.id AND ac.change_type = 'created'
);

-- Entri 'cancelled' dari log yang ada
INSERT INTO application_changes (
    id, application_id, user_id, change_type, from_values, to_values,
    reason, fee_difference, actor_id, actor_role, source, created_at
)
SELECT 
    uuid_generate_v4(),
    l.application_id,
    a.user_id,
    'cancelled',
    '{}'::jsonb,
    '{}'::jsonb,
    COALESCE(l.notes, 'Pendaftaran dibatalkan oleh calon mahasiswa'),
    0,
    COALESCE(l.changed_by, a.user_id),
    'calon_mahasiswa',
    'applicant',
    l.created_at
FROM application_status_logs l
JOIN applications a ON a.id = l.application_id
WHERE l.to_status = 'cancelled'
AND NOT EXISTS (
    SELECT 1 FROM application_changes ac 
    WHERE ac.application_id = l.application_id AND ac.change_type = 'cancelled'
);
