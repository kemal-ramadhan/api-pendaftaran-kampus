-- Migrasi up: Pembuatan tabel skema scoring service
CREATE TABLE IF NOT EXISTS score_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path_id UUID NOT NULL,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    weight_percent NUMERIC(5,2) NOT NULL,
    max_score NUMERIC(5,2) NOT NULL DEFAULT 100.00,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_path_component UNIQUE(path_id, code),
    CONSTRAINT chk_weight_positive CHECK(weight_percent > 0 AND weight_percent <= 100),
    CONSTRAINT chk_max_score CHECK(max_score > 0)
);

CREATE INDEX IF NOT EXISTS idx_score_components_path ON score_components(path_id);

CREATE TABLE IF NOT EXISTS scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL,
    component_code VARCHAR(50) NOT NULL,
    score NUMERIC(5,2) NOT NULL,
    input_by UUID NOT NULL,
    input_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_app_component UNIQUE(application_id, component_code),
    CONSTRAINT chk_score_range CHECK(score >= 0)
);

CREATE INDEX IF NOT EXISTS idx_scores_app ON scores(application_id);

CREATE TABLE IF NOT EXISTS score_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL UNIQUE,
    period_id UUID NOT NULL,
    path_id UUID NOT NULL,
    study_program_code VARCHAR(50) NOT NULL,
    final_score NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    rank_in_program INT NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_score_summaries_filter ON score_summaries(period_id, path_id, study_program_code);
CREATE INDEX IF NOT EXISTS idx_score_summaries_rank ON score_summaries(period_id, path_id, study_program_code, final_score DESC);

CREATE TABLE IF NOT EXISTS program_locks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id UUID NOT NULL,
    path_id UUID NOT NULL,
    study_program_code VARCHAR(50) NOT NULL,
    is_locked BOOLEAN NOT NULL DEFAULT false,
    locked_by UUID,
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_program_lock UNIQUE(period_id, path_id, study_program_code)
);

CREATE INDEX IF NOT EXISTS idx_program_locks_lookup ON program_locks(period_id, path_id, study_program_code);
