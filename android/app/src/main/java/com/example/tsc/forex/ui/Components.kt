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
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import com.example.tsc.forex.ui.LocalizedText as Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.animation.animateColorAsState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.ui.theme.Th

@Composable
fun Card(
    modifier: Modifier = Modifier,
    padding: PaddingValues = PaddingValues(14.dp),
    content: @Composable () -> Unit,
) {
    Surface(
        modifier = modifier,
        color = Th.surface,
        contentColor = Th.text,
        shape = RoundedCornerShape(18.dp),
        border = androidx.compose.foundation.BorderStroke(1.dp, Th.stroke),
        tonalElevation = 0.dp,
        shadowElevation = 0.dp,
    ) {
        Box(Modifier.padding(padding)) { content() }
    }
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
fun ScreenHeader(
    title: String,
    subtitle: String? = null,
    leading: (@Composable () -> Unit)? = null,
    trailing: (@Composable () -> Unit)? = null,
) {
    Box(
        Modifier
            .fillMaxWidth()
            .statusBarsPadding()
            .heightIn(min = 76.dp)
            .padding(horizontal = 12.dp, vertical = 10.dp),
    ) {
        Column(
            Modifier
                .align(Alignment.Center)
                .padding(horizontal = 52.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Text(
                title,
                color = Th.text,
                style = MaterialTheme.typography.headlineSmall,
                textAlign = TextAlign.Center,
                maxLines = 1,
            )
            if (subtitle != null) {
                Text(
                    subtitle,
                    color = Th.textDim,
                    style = MaterialTheme.typography.bodySmall,
                    textAlign = TextAlign.Center,
                    maxLines = 2,
                )
            }
        }
        if (leading != null) {
            Box(
                Modifier
                    .align(Alignment.CenterStart)
                    .sizeIn(minWidth = 48.dp, minHeight = 48.dp),
                contentAlignment = Alignment.Center,
            ) {
                leading()
            }
        }
        if (trailing != null) {
            Box(
                Modifier
                    .align(Alignment.CenterEnd)
                    .sizeIn(minWidth = 48.dp, minHeight = 48.dp),
                contentAlignment = Alignment.Center,
            ) {
                trailing()
            }
        }
    }
}

@Composable
fun StatTile(label: String, value: String, valueColor: Color = Th.text, modifier: Modifier = Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(label, color = Th.textDim, style = MaterialTheme.typography.labelMedium, maxLines = 1)
        Text(
            value,
            color = valueColor,
            style = MaterialTheme.typography.titleSmall,
            maxLines = 1,
        )
    }
}

@Composable
fun PrimaryActionButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    leading: (@Composable () -> Unit)? = null,
) {
    Button(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier
            .fillMaxWidth()
            .heightIn(min = 52.dp),
        shape = RoundedCornerShape(14.dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = Th.accent,
            contentColor = Th.bg,
            disabledContainerColor = Th.surfaceAlt,
            disabledContentColor = Th.textFaint,
        ),
        contentPadding = PaddingValues(horizontal = 18.dp, vertical = 13.dp),
    ) {
        if (leading != null) {
            leading()
            androidx.compose.foundation.layout.Spacer(Modifier.size(8.dp))
        }
        Text(text, style = MaterialTheme.typography.labelLarge, textAlign = TextAlign.Center)
    }
}

@Composable
fun SecondaryActionButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    OutlinedButton(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier
            .fillMaxWidth()
            .heightIn(min = 50.dp),
        shape = RoundedCornerShape(14.dp),
        border = androidx.compose.foundation.BorderStroke(1.dp, Th.stroke),
        colors = ButtonDefaults.outlinedButtonColors(
            contentColor = Th.text,
            disabledContentColor = Th.textFaint,
        ),
    ) {
        Text(text, style = MaterialTheme.typography.labelLarge, textAlign = TextAlign.Center)
    }
}

@Composable
fun SegmentedControl(
    options: List<String>,
    selectedIndex: Int,
    onSelected: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        color = Th.bg,
        shape = RoundedCornerShape(14.dp),
        border = androidx.compose.foundation.BorderStroke(1.dp, Th.stroke.copy(alpha = 0.8f)),
    ) {
        Row(
            Modifier.padding(4.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            options.forEachIndexed { index, option ->
                val selected = index == selectedIndex
                val background by animateColorAsState(
                    if (selected) Th.surfaceAlt else Color.Transparent,
                    label = "segmentBackground",
                )
                Surface(
                    onClick = { onSelected(index) },
                    modifier = Modifier
                        .weight(1f)
                        .heightIn(min = 42.dp),
                    color = background,
                    contentColor = if (selected) Th.text else Th.textDim,
                    shape = RoundedCornerShape(10.dp),
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Text(
                            option,
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = if (selected) FontWeight.Bold else FontWeight.SemiBold,
                            textAlign = TextAlign.Center,
                            maxLines = 1,
                        )
                    }
                }
            }
        }
    }
}

@Suppress("unused")
private fun mid(a: Offset, b: Offset) = Offset((a.x + b.x) / 2, (a.y + b.y) / 2)
