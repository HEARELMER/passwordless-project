package com.passwordless.authenticator.ui.login

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.passwordless.authenticator.repository.AuthRepository
import com.passwordless.authenticator.utils.AuthResult
import kotlinx.coroutines.launch

class LoginViewModel : ViewModel() {

    private val _loginState = MutableLiveData<AuthResult<String>>()
    val loginState: LiveData<AuthResult<String>> = _loginState

    fun login(repository: AuthRepository, userId: String) {
        if (userId.isBlank()) {
            _loginState.value = AuthResult.Error("Ingresa tu identificador")
            return
        }
        _loginState.value = AuthResult.Loading
        viewModelScope.launch {
            _loginState.postValue(repository.login(userId.trim()))
        }
    }

    fun reset() {
        _loginState.value = AuthResult.Loading
    }
}
