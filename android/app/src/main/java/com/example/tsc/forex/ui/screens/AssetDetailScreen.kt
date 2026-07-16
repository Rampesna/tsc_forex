package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.outlined.StarBorder
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.Candle
import com.example.tsc.forex.data.InstrumentDetail
import com.example.tsc.forex.data.InstrumentStats
import com.example.tsc.forex.data.MarketInsight
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ChangePill
import com.example.tsc.forex.ui.ChartMode
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.PriceChart
import com.example.tsc.forex.ui.StatTile
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

/** A chart time range: label + aggregation interval + lookback window (days). */
private data class Tf(val key: String, val label: String, val interval: String, val days: Long)

// Binance-style ranges. Long ranges use daily/weekly/monthly buckets to match
// backfilled history; short ranges use intraday buckets from the live feed.
private val TIMEFRAMES = listOf(
    Tf("1G", "1G", "5m", 1),
    Tf("1H", "1H", "1h", 7),
    Tf("1A", "1A", "1d", 30),
    Tf("1Y", "1Y", "1d", 365),
    Tf("5Y", "5Y", "1w", 365 * 5),
    Tf("MAX", "MAX", "1mo", 365 * 40),
)

/** A single label/value metric shown in the detail grid. */
private data class Metric(val label: String, val value: String)

// Sliding-window sizing: page this many candles and keep at most `MAX_WINDOW`
// rendered so the chart stays smooth no matter how far the user scrolls back.
private const val PAGE_SIZE = 300
private const val MAX_WINDOW = 900

@Composable
private fun ResearchTitle(title: String) {
    Text(title, color = Th.textFaint, fontSize = 9.sp, fontWeight = FontWeight.Bold)
}

@Composable
private fun ResearchBullets(title: String, items: List<String>) {
    if (items.isEmpty()) return
    Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
        ResearchTitle(title)
        items.forEach { item -> Text("• $item", color = Th.textDim, fontSize = 11.sp) }
    }
}

private fun LikelihoodLabel(value: String) = when (value) {
    "high" -> "YÜKSEK OLASILIK"
    "low" -> "DÜŞÜK OLASILIK"
    else -> "ORTA OLASILIK"
}

@Composable
fun AssetDetailScreen(code: String, auth: AuthStore, store: PriceStore, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val uriHandler = LocalUriHandler.current
    val ip = store.prices[code]
    var rangeKey by remember { mutableStateOf("1G") }
    var candles by remember { mutableStateOf<List<Candle>>(emptyList()) }
    var mode by remember { mutableStateOf(ChartMode.LINE) }
    var stats by remember { mutableStateOf<InstrumentStats?>(null) }
    var detail by remember { mutableStateOf<InstrumentDetail?>(null) }
    var showAlert by remember { mutableStateOf(false) }
    var toast by remember { mutableStateOf<String?>(null) }
    var isFav by remember { mutableStateOf(store.isFavorite(code)) }
    var aiInsight by remember { mutableStateOf<MarketInsight?>(null) }
    var aiLoading by remember { mutableStateOf(false) }
    var aiError by remember { mutableStateOf<String?>(null) }

    // Window paging bookkeeping.
    var loadingOlder by remember { mutableStateOf(false) }
    var loadingNewer by remember { mutableStateOf(false) }
    var hasMoreHistory by remember { mutableStateOf(true) }
    var hasMoreFuture by remember { mutableStateOf(false) }
    var chartReset by remember { mutableStateOf(0) }

    fun stepMs(interval: String): Long = when (interval) {
        "5m" -> 300_000L; "1h" -> 3_600_000L; "1d" -> 86_400_000L
        "1w" -> 604_800_000L; "1mo" -> 2_629_800_000L; else -> 300_000L
    }

    LaunchedEffect(code, rangeKey) {
        val api = auth.newApiClient()
        val tf = TIMEFRAMES.firstOrNull { it.key == rangeKey } ?: TIMEFRAMES[0]
        val fromMs = System.currentTimeMillis() - tf.days * 86_400_000L
        hasMoreHistory = true
        hasMoreFuture = false
        val loaded = runCatching { api.candles(code, tf.interval, fromMs = fromMs) }.getOrElse { emptyList() }
        candles = loaded.takeLast(MAX_WINDOW)
        chartReset++
        stats = runCatching { api.stats(code) }.getOrNull()
    }
    LaunchedEffect(code) {
        detail = runCatching { auth.newApiClient().detail(code) }.getOrNull()
    }

    // Live-fold the WS price into the last candle when the present is on screen.
    val displayCandles: List<Candle> = run {
        val p = ip?.price ?: 0.0
        val last = candles.lastOrNull()
        if (hasMoreFuture || p <= 0.0 || last == null) candles
        else candles.toMutableList().also {
            it[it.lastIndex] = last.copy(high = maxOf(last.high, p), low = minOf(last.low, p), close = p)
        }
    }

    suspend fun loadOlder() {
        if (loadingOlder || loadingNewer || !hasMoreHistory) return
        val oldest = candles.firstOrNull()?.epochMs ?: return
        loadingOlder = true
        try {
            val tf = TIMEFRAMES.firstOrNull { it.key == rangeKey } ?: TIMEFRAMES[0]
            val older = runCatching { auth.newApiClient().candles(code, tf.interval, toMs = oldest - 1, limit = PAGE_SIZE) }
                .getOrElse { emptyList() }
            if (older.isEmpty()) { hasMoreHistory = false; return }
            val merged = (older + candles).distinctBy { it.epochMs }.sortedBy { it.epochMs }
            candles = if (merged.size > MAX_WINDOW) { hasMoreFuture = true; merged.take(MAX_WINDOW) } else merged
            if (older.size < PAGE_SIZE) hasMoreHistory = false
        } finally { loadingOlder = false }
    }

    suspend fun loadNewer() {
        if (loadingNewer || loadingOlder || !hasMoreFuture) return
        val newest = candles.lastOrNull()?.epochMs ?: return
        loadingNewer = true
        try {
            val tf = TIMEFRAMES.firstOrNull { it.key == rangeKey } ?: TIMEFRAMES[0]
            val newer = runCatching { auth.newApiClient().candles(code, tf.interval, fromMs = newest + 1, limit = PAGE_SIZE) }
                .getOrElse { emptyList() }
            if (newer.isEmpty()) { hasMoreFuture = false; return }
            val merged = (candles + newer).distinctBy { it.epochMs }.sortedBy { it.epochMs }
            candles = if (merged.size > MAX_WINDOW) { hasMoreHistory = true; merged.takeLast(MAX_WINDOW) } else merged
            val step = stepMs(tf.interval)
            if (newer.size < PAGE_SIZE || (candles.lastOrNull()?.epochMs ?: 0L) >= System.currentTimeMillis() - step * 2) {
                hasMoreFuture = false
            }
        } finally { loadingNewer = false }
    }

    Column(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
            .verticalScroll(rememberScrollState()),
    ) {
        Row(
            Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, "geri", tint = Th.text)
            }
            Text(
                ip?.displayName ?: code,
                color = Th.text, fontSize = 18.sp, fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = {
                store.toggleFavorite(code); isFav = store.isFavorite(code)
            }) {
                Icon(
                    if (isFav) Icons.Filled.Star else Icons.Outlined.StarBorder,
                    "favori", tint = if (isFav) Th.accent else Th.textFaint,
                )
            }
            IconButton(onClick = { showAlert = true }) {
                Icon(Icons.Filled.Notifications, "alarm", tint = Th.textFaint)
            }
        }

        Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            val ccy = ip?.currency
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Row(verticalAlignment = Alignment.Bottom) {
                    Text(
                        Fmt.money(ip?.price ?: 0.0, ccy),
                        color = Th.text, fontSize = 34.sp, fontWeight = FontWeight.Bold,
                    )
                    Box(Modifier.padding(start = 10.dp, bottom = 6.dp)) { ChangePill(ip?.changePct ?: 0.0) }
                }
                if (ip?.isUSD == true) {
                    Text("≈ ${Fmt.tl(ip.tryValue)}", color = Th.textDim, fontSize = 13.sp)
                }
            }

            // Chart-mode toggle + timeframe selector.
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    listOf(ChartMode.LINE to "Çizgi", ChartMode.CANDLE to "Mum").forEach { (m, label) ->
                        val sel = m == mode
                        Box(
                            Modifier
                                .clip(RoundedCornerShape(8.dp))
                                .background(if (sel) Th.accent.copy(alpha = 0.18f) else Th.surface)
                                .clickable { mode = m }
                                .padding(horizontal = 12.dp, vertical = 6.dp),
                        ) { Text(label, color = if (sel) Th.accent else Th.textDim, fontSize = 12.sp) }
                    }
                }
                Spacer(Modifier.weight(1f))
                if (loadingOlder || loadingNewer) {
                    Text(if (loadingOlder) "Geçmiş yükleniyor…" else "Güncel yükleniyor…",
                        color = Th.textDim, fontSize = 10.sp)
                }
            }

            Card(padding = androidx.compose.foundation.layout.PaddingValues(8.dp)) {
                PriceChart(
                    candles = displayCandles,
                    mode = mode,
                    trendUp = (ip?.changePct ?: 0.0) >= 0,
                    currency = ccy,
                    resetToken = chartReset,
                    canLoadOlder = hasMoreHistory,
                    canLoadNewer = hasMoreFuture,
                    onReachStart = { scope.launch { loadOlder() } },
                    onReachEnd = { scope.launch { loadNewer() } },
                )
            }

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                TIMEFRAMES.forEach { tf ->
                    val selected = tf.key == rangeKey
                    Box(
                        Modifier
                            .clip(RoundedCornerShape(8.dp))
                            .background(if (selected) Th.accent.copy(alpha = 0.18f) else Th.surface)
                            .clickable { rangeKey = tf.key }
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    ) {
                        Text(tf.label, color = if (selected) Th.accent else Th.textDim, fontSize = 12.sp)
                    }
                }
            }

            stats?.let { s ->
                Card {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        StatTile("24s Açılış", Fmt.money(s.open, ccy))
                        StatTile("24s Yüksek", Fmt.money(s.high, ccy), valueColor = Th.up)
                        StatTile("24s Düşük", Fmt.money(s.low, ccy), valueColor = Th.down)
                    }
                }
            }

            ip?.let {
                Card {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        StatTile("Alış", Fmt.money(it.bid, ccy), valueColor = Th.up)
                        StatTile("Satış", Fmt.money(it.ask, ccy), valueColor = Th.down)
                        StatTile("Sağlayıcı", it.provider.ifEmpty { "-" }, valueColor = Th.textDim)
                    }
                }
            }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("✦", color = Th.accent, fontSize = 18.sp, fontWeight = FontWeight.Bold)
                        Text(
                            "  TSC DERİN ARAŞTIRMA",
                            color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold,
                        )
                        Text(
                            "  ${aiInsight?.model?.uppercase()?.ifEmpty { "GPT-5.6 SOL" } ?: "GPT-5.6 SOL"}", color = Th.textFaint, fontSize = 9.sp,
                            modifier = Modifier.weight(1f),
                        )
                        aiInsight?.let { insight ->
                            Text(
                                insight.stanceLabel,
                                color = when (insight.stance) {
                                    "positive" -> Th.up
                                    "cautious" -> Th.down
                                    else -> Th.accent
                                },
                                fontSize = 10.sp,
                                fontWeight = FontWeight.Bold,
                            )
                        }
                    }

                    aiInsight?.let { insight ->
                        Text(insight.title, color = Th.text, fontSize = 17.sp, fontWeight = FontWeight.Bold)
                        Text(insight.summary, color = Th.textDim, fontSize = 13.sp)
                        insight.factors.forEach { factor ->
                            Text("• $factor", color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.Medium)
                        }

                        if (insight.timeHorizons.isNotEmpty()) {
                            ResearchTitle("ZAMAN DİLİMLERİ")
                            insight.timeHorizons.forEach { horizon ->
                                Column(
                                    Modifier.fillMaxWidth().clip(RoundedCornerShape(9.dp)).background(Th.surfaceAlt).padding(10.dp),
                                    verticalArrangement = Arrangement.spacedBy(3.dp),
                                ) {
                                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                        Text(horizon.horizon, color = Th.accent, fontSize = 10.sp, fontWeight = FontWeight.Bold)
                                        Text(horizon.outlook, color = Th.text, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                                    }
                                    Text(horizon.rationale, color = Th.textDim, fontSize = 11.sp)
                                }
                            }
                        }
                        ResearchBullets("MAKRO SÜRÜCÜLER", insight.macroDrivers)
                        ResearchBullets("ÇAPRAZ VARLIK İLİŞKİLERİ", insight.crossAssetAnalysis)
                        ResearchBullets("JEOPOLİTİK RİSKLER", insight.geopoliticalRisks)

                        if (insight.scenarios.isNotEmpty()) {
                            ResearchTitle("SENARYOLAR")
                            insight.scenarios.forEach { scenario ->
                                Column(
                                    Modifier.fillMaxWidth().clip(RoundedCornerShape(9.dp)).background(Th.surfaceAlt).padding(10.dp),
                                    verticalArrangement = Arrangement.spacedBy(3.dp),
                                ) {
                                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                        Text(scenario.name, color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                                        Text(LikelihoodLabel(scenario.likelihood), color = Th.accent, fontSize = 9.sp, fontWeight = FontWeight.Bold)
                                    }
                                    Text("Tetik: ${scenario.trigger}", color = Th.textDim, fontSize = 10.sp, fontWeight = FontWeight.Medium)
                                    Text(scenario.implication, color = Th.textDim, fontSize = 11.sp)
                                }
                            }
                        }
                        Text("⚠ ${insight.riskNote}", color = Th.textDim, fontSize = 11.sp)
                        if (insight.sources.isNotEmpty()) {
                            ResearchTitle("ARAŞTIRMA KAYNAKLARI")
                            insight.sources.take(8).forEach { source ->
                                Text(
                                    "↗ ${source.title}", color = Th.accent, fontSize = 10.sp, fontWeight = FontWeight.Medium,
                                    modifier = Modifier.clickable(enabled = source.url.startsWith("http")) {
                                        runCatching { uriHandler.openUri(source.url) }
                                    },
                                )
                            }
                        }
                        Text(insight.disclaimer, color = Th.textFaint, fontSize = 9.sp)
                    } ?: Text(
                        "Çok zaman dilimli TSC verisini ilişkili piyasalarla karşılaştırır; gerektiğinde güncel makro ve jeopolitik gelişmeleri web üzerinden kaynaklı biçimde araştırır.",
                        color = Th.textDim, fontSize = 13.sp,
                    )

                    aiError?.let { Text(it, color = Th.down, fontSize = 11.sp) }

                    Box(
                        Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(10.dp))
                            .background(if (auth.isLoggedIn) Th.accent else Th.accent.copy(alpha = 0.45f))
                            .clickable(enabled = auth.isLoggedIn && !aiLoading) {
                                scope.launch {
                                    aiLoading = true
                                    aiError = null
                                    runCatching { auth.newApiClient().marketInsight(code) }
                                        .onSuccess { aiInsight = it }
                                        .onFailure { aiError = it.message ?: "AI yorumu şu anda alınamadı." }
                                    aiLoading = false
                                }
                            }
                            .padding(vertical = 11.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            if (!auth.isLoggedIn) "AI yorumu için giriş yap"
                            else if (aiLoading) "Kaynaklar araştırılıyor…"
                            else "✦  Derin araştırmayı başlat",
                            color = Th.bg, fontSize = 13.sp, fontWeight = FontWeight.Bold,
                        )
                    }
                }
            }

            // Per-kind rich detail (crypto/stock/gold/currency).
            DetailSections(detail)

            Box(Modifier.padding(bottom = 24.dp)) {
                toast?.let { Text(it, color = Th.up, fontSize = 12.sp) }
            }
        }
    }

    if (showAlert) {
        AlertDialogCreate(
            currentPrice = ip?.price ?: 0.0,
            currency = ip?.currency,
            onDismiss = { showAlert = false },
            onCreate = { threshold, direction ->
                showAlert = false
                scope.launch {
                    val ok = runCatching {
                        auth.newApiClient().createAlert(code, threshold, direction)
                    }.isSuccess
                    toast = if (ok) "Alarm oluşturuldu." else "Alarm oluşturulamadı."
                }
            },
        )
    }
}

@Composable
private fun DetailSections(d: InstrumentDetail?) {
    if (d == null) {
        // Facebook-style skeleton while the rich data loads.
        MetricGridCard("Piyasa Verileri", null)
        return
    }
    when (d.kind) {
        "crypto" -> {
            MetricGridCard("Piyasa Verileri", cryptoMarket(d))
            MetricGridCard("24 Saat", crypto24h(d))
            MetricGridCard("Arz", cryptoSupply(d))
        }
        "stock" -> {
            MetricGridCard("Hisse Verileri", stockStats(d))
            MetricGridCard("Bilgi", stockInfo(d))
        }
        "gold" -> MetricGridCard("Altın Verileri", goldMetrics(d))
        "currency" -> MetricGridCard("Kur Verileri", currencyMetrics(d))
    }
}

// ---- Metric builders (per kind) ----

private fun cryptoMarket(d: InstrumentDetail): List<Metric> = buildList {
    d.marketCapUsd?.let { add(Metric("Piyasa Değeri", Fmt.usd(it))) }
    d.fdvUsd?.let { add(Metric("Tam Seyrelt. Değer", Fmt.usd(it))) }
    d.rank?.let { add(Metric("Sıralama", "#$it")) }
    d.dominancePct?.let { add(Metric("Dominans", Fmt.pctUnsigned(it))) }
    d.priceUsd?.let { add(Metric("Fiyat (USD)", Fmt.usd(it))) }
}

private fun crypto24h(d: InstrumentDetail): List<Metric> = buildList {
    d.high24h?.let { add(Metric("24s Yüksek", Fmt.usd(it))) }
    d.low24h?.let { add(Metric("24s Düşük", Fmt.usd(it))) }
    d.changePct24h?.let { add(Metric("24s Değişim", Fmt.pct(it))) }
    d.quoteVolumeUsd?.let { add(Metric("24s Hacim", Fmt.usd(it))) }
    d.trades?.let { add(Metric("İşlem Sayısı", Fmt.compact(it.toDouble()))) }
}

private fun cryptoSupply(d: InstrumentDetail): List<Metric> = buildList {
    d.circulatingSupply?.let { add(Metric("Dolaşımdaki Arz", Fmt.compact(it))) }
    d.totalSupply?.let { add(Metric("Toplam Arz", Fmt.compact(it))) }
    d.maxSupply?.let { add(Metric("Maks. Arz", Fmt.compact(it))) }
    d.athUsd?.let { add(Metric("ATH", Fmt.usd(it) + (d.athDate?.let { dt -> " · ${Fmt.dateShort(dt)}" } ?: ""))) }
    d.atlUsd?.let { add(Metric("ATL", Fmt.usd(it) + (d.atlDate?.let { dt -> " · ${Fmt.dateShort(dt)}" } ?: ""))) }
}

private fun stockStats(d: InstrumentDetail): List<Metric> = buildList {
    val c = d.currency
    if (c == "USD") d.priceTry?.let { if (it > 0) add(Metric("Fiyat (₺)", Fmt.tl(it))) }
    d.dayHigh?.let { add(Metric("Gün Yüksek", Fmt.money(it, c))) }
    d.dayLow?.let { add(Metric("Gün Düşük", Fmt.money(it, c))) }
    d.week52High?.let { add(Metric("52h Yüksek", Fmt.money(it, c))) }
    d.week52Low?.let { add(Metric("52h Düşük", Fmt.money(it, c))) }
    d.volume?.let { add(Metric("Hacim", Fmt.compact(it))) }
}

private fun stockInfo(d: InstrumentDetail): List<Metric> = buildList {
    d.longName?.let { add(Metric("Şirket", it)) }
    d.exchange?.let { add(Metric("Borsa", if (it == "IST") "BIST" else it)) }
    d.priceUsd?.let { if (d.currency == "USD") add(Metric("Fiyat (USD)", Fmt.usd(it))) }
}

private fun goldMetrics(d: InstrumentDetail): List<Metric> = buildList {
    d.gramHasTry?.let { add(Metric("Gram Has (₺)", Fmt.tl(it))) }
    d.onsTry?.let { add(Metric("Ons (₺)", Fmt.tl(it))) }
    d.onsUsd?.let { add(Metric("Ons (USD)", Fmt.usd(it))) }
    d.usdTry?.let { add(Metric("USD/TRY", Fmt.number(it))) }
    d.gramWeight?.let { add(Metric("Gram Ağırlık", Fmt.number(it))) }
    d.purityLabel?.let { add(Metric("Ayar", it)) }
}

private fun currencyMetrics(d: InstrumentDetail): List<Metric> = buildList {
    d.iso?.let { add(Metric("Kod", it)) }
    d.priceTry?.let { add(Metric("Fiyat (₺)", Fmt.tl(it))) }
    d.inverse?.let { add(Metric("Ters Kur", Fmt.number(it))) }
    d.usdCross?.let { add(Metric("USD Çapraz", Fmt.number(it))) }
}

// ---- Metric grid card ----

@Composable
private fun MetricGridCard(title: String, metrics: List<Metric>?) {
    // Hide sections that ended up with no data (once loaded).
    if (metrics != null && metrics.isEmpty()) return
    Card {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(title.uppercase(), color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            val rows = (metrics ?: List(4) { Metric("", "") }).chunked(2)
            rows.forEach { row ->
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    row.forEach { m -> MetricCell(m, metrics == null, Modifier.weight(1f)) }
                    if (row.size == 1) Box(Modifier.weight(1f))
                }
            }
        }
    }
}

@Composable
private fun MetricCell(m: Metric, skeleton: Boolean, modifier: Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(4.dp)) {
        if (skeleton) {
            Box(Modifier.fillMaxWidth(0.6f).clip(RoundedCornerShape(4.dp)).background(Th.surfaceAlt).padding(6.dp)) {
                Text("", fontSize = 11.sp)
            }
            Box(Modifier.fillMaxWidth(0.85f).clip(RoundedCornerShape(4.dp)).background(Th.surfaceAlt).padding(8.dp)) {
                Text("", fontSize = 14.sp)
            }
        } else {
            Text(m.label, color = Th.textDim, fontSize = 11.sp, maxLines = 1)
            Text(m.value, color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, maxLines = 1)
        }
    }
}

@Composable
private fun AlertDialogCreate(currentPrice: Double, currency: String?, onDismiss: () -> Unit, onCreate: (Double, String) -> Unit) {
    var threshold by remember { mutableStateOf(if (currentPrice > 0) Fmt.number(currentPrice).replace(",", ".").replace(" ", "") else "") }
    var direction by remember { mutableStateOf("above") }
    val sym = if (currency == "USD") "$" else "₺"
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = Th.surface,
        title = { Text("Alarm kur", color = Th.text) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedTextField(
                    value = threshold,
                    onValueChange = { threshold = it },
                    label = { Text("Eşik fiyatı ($sym)") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    listOf("above" to "Üstüne çıkınca", "below" to "Altına inince").forEach { (v, label) ->
                        val sel = v == direction
                        Box(
                            Modifier
                                .clip(RoundedCornerShape(8.dp))
                                .background(if (sel) Th.accent.copy(alpha = 0.18f) else Th.surfaceAlt)
                                .clickable { direction = v }
                                .padding(horizontal = 12.dp, vertical = 8.dp),
                        ) { Text(label, color = if (sel) Th.accent else Th.textDim, fontSize = 12.sp) }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                val t = threshold.replace(",", ".").toDoubleOrNull()
                if (t != null && t > 0) onCreate(t, direction)
            }) { Text("Oluştur", color = Th.accent) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Vazgeç", color = Th.textDim) } },
    )
}
