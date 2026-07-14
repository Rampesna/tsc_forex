package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.InstrumentDetail
import com.example.tsc.forex.data.InstrumentStats
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ChangePill
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.PriceChart
import com.example.tsc.forex.ui.StatTile
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

private val TIMEFRAMES = listOf("1m" to "1dk", "5m" to "5dk", "15m" to "15dk", "1h" to "1sa", "1d" to "1g")

/** A single label/value metric shown in the detail grid. */
private data class Metric(val label: String, val value: String)

@Composable
fun AssetDetailScreen(code: String, auth: AuthStore, store: PriceStore, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val ip = store.prices[code]
    var interval by remember { mutableStateOf("5m") }
    var series by remember { mutableStateOf<List<Double>>(emptyList()) }
    var stats by remember { mutableStateOf<InstrumentStats?>(null) }
    var detail by remember { mutableStateOf<InstrumentDetail?>(null) }
    var showAlert by remember { mutableStateOf(false) }
    var toast by remember { mutableStateOf<String?>(null) }
    var isFav by remember { mutableStateOf(store.isFavorite(code)) }

    LaunchedEffect(code, interval) {
        val api = auth.newApiClient()
        series = runCatching { api.candles(code, interval).map { it.close } }
            .getOrElse { store.sparklines[code].orEmpty() }
        if (series.isEmpty()) series = store.sparklines[code].orEmpty()
        stats = runCatching { api.stats(code) }.getOrNull()
    }
    LaunchedEffect(code) {
        detail = runCatching { auth.newApiClient().detail(code) }.getOrNull()
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
            Row(verticalAlignment = Alignment.Bottom) {
                Text(
                    Fmt.tl(ip?.price ?: 0.0),
                    color = Th.text, fontSize = 34.sp, fontWeight = FontWeight.Bold,
                )
                Box(Modifier.padding(start = 10.dp, bottom = 6.dp)) { ChangePill(ip?.changePct ?: 0.0) }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                TIMEFRAMES.forEach { (key, label) ->
                    val selected = key == interval
                    Box(
                        Modifier
                            .clip(RoundedCornerShape(8.dp))
                            .background(if (selected) Th.accent.copy(alpha = 0.18f) else Th.surface)
                            .clickable { interval = key }
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    ) {
                        Text(label, color = if (selected) Th.accent else Th.textDim, fontSize = 12.sp)
                    }
                }
            }

            Card(padding = androidx.compose.foundation.layout.PaddingValues(8.dp)) {
                PriceChart(series)
            }

            stats?.let { s ->
                Card {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        StatTile("24s Açılış", Fmt.tl(s.open))
                        StatTile("24s Yüksek", Fmt.tl(s.high), valueColor = Th.up)
                        StatTile("24s Düşük", Fmt.tl(s.low), valueColor = Th.down)
                    }
                }
            }

            ip?.let {
                Card {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        StatTile("Alış", Fmt.tl(it.bid), valueColor = Th.up)
                        StatTile("Satış", Fmt.tl(it.ask), valueColor = Th.down)
                        StatTile("Sağlayıcı", it.provider.ifEmpty { "-" }, valueColor = Th.textDim)
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
    d.dayHigh?.let { add(Metric("Gün Yüksek", Fmt.tl(it))) }
    d.dayLow?.let { add(Metric("Gün Düşük", Fmt.tl(it))) }
    d.week52High?.let { add(Metric("52h Yüksek", Fmt.tl(it))) }
    d.week52Low?.let { add(Metric("52h Düşük", Fmt.tl(it))) }
    d.volume?.let { add(Metric("Hacim", Fmt.compact(it))) }
}

private fun stockInfo(d: InstrumentDetail): List<Metric> = buildList {
    d.longName?.let { add(Metric("Şirket", it)) }
    d.exchange?.let { add(Metric("Borsa", it)) }
    d.priceUsd?.let { add(Metric("Fiyat (USD)", Fmt.usd(it))) }
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
private fun AlertDialogCreate(currentPrice: Double, onDismiss: () -> Unit, onCreate: (Double, String) -> Unit) {
    var threshold by remember { mutableStateOf(if (currentPrice > 0) Fmt.number(currentPrice).replace(",", ".").replace(" ", "") else "") }
    var direction by remember { mutableStateOf("above") }
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = Th.surface,
        title = { Text("Alarm kur", color = Th.text) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedTextField(
                    value = threshold,
                    onValueChange = { threshold = it },
                    label = { Text("Eşik fiyatı (₺)") },
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
