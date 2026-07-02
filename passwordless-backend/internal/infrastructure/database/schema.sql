CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username    VARCHAR(255) UNIQUE NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    preferences JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgname = 'trg_users_updated_at'
    ) THEN
        CREATE TRIGGER trg_users_updated_at
        BEFORE UPDATE ON users
        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
    END IF;
END;
$$;

CREATE TABLE IF NOT EXISTS credentials (
    id                    BYTEA        PRIMARY KEY,
    user_id               UUID         NOT NULL,
    public_key            BYTEA        NOT NULL,
    attestation_type      VARCHAR(50)  NOT NULL,
    aaguid                BYTEA,
    sign_count            BIGINT       NOT NULL DEFAULT 0,
    is_active             BOOLEAN      NOT NULL DEFAULT TRUE,
    last_used_at          TIMESTAMPTZ,
    last_login_ip         INET,
    client_info           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    raw_registration_data JSONB,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_credentials_user_id ON credentials (user_id);
CREATE INDEX IF NOT EXISTS idx_credentials_user_active ON credentials (user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_credentials_client_info ON credentials USING GIN (client_info);

DROP TABLE IF EXISTS webauthn_sessions CASCADE;
CREATE TABLE webauthn_sessions (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID         NOT NULL,
    challenge    VARCHAR(255) NOT NULL,
    type         VARCHAR(20)  NOT NULL CHECK (type IN ('REGISTRATION', 'AUTHENTICATION')),
    status       VARCHAR(20)  NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'COMPLETED', 'EXPIRED', 'FAILED')),
    session_data JSONB        NOT NULL,
    user_agent   TEXT,
    ip_address   INET,
    expires_at   TIMESTAMPTZ  NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON webauthn_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user_status ON webauthn_sessions (user_id, status);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON webauthn_sessions (expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_challenge ON webauthn_sessions (challenge);
