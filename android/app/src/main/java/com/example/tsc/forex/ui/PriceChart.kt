package com.example.tsc.forex.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.ui.theme.Th

/** Line + subtle area chart for a series of closing prices. */
@Composable
fun PriceChart(values: List<Double>, modifier: Modifier = Modifier) {
    if (values.size < 2) {
        Box(
            modifier
                .fillMaxWidth()
                .height(220.dp),
            contentAlignment = Alignment.Center,
        ) { Text("Grafik için yeterli veri yok", color = Th.textDim, fontSize = 13.sp) }
        return
    }
    val up = values.last() >= values.first()
    val line = if (up) Th.up else Th.down
    Canvas(
        modifier
            .fillMaxWidth()
            .height(220.dp)
    ) {
        val minV = values.min()
        val maxV = values.max()
        val range = (maxV - minV).takeIf { it > 0 } ?: 1.0
        val dx = size.width / (values.size - 1)

        fun y(v: Double) = size.height - ((v - minV) / range).toFloat() * (size.height * 0.9f) - size.height * 0.05f

        val line1 = Path()
        val area = Path()
        values.forEachIndexed { i, v ->
            val x = dx * i
            val yy = y(v)
            if (i == 0) {
                line1.moveTo(x, yy)
                area.moveTo(x, size.height)
                area.lineTo(x, yy)
            } else {
                line1.lineTo(x, yy)
                area.lineTo(x, yy)
            }
        }
        area.lineTo(size.width, size.height)
        area.close()

        drawPath(
            area,
            brush = Brush.verticalGradient(
                listOf(line.copy(alpha = 0.22f), line.copy(alpha = 0.0f)),
                startY = 0f, endY = size.height,
            ),
        )
        drawPath(line1, color = line, style = Stroke(width = 3f, cap = StrokeCap.Round))
    }
}

@Suppress("unused")
private fun p(x: Float, y: Float) = Offset(x, y)
