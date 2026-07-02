package com.passwordless.authenticator.network.models

import com.google.gson.annotations.SerializedName
import com.google.gson.JsonObject

// ─── REGISTRO ──────────────────────────────────────────────────────────────

data class RegisterBeginRequest(
    val username: String,
    val email: String
)

data class RegisterBeginResponse(
    @SerializedName("session_id") val sessionId: String,
    val publicKey: JsonObject
)

data class RegisterFinishResponse(
    val status: String
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
    val token: String,
    @SerializedName("user_id") val userId: String,
    @SerializedName("expires_in") val expiresIn: Int
)

// ─── ERROR GENÉRICO ────────────────────────────────────────────────────────

data class ApiError(
    val error: String,
    val details: String?
)
