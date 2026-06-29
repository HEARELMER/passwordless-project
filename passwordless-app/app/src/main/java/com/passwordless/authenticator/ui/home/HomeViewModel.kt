package com.passwordless.authenticator.ui.home

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel

class HomeViewModel : ViewModel() {
    private val _sessionToken = MutableLiveData<String?>()
    val sessionToken: LiveData<String?> = _sessionToken

    private val _userId = MutableLiveData<String?>()
    val userId: LiveData<String?> = _userId

    fun loadSession(token: String?, userId: String?) {
        _sessionToken.value = token
        _userId.value = userId
    }
}
