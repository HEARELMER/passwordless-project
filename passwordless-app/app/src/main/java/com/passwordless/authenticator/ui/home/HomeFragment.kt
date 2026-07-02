package com.passwordless.authenticator.ui.home

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.fragment.app.viewModels
import androidx.navigation.fragment.findNavController
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.passwordless.authenticator.R
import com.passwordless.authenticator.databinding.FragmentHomeBinding
import com.passwordless.authenticator.utils.SessionManager

class HomeFragment : Fragment() {

    private var _binding: FragmentHomeBinding? = null
    private val binding get() = _binding!!
    private val viewModel: HomeViewModel by viewModels()

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentHomeBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        val session = SessionManager(requireContext())
        viewModel.loadSession(session.sessionToken, session.userId)

        viewModel.userId.observe(viewLifecycleOwner) { uid ->
            binding.tvUserEmail.text = "Hola, ${uid ?: "Usuario"}"
        }

        viewModel.sessionToken.observe(viewLifecycleOwner) { token ->
            binding.tvJwtToken.text = if (token != null) {
                "Token JWT:\n${token.take(60)}…"
            } else {
                "Sesión activa (sin token)"
            }
        }

        binding.btnLogout.setOnClickListener {
            session.clearSession()
            findNavController().navigate(R.id.action_home_to_welcome)
        }

        binding.btnRevoke.setOnClickListener {
            MaterialAlertDialogBuilder(requireContext())
                .setTitle("Revocar credenciales")
                .setMessage("Esto eliminará tu llave de acceso de este dispositivo. Tendrás que registrarte nuevamente.")
                .setPositiveButton("Revocar") { _, _ ->
                    val userId = session.userId
                    session.clearAll()
                    findNavController().navigate(R.id.action_home_to_welcome)
                }
                .setNegativeButton("Cancelar", null)
                .show()
        }
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
