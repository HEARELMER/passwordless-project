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

    fun register(repository: AuthRepository, username: String, email: String) {
        if (username.isBlank() || email.isBlank()) {
            _registerState.value = AuthResult.Error("Todos los campos son obligatorios")
            return
        }
        _registerState.value = AuthResult.Loading
        viewModelScope.launch {
            _registerState.postValue(repository.register(username.trim(), email.trim()))
        }
    }

    fun reset() {
        _registerState.value = AuthResult.Loading
    }
}
