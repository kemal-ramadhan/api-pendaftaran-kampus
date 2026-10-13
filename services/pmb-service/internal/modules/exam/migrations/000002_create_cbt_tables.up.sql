-- Migration 000002: Create CBT Tables (CR-003)
CREATE TABLE IF NOT EXISTS question_banks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(100) NOT NULL,
    path_id UUID NULL,
    study_program_code VARCHAR(50) NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_id UUID NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    type VARCHAR(30) NOT NULL, -- single_choice, multiple_choice, true_false, essay
    content TEXT NOT NULL,
    image_object_key VARCHAR(255) NULL,
    points INT NOT NULL DEFAULT 10,
    difficulty VARCHAR(20) NOT NULL DEFAULT 'medium',
    explanation TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_questions_bank_id ON questions(bank_id);

CREATE TABLE IF NOT EXISTS question_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    label VARCHAR(5) NOT NULL,
    content TEXT NOT NULL,
    image_object_key VARCHAR(255) NULL,
    is_correct BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_question_options_q_id ON question_options(question_id);

CREATE TABLE IF NOT EXISTS cbt_exams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    instructions TEXT NULL,
    duration_minutes INT NOT NULL DEFAULT 90,
    question_count INT NOT NULL DEFAULT 50,
    randomize_questions BOOLEAN NOT NULL DEFAULT true,
    randomize_options BOOLEAN NOT NULL DEFAULT true,
    show_result_to_participant BOOLEAN NOT NULL DEFAULT false,
    requires_access_token BOOLEAN NOT NULL DEFAULT true,
    access_token_hash VARCHAR(255) NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT uq_cbt_session UNIQUE(session_id)
);

CREATE TABLE IF NOT EXISTS cbt_exam_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cbt_exam_id UUID NOT NULL REFERENCES cbt_exams(id) ON DELETE CASCADE,
    bank_id UUID NOT NULL REFERENCES question_banks(id) ON DELETE RESTRICT,
    difficulty VARCHAR(20) NULL,
    count INT NOT NULL DEFAULT 10
);

CREATE TABLE IF NOT EXISTS cbt_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cbt_exam_id UUID NOT NULL REFERENCES cbt_exams(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'not_started',
    started_at TIMESTAMPTZ NULL,
    deadline_at TIMESTAMPTZ NULL,
    submitted_at TIMESTAMPTZ NULL,
    question_order TEXT NULL,
    objective_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    essay_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    final_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    graded_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_exam_participant UNIQUE(cbt_exam_id, participant_id)
);
CREATE INDEX IF NOT EXISTS idx_cbt_attempts_user ON cbt_attempts(user_id);

CREATE TABLE IF NOT EXISTS cbt_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID NOT NULL REFERENCES cbt_attempts(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    selected_option_ids TEXT NULL,
    essay_text TEXT NULL,
    is_flagged BOOLEAN NOT NULL DEFAULT false,
    is_correct BOOLEAN NULL,
    points_awarded NUMERIC(5,2) NOT NULL DEFAULT 0,
    answered_at TIMESTAMPTZ NULL,
    graded_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_attempt_question UNIQUE(attempt_id, question_id)
);

CREATE TABLE IF NOT EXISTS cbt_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID NOT NULL REFERENCES cbt_attempts(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    client_time TIMESTAMPTZ NULL,
    server_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT NULL
);
CREATE INDEX IF NOT EXISTS idx_cbt_events_attempt ON cbt_events(attempt_id);
