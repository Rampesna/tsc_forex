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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
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

@Composable
fun AssetDetailScreen(code: String, auth: AuthStore, store: PriceStore, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val ip = store.prices[code]
    var interval by remember { mutableStateOf("5m") }
    var series by remember { mutableStateOf<List<Double>>(emptyList()) }
    var stats by remember { mutableStateOf<InstrumentStats?>(null) }
    var showAlert by remember { mutableStateOf(false) }
    var toast by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(code, interval) {
        val api = auth.newApiClient()
        series = runCatching { api.candles(code, interval).map { it.close } }
            .getOrElse { store.sparklines[code].orEmpty() }
        if (series.isEmpty()) series = store.sparklines[code].orEmpty()
        stats = runCatching { api.stats(code) }.getOrNull()
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
            Text(ip?.displayName ?: code, color = Th.text, fontSize = 18.sp, fontWeight = FontWeight.Bold)
        }

        Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Row(verticalAlignment = Alignment.Bottom) {
                Text(
                    Fmt.tl(ip?.price ?: 0.0),
                    color = Th.text, fontSize = 34.sp, fontWeight = FontWeight.Bold,
                )
                Box(Modifier.padding(start = 10.dp, bottom = 6.dp)) { ChangePill(ip?.changePct ?: 0.0) }
            }

            // Timeframe selector
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

            // 24h stats
            stats?.let { s ->
                Card {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        StatTile("24s Açılış", Fmt.tl(s.open))
                        StatTile("24s Yüksek", Fmt.tl(s.high), valueColor = Th.up)
                        StatTile("24s Düşük", Fmt.tl(s.low), valueColor = Th.down)
                    }
                }
            }

            // Bid / Ask
            ip?.let {
                Card {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        StatTile("Alış", Fmt.tl(it.bid), valueColor = Th.up)
                        StatTile("Satış", Fmt.tl(it.ask), valueColor = Th.down)
                        StatTile("Sağlayıcı", it.provider.ifEmpty { "-" }, valueColor = Th.textDim)
                    }
                }
            }

            // Actions
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.padding(bottom = 24.dp)) {
                OutlinedButton(
                    onClick = { store.toggleFavorite(code) },
                    modifier = Modifier.weight(1f),
                    shape = RoundedCornerShape(12.dp),
                ) {
                    Text(if (store.isFavorite(code)) "Favoriden çıkar" else "Favori ekle", color = Th.text)
                }
                Button(
                    onClick = { showAlert = true },
                    modifier = Modifier.weight(1f),
                    colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
                    shape = RoundedCornerShape(12.dp),
                ) { Text("Alarm kur", fontWeight = FontWeight.Bold) }
            }

            toast?.let { Text(it, color = Th.up, fontSize = 12.sp, modifier = Modifier.padding(bottom = 12.dp)) }
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
