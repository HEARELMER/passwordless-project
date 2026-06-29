package com.passwordless.authenticator.ui.register

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.passwordless.authenticator.repository.AuthRepository
import com.passwordless.authenticator.utils.AuthResult
import kotlinx.coroutines.launch

class RegisterViewModel : ViewModel() {

    private val _registerState = MutableLiveData<AuthResult<String>>()
    val registerState: LiveData<AuthResult<String>> = _registerState

    fun register(repository: AuthRepository, userId: String, displayName: String) {
        if (userId.isBlank()) {
            _registerState.value = AuthResult.Error("El identificador no puede estar vacío")
            return
        }
        _registerState.value = AuthResult.Loading
        viewModelScope.launch {
            _registerState.postValue(repository.register(userId.trim(), displayName.trim().ifBlank { userId.trim() }))
        }
    }

    fun reset() {
        _registerState.value = AuthResult.Loading
    }
}
