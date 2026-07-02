package com.passwordless.authenticator.crypto

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.PublicKey
import java.security.Signature

/**
 * Gestiona el ciclo de vida de las llaves criptográficas dentro del AndroidKeyStore.
 *
 * Seguridad:
 * - Algoritmo: ECDSA con curva P-256 (secp256r1), compatible con WebAuthn/FIDO2.
 * - Las llaves se almacenan en el TEE (Trusted Execution Environment) del dispositivo.
 * - La llave privada NUNCA abandona el chip seguro.
 * - Se requiere autenticación biométrica fuerte para usar la llave privada.
 *
 * Nota sobre StrongBox: Se usa TEE estándar (disponible en todos los dispositivos con
 * minSdk=28). StrongBox requiere hardware dedicado adicional (disponible solo en
 * dispositivos premium) y se puede habilitar en una fase posterior mediante un módulo
 * de extensión de seguridad.
 */
object KeyManager {

    private const val KEYSTORE_PROVIDER = "AndroidKeyStore"
    private const val KEY_ALGORITHM = KeyProperties.KEY_ALGORITHM_EC
    const val SIGNATURE_ALGORITHM = "SHA256withECDSA"

    /**
     * Genera un par de llaves ECDSA P-256 en el AndroidKeyStore (TEE).
     * Requiere autenticación biométrica fuerte para firmar con la llave privada.
     *
     * @param alias Identificador único de la llave (ej: userId del usuario)
     * @return La llave pública en formato X.509 (puede enviarse al servidor)
     */
    fun generateKeyPair(alias: String): PublicKey {
        val keyPairGenerator = KeyPairGenerator.getInstance(KEY_ALGORITHM, KEYSTORE_PROVIDER)

        val keySpec = KeyGenParameterSpec.Builder(
            alias,
            KeyProperties.PURPOSE_SIGN or KeyProperties.PURPOSE_VERIFY
        )
            .setDigests(KeyProperties.DIGEST_SHA256)
            // La llave solo se puede usar previa autenticación biométrica fuerte
            .setUserAuthenticationRequired(true)
            // 0 segundos = se requiere biometría en CADA uso de la llave (más seguro)
            .setUserAuthenticationParameters(0, KeyProperties.AUTH_BIOMETRIC_STRONG)
            // Invalida la llave si se registra una nueva huella en el dispositivo
            .setInvalidatedByBiometricEnrollment(true)
            .build()

        keyPairGenerator.initialize(keySpec)
        return keyPairGenerator.generateKeyPair().public
    }

    /**
     * Obtiene un objeto [Signature] inicializado con la llave privada del alias.
     * Este objeto DEBE pasarse a BiometricPrompt como CryptoObject.
     * Solo después de la autenticación biométrica exitosa el objeto Signature
     * podrá ser usado para firmar datos.
     *
     * @param alias Identificador de la llave (debe existir en el KeyStore)
     * @return Signature inicializado listo para BiometricPrompt
     * @throws IllegalStateException si no existe llave para el alias dado
     */
    fun getSignatureForSigning(alias: String): Signature {
        val keyStore = KeyStore.getInstance(KEYSTORE_PROVIDER).apply { load(null) }
        val privateKey = keyStore.getKey(alias, null)
            ?: throw IllegalStateException("No se encontró llave privada para: $alias")

        return Signature.getInstance(SIGNATURE_ALGORITHM).apply {
            initSign(privateKey as java.security.PrivateKey)
        }
    }

    /** Verifica si existe una llave con el alias dado en el KeyStore */
    fun hasKey(alias: String): Boolean {
        val keyStore = KeyStore.getInstance(KEYSTORE_PROVIDER).apply { load(null) }
        return keyStore.containsAlias(alias)
    }

    /** Obtiene la llave pública almacenada para el alias dado */
    fun getPublicKey(alias: String): PublicKey? {
        val keyStore = KeyStore.getInstance(KEYSTORE_PROVIDER).apply { load(null) }
        return keyStore.getCertificate(alias)?.publicKey
    }

    /** Elimina la llave del KeyStore (revocación de credenciales del dispositivo) */
    fun deleteKey(alias: String) {
        val keyStore = KeyStore.getInstance(KEYSTORE_PROVIDER).apply { load(null) }
        if (keyStore.containsAlias(alias)) keyStore.deleteEntry(alias)
    }
}
