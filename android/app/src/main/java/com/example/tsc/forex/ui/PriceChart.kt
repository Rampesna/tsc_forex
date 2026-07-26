package com.example.tsc.forex.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectDragGesturesAfterLongPress
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import com.example.tsc.forex.ui.LocalizedText as Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.Candle
import com.example.tsc.forex.ui.theme.Th
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min

enum class ChartMode { LINE, CANDLE }

/** Extra polyline drawn on the same time/price axes as the primary series. */
data class ChartOverlaySeries(
    val title: String,
    val color: Color,
    /** Epoch ms → value pairs (same time base as candles). */
    val points: List<Pair<Long, Double>>,
)

private val MA7_COLOR = Color(0xFFF3C43D)   // gold
private val MA25_COLOR = Color(0xFFE559B7)  // pink

/**
 * Interactive price chart drawn on a Compose Canvas:
 *   • one-finger drag = pan, two-finger pinch = zoom
 *   • tap / long-press + drag = crosshair scrub
 *   • optional MA overlays, volume pane, and extra line series
 *
 * Viewport is time-anchored (epoch ms). Edge paging via [onReachStart]/[onReachEnd].
 */
@Composable
fun PriceChart(
    candles: List<Candle>,
    mode: ChartMode,
    trendUp: Boolean,
    currency: String?,
    resetToken: Int,
    canLoadOlder: Boolean = false,
    canLoadNewer: Boolean = false,
    onReachStart: () -> Unit = {},
    onReachEnd: () -> Unit = {},
    modifier: Modifier = Modifier,
    heightDp: Int = 250,
    seriesTitle: String = "Fiyat",
    seriesColor: Color? = null,
    overlaySeries: List<ChartOverlaySeries> = emptyList(),
    showsMovingAverages: Boolean = true,
    showsVolume: Boolean = true,
    showsOHLCSelection: Boolean = true,
    stickySelection: Boolean = false,
    emptyTitle: String = "Grafik verisi birikiyor…",
    emptySubtitle: String = "Kısa süre içinde mum çubukları oluşacak.",
    interactionHint: String = "Kaydır • İki parmakla yakınlaştır • Basılı tut: detay",
    onSelectionChange: ((Long?) -> Unit)? = null,
) {
    if (candles.size < 2) {
        Box(
            modifier.fillMaxWidth().height(heightDp.dp),
            contentAlignment = Alignment.Center,
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(emptyTitle, color = Th.textDim, fontSize = 13.sp)
                Text(emptySubtitle, color = Th.textFaint, fontSize = 11.sp)
            }
        }
        return
    }

    val measurer = rememberTextMeasurer()
    val hostView = LocalView.current
    val density = LocalDensity.current
    val axisGutterPx = with(density) {
        (if (fontScale > 1.15f) 68.dp else 58.dp).toPx()
    }
    val lineColor = seriesColor ?: if (trendUp) Th.up else Th.down
    val multiSeries = overlaySeries.isNotEmpty()

    val firstMs = candles.first().epochMs.toDouble()
    val lastMs = candles.last().epochMs.toDouble()
    val step = remember(candles.size, candles.firstOrNull()?.epochMs, candles.lastOrNull()?.epochMs) {
        candleStep(candles)
    }
    val fullSpan = max(lastMs - firstMs, step)

    // Time-anchored viewport.
    var viewStartMs by remember { mutableStateOf(0.0) }
    var viewSpanMs by remember { mutableStateOf(0.0) }
    var selectedIdx by remember { mutableStateOf<Int?>(null) }

    val minSpan = step * 6
    val maxSpan = fullSpan + step

    fun recenter() {
        viewSpanMs = min(step * 40, maxSpan).coerceAtLeast(minSpan)
        viewStartMs = lastMs - viewSpanMs + step
        selectedIdx = null
    }

    // Re-center only on timeframe switch (resetToken), never on window paging.
    LaunchedEffect(resetToken) { recenter() }
    if (viewSpanMs <= 0.0) recenter()

    LaunchedEffect(selectedIdx, candles) {
        onSelectionChange?.invoke(selectedIdx?.let { candles.getOrNull(it)?.epochMs })
    }

    // Clamp the viewport into the loaded range.
    fun clamp() {
        viewSpanMs = viewSpanMs.coerceIn(minSpan, maxSpan)
        val minStart = firstMs - viewSpanMs * 0.15
        val maxStart = lastMs + step - viewSpanMs * 0.85
        viewStartMs = viewStartMs.coerceIn(minStart, maxStart)
    }

    fun maybePage() {
        if (canLoadOlder && viewStartMs <= firstMs + step * 3) onReachStart()
        if (canLoadNewer && viewStartMs + viewSpanMs >= lastMs - step * 3) onReachEnd()
    }

    val ma7 = remember(candles.size, candles.firstOrNull()?.epochMs, candles.lastOrNull()?.close, showsMovingAverages) {
        if (showsMovingAverages) movingAverage(candles, 7) else emptyList()
    }
    val ma25 = remember(candles.size, candles.firstOrNull()?.epochMs, candles.lastOrNull()?.close, showsMovingAverages) {
        if (showsMovingAverages) movingAverage(candles, 25) else emptyList()
    }

    Column(
        modifier
            .fillMaxWidth()
            .pointerInput(Unit) {
                // Keep parent vertical scroll from stealing chart pan/pinch.
                awaitPointerEventScope {
                    while (true) {
                        val event = awaitPointerEvent()
                        if (event.changes.any { it.pressed }) {
                            (hostView.parent as? android.view.ViewParent)
                                ?.requestDisallowInterceptTouchEvent(true)
                        }
                    }
                }
            },
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        // Header: legend or crosshair readout.
        val sel = selectedIdx?.let { candles.getOrNull(it) }
        Row(Modifier.fillMaxWidth().heightIn(min = 38.dp), verticalAlignment = Alignment.CenterVertically) {
            if (sel != null) {
                if (showsOHLCSelection) {
                    val chgPct = if (sel.open != 0.0) (sel.close - sel.open) / sel.open * 100 else 0.0
                    Column {
                        Text(fmtDateTime(sel.epochMs, fullSpan), color = Th.textDim, fontSize = 10.sp)
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            ohlc("A", sel.open); ohlc("Y", sel.high); ohlc("D", sel.low); ohlc("K", sel.close)
                        }
                    }
                    Spacer(Modifier.weight(1f))
                    Text(Fmt.signedPct(chgPct), color = Th.trend(chgPct), fontSize = 12.sp, fontWeight = FontWeight.Bold)
                } else {
                    Text(fmtDateTime(sel.epochMs, fullSpan), color = Th.textDim, fontSize = 11.sp)
                    Spacer(Modifier.weight(1f))
                    Text(Fmt.money(sel.close, currency), color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.Bold)
                }
            } else if (multiSeries) {
                LegendDot(lineColor, seriesTitle)
                overlaySeries.forEach { series ->
                    Spacer(Modifier.size(8.dp))
                    LegendDot(series.color, series.title)
                }
                Spacer(Modifier.weight(1f))
            } else {
                LegendDot(lineColor, seriesTitle)
                if (showsMovingAverages) {
                    Spacer(Modifier.size(10.dp))
                    LegendDot(MA7_COLOR, "MA7"); Spacer(Modifier.size(10.dp))
                    LegendDot(MA25_COLOR, "MA25")
                }
                Spacer(Modifier.weight(1f))
                Text(Fmt.money(candles.last().close, currency), color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.Bold)
            }
        }

        // Main price canvas.
        Canvas(
            Modifier
                .fillMaxWidth()
                .height(heightDp.dp)
                .pointerInput(resetToken, candles.size) {
                    detectTransformGestures { _, pan, zoom, _ ->
                        val w = (size.width.toFloat() - axisGutterPx).coerceAtLeast(1f)
                        viewStartMs -= pan.x / w * viewSpanMs
                        if (zoom != 1f) {
                            val center = viewStartMs + viewSpanMs / 2
                            viewSpanMs = (viewSpanMs / zoom)
                            viewStartMs = center - viewSpanMs / 2
                        }
                        clamp()
                        selectedIdx = null
                        maybePage()
                    }
                }
                .pointerInput(resetToken, candles.size, stickySelection) {
                    detectTapGestures { pos ->
                        val hit = nearestIndex(
                            candles, pos.x, size.width.toFloat(), axisGutterPx, viewStartMs, viewSpanMs,
                        )
                        selectedIdx = if (stickySelection && selectedIdx == hit) null else hit
                    }
                }
                .pointerInput(resetToken, candles.size, stickySelection) {
                    detectDragGesturesAfterLongPress(
                        onDragStart = { pos ->
                            selectedIdx = nearestIndex(
                                candles, pos.x, size.width.toFloat(), axisGutterPx, viewStartMs, viewSpanMs,
                            )
                        },
                        onDrag = { change, _ ->
                            selectedIdx = nearestIndex(
                                candles,
                                change.position.x,
                                size.width.toFloat(),
                                axisGutterPx,
                                viewStartMs,
                                viewSpanMs,
                            )
                        },
                        onDragEnd = { if (!stickySelection) selectedIdx = null },
                        onDragCancel = { if (!stickySelection) selectedIdx = null },
                    )
                },
        ) {
            drawPricePane(
                candles = candles, mode = mode, trendUp = trendUp,
                viewStartMs = viewStartMs, viewSpanMs = viewSpanMs, step = step,
                lineColor = lineColor, ma7 = ma7, ma25 = ma25,
                overlaySeries = overlaySeries,
                showsMovingAverages = showsMovingAverages && !multiSeries,
                selected = selectedIdx?.let { candles.getOrNull(it) },
                currency = currency, measurer = measurer, fullSpan = fullSpan,
                axisGutterPx = axisGutterPx,
            )
        }

        if (showsVolume) {
            Canvas(Modifier.fillMaxWidth().height(44.dp)) {
                drawVolumePane(candles, viewStartMs, viewSpanMs, step, axisGutterPx)
            }
        }

        Canvas(Modifier.fillMaxWidth().height(30.dp)) {
            drawTimeAxis(
                viewStartMs = viewStartMs,
                viewSpanMs = viewSpanMs,
                axisGutterPx = axisGutterPx,
                measurer = measurer,
                fullSpan = fullSpan,
            )
        }

        Text(
            interactionHint,
            color = Th.textFaint, fontSize = 9.sp,
            modifier = Modifier.fillMaxWidth().padding(top = 2.dp),
        )
    }
}

@Composable
private fun LegendDot(color: Color, label: String) {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
        Box(Modifier.size(width = 10.dp, height = 3.dp).clip(RoundedCornerShape(1.dp)).background(color))
        Text(label, color = Th.textDim, fontSize = 10.sp, fontWeight = FontWeight.Medium)
    }
}

@Composable
private fun ohlc(k: String, v: Double) {
    Row(horizontalArrangement = Arrangement.spacedBy(2.dp)) {
        Text(k, color = Th.textFaint, fontSize = 9.sp)
        Text(Fmt.compactPrice(v), color = Th.text, fontSize = 10.sp, fontWeight = FontWeight.SemiBold)
    }
}

// MARK: - Canvas drawing

private fun DrawScope.drawPricePane(
    candles: List<Candle>,
    mode: ChartMode,
    trendUp: Boolean,
    viewStartMs: Double,
    viewSpanMs: Double,
    step: Double,
    lineColor: Color,
    ma7: List<Pair<Long, Double>>,
    ma25: List<Pair<Long, Double>>,
    overlaySeries: List<ChartOverlaySeries>,
    showsMovingAverages: Boolean,
    selected: Candle?,
    currency: String?,
    measurer: androidx.compose.ui.text.TextMeasurer,
    fullSpan: Double,
    axisGutterPx: Float,
) {
    val w = size.width
    val h = size.height
    val plotRight = (w - axisGutterPx).coerceAtLeast(w * 0.55f)
    if (viewSpanMs <= 0) return

    fun xOf(ms: Double): Float = ((ms - viewStartMs) / viewSpanMs * plotRight).toFloat()

    // Visible candles (+1 padding each side).
    val visible = candles.filter {
        val ms = it.epochMs.toDouble()
        ms >= viewStartMs - step && ms <= viewStartMs + viewSpanMs + step
    }
    if (visible.isEmpty()) return

    var lo = Double.MAX_VALUE; var hi = -Double.MAX_VALUE
    for (c in visible) { lo = min(lo, c.low); hi = max(hi, c.high) }
    for (series in overlaySeries) {
        for ((ms, v) in series.points) {
            if (ms.toDouble() < viewStartMs - step || ms.toDouble() > viewStartMs + viewSpanMs + step) continue
            lo = min(lo, v); hi = max(hi, v)
        }
    }
    if (hi <= lo) { hi = lo + 1 }
    val pad = (hi - lo) * 0.10
    lo -= pad; hi += pad
    val range = (hi - lo).coerceAtLeast(1e-9)
    fun yOf(v: Double): Float = (h - ((v - lo) / range) * h).toFloat()

    val gridColor = Th.stroke.copy(alpha = 0.5f)
    val labelStyle = TextStyle(color = Th.textDim, fontSize = 9.sp)

    // Horizontal grid + right price labels.
    val gridLines = 4
    for (i in 0..gridLines) {
        val v = lo + range * i / gridLines
        val y = yOf(v)
        drawLine(gridColor, Offset(0f, y), Offset(plotRight, y), strokeWidth = 1f)
        val tl = measurer.measure(Fmt.compactPrice(v), labelStyle)
        val labelY = (y - tl.size.height / 2)
            .coerceIn(0f, (h - tl.size.height).coerceAtLeast(0f))
        drawText(tl, topLeft = Offset(plotRight + 8f, labelY))
    }

    // Vertical grid + bottom time labels (a few evenly spaced).
    val timeTicks = 4
    for (i in 0..timeTicks) {
        val ms = viewStartMs + viewSpanMs * i / timeTicks
        val x = xOf(ms)
        if (x < 0 || x > plotRight) continue
        drawLine(gridColor.copy(alpha = 0.35f), Offset(x, 0f), Offset(x, h), strokeWidth = 1f)
    }

    val barW = (plotRight / (viewSpanMs / step).toFloat()).coerceIn(2f, 18f) * 0.62f

    when (mode) {
        ChartMode.LINE -> {
            val line = Path(); val area = Path()
            visible.forEachIndexed { i, c ->
                val x = xOf(c.epochMs.toDouble()); val y = yOf(c.close)
                if (i == 0) { line.moveTo(x, y); area.moveTo(x, h); area.lineTo(x, y) }
                else { line.lineTo(x, y); area.lineTo(x, y) }
            }
            area.lineTo(xOf(visible.last().epochMs.toDouble()), h); area.close()
            val areaAlpha = if (overlaySeries.isEmpty()) 0.22f else 0.10f
            drawPath(area, Brush.verticalGradient(
                listOf(lineColor.copy(alpha = areaAlpha), lineColor.copy(alpha = 0f)), startY = 0f, endY = h))
            drawPath(line, color = lineColor, style = Stroke(width = 2.5f, cap = StrokeCap.Round))
        }
        ChartMode.CANDLE -> {
            for (c in visible) {
                val x = xOf(c.epochMs.toDouble())
                val up = c.close >= c.open
                val col = if (up) Th.up else Th.down
                // Wick.
                drawLine(col, Offset(x, yOf(c.high)), Offset(x, yOf(c.low)), strokeWidth = 1.4f)
                // Body.
                val top = yOf(max(c.open, c.close))
                val bot = yOf(min(c.open, c.close))
                drawRect(col, topLeft = Offset(x - barW / 2, top),
                    size = androidx.compose.ui.geometry.Size(barW, (bot - top).coerceAtLeast(1.2f)))
            }
        }
    }

    for (series in overlaySeries) {
        drawMa(series.points, series.color, viewStartMs, viewSpanMs, plotRight, lo, range, h)
    }

    if (showsMovingAverages) {
        drawMa(ma7, MA7_COLOR, viewStartMs, viewSpanMs, plotRight, lo, range, h)
        drawMa(ma25, MA25_COLOR, viewStartMs, viewSpanMs, plotRight, lo, range, h)
    }

    // Last price dashed line + tag (skip when multi-series — too noisy).
    if (overlaySeries.isEmpty()) {
        val last = candles.last()
        val lastY = yOf(last.close)
        if (lastY in 0f..h) {
            drawLine(lineColor.copy(alpha = 0.4f), Offset(0f, lastY), Offset(plotRight, lastY),
                strokeWidth = 1f, pathEffect = PathEffect.dashPathEffect(floatArrayOf(4f, 4f)))
            drawTag(measurer, Fmt.compactPrice(last.close), plotRight + 2f, lastY, lineColor, Color.White)
        }
    }

    // Crosshair.
    if (selected != null) {
        val cx = xOf(selected.epochMs.toDouble())
        val cy = yOf(selected.close)
        val dash = PathEffect.dashPathEffect(floatArrayOf(4f, 4f))
        if (cx in 0f..plotRight) {
            drawLine(Th.textDim, Offset(cx, 0f), Offset(cx, h), strokeWidth = 1f, pathEffect = dash)
        }
        drawLine(Th.textDim, Offset(0f, cy), Offset(plotRight, cy), strokeWidth = 1f, pathEffect = dash)
        drawTag(measurer, Fmt.compactPrice(selected.close), plotRight + 2f, cy, Th.surfaceAlt, Th.text)
    }
}

private fun DrawScope.drawMa(
    ma: List<Pair<Long, Double>>,
    color: Color,
    viewStartMs: Double,
    viewSpanMs: Double,
    plotRight: Float,
    lo: Double,
    range: Double,
    h: Float,
) {
    if (ma.size < 2) return
    val path = Path()
    var started = false
    for ((ms, v) in ma) {
        val x = ((ms.toDouble() - viewStartMs) / viewSpanMs * plotRight).toFloat()
        val y = (h - ((v - lo) / range) * h).toFloat()
        if (!started) { path.moveTo(x, y); started = true } else path.lineTo(x, y)
    }
    drawPath(path, color = color.copy(alpha = 0.9f), style = Stroke(width = 1.4f))
}

private fun DrawScope.drawTag(
    measurer: androidx.compose.ui.text.TextMeasurer,
    text: String,
    x: Float,
    y: Float,
    bg: Color,
    fg: Color,
) {
    val style = TextStyle(color = fg, fontSize = 9.sp, fontWeight = FontWeight.Bold)
    val tl = measurer.measure(text, style)
    val padX = 4f; val padY = 2f
    val tagWidth = tl.size.width + padX * 2
    val tagHeight = tl.size.height + padY * 2
    val left = x.coerceIn(0f, (size.width - tagWidth).coerceAtLeast(0f))
    val top = (y - tagHeight / 2).coerceIn(0f, (size.height - tagHeight).coerceAtLeast(0f))
    val rect = Rect(left, top, left + tagWidth, top + tagHeight)
    drawRoundRect(bg, topLeft = Offset(rect.left, rect.top),
        size = androidx.compose.ui.geometry.Size(rect.width, rect.height),
        cornerRadius = androidx.compose.ui.geometry.CornerRadius(4f, 4f))
    drawText(tl, topLeft = Offset(left + padX, top + padY))
}

private fun DrawScope.drawVolumePane(
    candles: List<Candle>,
    viewStartMs: Double,
    viewSpanMs: Double,
    step: Double,
    axisGutterPx: Float,
) {
    if (viewSpanMs <= 0) return
    val w = size.width; val h = size.height
    val plotRight = (w - axisGutterPx).coerceAtLeast(w * 0.55f)
    drawRect(Th.surface.copy(alpha = 0.4f), size = androidx.compose.ui.geometry.Size(plotRight, h))
    val visible = candles.filter {
        val ms = it.epochMs.toDouble()
        ms >= viewStartMs - step && ms <= viewStartMs + viewSpanMs + step
    }
    val maxCount = visible.maxOfOrNull { it.count }?.coerceAtLeast(1) ?: return
    val barW = (plotRight / (viewSpanMs / step).toFloat()).coerceIn(2f, 18f) * 0.62f
    for (c in visible) {
        if (c.count <= 0) continue
        val x = ((c.epochMs.toDouble() - viewStartMs) / viewSpanMs * plotRight).toFloat()
        val barH = (c.count.toFloat() / maxCount) * h
        val col = (if (c.close >= c.open) Th.up else Th.down).copy(alpha = 0.5f)
        drawRect(col, topLeft = Offset(x - barW / 2, h - barH),
            size = androidx.compose.ui.geometry.Size(barW, barH))
    }
}

private fun DrawScope.drawTimeAxis(
    viewStartMs: Double,
    viewSpanMs: Double,
    axisGutterPx: Float,
    measurer: androidx.compose.ui.text.TextMeasurer,
    fullSpan: Double,
) {
    if (viewSpanMs <= 0) return
    val plotRight = (size.width - axisGutterPx).coerceAtLeast(size.width * 0.55f)
    val style = TextStyle(color = Th.textFaint, fontSize = 9.sp)
    val ticks = 4
    for (i in 0..ticks) {
        val ms = (viewStartMs + viewSpanMs * i / ticks).toLong()
        val textLayout = measurer.measure(fmtDateTime(ms, fullSpan), style)
        val x = plotRight * i / ticks
        val left = (x - textLayout.size.width / 2)
            .coerceIn(0f, (plotRight - textLayout.size.width).coerceAtLeast(0f))
        drawText(
            textLayout,
            topLeft = Offset(left, (size.height - textLayout.size.height) / 2),
        )
    }
}

// MARK: - Helpers

private fun nearestIndex(
    candles: List<Candle>,
    xPx: Float,
    width: Float,
    axisGutterPx: Float,
    viewStartMs: Double,
    viewSpanMs: Double,
): Int? {
    if (candles.isEmpty() || width <= 0) return null
    val plotRight = (width - axisGutterPx).coerceAtLeast(width * 0.55f)
    val ms = viewStartMs + (xPx / plotRight).coerceIn(0f, 1f) * viewSpanMs
    var best = -1; var bestD = Double.MAX_VALUE
    for (i in candles.indices) {
        val d = abs(candles[i].epochMs.toDouble() - ms)
        if (d < bestD) { bestD = d; best = i }
    }
    return best.takeIf { it >= 0 }
}

private fun candleStep(candles: List<Candle>): Double {
    if (candles.size < 2) return 60_000.0
    var minGap = Double.MAX_VALUE
    for (i in 1 until candles.size) {
        val g = (candles[i].epochMs - candles[i - 1].epochMs).toDouble()
        if (g > 0) minGap = min(minGap, g)
    }
    return if (minGap == Double.MAX_VALUE) 60_000.0 else minGap
}

private fun movingAverage(candles: List<Candle>, period: Int): List<Pair<Long, Double>> {
    if (candles.size < period) return emptyList()
    val out = ArrayList<Pair<Long, Double>>(candles.size - period + 1)
    var window = 0.0
    for (i in candles.indices) {
        window += candles[i].close
        if (i >= period) window -= candles[i - period].close
        if (i >= period - 1) out.add(candles[i].epochMs to window / period)
    }
    return out
}

private val dayFmt = DateTimeFormatter.ofPattern("dd MMM").withZone(ZoneId.systemDefault())
private val monthYearFmt = DateTimeFormatter.ofPattern("MMM yy").withZone(ZoneId.systemDefault())
private val fullFmt = DateTimeFormatter.ofPattern("dd MMM HH:mm").withZone(ZoneId.systemDefault())

private fun fmtDateTime(ms: Long, fullSpan: Double): String {
    if (ms <= 0) return ""
    val inst = Instant.ofEpochMilli(ms)
    return when {
        fullSpan <= 3.0 * 86_400_000 -> fullFmt.format(inst)
        fullSpan <= 365.0 * 86_400_000 -> dayFmt.format(inst)
        else -> monthYearFmt.format(inst)
    }
}
