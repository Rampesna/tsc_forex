package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.PriceAlert
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.AssetBadge
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.PrimaryActionButton
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.SecondaryActionButton
import com.example.tsc.forex.ui.SegmentedControl
import com.example.tsc.forex.ui.ToastOnChange
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

@Composable
fun AlertsScreen(auth: AuthStore, store: PriceStore) {
    val scope = rememberCoroutineScope()
    var alerts by remember { mutableStateOf<List<PriceAlert>>(emptyList()) }
    var loaded by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var editing by remember { mutableStateOf<PriceAlert?>(null) }

    suspend fun load() {
        runCatching { auth.newApiClient().alerts().filter { it.active } }
            .onSuccess { alerts = it; error = null }
            .onFailure { error = it.message }
        loaded = true
    }

    LaunchedEffect(Unit) { load() }
    LaunchedEffect(store.liveNotifications.size) {
        if (loaded) load()
    }
    ToastOnChange(error) { error = null }

    Column(Modifier.fillMaxSize().background(Th.bg)) {
        ScreenHeader("Alarmlar", "Eşik geçilince bir kez bildirir")
        if (loaded && alerts.isEmpty()) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Column(
                    Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 48.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Text("Alarm yok", color = Th.text, fontSize = 16.sp, fontWeight = FontWeight.SemiBold)
                    Text(
                        "Piyasa sekmesinden bir varlık açıp tek seferlik fiyat alarmı kurabilirsiniz.",
                        color = Th.textDim,
                        fontSize = 13.sp,
                        textAlign = TextAlign.Center,
                        modifier = Modifier.padding(horizontal = 40.dp),
                    )
                    SecondaryActionButton(
                        text = "Yenile",
                        onClick = { scope.launch { load() } },
                    )
                }
            }
        } else {
            LazyColumn(
                Modifier.padding(horizontal = 16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                item {
                    Card {
                        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text("Tek seferlik alarmlar", color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.SemiBold)
                            Text(
                                "Eşik gerçekleşip bildirim oluşturulduğunda alarm otomatik olarak aktif listeden kaldırılır.",
                                color = Th.textDim,
                                fontSize = 11.sp,
                            )
                        }
                    }
                }
                items(alerts, key = { it.id }) { a ->
                    val name = store.prices[a.code]?.displayName ?: a.code
                    Card {
                        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                            AssetBadge(a.code, a.code.contains("GOLD", true) || a.code.startsWith("GRAM"))
                            Column(Modifier.weight(1f).padding(start = 12.dp)) {
                                Text(name, color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
                                Text(
                                    if (a.direction == "above") "Üstüne çıkınca" else "Altına inince",
                                    color = Th.textDim,
                                    fontSize = 12.sp,
                                )
                            }
                            Column(horizontalAlignment = Alignment.End) {
                                Text(Fmt.tl(a.threshold), color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
                                Row {
                                    IconButton(onClick = { editing = a }) {
                                        Icon(Icons.Filled.Edit, "Düzenle", tint = Th.accent)
                                    }
                                    IconButton(onClick = {
                                        scope.launch {
                                            if (runCatching { auth.newApiClient().deleteAlert(a.id) }.isSuccess) {
                                                alerts = alerts.filter { it.id != a.id }
                                            }
                                        }
                                    }) { Icon(Icons.Filled.Delete, "Sil", tint = Th.down) }
                                }
                            }
                        }
                    }
                }
                item { Spacer(Modifier.padding(40.dp)) }
            }
        }
    }

    editing?.let { alert ->
        AlertEditSheet(
            alert = alert,
            displayName = store.prices[alert.code]?.displayName ?: alert.code,
            currentPrice = store.prices[alert.code]?.price,
            auth = auth,
            onDismiss = { editing = null },
            onSaved = { updated ->
                alerts = alerts.map { if (it.id == updated.id) updated else it }
                editing = null
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AlertEditSheet(
    alert: PriceAlert,
    displayName: String,
    currentPrice: Double?,
    auth: AuthStore,
    onDismiss: () -> Unit,
    onSaved: (PriceAlert) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    var direction by remember { mutableStateOf(alert.direction) }
    var threshold by remember {
        mutableStateOf(
            alert.threshold.toString().replace('.', ','),
        )
    }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val parsed = threshold.trim().replace(',', '.').toDoubleOrNull()?.takeIf { it > 0 }

    ToastOnChange(error) { error = null }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState, containerColor = Th.bg) {
        Column(
            Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp)
                .padding(bottom = 32.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text("Alarmı Düzenle", color = Th.text, fontSize = 20.sp, fontWeight = FontWeight.Bold)
            Text("$displayName · Tek seferlik", color = Th.textDim, fontSize = 12.sp)
            SegmentedControl(
                options = listOf("Üstüne çıkınca", "Altına inince"),
                selectedIndex = if (direction == "above") 0 else 1,
                onSelected = { direction = if (it == 0) "above" else "below" },
            )
            OutlinedTextField(
                value = threshold,
                onValueChange = { threshold = it },
                label = { Text("Fiyat eşiği (₺)") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                modifier = Modifier.fillMaxWidth(),
            )
            currentPrice?.let {
                Text("Güncel fiyat: ${Fmt.tl(it)}", color = Th.textDim, fontSize = 11.sp)
            }
            if (parsed != null && currentPrice != null) {
                if (direction == "above" && currentPrice >= parsed) {
                    Text(
                        "Güncel fiyat bu eşiğin üzerinde. Alarm ilk fiyat güncellemesinde tetiklenebilir.",
                        color = Th.accent,
                        fontSize = 11.sp,
                    )
                } else if (direction == "below" && currentPrice <= parsed) {
                    Text(
                        "Güncel fiyat bu eşiğin altında. Alarm ilk fiyat güncellemesinde tetiklenebilir.",
                        color = Th.accent,
                        fontSize = 11.sp,
                    )
                }
            }
            PrimaryActionButton(
                text = if (saving) "Kaydediliyor…" else "Değişiklikleri kaydet",
                onClick = {
                    val value = parsed ?: return@PrimaryActionButton
                    scope.launch {
                        saving = true
                        error = null
                        runCatching { auth.newApiClient().updateAlert(alert.id, direction, value) }
                            .onSuccess(onSaved)
                            .onFailure { error = it.message ?: "Kaydedilemedi" }
                        saving = false
                    }
                },
                enabled = parsed != null && !saving,
            )
        }
    }
}
