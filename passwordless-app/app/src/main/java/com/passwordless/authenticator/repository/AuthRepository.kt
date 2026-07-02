package com.passwordless.authenticator.repository

import android.util.Log
import com.passwordless.authenticator.biometric.BiometricHelper
import com.passwordless.authenticator.biometric.BiometricResult
import com.passwordless.authenticator.crypto.CryptoEngine
import com.passwordless.authenticator.crypto.KeyManager
import com.passwordless.authenticator.network.ApiClient
import com.passwordless.authenticator.network.models.*
import com.passwordless.authenticator.utils.AuthResult
import com.passwordless.authenticator.utils.SessionManager
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import kotlin.coroutines.resume

private const val TAG = "AuthRepository"

/**
 * Orquestador central de los flujos criptográficos.
 *
 * ─── FLUJO DE REGISTRO ────────────────────────────────────────────────────
 * 1. POST /auth/register/begin → recibe challenge
 * 2. Genera par de llaves ECDSA en AndroidKeyStore
 * 3. Biometría autoriza la firma del challenge
 * 4. POST /auth/register/finish → envía llave pública + firma
 *
 * ─── FLUJO DE LOGIN ───────────────────────────────────────────────────────
 * 1. POST /auth/login/begin → recibe challenge
 * 2. Recupera Signature con llave privada del KeyStore
 * 3. Biometría autoriza la firma del challenge
 * 4. POST /auth/login/finish → envía firma → recibe JWT
 */
class AuthRepository(
    private val sessionManager: SessionManager,
    private val biometricHelper: BiometricHelper
) {
    private val api = ApiClient.apiService

    // ─── REGISTRO ──────────────────────────────────────────────────────────

    suspend fun register(userId: String, displayName: String): AuthResult<String> =
        withContext(Dispatchers.IO) {
            try {
                // 1. Solicitar challenge de registro
                val beginResp = api.registerBegin(RegisterBeginRequest(userId, displayName))
                if (!beginResp.isSuccessful) {
                    return@withContext AuthResult.Error("Error al iniciar registro: ${beginResp.code()}")
                }
                val beginData = beginResp.body()!!
                val challengeBytes = CryptoEngine.decodeBase64UrlSafe(beginData.challenge)

                // 2. Generar par de llaves en el AndroidKeyStore
                val publicKey = KeyManager.generateKeyPair(userId)
                val publicKeyEncoded = CryptoEngine.encodePublicKey(publicKey)

                // 3. Preparar Signature para BiometricPrompt
                val signature = KeyManager.getSignatureForSigning(userId)

                // 4. Autenticación biométrica (volver al hilo principal para UI)
                val biometricResult = withContext(Dispatchers.Main) {
                    suspendCancellableCoroutine { cont ->
                        biometricHelper.authenticate(
                            signature = signature,
                            title = "Registrar dispositivo",
                            subtitle = "Crea tu llave de acceso",
                            description = "Confirma tu identidad para asociar este dispositivo a tu cuenta",
                            onResult = { cont.resume(it) }
                        )
                    }
                }

                when (biometricResult) {
                    is BiometricResult.Cancelled -> return@withContext AuthResult.Error("Registro cancelado")
                    is BiometricResult.Error -> return@withContext AuthResult.Error(biometricResult.message)
                    is BiometricResult.Success -> {
                        // 5. Firmar el challenge con la Signature autorizada biométricamente
                        val authorizedSignature = biometricResult.cryptoObject.signature!!
                        val signatureBytes = CryptoEngine.signChallenge(authorizedSignature, challengeBytes)
                        val signatureEncoded = CryptoEngine.encodeBase64UrlSafe(signatureBytes)
                        val clientDataHash = CryptoEngine.sha256Base64(challengeBytes)

                        // 6. Enviar llave pública y firma al servidor
                        val finishResp = api.registerFinish(
                            RegisterFinishRequest(
                                userId = userId,
                                publicKey = publicKeyEncoded,
                                signature = signatureEncoded,
                                clientDataHash = clientDataHash
                            )
                        )

                        if (finishResp.isSuccessful && finishResp.body()?.success == true) {
                            sessionManager.userId = userId
                            sessionManager.displayName = displayName
                            sessionManager.isRegistered = true
                            AuthResult.Success("Registro exitoso")
                        } else {
                            // Limpiar llave si el servidor rechazó
                            KeyManager.deleteKey(userId)
                            AuthResult.Error("El servidor rechazó el registro: ${finishResp.body()?.message}")
                        }
                    }
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
                // Verificar que existe llave local para este usuario
                if (!KeyManager.hasKey(userId)) {
                    return@withContext AuthResult.Error("No hay credenciales registradas para este usuario en este dispositivo")
                }

                // 1. Solicitar challenge de login
                val beginResp = api.loginBegin(LoginBeginRequest(userId))
                if (!beginResp.isSuccessful) {
                    return@withContext AuthResult.Error("Error al iniciar sesión: ${beginResp.code()}")
                }
                val beginData = beginResp.body()!!
                val challengeBytes = CryptoEngine.decodeBase64UrlSafe(beginData.challenge)

                // 2. Preparar Signature con la llave privada existente
                val signature = KeyManager.getSignatureForSigning(userId)

                // 3. Autenticación biométrica
                val biometricResult = withContext(Dispatchers.Main) {
                    suspendCancellableCoroutine { cont ->
                        biometricHelper.authenticate(
                            signature = signature,
                            title = "Iniciar sesión",
                            subtitle = "Verifica tu identidad",
                            description = "Usa tu huella o rostro para acceder de forma segura",
                            onResult = { cont.resume(it) }
                        )
                    }
                }

                when (biometricResult) {
                    is BiometricResult.Cancelled -> return@withContext AuthResult.Error("Inicio de sesión cancelado")
                    is BiometricResult.Error -> return@withContext AuthResult.Error(biometricResult.message)
                    is BiometricResult.Success -> {
                        // 4. Firmar challenge
                        val authorizedSignature = biometricResult.cryptoObject.signature!!
                        val signatureBytes = CryptoEngine.signChallenge(authorizedSignature, challengeBytes)
                        val signatureEncoded = CryptoEngine.encodeBase64UrlSafe(signatureBytes)
                        val clientDataHash = CryptoEngine.sha256Base64(challengeBytes)

                        // 5. Enviar firma al servidor
                        val finishResp = api.loginFinish(
                            LoginFinishRequest(
                                userId = userId,
                                signature = signatureEncoded,
                                clientDataHash = clientDataHash
                            )
                        )

                        val body = finishResp.body()
                        if (finishResp.isSuccessful && body?.success == true) {
                            sessionManager.sessionToken = body.token
                            sessionManager.userId = userId
                            AuthResult.Success(body.token ?: "Sesión iniciada")
                        } else {
                            AuthResult.Error("Verificación fallida: ${body?.message ?: finishResp.code()}")
                        }
                    }
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
        KeyManager.deleteKey(userId)
        sessionManager.clearAll()
    }
}
