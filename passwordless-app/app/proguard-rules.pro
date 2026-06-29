# ProGuard rules para la app Passwordless
# Mantener clases de modelos de red (Gson necesita los nombres)
-keep class com.passwordless.authenticator.network.models.** { *; }

# Retrofit
-keepattributes Signature
-keepattributes Exceptions
-keep class retrofit2.** { *; }
-keepclasseswithmembers class * {
    @retrofit2.http.* <methods>;
}

# OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**

# Gson
-keep class com.google.gson.** { *; }
-keepattributes *Annotation*

# AndroidKeyStore y Biometría (no ofuscar)
-keep class android.security.keystore.** { *; }
-keep class androidx.biometric.** { *; }

# Coroutines
-keepclassmembernames class kotlinx.** { volatile <fields>; }
