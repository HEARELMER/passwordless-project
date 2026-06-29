# Esquema de Base de Datos PostgreSQL: Sistema FIDO2 Passwordless

**Motor:** PostgreSQL 15+  
**Diseño:** Relaciones manejadas por aplicación (sin Foreign Keys físicas, optimizado con índices).  
**Características aprovechadas:** `UUID` nativo, `BYTEA` (binarios), `JSONB` (documentos), `INET` (redes), `TIMESTAMPTZ` (zonas horarias).

---

## 1. Entidad: `users` (Usuarios)

Almacena la identidad base. `JSONB` permite metadatos flexibles sin alterar el esquema.

```sql
CREATE TABLE IF NOT EXISTS users (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username    VARCHAR(255) UNIQUE NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    preferences JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

> **Por qué `TIMESTAMPTZ`:** En sistemas de autenticación globales, siempre se guarda la hora con zona horaria (UTC) para evitar inconsistencias en auditorías.

El campo `updated_at` se mantiene automáticamente mediante un trigger (`trg_users_updated_at`) que invoca la función `set_updated_at()`.

---

## 2. Entidad: `credentials` (Llaves de Dispositivos / Auditoría)

Tabla maestra que une la criptografía del celular (Autenticador) con el contexto de la conexión web (Cliente).

```sql
CREATE TABLE IF NOT EXISTS credentials (
    id                    BYTEA       PRIMARY KEY,
    user_id               UUID        NOT NULL,
    public_key            BYTEA       NOT NULL,
    attestation_type      VARCHAR(50) NOT NULL,
    aaguid                BYTEA,
    sign_count            BIGINT      NOT NULL DEFAULT 0,

    -- Campos de auditoría avanzada
    last_used_at          TIMESTAMPTZ,
    last_login_ip         INET,
    client_info           JSONB       NOT NULL DEFAULT '{}'::jsonb,
    raw_registration_data JSONB,

    created_at            TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_credentials_user_id    ON credentials (user_id);
CREATE INDEX IF NOT EXISTS idx_credentials_client_info ON credentials USING GIN (client_info);
```

> **Por qué `INET`:** PostgreSQL valida automáticamente que el valor sea una IP real (IPv4/IPv6).  
> **Importante (pgx/v5):** El driver pgx v5 mapea columnas `INET` al tipo Go `netip.Addr`, **no** a `net.IP`. Usar `net.IP` en el `Scan` causa un error en runtime.

> **Por qué `JSONB` en `client_info`:** El User-Agent cambia frecuentemente. Con `JSONB` el backend puede insertar un objeto JSON parseado directamente. El índice GIN permite consultas ultra-rápidas dentro del JSON, por ejemplo: `WHERE client_info @> '{"os": "Windows"}'`.

---

## 3. Entidad: `webauthn_sessions` (Desafíos Temporales)

Maneja el estado efímero del protocolo WebAuthn.

```sql
CREATE TABLE IF NOT EXISTS webauthn_sessions (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID         NOT NULL,
    challenge    VARCHAR(255) NOT NULL,
    type         VARCHAR(20)  NOT NULL CHECK (type IN ('REGISTRATION', 'AUTHENTICATION')),
    session_data JSONB        NOT NULL,
    expires_at   TIMESTAMPTZ  NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id   ON webauthn_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON webauthn_sessions (expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_challenge  ON webauthn_sessions (challenge);
```

> **Por qué `session_data JSONB`:** La librería `go-webauthn` genera un objeto de sesión completo con variables internas. En lugar de crear 10 columnas, se serializa ese objeto en un solo campo JSONB.  
> **Por qué el `CHECK` en `type`:** Garantiza a nivel de base de datos que nunca exista un tipo de sesión inválido.

---

## 4. Extensión requerida

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

`gen_random_uuid()` requiere `pgcrypto` en PostgreSQL < 14. En versiones 14+ está disponible de forma nativa; la instrucción es inofensiva si ya existe.

---

## 5. Flujo de Auditoría (Ejemplo de Actualización)

Cuando un usuario inicia sesión, el servidor Go ejecuta:

```sql
UPDATE credentials
SET
    sign_count    = $1,
    last_used_at  = CURRENT_TIMESTAMP,
    last_login_ip = $2,        -- '190.23.1.5' (Go pasa netip.Addr)
    client_info   = $3         -- '{"os": "Windows", "browser": "Chrome"}'::jsonb
WHERE id = $4;                 -- ID de la credencial usada (BYTEA)
```

---

## 6. Resumen de decisiones de tipos

| Campo | Tipo PostgreSQL | Tipo Go (pgx/v5) | Razón |
|---|---|---|---|
| `id` (users) | `UUID` | `uuid.UUID` | Identificador global único sin colisiones |
| `id` (credentials) | `BYTEA` | `[]byte` | El Credential ID es binario arbitrario |
| `public_key` | `BYTEA` | `[]byte` | Datos criptográficos en formato DER/COSE |
| `last_login_ip` | `INET` | `*netip.Addr` | Validación nativa de IP; `nil` = NULL |
| `preferences`, `client_info`, `session_data` | `JSONB` | `map[string]any` | Documentos flexibles con índice GIN |
| `created_at`, `updated_at` | `TIMESTAMPTZ` | `time.Time` | Siempre UTC, auditado y sin ambigüedad |