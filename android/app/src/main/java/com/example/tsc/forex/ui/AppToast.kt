package com.example.tsc.forex.ui

import android.content.Context
import android.widget.Toast
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.platform.LocalContext

/** Shows a short Android system toast. Prefer this over inline error text. */
fun Context.appToast(message: String, long: Boolean = true) {
    val text = message.trim()
    if (text.isEmpty()) return
    Toast.makeText(this, text, if (long) Toast.LENGTH_LONG else Toast.LENGTH_SHORT).show()
}

/**
 * Observes [message]; when non-null/blank, shows a toast and calls [onConsumed]
 * so the source can clear state (avoids sticky on-screen error banners).
 */
@Composable
fun ToastOnChange(message: String?, onConsumed: () -> Unit) {
    val context = LocalContext.current
    LaunchedEffect(message) {
        val msg = message?.trim()?.takeIf { it.isNotEmpty() } ?: return@LaunchedEffect
        context.appToast(msg)
        onConsumed()
    }
}
