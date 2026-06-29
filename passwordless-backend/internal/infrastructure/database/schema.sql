-- ============================================================
-- Extensiones requeridas
-- ============================================================
-- gen_random_uuid() requiere pgcrypto en PostgreSQL < 14.
-- En PG 14+ está disponible como función nativa; la extensión
-- es inofensiva si ya existe.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- 1. Tabla: users
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username    VARCHAR(255) UNIQUE NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    -- JSONB: Metadatos flexibles (roles, preferencias) sin alterar el esquema.
    preferences JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- updated_at se actualiza automáticamente mediante trigger (ver abajo).
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Función y trigger para mantener updated_at sincronizado.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- El trigger se crea solo si no existe (idempotente).
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

-- ============================================================
-- 2. Tabla: credentials  (Llaves Públicas / Auditoría)
-- ============================================================
CREATE TABLE IF NOT EXISTS credentials (
    -- BYTEA: El Credential ID generado por el Enclave Seguro del celular.
    id                    BYTEA        PRIMARY KEY,
    -- Relación lógica con users.id (sin FK física para máximo rendimiento).
    user_id               UUID         NOT NULL,
    -- Llave pública ECDSA P-256 en formato COSE/DER.
    public_key            BYTEA        NOT NULL,
    -- Tipo de atestación: "packed", "none", "android-key", etc.
    attestation_type      VARCHAR(50)  NOT NULL,
    -- AAGUID: Identifica el modelo/fabricante del autenticador.
    aaguid                BYTEA,
    -- Contador de firmas: detecta clonación del dispositivo (CRÍTICO FIDO2).
    sign_count            BIGINT       NOT NULL DEFAULT 0,

    -- ── Campos de auditoría avanzada ─────────────────────────────────────
    -- Última autenticación exitosa.
    last_used_at          TIMESTAMPTZ,
    -- INET: PostgreSQL valida que sea una IP real (IPv4/IPv6).
    last_login_ip         INET,
    -- JSONB: User-Agent parseado. Ej: {"os":"Windows","browser":"Chrome"}
    client_info           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    -- JSONB: Datos crudos del registro (útil para depuración y auditoría).
    raw_registration_data JSONB,

    created_at            TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Índice principal: buscar todas las credenciales de un usuario.
CREATE INDEX IF NOT EXISTS idx_credentials_user_id
    ON credentials (user_id);

-- Índice GIN: consultas rápidas dentro del JSON de client_info.
-- Ej: WHERE client_info @> '{"os": "Windows"}'
CREATE INDEX IF NOT EXISTS idx_credentials_client_info
    ON credentials USING GIN (client_info);

-- ============================================================
-- 3. Tabla: webauthn_sessions  (Desafíos Temporales)
-- ============================================================
CREATE TABLE IF NOT EXISTS webauthn_sessions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Relación lógica con users.id.
    user_id      UUID        NOT NULL,
    -- El desafío aleatorio en Base64URL (enviado al autenticador).
    challenge    VARCHAR(255) NOT NULL,
    -- Tipo de operación: 'REGISTRATION' o 'AUTHENTICATION'.
    type         VARCHAR(20) NOT NULL CHECK (type IN ('REGISTRATION', 'AUTHENTICATION')),
    -- JSONB: Estado interno de la librería WebAuthn (go-webauthn serializa aquí).
    session_data JSONB       NOT NULL,
    -- La sesión muere en esta fecha. El servidor DEBE rechazar sesiones expiradas.
    expires_at   TIMESTAMPTZ NOT NULL
);

-- Índice: encontrar la sesión activa de un usuario por tipo.
CREATE INDEX IF NOT EXISTS idx_sessions_user_id
    ON webauthn_sessions (user_id);

-- Índice: limpiar sesiones expiradas eficientemente (DELETE WHERE expires_at < NOW()).
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at
    ON webauthn_sessions (expires_at);

-- Índice: lookup por challenge (usado en la verificación de autenticación).
CREATE INDEX IF NOT EXISTS idx_sessions_challenge
    ON webauthn_sessions (challenge);
