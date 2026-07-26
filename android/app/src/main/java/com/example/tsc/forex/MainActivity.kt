package com.example.tsc.forex

import android.content.Context
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import android.graphics.Color
import androidx.compose.runtime.LaunchedEffect
import com.example.tsc.forex.ui.RootScaffold
import com.example.tsc.forex.ui.AppLocale
import com.example.tsc.forex.ui.theme.TscForexTheme

class MainActivity : ComponentActivity() {
    override fun attachBaseContext(newBase: Context) {
        super.attachBaseContext(AppLocale.wrap(newBase))
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(Color.TRANSPARENT),
            navigationBarStyle = SystemBarStyle.dark(Color.TRANSPARENT),
        )
        val app = application as TscApp

        setContent {
            TscForexTheme {
                // Markets are public; other tabs soft-gate to login (iOS parity).
                LaunchedEffect(Unit) {
                    app.prices.start()
                    if (app.auth.isLoggedIn) {
                        app.auth.loadProfile()
                        app.billing.refreshNow()
                    }
                }
                RootScaffold(
                    auth = app.auth,
                    prices = app.prices,
                    portfolio = app.portfolio,
                    billing = app.billing,
                    prefs = app.prefs,
                )
            }
        }
    }
}
