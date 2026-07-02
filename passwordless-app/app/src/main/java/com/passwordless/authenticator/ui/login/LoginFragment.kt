package com.passwordless.authenticator.ui.login

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import androidx.fragment.app.viewModels
import androidx.navigation.fragment.findNavController
import com.google.android.material.snackbar.Snackbar
import com.passwordless.authenticator.R
import com.passwordless.authenticator.databinding.FragmentLoginBinding
import com.passwordless.authenticator.repository.AuthRepository
import com.passwordless.authenticator.utils.AuthResult
import com.passwordless.authenticator.utils.SessionManager

class LoginFragment : Fragment() {

    private var _binding: FragmentLoginBinding? = null
    private val binding get() = _binding!!
    private val viewModel: LoginViewModel by viewModels()

    private lateinit var repository: AuthRepository

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentLoginBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        val sessionManager = SessionManager(requireContext())
        repository = AuthRepository(sessionManager, requireActivity())

        binding.btnLogin.setOnClickListener {
            val userId = binding.etUsername.text.toString()
            viewModel.login(repository, userId)
        }

        viewModel.loginState.observe(viewLifecycleOwner) { result ->
            when (result) {
                is AuthResult.Loading -> {
                    binding.progressBar.isVisible = true
                    binding.btnLogin.isEnabled = false
                }
                is AuthResult.Success -> {
                    binding.progressBar.isVisible = false
                    binding.btnLogin.isEnabled = true
                    Snackbar.make(binding.root, "Bienvenido", Snackbar.LENGTH_SHORT).show()
                    findNavController().navigate(R.id.action_login_to_home)
                }
                is AuthResult.Error -> {
                    binding.progressBar.isVisible = false
                    binding.btnLogin.isEnabled = true
                    Snackbar.make(binding.root, "${result.message}", Snackbar.LENGTH_LONG).show()
                }
            }
        }
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
