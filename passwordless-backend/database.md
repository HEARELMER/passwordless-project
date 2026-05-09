# Esquema de Base de Datos PostgreSQL: Sistema FIDO2 Passwordless
**Motor:** PostgreSQL 15+
**Diseño:** Relaciones manejadas por aplicación (Sin Foreign Keys físicas, optimizado con índices).
**Características aprovechadas:** `UUID` nativo, `BYTEA` (binarios), `JSONB` (documentos), `INET` (redes) y `TIMESTAMPTZ` (zonas horarias).

---

## 1. Entidad: `users` (Usuarios)
Almacena la identidad base. Utilizamos `JSONB` para permitir metadatos flexibles sin tener que alterar la tabla en el futuro.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    -- JSONB: Perfecto para guardar preferencias del usuario, 
    -- roles, o datos de perfil extra sin romper el esquema relacional.
    preferences JSONB DEFAULT '{}'::jsonb, 
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
Justificación TIMESTAMPTZ: En sistemas de autenticación globales, SIEMPRE se debe guardar la hora con su zona horaria (UTC) para evitar inconsistencias en las auditorías.

2. Entidad: credentials (Llaves de Dispositivos / Auditoría)
Esta es la tabla maestra. Aquí unimos la criptografía del celular (Autenticador) con el contexto de la conexión web (Cliente).

SQL
CREATE TABLE credentials (
    id BYTEA PRIMARY KEY,                      -- Credential ID generado por el celular
    user_id UUID NOT NULL,                     -- Relación lógica con users.id
    public_key BYTEA NOT NULL,                 -- Llave pública ECDSA
    attestation_type VARCHAR(50) NOT NULL,     -- Ej: "packed", "none", "android-key"
    aaguid BYTEA,                              -- ID del modelo de hardware (celular)
    sign_count BIGINT NOT NULL DEFAULT 0,      -- Prevención de clonación (Crítico)
    
    -- CAMPOS DE AUDITORÍA AVANZADA
    last_used_at TIMESTAMPTZ,                  -- Cuándo fue el último login exitoso
    last_login_ip INET,                        -- Tipo INET: Valida que sea una IP real (IPv4/IPv6)
    
    -- JSONB: Guardamos el User-Agent parseado del cliente web.
    -- Ejemplo de contenido: {"os": "Windows 11", "browser": "Chrome", "is_mobile": false}
    client_info JSONB DEFAULT '{}'::jsonb,     
    
    -- JSONB: (Opcional) Guardar datos crudos del registro por si hay que debugear.
    raw_registration_data JSONB,               
    
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Índices de optimización
CREATE INDEX idx_credentials_user_id ON credentials(user_id);
CREATE INDEX idx_credentials_client_info ON credentials USING GIN (client_info);
Justificación INET: Postgres optimiza las IPs. Te permite hacer consultas como "Bloquea todas las llaves que se conecten desde la subred 192.168.1.0/24".

Justificación JSONB en client_info: El User-Agent cambia mucho y a veces quieres guardar más cosas (ej. resolución de pantalla o idioma). Al usar JSONB, tu backend en Go puede parsear el header HTTP y meter un objeto JSON directamente ahí. Además, con el índice GIN, puedes hacer consultas SQL ultra rápidas dentro de ese JSON.

3. Entidad: webauthn_sessions (Desafíos Temporales)
Maneja el estado efímero del protocolo.

SQL
CREATE TABLE webauthn_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,                     -- Relación lógica con users.id
    challenge VARCHAR(255) NOT NULL,           -- El desafío en Base64URL
    type VARCHAR(20) NOT NULL,                 -- 'REGISTRATION' o 'AUTHENTICATION'
    
    -- JSONB: La librería de Go necesita recordar las opciones exactas
    -- que le mandó al cliente (ej. si le exigió huella o no) para validarlas a la vuelta.
    session_data JSONB NOT NULL,               
    
    expires_at TIMESTAMPTZ NOT NULL            -- Fecha exacta de muerte del desafío
);

-- Índices de optimización
CREATE INDEX idx_sessions_user_id ON webauthn_sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON webauthn_sessions(expires_at);
Justificación session_data JSONB: Las librerías de WebAuthn en Go (como go-webauthn) generan un objeto de sesión completo con variables internas. En lugar de crear 10 columnas en esta tabla, guardamos ese objeto serializado en un solo campo JSONB.

4. Flujo de Auditoría (Ejemplo de Actualización)
Cuando un usuario inicia sesión, tu servidor Go hará algo como esto:

Extrae la IP de la petición (ej. 190.23.1.5).

Lee el header User-Agent (ej. Mozilla/5.0 (Windows NT 10.0; Win64; x64)...).

Usa una librería en Go para parsearlo a: {"os": "Windows", "browser": "Chrome"}.

Si la criptografía (la firma digital) es correcta, ejecuta esta consulta de actualización:

SQL
UPDATE credentials 
SET 
    sign_count = $1, 
    last_used_at = CURRENT_TIMESTAMP,
    last_login_ip = $2,               -- '190.23.1.5'
    client_info = $3                  -- '{"os": "Windows", "browser": "Chrome"}'::jsonb
WHERE id = $4;                        -- ID de la credencial usada