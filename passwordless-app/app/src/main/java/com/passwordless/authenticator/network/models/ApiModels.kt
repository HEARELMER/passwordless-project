package com.passwordless.authenticator.network.models

import com.google.gson.annotations.SerializedName
import com.google.gson.JsonObject

// ─── REGISTRO ──────────────────────────────────────────────────────────────

data class RegisterBeginRequest(
    @SerializedName("username")
    val userId: String,
    @SerializedName("email")
    val displayName: String
)

data class RegisterBeginResponse(
    @SerializedName("session_id") val sessionId: String,
    val publicKey: JsonObject
)

data class RegisterFinishResponse(
    val success: Boolean,
    val message: String,
    val credentialId: String?
)

// ─── LOGIN ─────────────────────────────────────────────────────────────────

data class LoginBeginRequest(
    @SerializedName("username")
    val userId: String
)

data class LoginBeginResponse(
    @SerializedName("session_id") val sessionId: String,
    val publicKey: JsonObject
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
