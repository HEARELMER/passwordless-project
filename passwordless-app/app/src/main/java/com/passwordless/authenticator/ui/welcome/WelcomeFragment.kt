package com.passwordless.authenticator.ui.welcome

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.navigation.fragment.findNavController
import com.passwordless.authenticator.R
import com.passwordless.authenticator.databinding.FragmentWelcomeBinding
import com.passwordless.authenticator.utils.SessionManager

class WelcomeFragment : Fragment() {

    private var _binding: FragmentWelcomeBinding? = null
    private val binding get() = _binding!!

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentWelcomeBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        val session = SessionManager(requireContext())

        // Si ya hay sesión activa, ir directamente al Home
        if (session.isLoggedIn) {
            findNavController().navigate(R.id.action_welcome_to_home)
            return
        }

        binding.btnRegister.setOnClickListener {
            findNavController().navigate(R.id.action_welcome_to_register)
        }

        binding.btnLogin.setOnClickListener {
            findNavController().navigate(R.id.action_welcome_to_login)
        }
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
