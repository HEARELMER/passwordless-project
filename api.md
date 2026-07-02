# Explicación Detallada de los Endpoints (Mobile API)

A continuación se detalla qué hace cada endpoint de la API Passwordless WebAuthn.
Esta API es "Stateless", lo que significa que el estado temporal entre peticiones (begin/finish) se comunica explícitamente vía JSON y Headers, sin usar Cookies.

---

## 1. `POST /api/webauthn/register/begin`

**Para qué sirve:** 
Es el **primer paso** para registrar a un nuevo usuario o añadir un nuevo dispositivo. El backend toma el correo y nombre, y genera un "desafío" criptográfico. También devuelve un `session_id` temporal.

**Ejemplo de Body (Lo que envías desde la App):**
```json
{
  "username": "elmer_123",
  "email": "elmer@midominio.com"
}
```

**Ejemplo de Respuesta (Lo que devuelve el Backend):**
```json
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "publicKey": {
    "challenge": "j4K9Lq2zP...", 
    "rp": {
      "name": "Passwordless App",
      "id": "localhost"
    },
    "user": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "name": "elmer_123",
      "displayName": "elmer_123"
    },
    "pubKeyCredParams": [
      { "type": "public-key", "alg": -7 },
      { "type": "public-key", "alg": -257 }
    ],
    "timeout": 60000
  }
}
```
*   **Nota:** Debes guardar el `session_id` y usar el objeto `publicKey` para pasárselo al cliente FIDO2 de Google Play Services (o equivalente).

---

## 2. `POST /api/webauthn/register/finish`

**Para qué sirve:** 
Es el **segundo paso** del registro. Recibe el JSON con la respuesta al "desafío" y la llave pública. Valida criptográficamente los datos y guarda la Passkey en la base de datos. Requiere que envíes el Header `X-Session-ID` para saber a qué sesión pertenece este intento.

**Headers Obligatorios:**
`X-Session-ID: <session_id_del_paso_anterior>`

**Ejemplo de Body (Generado por FIDO2):**
*No lo escribes tú a mano, es el resultado del escáner biométrico.*
```json
{
  "id": "base64-url-encoded-credential-id",
  "rawId": "base64-url-encoded-credential-id",
  "type": "public-key",
  "response": {
    "clientDataJSON": "base64-data-sobre-el-desafio",
    "attestationObject": "base64-data-con-la-llave-publica"
  }
}
```

**Ejemplo de Respuesta:**
```json
{
  "status": "ok"
}
```

---

## 3. `POST /api/webauthn/login/begin`

**Para qué sirve:** 
Es el **primer paso** para Iniciar Sesión. El servidor crea un desafío, devuelve las credenciales permitidas (Passkeys vinculados), e inyecta un `session_id`.

**Ejemplo de Body (Lo que envías):**
```json
{
  "username": "elmer_123"
}
```

**Ejemplo de Respuesta:**
```json
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "publicKey": {
    "challenge": "XyZ123abc...",
    "timeout": 60000,
    "rpId": "localhost",
    "allowCredentials": [
      {
        "type": "public-key",
        "id": "base64-url-encoded-credential-id-que-guardamos-antes"
      }
    ],
    "userVerification": "preferred"
  }
}
```
*   **Nota:** Extrae el `publicKey` para la API de FIDO2 y guarda el `session_id` temporalmente para el paso 2.

---

## 4. `POST /api/webauthn/login/finish`

**Para qué sirve:** 
Es el **segundo y último paso** del Login. Recibe la firma y la valida. Si la firma es válida, retorna el Token JWT.

**Headers Obligatorios:**
`X-Session-ID: <session_id_del_paso_anterior>`

**Ejemplo de Body (Generado por FIDO2):**
```json
{
  "id": "base64-credential-id",
  "rawId": "base64-credential-id",
  "type": "public-key",
  "response": {
    "clientDataJSON": "base64-encoded-data",
    "authenticatorData": "base64-encoded-data",
    "signature": "firma-criptografica-unica-base64",
    "userHandle": "123e4567-e89b-12d3-a456-426614174000"
  }
}
```

**Ejemplo de Respuesta:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2...",
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "expires_in": 86400
}
```

---

## 5. `GET /api/me`

**Para qué sirve:** 
Devuelve la información de perfil del usuario autenticado.

**Headers Obligatorios:**
`Authorization: Bearer <token_jwt>`

**Ejemplo de Respuesta:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "elmer_123",
  "email": "elmer@midominio.com",
  "created_at": "2026-06-29T10:00:00Z"
}
```

---

## 6. `GET /api/credentials`

**Para qué sirve:** 
Muestra una lista de todos los dispositivos de seguridad (Passkeys) que el usuario tiene vinculados a su cuenta.

**Headers Obligatorios:**
`Authorization: Bearer <token_jwt>`

**Ejemplo de Respuesta:**
```json
[
  {
    "id": "base64-credential-id",
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "is_active": true,
    "last_used_at": "2026-06-29T10:30:00Z",
    "client_info": {
      "user_agent": "okhttp/4.10.0"
    },
    "created_at": "2026-06-29T10:00:00Z"
  }
]
```

---

## 7. `DELETE /api/credentials/{id}`

**Para qué sirve:** 
Permite al usuario revocar el acceso a un dispositivo específico eliminando su llave vinculada.

**Headers Obligatorios:**
`Authorization: Bearer <token_jwt>`

**Cómo se envía en la URL:** 
Debes reemplazar `{id}` por el id exacto de la credencial (el base64).

**Ejemplo de Respuesta:**
```json
{
  "status": "ok"
}
```
