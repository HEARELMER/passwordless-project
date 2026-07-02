package com.passwordless.authenticator.network

import com.passwordless.authenticator.network.models.*
import okhttp3.RequestBody
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

/**
 * Definición de los endpoints del Backend Go (Relying Party).
 * Todos los métodos son suspending para uso con Coroutines.
 */
interface ApiService {

    // ─── REGISTRO ──────────────────────────────────────────────────────────

    /** Fase 1 del registro: solicita un challenge aleatorio al servidor */
    @POST("api/webauthn/register/begin")
    suspend fun registerBegin(
        @Body request: RegisterBeginRequest
    ): Response<RegisterBeginResponse>

    /** Fase 2 del registro: envía la llave pública y la firma del challenge */
    @POST("api/webauthn/register/finish")
    suspend fun registerFinish(
        @retrofit2.http.Header("X-Session-ID") sessionId: String,
        @Body request: RequestBody
    ): Response<RegisterFinishResponse>

    // ─── LOGIN ─────────────────────────────────────────────────────────────

    /** Fase 1 del login: solicita un challenge nuevo al servidor */
    @POST("api/webauthn/login/begin")
    suspend fun loginBegin(
        @Body request: LoginBeginRequest
    ): Response<LoginBeginResponse>

    /** Fase 2 del login: envía la firma digital para verificación */
    @POST("api/webauthn/login/finish")
    suspend fun loginFinish(
        @retrofit2.http.Header("X-Session-ID") sessionId: String,
        @Body request: RequestBody
    ): Response<LoginFinishResponse>
}
