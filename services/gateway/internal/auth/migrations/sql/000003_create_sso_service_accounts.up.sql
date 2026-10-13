-- CR-012: Auth Dua Versi (Core/SSO vs PMB) & Service Accounts

CREATE TABLE IF NOT EXISTS service_accounts (
    id UUID PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    product_group VARCHAR(50) NOT NULL DEFAULT 'pmb',
    base_url VARCHAR(255),
    audience VARCHAR(100) NOT NULL DEFAULT 'pmb-api',
    client_type VARCHAR(30) NOT NULL DEFAULT 'public',
    client_secret_hash VARCHAR(255),
    redirect_uris TEXT,
    post_logout_redirect_uris TEXT,
    allowed_scopes VARCHAR(255) DEFAULT 'openid profile email',
    first_party BOOLEAN DEFAULT TRUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS user_accounts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    service_account_id UUID NOT NULL REFERENCES service_accounts(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_access_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uq_user_service UNIQUE(user_id, service_account_id)
);

CREATE TABLE IF NOT EXISTS user_account_roles (
    user_account_id UUID NOT NULL REFERENCES user_accounts(id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope_type VARCHAR(30) NOT NULL DEFAULT '',
    scope_id VARCHAR(50) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY(user_account_id, role_id, scope_type, scope_id)
);

CREATE TABLE IF NOT EXISTS signing_keys (
    kid VARCHAR(50) PRIMARY KEY,
    public_key TEXT NOT NULL,
    algorithm VARCHAR(20) NOT NULL DEFAULT 'RS256',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ
);

ALTER TABLE roles ADD COLUMN IF NOT EXISTS service_account_id UUID;
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS service_account_id UUID;
