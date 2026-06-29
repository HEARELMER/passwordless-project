package com.passwordless.authenticator.biometric

import androidx.biometric.BiometricManager
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import java.security.Signature

/** Resultado de la operación biométrica */
sealed class BiometricResult {
    /** Autenticación exitosa; cryptoObject contiene la Signature firmada */
    data class Success(val cryptoObject: BiometricPrompt.CryptoObject) : BiometricResult()
    /** Error del sistema biométrico */
    data class Error(val errorCode: Int, val message: String) : BiometricResult()
    /** El usuario canceló */
    object Cancelled : BiometricResult()
}

/**
 * Helper que encapsula BiometricPrompt.
 *
 * El flujo crítico de seguridad es:
 *   1. Se inicializa un objeto Signature con la llave privada del KeyStore.
 *   2. Se envuelve en BiometricPrompt.CryptoObject(signature).
 *   3. BiometricPrompt valida la biometría Y autoriza la firma en el mismo acto atómico.
 *   4. Solo si la biometría es válida, el CryptoObject "desbloqueado" se devuelve en onSuccess.
 */
class BiometricHelper(private val fragment: Fragment) {

    private var pendingCallback: ((BiometricResult) -> Unit)? = null

    private val authCallback = object : BiometricPrompt.AuthenticationCallback() {
        override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
            val crypto = result.cryptoObject
            if (crypto != null) {
                pendingCallback?.invoke(BiometricResult.Success(crypto))
            } else {
                pendingCallback?.invoke(BiometricResult.Error(-1, "CryptoObject no disponible"))
            }
            pendingCallback = null
        }

        override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
            val result = if (
                errorCode == BiometricPrompt.ERROR_USER_CANCELED ||
                errorCode == BiometricPrompt.ERROR_NEGATIVE_BUTTON
            ) BiometricResult.Cancelled
            else BiometricResult.Error(errorCode, errString.toString())

            pendingCallback?.invoke(result)
            pendingCallback = null
        }

        override fun onAuthenticationFailed() {
            // El sistema gestiona los reintentos automáticamente; no se propaga aquí
        }
    }

    private val biometricPrompt: BiometricPrompt by lazy {
        val executor = ContextCompat.getMainExecutor(fragment.requireContext())
        BiometricPrompt(fragment, executor, authCallback)
    }

    /**
     * Muestra el diálogo biométrico vinculado criptográficamente a [signature].
     *
     * @param signature  Objeto Signature ya inicializado (de KeyManager)
     * @param title      Título del diálogo
     * @param subtitle   Subtítulo
     * @param description Descripción de la operación
     * @param onResult   Callback con el resultado
     */
    fun authenticate(
        signature: Signature,
        title: String = "Verificación biométrica",
        subtitle: String = "Confirma tu identidad",
        description: String = "Usa tu huella o reconocimiento facial para continuar",
        onResult: (BiometricResult) -> Unit
    ) {
        pendingCallback = onResult
        val cryptoObject = BiometricPrompt.CryptoObject(signature)
        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setTitle(title)
            .setSubtitle(subtitle)
            .setDescription(description)
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .setNegativeButtonText("Cancelar")
            .build()

        biometricPrompt.authenticate(promptInfo, cryptoObject)
    }

    /** Verifica si el dispositivo tiene biometría fuerte configurada */
    fun isAvailable(): Boolean {
        val mgr = BiometricManager.from(fragment.requireContext())
        return mgr.canAuthenticate(BiometricManager.Authenticators.BIOMETRIC_STRONG) ==
                BiometricManager.BIOMETRIC_SUCCESS
    }
}
