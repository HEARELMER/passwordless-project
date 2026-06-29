package com.passwordless.authenticator.utils

/** Resultado genérico para operaciones que pueden fallar */
sealed class AuthResult<out T> {
    data class Success<T>(val data: T) : AuthResult<T>()
    data class Error(val message: String, val cause: Throwable? = null) : AuthResult<Nothing>()
    object Loading : AuthResult<Nothing>()
}
