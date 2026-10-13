-- Migration up: create notification tables
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS notification_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    idempotency_key VARCHAR(100) UNIQUE NOT NULL,
    user_id UUID,
    email VARCHAR(255),
    phone VARCHAR(30),
    channel VARCHAR(20) NOT NULL,
    template VARCHAR(100) NOT NULL,
    data TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    retries INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notification_logs_user_id ON notification_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_logs_status ON notification_logs(status);
CREATE INDEX IF NOT EXISTS idx_notification_logs_idempotency_key ON notification_logs(idempotency_key);
