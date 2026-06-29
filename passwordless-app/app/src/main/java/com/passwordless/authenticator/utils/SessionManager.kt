package com.passwordless.authenticator.utils

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

/**
 * Gestiona la sesión del usuario con almacenamiento cifrado.
 * Usa EncryptedSharedPreferences respaldado por AndroidKeyStore.
 */
class SessionManager(context: Context) {

    companion object {
        private const val PREFS_FILE = "secure_session"
        private const val KEY_TOKEN = "session_token"
        private const val KEY_USER_ID = "user_id"
        private const val KEY_DISPLAY_NAME = "display_name"
        private const val KEY_REGISTERED = "is_registered"
    }

    private val masterKey = MasterKey.Builder(context)
        .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
        .build()

    private val prefs: SharedPreferences = EncryptedSharedPreferences.create(
        context,
        PREFS_FILE,
        masterKey,
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )

    var sessionToken: String?
        get() = prefs.getString(KEY_TOKEN, null)
        set(value) = prefs.edit().putString(KEY_TOKEN, value).apply()

    var userId: String?
        get() = prefs.getString(KEY_USER_ID, null)
        set(value) = prefs.edit().putString(KEY_USER_ID, value).apply()

    var displayName: String?
        get() = prefs.getString(KEY_DISPLAY_NAME, null)
        set(value) = prefs.edit().putString(KEY_DISPLAY_NAME, value).apply()

    var isRegistered: Boolean
        get() = prefs.getBoolean(KEY_REGISTERED, false)
        set(value) = prefs.edit().putBoolean(KEY_REGISTERED, value).apply()

    val isLoggedIn: Boolean
        get() = sessionToken != null

    /** Cierra la sesión borrando el token (las llaves del KeyStore permanecen) */
    fun clearSession() {
        prefs.edit()
            .remove(KEY_TOKEN)
            .apply()
    }

    /** Borra todos los datos del usuario (para "olvidar dispositivo") */
    fun clearAll() {
        prefs.edit().clear().apply()
    }
}
