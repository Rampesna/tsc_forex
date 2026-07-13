package com.example.tsc.forex.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable

private val TscColorScheme = darkColorScheme(
    primary = Th.accent,
    onPrimary = Th.bg,
    secondary = Th.up,
    background = Th.bg,
    onBackground = Th.text,
    surface = Th.surface,
    onSurface = Th.text,
    surfaceVariant = Th.surfaceAlt,
    onSurfaceVariant = Th.textDim,
    outline = Th.stroke,
    error = Th.down,
)

/** The app is intentionally always-dark (a trading UI convention). */
@Composable
fun TscForexTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = TscColorScheme,
        typography = Typography,
        content = content,
    )
}
