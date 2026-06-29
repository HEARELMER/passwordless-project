package com.passwordless.authenticator.network

import com.passwordless.authenticator.BuildConfig
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

/**
 * Singleton que provee la instancia configurada de Retrofit + OkHttp.
 *
 * Configuración de seguridad:
 * - Logging solo en DEBUG (nunca en producción)
 * - Timeouts estrictos
 * - Base URL configurable por buildType (ver build.gradle.kts)
 *
 * NOTA: Para producción, añadir Certificate Pinning con OkHttp CertificatePinner.
 * Ejemplo:
 *   val pinner = CertificatePinner.Builder()
 *       .add("api.tu-dominio.com", "sha256/AAAA...==")
 *       .build()
 *   OkHttpClient.Builder().certificatePinner(pinner)
 */
object ApiClient {

    private val loggingInterceptor = HttpLoggingInterceptor().apply {
        level = if (BuildConfig.DEBUG) {
            HttpLoggingInterceptor.Level.BODY
        } else {
            HttpLoggingInterceptor.Level.NONE
        }
    }

    private val okHttpClient = OkHttpClient.Builder()
        .addInterceptor(loggingInterceptor)
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        // TODO (producción): añadir .certificatePinner(pinner) con el hash del cert del servidor
        .build()

    private val retrofit = Retrofit.Builder()
        .baseUrl(BuildConfig.BASE_URL)
        .client(okHttpClient)
        .addConverterFactory(GsonConverterFactory.create())
        .build()

    val apiService: ApiService = retrofit.create(ApiService::class.java)
}
