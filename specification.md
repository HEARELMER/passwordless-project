# Especificación Lógica del Backend: Sistema Passwordless (FIDO2 / WebAuthn)
**Tecnología Principal:** Go (Golang)
**Base de Datos:** PostgreSQL

---

## 1. Modelo de Base de Datos (PostgreSQL)

Para manejar WebAuthn, necesitamos almacenar a los usuarios, sus llaves públicas (credenciales) y un mecanismo temporal para guardar los "desafíos" mientras el usuario aprueba la biometría en su celular.

### Tabla: `users`
Almacena la información básica de identidad.
* `id` (UUID, Primary Key): Identificador interno del usuario. (Equivale al `User Handle` en FIDO2).
* `username` (VARCHAR, Unique): El identificador que ingresa el usuario (ej. correo o alias).
* `created_at` (TIMESTAMP).

### Tabla: `credentials`
Almacena las Llaves Públicas generadas por la App Autenticadora de Kotlin. **Un usuario puede tener múltiples credenciales** (ej. si registra un celular nuevo).
* `credential_id` (BYTEA / Base64URL String, Primary Key): El ID único generado por el celular para esta llave específica.
* `user_id` (UUID, Foreign Key -> users.id).
* `public_key` (BYTEA): La llave pública en formato binario (generalmente codificada en formato COSE).
* `attestation_type` (VARCHAR): El formato de atestación (ej. "packed", "none").
* `aaguid` (BYTEA): Identificador del modelo de hardware del celular (opcional, útil para auditar qué dispositivos se conectan).
* `sign_count` (INTEGER): Contador de firmas. **CRÍTICO PARA SEGURIDAD**. FIDO2 exige llevar un conteo de cuántas veces se ha usado la llave para evitar que un dispositivo clonado intente iniciar sesión.
* `created_at` (TIMESTAMP).

### Tabla: `webauthn_sessions` (Desafíos Temporales)
Cuando el servidor envía un desafío, debe recordarlo para validarlo cuando regrese la firma.
* `session_id` (UUID, Primary Key).
* `user_id` (UUID, Foreign Key -> users.id).
* `challenge` (VARCHAR): El string aleatorio en Base64URL enviado al celular.
* `type` (VARCHAR): "REGISTRATION" o "AUTHENTICATION".
* `expires_at` (TIMESTAMP): Caducidad del desafío (ej. 2 minutos).

---

## 2. Flujo Lógico y Endpoints de la API

El protocolo requiere 4 endpoints obligatorios divididos en las dos ceremonias principales.

### Ceremonia A: Registro de Nueva Credencial

#### Endpoint 1: Iniciar Registro (`POST /api/webauthn/register/begin`)
**Objetivo:** El servidor genera un desafío para que el celular lo firme y cree una nueva llave.

1. **Entrada:** Recibe JSON con `username`.
2. **Lógica de Backend:**
   * Busca al usuario en la DB `users`. Si no existe, lo crea provisionalmente.
   * Busca en `credentials` si el usuario ya tiene llaves (para excluirlas y que no registre la misma llave dos veces).
   * Genera un `challenge` criptográficamente seguro (32 bytes aleatorios).
   * Guarda este `challenge` en la tabla `webauthn_sessions` atado al `user_id`.
3. **Salida (JSON enviado a Kotlin):**
   * `rp`: Información del Servidor (Nombre y Dominio).
   * `user`: Información del usuario (ID, username).
   * `challenge`: El desafío generado.
   * `pubKeyCredParams`: Tipos de algoritmos aceptados (ej. `ES256` para curva elíptica ECDSA).
   * `authenticatorSelection`: Configurado para exigir validación de usuario (Biometría obligatoria) y que la llave resida en el dispositivo (`residentKey: "required"`).

#### Endpoint 2: Finalizar Registro (`POST /api/webauthn/register/finish`)
**Objetivo:** Recibir la Llave Pública creada y validarla.

1. **Entrada:** JSON (`AttestationResponse`) proveniente de Kotlin, que contiene el `credential_id`, la `public_key` encriptada, el `clientDataJSON` y la firma de atestación.
2. **Lógica de Backend (Validación Estricta):**
   * Recupera el `challenge` guardado en `webauthn_sessions` para este usuario.
   * **Verificación 1:** El `challenge` devuelto por el celular debe coincidir EXACTAMENTE con el guardado en la DB.
   * **Verificación 2:** El dominio de origen (`origin`) en el `clientDataJSON` debe ser exactamente tu dominio o App Package Name. (Previene Phishing).
   * **Verificación 3:** Extrae y decodifica la Llave Pública (`public_key`) del formato COSE.
3. **Transacción en DB:**
   * `INSERT INTO credentials` guardando el `credential_id`, el `user_id`, la `public_key` decodificada y configurando `sign_count = 0`.
   * Borra el desafío de `webauthn_sessions`.
4. **Salida:** Respuesta `200 OK`. (Usuario registrado exitosamente).

---

### Ceremonia B: Inicio de Sesión (Autenticación)

#### Endpoint 3: Iniciar Login (`POST /api/webauthn/login/begin`)
**Objetivo:** El servidor emite un desafío que solo la Llave Privada correcta podrá resolver.

1. **Entrada:** JSON con `username`.
2. **Lógica de Backend:**
   * Busca al usuario en `users`. Si no existe, aborta.
   * Busca todas las credenciales activas del usuario en `credentials`. Obtiene la lista de `credential_id`s.
   * Genera un NUEVO `challenge` criptográficamente seguro (32 bytes).
   * Guarda el `challenge` en `webauthn_sessions`.
3. **Salida (JSON enviado a Kotlin):**
   * `challenge`: El nuevo desafío.
   * `allowCredentials`: Una lista de los `credential_id` válidos para ese usuario. El celular buscará en su chip seguro si tiene la llave privada que corresponde a alguno de esos IDs.

#### Endpoint 4: Finalizar Login (`POST /api/webauthn/login/finish`)
**Objetivo:** Validar la Firma Digital usando la Llave Pública de la Base de Datos.

1. **Entrada:** JSON (`AssertionResponse`) que contiene el `credential_id` usado, el `authenticatorData`, el `clientDataJSON` (que contiene el desafío), y la **Firma Digital (`signature`)**.
2. **Lógica de Backend (Validación Matemática):**
   * Lee el `credential_id` entrante y busca la `public_key` correspondiente en la tabla `credentials`. Si no existe, aborta.
   * Recupera el `challenge` de `webauthn_sessions`.
   * **Verificación Criptográfica (El núcleo del sistema):** El servidor usa el algoritmo ECDSA con la `public_key` de la base de datos para verificar matemáticamente si la **Firma Digital** fue creada por la Llave Privada correspondiente usando el *authenticatorData* y el *hash del clientDataJSON*.
   * **Prevención de Clonación (`sign_count`):** Lee el contador de firmas enviado por el celular. Debe ser ESTRICTAMENTE MAYOR que el `sign_count` guardado en la DB. Si es menor o igual, ¡alguien clonó la llave! Aborta.
3. **Transacción en DB:**
   * Actualiza el `sign_count` en la tabla `credentials` con el nuevo valor recibido.
   * Borra la sesión en `webauthn_sessions`.
4. **Salida:** Respuesta `200 OK` devolviendo un **Token JWT** o una Cookie de sesión HTTP-Only para que el usuario navegue en la aplicación cliente.

---

## 3. Resumen de Reglas de Negocio en el Backend
* **Aislamiento:** El backend **nunca** recibe biometría, PINs ni contraseñas. Solo procesa matemáticas (firmas de curva elíptica).
* **Eficiencia (Go):** Las funciones `Verify()` en Go son sincrónicas y bloqueantes debido al cálculo matemático. El modelo de *Goroutines* permite que si 100 personas inician sesión a la vez, el servidor pueda calcular 100 firmas asíncronamente sin que el sistema colapse.
* **Tiempos de Vida:** Los desafíos en la tabla `webauthn_sessions` deben tener un trabajo programado (Cron/Worker) que los elimine si el usuario no completó el proceso biométrico en 2 minutos, para liberar espacio en PostgreSQL y evitar ataques de repetición prolongados.