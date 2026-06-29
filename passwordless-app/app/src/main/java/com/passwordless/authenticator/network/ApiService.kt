package com.passwordless.authenticator.network

import com.passwordless.authenticator.network.models.*
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
    @POST("auth/register/begin")
    suspend fun registerBegin(
        @Body request: RegisterBeginRequest
    ): Response<RegisterBeginResponse>

    /** Fase 2 del registro: envía la llave pública y la firma del challenge */
    @POST("auth/register/finish")
    suspend fun registerFinish(
        @Body request: RegisterFinishRequest
    ): Response<RegisterFinishResponse>

    // ─── LOGIN ─────────────────────────────────────────────────────────────

    /** Fase 1 del login: solicita un challenge nuevo al servidor */
    @POST("auth/login/begin")
    suspend fun loginBegin(
        @Body request: LoginBeginRequest
    ): Response<LoginBeginResponse>

    /** Fase 2 del login: envía la firma digital para verificación */
    @POST("auth/login/finish")
    suspend fun loginFinish(
        @Body request: LoginFinishRequest
    ): Response<LoginFinishResponse>
}
