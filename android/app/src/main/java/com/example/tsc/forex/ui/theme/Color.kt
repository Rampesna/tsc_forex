package com.example.tsc.forex.ui.theme

import androidx.compose.ui.graphics.Color

/** Binance-inspired dark palette shared across the app. */
object Th {
    val bg = Color(0xFF0B0E11)
    val surface = Color(0xFF161A1E)
    val surfaceAlt = Color(0xFF1E242B)
    val stroke = Color(0xFF262D35)

    val text = Color(0xFFEAECEF)
    val textDim = Color(0xFF848E9C)
    val textFaint = Color(0xFF5E6673)

    val up = Color(0xFF0ECB81)
    val down = Color(0xFFF6465D)
    // Brand palette from the logo (deep teal #003D4D). The deep tone is illegible
    // on the near-black bg, so interactive elements use a readable brand teal and
    // brandDeep is reserved for tiles/gradients.
    val brandDeep = Color(0xFF003D4D)
    val accent = Color(0xFF17A2B8)   // readable brand teal
    val accentDim = Color(0xFF07333B)

    fun trend(v: Double) = when {
        v > 0 -> up
        v < 0 -> down
        else -> textDim
    }
}
