package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.StarBorder
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import com.example.tsc.forex.ui.LocalizedText as Text
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
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.ApiException
import com.example.tsc.forex.data.Candle
import com.example.tsc.forex.data.DeviceIdentity
import com.example.tsc.forex.data.InstrumentDetail
import com.example.tsc.forex.data.InstrumentStats
import com.example.tsc.forex.data.MarketInsight
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.BillingStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ChartMode
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.PriceChart
import com.example.tsc.forex.ui.PrimaryActionButton
import com.example.tsc.forex.ui.SegmentedControl
import com.example.tsc.forex.ui.StatTile
import com.example.tsc.forex.ui.ToastOnChange
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
private data class Metric(
    val label: String,
    val value: String,
    val tint: androidx.compose.ui.graphics.Color? = null,
    val sub: String? = null,
)

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
fun AssetDetailScreen(
    code: String,
    auth: AuthStore,
    store: PriceStore,
    billing: BillingStore,
    prefs: Prefs,
    onBack: () -> Unit,
    onRequestAuth: () -> Unit = {},
) {
    val scope = rememberCoroutineScope()
    val uriHandler = LocalUriHandler.current
    val ip = store.prices[code]
    var rangeKey by remember { mutableStateOf("1G") }
    var candles by remember { mutableStateOf<List<Candle>>(emptyList()) }
    var chartLoading by remember { mutableStateOf(true) }
    var chartError by remember { mutableStateOf<String?>(null) }
    var chartReload by remember { mutableStateOf(0) }
    var mode by remember { mutableStateOf(ChartMode.LINE) }
    var stats by remember { mutableStateOf<InstrumentStats?>(null) }
    var detail by remember { mutableStateOf<InstrumentDetail?>(null) }
    var showAlert by remember { mutableStateOf(false) }
    var toast by remember { mutableStateOf<String?>(null) }
    var isFav by remember { mutableStateOf(store.isFavorite(code)) }
    var aiInsight by remember { mutableStateOf<MarketInsight?>(null) }
    var aiLoading by remember { mutableStateOf(false) }
    var aiError by remember { mutableStateOf<String?>(null) }
    var showAIDisclosure by remember { mutableStateOf(false) }
    var showPaywall by remember { mutableStateOf(false) }
    var acceptedAI by remember { mutableStateOf(false) }

    LaunchedEffect(auth.userId) {
        acceptedAI = prefs.aiResearchConsent(auth.userId.orEmpty())
    }

    suspend fun loadAIInsight() {
        if (!auth.isLoggedIn) {
            aiError = "Bu özellik için hesabınızla giriş yapın."
            return
        }
        val userId = auth.ensureUserId()
        if (userId.isNullOrBlank()) {
            aiError = "Hesap bilgileri doğrulanamadı. Profilinizi yenileyip tekrar deneyin."
            return
        }
        val status = billing.serverStatus
        if (status?.plan?.code == "free" &&
            status.freeResearchAvailable &&
            !billing.freeTrialReady
        ) {
            aiError = "Ücretsiz araştırma için cihaz doğrulama şu anda Android’de kullanılamıyor. iOS aboneliğiniz varsa aynı hesapla devam edebilir veya plan seçebilirsiniz."
            showPaywall = true
            return
        }
        aiLoading = true
        aiError = null
        val op = "ai-research"
        val resource = code.uppercase()
        val requestId = prefs.pendingRequestId(userId, op, resource)
        try {
            aiInsight = auth.newApiClient().marketInsight(
                code = code,
                deviceId = DeviceIdentity.id(prefs),
                requestId = requestId,
            )
            prefs.clearPendingRequestId(userId, op, resource)
            billing.refreshNow()
        } catch (e: ApiException) {
            if (!e.shouldRetainRequestId) prefs.clearPendingRequestId(userId, op, resource)
            when {
                e.code == 402 -> {
                    aiError = "Bu dönem için AI araştırma hakkınız doldu. Bir plan seçerek devam edebilirsiniz."
                    showPaywall = true
                }
                e.code == 429 ->
                    aiError = "Çok sık istek gönderildi. Kısa bir süre bekleyip aynı araştırmayı yeniden deneyin."
                e.code == 409 && e.serverCode == "operation_in_progress" ->
                    aiError = "Araştırma işlemi halen tamamlanıyor. Biraz sonra tekrar deneyin; mevcut hakkınız yeniden kullanılmayacak."
                e.code == 403 && e.serverCode == "device_attestation_required" -> {
                    aiError = "Ücretsiz AI hakkı için doğrulanmış iOS uygulaması gerekir. Aboneliğiniz varsa durum yenileyin veya plan seçin."
                    showPaywall = true
                }
                else -> aiError = e.message ?: "AI yorumu şu anda alınamadı."
            }
        } catch (e: Exception) {
            prefs.clearPendingRequestId(userId, op, resource)
            aiError = e.message ?: "AI yorumu şu anda alınamadı."
        } finally {
            aiLoading = false
        }
    }

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

    LaunchedEffect(code, rangeKey, chartReload) {
        val api = auth.newApiClient()
        val tf = TIMEFRAMES.firstOrNull { it.key == rangeKey } ?: TIMEFRAMES[0]
        val fromMs = System.currentTimeMillis() - tf.days * 86_400_000L
        hasMoreHistory = true
        hasMoreFuture = false
        chartLoading = true
        chartError = null
        try {
            val loaded = api.candles(
                code,
                tf.interval,
                fromMs = fromMs,
                latestSession = rangeKey == "1G",
            )
            candles = loaded.takeLast(MAX_WINDOW)
            chartReset++
        } catch (_: Exception) {
            candles = emptyList()
            chartError = "Grafik verisi şu anda alınamadı."
        } finally {
            chartLoading = false
        }
        stats = runCatching { api.stats(code) }.getOrNull()
    }
    LaunchedEffect(code) {
        detail = runCatching { auth.newApiClient().detail(code) }.getOrNull()
    }

    // Live-fold WS price into the last candle. Round price key to cut Compose redraw thrash.
    val livePriceKey = ((ip?.price ?: 0.0) * 100).toLong()
    val displayCandles = remember(candles, livePriceKey, hasMoreFuture) {
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

    val ccy = ip?.currency
    val changePct = ip?.changePct ?: 0.0
    val changeAmt = ip?.change ?: 0.0
    val trendColor = if (changePct >= 0) Th.up else Th.down

    fun statVal(v: Double?): String {
        if (v == null || v <= 0) return "—"
        return Fmt.money(v, ccy)
    }

    Column(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
            .verticalScroll(rememberScrollState()),
    ) {
        // iOS toolbar: back · centered name/code · star + bell
        Box(
            Modifier
                .fillMaxWidth()
                .statusBarsPadding()
                .padding(horizontal = 4.dp, vertical = 4.dp),
        ) {
            IconButton(onClick = onBack, modifier = Modifier.align(Alignment.CenterStart)) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, "geri", tint = Th.text)
            }
            Column(
                Modifier
                    .align(Alignment.Center)
                    .padding(horizontal = 108.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    ip?.displayName ?: code,
                    color = Th.text,
                    style = MaterialTheme.typography.titleMedium,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                )
                Text(code, color = Th.textDim, style = MaterialTheme.typography.labelSmall, maxLines = 1)
            }
            Row(Modifier.align(Alignment.CenterEnd)) {
                IconButton(onClick = {
                    if (!auth.isLoggedIn) onRequestAuth()
                    else {
                        store.toggleFavorite(code)
                        isFav = store.isFavorite(code)
                    }
                }) {
                    Icon(
                        if (isFav) Icons.Filled.Star else Icons.Outlined.StarBorder,
                        "favori", tint = if (isFav) Th.accent else Th.textDim,
                    )
                }
                IconButton(onClick = {
                    if (!auth.isLoggedIn) onRequestAuth() else showAlert = true
                }) {
                    Icon(Icons.Outlined.Notifications, "alarm", tint = Th.textDim)
                }
            }
        }

        Column(
            Modifier.padding(horizontal = 16.dp).padding(top = 8.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            // 1) priceHeader
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                BoxWithConstraints(Modifier.fillMaxWidth()) {
                    Text(
                        ip?.let { Fmt.money(it.price, ccy) } ?: "—",
                        color = if (ip == null) Th.textDim else Th.text,
                        fontSize = if (maxWidth < 360.dp) 30.sp else 36.sp,
                        fontWeight = FontWeight.Bold,
                        maxLines = 1,
                        overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                    )
                }
                if (ip?.isUSD == true) {
                    Text("≈ ${Fmt.tl(ip.tryValue)}", color = Th.textDim, fontSize = 14.sp, fontWeight = FontWeight.Medium)
                }
                if (ip != null) {
                    val unit = if (ip.isUSD) "$" else "₺"
                    Text(
                        "${Fmt.signed(changeAmt)} $unit  ${Fmt.signedPct(changePct)}",
                        color = trendColor,
                        fontSize = 15.sp,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }

            // 2) stats24h — Yüksek · Düşük · Açılış · Son
            Card {
                BoxWithConstraints(Modifier.fillMaxWidth()) {
                    val tiles = listOf(
                        Triple("24s Yüksek", statVal(stats?.high), Th.up),
                        Triple("24s Düşük", statVal(stats?.low), Th.down),
                        Triple("24s Açılış", statVal(stats?.open), Th.text),
                        Triple("24s Son", statVal(stats?.last ?: ip?.price), Th.text),
                    )
                    if (maxWidth < 430.dp) {
                        Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
                            tiles.chunked(2).forEach { row ->
                                Row(
                                    Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.spacedBy(16.dp),
                                ) {
                                    row.forEach { (label, value, color) ->
                                        StatTile(label, value, color, Modifier.weight(1f))
                                    }
                                }
                            }
                        }
                    } else {
                        Row(
                            Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                        ) {
                            tiles.forEach { (label, value, color) ->
                                StatTile(label, value, color, Modifier.weight(1f))
                            }
                        }
                    }
                }
            }

            // 3) chartCard — Çizgi/Mum + grafik + 1G…MAX
            Card(padding = PaddingValues(14.dp)) {
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                        SegmentedControl(
                            options = listOf("Çizgi", "Mum"),
                            selectedIndex = if (mode == ChartMode.LINE) 0 else 1,
                            onSelected = { mode = if (it == 0) ChartMode.LINE else ChartMode.CANDLE },
                            modifier = Modifier.weight(1f),
                        )
                        if (loadingOlder || loadingNewer) {
                            Spacer(Modifier.width(10.dp))
                            CircularProgressIndicator(
                                modifier = Modifier.height(16.dp).width(16.dp),
                                color = Th.accent,
                                strokeWidth = 2.dp,
                            )
                        }
                    }
                    BoxWithConstraints(Modifier.fillMaxWidth()) {
                        when {
                            chartLoading -> Box(
                                Modifier
                                    .fillMaxWidth()
                                    .height(if (maxWidth < 360.dp) 210.dp else 240.dp),
                                contentAlignment = Alignment.Center,
                            ) {
                                Column(
                                    horizontalAlignment = Alignment.CenterHorizontally,
                                    verticalArrangement = Arrangement.spacedBy(10.dp),
                                ) {
                                    CircularProgressIndicator(
                                        modifier = Modifier.size(24.dp),
                                        color = Th.accent,
                                        strokeWidth = 2.dp,
                                    )
                                    Text("Grafik hazırlanıyor…", color = Th.textDim, fontSize = 12.sp)
                                }
                            }
                            chartError != null -> Box(
                                Modifier
                                    .fillMaxWidth()
                                    .height(if (maxWidth < 360.dp) 210.dp else 240.dp),
                                contentAlignment = Alignment.Center,
                            ) {
                                Column(
                                    horizontalAlignment = Alignment.CenterHorizontally,
                                    verticalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    Text(chartError!!, color = Th.textDim, fontSize = 12.sp)
                                    TextButton(onClick = { chartReload++ }) {
                                        Text("Yeniden dene", color = Th.accent, fontWeight = FontWeight.Bold)
                                    }
                                }
                            }
                            else -> PriceChart(
                                candles = displayCandles,
                                mode = mode,
                                trendUp = changePct >= 0,
                                currency = ccy,
                                resetToken = chartReset,
                                canLoadOlder = hasMoreHistory,
                                canLoadNewer = hasMoreFuture,
                                onReachStart = { scope.launch { loadOlder() } },
                                onReachEnd = { scope.launch { loadNewer() } },
                                heightDp = if (maxWidth < 360.dp) 210 else 240,
                                emptyTitle = "Son işlem seansı bulunamadı",
                                emptySubtitle = "Piyasa açıldığında grafik otomatik güncellenecek.",
                            )
                        }
                        if (loadingOlder || loadingNewer) {
                            Text(
                                if (loadingOlder) "Geçmiş yükleniyor…" else "Güncel yükleniyor…",
                                color = Th.textDim,
                                fontSize = 10.sp,
                                modifier = Modifier
                                    .align(Alignment.TopStart)
                                    .padding(8.dp)
                                    .clip(RoundedCornerShape(20.dp))
                                    .background(Th.bg.copy(alpha = 0.85f))
                                    .padding(horizontal = 8.dp, vertical = 3.dp),
                            )
                        }
                    }
                    SegmentedControl(
                        options = TIMEFRAMES.map { it.label },
                        selectedIndex = TIMEFRAMES.indexOfFirst { it.key == rangeKey }.coerceAtLeast(0),
                        onSelected = { index -> rangeKey = TIMEFRAMES[index].key },
                    )
                }
            }

            // 4) AI PİYASA ARAŞTIRMASI
            Card(padding = PaddingValues(16.dp)) {
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(7.dp)) {
                        Text("✦", color = Th.accent, fontSize = 16.sp, fontWeight = FontWeight.Bold)
                        Text(
                            "AI PİYASA ARAŞTIRMASI",
                            color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold,
                        )
                        Spacer(Modifier.weight(1f))
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
                        insight.whatIWouldDo?.let { playbook ->
                            ResearchTitle("BEN OLSAM NE YAPARDIM")
                            Text(playbook.intro, color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                            listOf(
                                "Kısa vade" to playbook.shortTerm,
                                "Orta vade" to playbook.mediumTerm,
                                "Uzun vade" to playbook.longTerm,
                            ).forEach { (label, body) ->
                                Column(
                                    Modifier.fillMaxWidth().clip(RoundedCornerShape(9.dp)).background(Th.surfaceAlt).padding(10.dp),
                                    verticalArrangement = Arrangement.spacedBy(3.dp),
                                ) {
                                    Text(label, color = Th.accent, fontSize = 10.sp, fontWeight = FontWeight.Bold)
                                    Text(body, color = Th.text, fontSize = 12.sp)
                                }
                            }
                        }
                        if (insight.riskNote.isNotBlank()) {
                            Text("⚠ ${insight.riskNote}", color = Th.textDim, fontSize = 11.sp)
                        }
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
                        "Çok zaman dilimli piyasa verilerini ilişkili varlıklarla karşılaştırır; güncel makro ve jeopolitik gelişmeleri kaynaklarıyla araştırır. Sonuçlar hatalı veya eksik olabilir ve yatırım tavsiyesi değildir.",
                        color = Th.textDim, fontSize = 13.sp,
                    )

                    PrimaryActionButton(
                        text = when {
                            !auth.isLoggedIn -> "Araştırma için giriş yap"
                            aiLoading -> "Kaynaklar araştırılıyor…"
                            else -> "✦  Derin araştırmayı başlat"
                        },
                        enabled = !aiLoading,
                        onClick = {
                                if (!auth.isLoggedIn) {
                                    onRequestAuth()
                                    return@PrimaryActionButton
                                }
                                scope.launch {
                                    if (!acceptedAI) showAIDisclosure = true
                                    else loadAIInsight()
                                }
                        },
                    )
                }
            }

            // 5) kind-specific grids
            DetailSections(detail, code)

            // 6) HESAPLA
            InstantCalculatorCard(unitPrice = ip?.price ?: 0.0)

            Spacer(Modifier.height(24.dp))
        }
    }

    ToastOnChange(toast) { toast = null }
    ToastOnChange(aiError) { aiError = null }

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

    if (showAIDisclosure) {
        AlertDialog(
            onDismissRequest = { showAIDisclosure = false },
            containerColor = Th.surface,
            title = { Text("AI araştırma onayı", color = Th.text) },
            text = {
                Text(
                    "Derin araştırmada piyasa verileri ve açık web kaynakları OpenAI’a gönderilir. Kişisel profil ve portföy tutarları gönderilmez. Devam ederek bu paylaşımı onaylarsınız.",
                    color = Th.textDim,
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        val userId = auth.ensureUserId()
                        if (userId.isNullOrBlank()) {
                            aiError = "AI araştırma onayı hesabınıza kaydedilemedi. Profilinizi yenileyip tekrar deneyin."
                            showAIDisclosure = false
                            return@launch
                        }
                        prefs.setAiResearchConsent(userId, true)
                        acceptedAI = true
                        showAIDisclosure = false
                        loadAIInsight()
                    }
                }) { Text("Kabul et ve başlat", color = Th.accent, fontWeight = FontWeight.Bold) }
            },
            dismissButton = {
                TextButton(onClick = { showAIDisclosure = false }) {
                    Text("Vazgeç", color = Th.textDim)
                }
            },
        )
    }

    if (showPaywall) {
        PaywallSheet(billing = billing, onDismiss = { showPaywall = false })
    }
}

@Composable
private fun InstantCalculatorCard(unitPrice: Double) {
    var amount by remember { mutableStateOf("1") }
    val units = amount.replace(',', '.').toDoubleOrNull() ?: 0.0
    val result = units * unitPrice
    Card(padding = PaddingValues(14.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("HESAPLA", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            BoxWithConstraints(Modifier.fillMaxWidth()) {
                if (maxWidth < 360.dp) {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedTextField(
                            value = amount,
                            onValueChange = { amount = it },
                            label = { Text("Miktar") },
                            singleLine = true,
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Text(
                            "Sonuç: ₺${Fmt.compactPrice(result)}",
                            color = Th.accent,
                            style = MaterialTheme.typography.titleMedium,
                        )
                    }
                } else {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        OutlinedTextField(
                            value = amount,
                            onValueChange = { amount = it },
                            label = { Text("Miktar") },
                            singleLine = true,
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                            modifier = Modifier.weight(1f),
                        )
                        Column(Modifier.weight(1f), horizontalAlignment = Alignment.End) {
                            Text("Hesaplanan değer", color = Th.textDim, fontSize = 11.sp)
                            Text(
                                "₺${Fmt.compactPrice(result)}",
                                color = Th.accent,
                                style = MaterialTheme.typography.titleMedium,
                                maxLines = 1,
                            )
                        }
                    }
                }
            }
            val quickAmounts = listOf("0.25", "0.5", "1", "5", "10")
            SegmentedControl(
                options = quickAmounts,
                selectedIndex = quickAmounts.indexOf(amount),
                onSelected = { amount = quickAmounts[it] },
            )
        }
    }
}

@Composable
private fun DetailSections(d: InstrumentDetail?, code: String) {
    if (d == null) {
        MetricGridCard("Piyasa Verileri", null)
        return
    }
    when (d.kind) {
        "crypto" -> {
            MetricGridCard("Piyasa Verileri", cryptoMarket(d, code))
            MetricGridCard("24s İstatistikler", cryptoStats(d))
            MetricGridCard("Rekorlar", cryptoRecords(d))
        }
        "stock" -> {
            MetricGridCard("İstatistikler", stockStats(d))
            MetricGridCard("Şirket Bilgisi", stockInfo(d), columns = 1)
        }
        "gold" -> MetricGridCard("Altın Verileri", goldMetrics(d))
        "currency" -> MetricGridCard("Kur Verileri", currencyMetrics(d, code))
    }
}

private fun trimNum(v: Double): String =
    if (kotlin.math.abs(v - v.toLong()) < 1e-9) v.toLong().toString() else Fmt.number(v)

private fun cryptoMarket(d: InstrumentDetail, code: String): List<Metric> = buildList {
    d.marketCapUsd?.let { add(Metric("Piyasa Değeri", Fmt.usd(it))) }
    d.rank?.let { add(Metric("Sıralama", "#$it")) }
    d.dominancePct?.let { add(Metric("Dominans", Fmt.pctUnsigned(it))) }
    d.quoteVolumeUsd?.let { add(Metric("24s Hacim", Fmt.usd(it))) }
    d.circulatingSupply?.let { add(Metric("Dolaşan Arz", "${Fmt.compact(it)} $code")) }
    when {
        d.maxSupply != null -> add(Metric("Maks. Arz", "${Fmt.compact(d.maxSupply)} $code"))
        d.totalSupply != null -> add(Metric("Toplam Arz", "${Fmt.compact(d.totalSupply)} $code"))
    }
    d.fdvUsd?.let { add(Metric("FDV", Fmt.usd(it))) }
    d.trades?.let { add(Metric("İşlem (24s)", Fmt.compact(it.toDouble()))) }
}

private fun cryptoStats(d: InstrumentDetail): List<Metric> = buildList {
    d.high24h?.takeIf { it > 0 }?.let { add(Metric("24s Yüksek", Fmt.tl(it), tint = Th.up)) }
    d.low24h?.takeIf { it > 0 }?.let { add(Metric("24s Düşük", Fmt.tl(it), tint = Th.down)) }
    d.open24h?.takeIf { it > 0 }?.let { add(Metric("24s Açılış", Fmt.tl(it))) }
    d.weightedAvg?.takeIf { it > 0 }?.let { add(Metric("Ort. Fiyat", Fmt.tl(it))) }
    d.changePct24h?.let { add(Metric("24s Değişim", Fmt.signedPct(it), tint = if (it >= 0) Th.up else Th.down)) }
    d.priceUsd?.takeIf { it > 0 }?.let { add(Metric("USD Fiyat", Fmt.usd(it))) }
}

private fun cryptoRecords(d: InstrumentDetail): List<Metric> = buildList {
    d.athUsd?.takeIf { it > 0 }?.let {
        add(Metric("En Yüksek (ATH)", Fmt.usd(it), sub = d.athDate?.let(Fmt::dateShort)))
    }
    d.atlUsd?.takeIf { it > 0 }?.let {
        add(Metric("En Düşük (ATL)", Fmt.usd(it), sub = d.atlDate?.let(Fmt::dateShort)))
    }
}

private fun stockStats(d: InstrumentDetail): List<Metric> = buildList {
    val c = d.currency
    if (c == "USD") d.priceTry?.takeIf { it > 0 }?.let { add(Metric("Fiyat (₺)", Fmt.tl(it))) }
    d.dayHigh?.takeIf { it > 0 }?.let { add(Metric("Gün Yüksek", Fmt.money(it, c), tint = Th.up)) }
    d.dayLow?.takeIf { it > 0 }?.let { add(Metric("Gün Düşük", Fmt.money(it, c), tint = Th.down)) }
    d.week52High?.takeIf { it > 0 }?.let { add(Metric("52H Yüksek", Fmt.money(it, c))) }
    d.week52Low?.takeIf { it > 0 }?.let { add(Metric("52H Düşük", Fmt.money(it, c))) }
    d.prevClose?.takeIf { it > 0 }?.let { add(Metric("Önceki Kapanış", Fmt.money(it, c))) }
    d.volume?.takeIf { it > 0 }?.let { add(Metric("Hacim (Lot)", Fmt.compact(it))) }
}

private fun stockInfo(d: InstrumentDetail): List<Metric> = buildList {
    d.exchange?.let { add(Metric("Borsa", if (it == "IST") "BIST" else it)) }
    d.longName?.let { add(Metric("Şirket", it)) }
}

private fun goldMetrics(d: InstrumentDetail): List<Metric> = buildList {
    d.onsUsd?.takeIf { it > 0 }?.let { add(Metric("Ons (XAU/USD)", Fmt.usd(it))) }
    d.onsTry?.takeIf { it > 0 }?.let { add(Metric("Ons (₺)", Fmt.tl(it))) }
    d.gramHasTry?.takeIf { it > 0 }?.let { add(Metric("Gram Has Altın", Fmt.tl(it))) }
    d.usdTry?.takeIf { it > 0 }?.let { add(Metric("USD/TRY", Fmt.tl(it))) }
    d.purityLabel?.let { add(Metric("Saflık", it)) }
    d.gramWeight?.takeIf { it > 0 }?.let { add(Metric("Gram Ağırlığı", "${trimNum(it)} gr")) }
}

private fun currencyMetrics(d: InstrumentDetail, code: String): List<Metric> = buildList {
    val iso = d.iso ?: code.replace("TRY", "")
    d.priceTry?.takeIf { it > 0 }?.let { add(Metric("1 $iso", Fmt.tl(it))) }
    d.inverse?.takeIf { it > 0 }?.let { add(Metric("1 ₺", "${Fmt.compactPrice(it)} $iso")) }
    d.usdCross?.takeIf { it > 0 }?.let { add(Metric("$iso/USD", Fmt.usd(it))) }
    d.usdTry?.takeIf { it > 0 }?.let { add(Metric("USD/TRY", Fmt.tl(it))) }
}

@Composable
private fun MetricGridCard(title: String, metrics: List<Metric>?, columns: Int = 2) {
    if (metrics != null && metrics.isEmpty()) return
    Card(padding = PaddingValues(14.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Text(title, color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.Bold)
            val rows = (metrics ?: List(4) { Metric("", "") }).chunked(columns)
            rows.forEach { row ->
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                    row.forEach { m -> MetricCell(m, metrics == null, Modifier.weight(1f)) }
                    repeat(columns - row.size) { Box(Modifier.weight(1f)) }
                }
            }
        }
    }
}

@Composable
private fun MetricCell(m: Metric, skeleton: Boolean, modifier: Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(3.dp)) {
        if (skeleton) {
            Box(Modifier.fillMaxWidth(0.6f).clip(RoundedCornerShape(4.dp)).background(Th.surfaceAlt).height(9.dp))
            Box(Modifier.fillMaxWidth(0.85f).clip(RoundedCornerShape(4.dp)).background(Th.surfaceAlt).height(15.dp))
        } else {
            Text(m.label, color = Th.textDim, fontSize = 11.sp, maxLines = 1)
            Text(m.value, color = m.tint ?: Th.text, fontSize = 15.sp, fontWeight = FontWeight.SemiBold, maxLines = 1)
            m.sub?.takeIf { it.isNotBlank() }?.let {
                Text(it, color = Th.textFaint, fontSize = 10.sp, maxLines = 1)
            }
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
                SegmentedControl(
                    options = listOf("Üstüne çıkınca", "Altına inince"),
                    selectedIndex = if (direction == "above") 0 else 1,
                    onSelected = { direction = if (it == 0) "above" else "below" },
                )
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
