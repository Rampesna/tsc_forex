package com.example.tsc.forex.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.ui.theme.Th

@Composable
fun Card(
    modifier: Modifier = Modifier,
    padding: PaddingValues = PaddingValues(14.dp),
    content: @Composable () -> Unit,
) {
    Box(
        modifier
            .clip(RoundedCornerShape(16.dp))
            .background(Th.surface)
            .border(1.dp, Th.stroke, RoundedCornerShape(16.dp))
            .padding(padding)
    ) { content() }
}

/** Small colored pill showing a percentage change. */
@Composable
fun ChangePill(pct: Double, modifier: Modifier = Modifier) {
    val c = Th.trend(pct)
    Box(
        modifier
            .clip(RoundedCornerShape(6.dp))
            .background(c.copy(alpha = 0.14f))
            .padding(horizontal = 6.dp, vertical = 2.dp)
    ) {
        Text(Fmt.pct(pct), color = c, fontSize = 11.sp, fontWeight = FontWeight.SemiBold, maxLines = 1)
    }
}

@Composable
fun ConnectionDot(connected: Boolean) {
    Box(
        Modifier
            .size(8.dp)
            .clip(CircleShape)
            .background(if (connected) Th.up else Th.down)
    )
}

/** Round monogram badge for an instrument (first letters of its code). */
@Composable
fun AssetBadge(code: String, isGold: Boolean, modifier: Modifier = Modifier) {
    val label = if (isGold) "Au" else code.take(2)
    val bg = if (isGold) Th.accent.copy(alpha = 0.18f) else Th.surfaceAlt
    val fg = if (isGold) Th.accent else Th.text
    Box(
        modifier
            .size(36.dp)
            .clip(CircleShape)
            .background(bg),
        contentAlignment = Alignment.Center,
    ) {
        Text(label, color = fg, fontSize = 13.sp, fontWeight = FontWeight.Bold)
    }
}

/** Compact sparkline drawn as a thin line (no gradient fill). */
@Composable
fun Sparkline(points: List<Double>, up: Boolean, modifier: Modifier = Modifier) {
    if (points.size < 2) {
        Box(modifier)
        return
    }
    val color = if (up) Th.up else Th.down
    Canvas(modifier) {
        val minV = points.min()
        val maxV = points.max()
        val range = (maxV - minV).takeIf { it > 0 } ?: 1.0
        val dx = size.width / (points.size - 1)
        val path = Path()
        points.forEachIndexed { i, v ->
            val x = dx * i
            val y = size.height - ((v - minV) / range).toFloat() * size.height
            if (i == 0) path.moveTo(x, y) else path.lineTo(x, y)
        }
        drawPath(path, color = color, style = Stroke(width = 2f, cap = StrokeCap.Round))
    }
}

@Composable
fun ScreenHeader(title: String, subtitle: String? = null, trailing: (@Composable () -> Unit)? = null) {
    Row(
        Modifier
            .fillMaxWidth()
            .statusBarsPadding()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(title, color = Th.text, fontSize = 22.sp, fontWeight = FontWeight.Bold)
            if (subtitle != null) Text(subtitle, color = Th.textDim, fontSize = 12.sp)
        }
        if (trailing != null) trailing()
    }
}

@Composable
fun StatTile(label: String, value: String, valueColor: Color = Th.text, modifier: Modifier = Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(label, color = Th.textDim, fontSize = 11.sp)
        Text(value, color = valueColor, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, maxLines = 1)
    }
}

@Suppress("unused")
private fun mid(a: Offset, b: Offset) = Offset((a.x + b.x) / 2, (a.y + b.y) / 2)
