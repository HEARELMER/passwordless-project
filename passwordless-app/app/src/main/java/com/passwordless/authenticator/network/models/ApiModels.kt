package com.passwordless.authenticator.network.models

// ─── REGISTRO ──────────────────────────────────────────────────────────────

data class RegisterBeginRequest(
    val userId: String,
    val displayName: String
)

data class RegisterBeginResponse(
    val challenge: String,      // Base64 URL-safe
    val userId: String,
    val rpId: String            // Relying Party ID (dominio del servidor)
)

data class RegisterFinishRequest(
    val userId: String,
    val publicKey: String,      // Llave pública X.509 en Base64 URL-safe
    val signature: String,      // Firma DER del challenge en Base64 URL-safe
    val clientDataHash: String  // SHA-256 del challenge en Base64 URL-safe
)

data class RegisterFinishResponse(
    val success: Boolean,
    val message: String,
    val credentialId: String?
)

// ─── LOGIN ─────────────────────────────────────────────────────────────────

data class LoginBeginRequest(
    val userId: String
)

data class LoginBeginResponse(
    val challenge: String,      // Base64 URL-safe
    val userId: String,
    val credentialId: String
)

data class LoginFinishRequest(
    val userId: String,
    val signature: String,      // Firma DER del challenge en Base64 URL-safe
    val clientDataHash: String  // SHA-256 del challenge en Base64 URL-safe
)

data class LoginFinishResponse(
    val success: Boolean,
    val token: String?,         // JWT de sesión (si autenticación exitosa)
    val message: String
)

// ─── ERROR GENÉRICO ────────────────────────────────────────────────────────

data class ApiError(
    val error: String,
    val details: String?
)
