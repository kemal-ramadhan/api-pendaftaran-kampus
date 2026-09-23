-- Migration up: create identity tables
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Table persons
CREATE TABLE IF NOT EXISTS persons (
    user_id UUID PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL,
    nik TEXT NOT NULL,
    nik_hash VARCHAR(64) UNIQUE NOT NULL,
    birth_place VARCHAR(100),
    birth_date DATE,
    gender VARCHAR(10),
    religion VARCHAR(50),
    nationality VARCHAR(50) DEFAULT 'WNI',
    phone VARCHAR(30),
    photo_object_key VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_persons_deleted_at ON persons(deleted_at);
CREATE INDEX IF NOT EXISTS idx_persons_nik_hash ON persons(nik_hash);

-- 2. Table addresses
CREATE TABLE IF NOT EXISTS addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES persons(user_id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    street TEXT,
    rt VARCHAR(10),
    rw VARCHAR(10),
    village VARCHAR(100),
    district VARCHAR(100),
    city VARCHAR(100),
    province VARCHAR(100),
    postal_code VARCHAR(10),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_addresses_user_type UNIQUE(user_id, type)
);

CREATE INDEX IF NOT EXISTS idx_addresses_user_id ON addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_addresses_deleted_at ON addresses(deleted_at);

-- 3. Table guardians
CREATE TABLE IF NOT EXISTS guardians (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES persons(user_id) ON DELETE CASCADE,
    relation VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    occupation VARCHAR(100),
    income_range VARCHAR(100),
    phone VARCHAR(30),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_guardians_user_relation UNIQUE(user_id, relation)
);

CREATE INDEX IF NOT EXISTS idx_guardians_user_id ON guardians(user_id);
CREATE INDEX IF NOT EXISTS idx_guardians_deleted_at ON guardians(deleted_at);

-- 4. Table education_histories
CREATE TABLE IF NOT EXISTS education_histories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES persons(user_id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL,
    school_name VARCHAR(255) NOT NULL,
    npsn VARCHAR(20),
    major VARCHAR(100),
    graduation_year INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_education_histories_user_id ON education_histories(user_id);
CREATE INDEX IF NOT EXISTS idx_education_histories_deleted_at ON education_histories(deleted_at);

-- 5. Table ktp_scans
CREATE TABLE IF NOT EXISTS ktp_scans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    object_key VARCHAR(255) NOT NULL,
    job_id VARCHAR(100),
    status VARCHAR(30) NOT NULL,
    quality_score NUMERIC(5,2) DEFAULT 0,
    enhanced BOOLEAN DEFAULT FALSE,
    reason VARCHAR(50),
    reason_message TEXT,
    field_confidences JSONB,
    ocr_data_raw TEXT,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    fields_edited_by_user JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ktp_scans_user_id ON ktp_scans(user_id);
CREATE INDEX IF NOT EXISTS idx_ktp_scans_status ON ktp_scans(status);
