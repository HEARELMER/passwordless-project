package com.passwordless.authenticator.repository

import android.content.Context
import android.util.Log
import androidx.credentials.CreatePublicKeyCredentialRequest
import androidx.credentials.CreatePublicKeyCredentialResponse
import androidx.credentials.CredentialManager
import androidx.credentials.GetCredentialRequest
import androidx.credentials.GetPublicKeyCredentialOption
import androidx.credentials.PublicKeyCredential
import com.passwordless.authenticator.network.ApiClient
import com.passwordless.authenticator.network.models.LoginBeginRequest
import com.passwordless.authenticator.network.models.RegisterBeginRequest
import com.passwordless.authenticator.utils.AuthResult
import com.passwordless.authenticator.utils.SessionManager
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody

private const val TAG = "AuthRepository"

/**
 * Orquestador central de los flujos de Passkeys (WebAuthn).
 *
 * ─── FLUJO DE REGISTRO ────────────────────────────────────────────────────
 * 1. POST /auth/register/begin → recibe challenge (options JSON)
 * 2. CredentialManager.createCredential() maneja la UI y biometría nativa
 * 3. POST /auth/register/finish → envía registrationResponseJson al backend
 *
 * ─── FLUJO DE LOGIN ───────────────────────────────────────────────────────
 * 1. POST /auth/login/begin → recibe challenge (options JSON)
 * 2. CredentialManager.getCredential() maneja la UI y biometría nativa
 * 3. POST /auth/login/finish → envía authenticationResponseJson al backend
 */
class AuthRepository(
    private val sessionManager: SessionManager,
    private val context: Context
) {
    private val api = ApiClient.apiService
    private val credentialManager = CredentialManager.create(context)

    // ─── REGISTRO ──────────────────────────────────────────────────────────

    suspend fun register(username: String, email: String): AuthResult<String> =
        withContext(Dispatchers.IO) {
            try {
                // 1. Solicitar opciones de registro al servidor
                val beginResp = api.registerBegin(RegisterBeginRequest(username, email))
                if (!beginResp.isSuccessful) {
                    return@withContext AuthResult.Error("Error al iniciar registro: ${beginResp.code()}")
                }
                val beginData = beginResp.body()!!
                
                // 2. Ejecutar CredentialManager (Passkeys UI Nativa)
                val requestJson = beginData.publicKey.toString()
                val createRequest = CreatePublicKeyCredentialRequest(requestJson)
                
                val result = try {
                    credentialManager.createCredential(context, createRequest)
                } catch (e: Exception) {
                    return@withContext AuthResult.Error("Registro cancelado o fallido: ${e.localizedMessage}")
                }
                
                val response = result as? CreatePublicKeyCredentialResponse
                if (response == null) {
                    return@withContext AuthResult.Error("Respuesta de credencial inválida")
                }
                
                // 3. Enviar respuesta JSON cruda al backend
                val mediaType = "application/json".toMediaType()
                val requestBody = response.registrationResponseJson.toRequestBody(mediaType)
                
                val finishResp = api.registerFinish(beginData.sessionId, requestBody)

                if (finishResp.isSuccessful && finishResp.body()?.status == "ok") {
                    sessionManager.userId = username
                    sessionManager.displayName = email
                    sessionManager.isRegistered = true
                    AuthResult.Success("Registro exitoso")
                } else {
                    val errBody = finishResp.errorBody()?.string()
                    AuthResult.Error("El servidor rechazó el registro: ${errBody ?: finishResp.code()}")
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error en registro", e)
                AuthResult.Error("Error inesperado: ${e.localizedMessage}", e)
            }
        }

    // ─── LOGIN ─────────────────────────────────────────────────────────────

    suspend fun login(userId: String): AuthResult<String> =
        withContext(Dispatchers.IO) {
            try {
                // 1. Solicitar opciones de login al servidor
                val beginResp = api.loginBegin(LoginBeginRequest(userId))
                if (!beginResp.isSuccessful) {
                    return@withContext AuthResult.Error("Error al iniciar sesión: ${beginResp.code()}")
                }
                val beginData = beginResp.body()!!
                
                // 2. Ejecutar CredentialManager (Passkeys UI Nativa)
                val publicKeyObj = beginData.publicKey.asJsonObject
                // HACK Samsung Pass: Remover allowCredentials para forzar Discoverable Credentials
                publicKeyObj.remove("allowCredentials")
                
                val requestJson = publicKeyObj.toString()
                val getOption = GetPublicKeyCredentialOption(requestJson)
                val getRequest = GetCredentialRequest(listOf(getOption))
                
                val result = try {
                    credentialManager.getCredential(context, getRequest)
                } catch (e: Exception) {
                    return@withContext AuthResult.Error("Inicio de sesión cancelado o fallido: ${e.localizedMessage}")
                }
                
                val credential = result.credential as? PublicKeyCredential
                if (credential == null) {
                    return@withContext AuthResult.Error("Credencial seleccionada inválida")
                }
                
                // 3. Enviar aserción JSON cruda al backend
                val mediaType = "application/json".toMediaType()
                val requestBody = credential.authenticationResponseJson.toRequestBody(mediaType)
                
                val finishResp = api.loginFinish(beginData.sessionId, requestBody)

                val body = finishResp.body()
                if (finishResp.isSuccessful && body != null && body.token.isNotEmpty()) {
                    sessionManager.sessionToken = body.token
                    sessionManager.userId = userId
                    AuthResult.Success(body.token)
                } else {
                    val errBody = finishResp.errorBody()?.string()
                    AuthResult.Error("Verificación fallida: ${errBody ?: finishResp.code()}")
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error en login", e)
                AuthResult.Error("Error inesperado: ${e.localizedMessage}", e)
            }
        }

    fun logout() {
        sessionManager.clearSession()
    }

    fun revokeCredentials(userId: String) {
        // En Passkeys puros, la llave vive en Google Password Manager.
        // No la eliminamos localmente. Solo limpiamos la sesión.
        sessionManager.clearAll()
    }
}
