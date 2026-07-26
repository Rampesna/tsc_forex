package com.example.tsc.forex.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.unit.dp

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

private val TscShapes = Shapes(
    extraSmall = RoundedCornerShape(8.dp),
    small = RoundedCornerShape(12.dp),
    medium = RoundedCornerShape(16.dp),
    large = RoundedCornerShape(22.dp),
    extraLarge = RoundedCornerShape(28.dp),
)

/** The app is intentionally always-dark (a trading UI convention). */
@Composable
fun TscForexTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = TscColorScheme,
        typography = Typography,
        shapes = TscShapes,
        content = content,
    )
}
