package com.passwordless.authenticator.crypto

import android.util.Base64
import java.security.MessageDigest
import java.security.Signature

/**
 * Motor criptográfico: firma desafíos y codifica datos en Base64 URL-safe
 * compatible con el estándar WebAuthn/FIDO2.
 */
object CryptoEngine {

    /**
     * Firma los bytes del desafío usando el objeto Signature ya autorizado
     * por BiometricPrompt (mediante CryptoObject).
     *
     * @param signature Objeto Signature autorizado biométricamente
     * @param challengeBytes Bytes del desafío recibido del servidor
     * @return Firma DER (Distinguished Encoding Rules) en bytes
     */
    fun signChallenge(signature: Signature, challengeBytes: ByteArray): ByteArray {
        signature.update(challengeBytes)
        return signature.sign()
    }

    /**
     * Calcula SHA-256 de los bytes y retorna en Base64 URL-safe.
     * Se usa para crear el clientDataHash enviado al servidor.
     */
    fun sha256Base64(data: ByteArray): String {
        val digest = MessageDigest.getInstance("SHA-256").digest(data)
        return encodeBase64UrlSafe(digest)
    }

    /** Codifica bytes a Base64 URL-safe sin padding (estándar WebAuthn) */
    fun encodeBase64UrlSafe(data: ByteArray): String =
        Base64.encodeToString(data, Base64.URL_SAFE or Base64.NO_WRAP or Base64.NO_PADDING)

    /** Decodifica Base64 URL-safe a bytes */
    fun decodeBase64UrlSafe(data: String): ByteArray =
        Base64.decode(data, Base64.URL_SAFE or Base64.NO_WRAP or Base64.NO_PADDING)

    /**
     * Codifica la llave pública en formato X.509 (SubjectPublicKeyInfo)
     * en Base64 URL-safe para enviar al servidor.
     */
    fun encodePublicKey(publicKey: java.security.PublicKey): String =
        encodeBase64UrlSafe(publicKey.encoded)
}
