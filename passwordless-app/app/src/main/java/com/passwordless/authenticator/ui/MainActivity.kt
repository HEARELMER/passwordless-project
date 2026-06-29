package com.passwordless.authenticator.ui

import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import com.passwordless.authenticator.databinding.ActivityMainBinding

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        // El tema usa NoActionBar — la navegación entre fragments se gestiona
        // directamente por NavController desde cada Fragment (findNavController())
    }
}
