package com.example.tsc.forex

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import com.example.tsc.forex.ui.RootScaffold
import com.example.tsc.forex.ui.screens.LoginScreen
import com.example.tsc.forex.ui.theme.TscForexTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val app = application as TscApp

        setContent {
            TscForexTheme {
                val loggedIn = app.auth.isLoggedIn
                if (loggedIn) {
                    LaunchedEffect(Unit) { app.prices.start() }
                    RootScaffold(app.auth, app.prices, app.portfolio, app.prefs)
                } else {
                    LoginScreen(app.auth, onLoggedIn = { app.prices.start() })
                }
            }
        }
    }
}
