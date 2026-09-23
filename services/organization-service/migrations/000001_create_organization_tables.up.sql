-- 000001_create_organization_tables.up.sql

CREATE TABLE IF NOT EXISTS faculties (
    id SERIAL PRIMARY KEY,
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_faculties_deleted_at ON faculties(deleted_at);

CREATE TABLE IF NOT EXISTS study_programs (
    id SERIAL PRIMARY KEY,
    faculty_id INTEGER NOT NULL REFERENCES faculties(id) ON DELETE RESTRICT,
    code VARCHAR(20) NOT NULL UNIQUE,
    nim_code VARCHAR(5) NOT NULL,
    name VARCHAR(100) NOT NULL,
    degree VARCHAR(10) NOT NULL,
    accreditation VARCHAR(30) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_study_programs_faculty_id ON study_programs(faculty_id);
CREATE INDEX IF NOT EXISTS idx_study_programs_deleted_at ON study_programs(deleted_at);
CREATE INDEX IF NOT EXISTS idx_study_programs_code ON study_programs(code);
