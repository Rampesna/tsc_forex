package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.PriceAlert
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.AssetBadge
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

@Composable
fun AlertsScreen(auth: AuthStore, store: PriceStore) {
    val scope = rememberCoroutineScope()
    var alerts by remember { mutableStateOf<List<PriceAlert>>(emptyList()) }
    var loaded by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        alerts = runCatching { auth.newApiClient().alerts() }.getOrDefault(emptyList())
        loaded = true
    }

    Column(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
    ) {
        ScreenHeader("Alarmlar", if (alerts.isEmpty()) null else "${alerts.size} aktif alarm")
        if (loaded && alerts.isEmpty()) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text("Henüz alarm kurmadın.\nBir varlık detayından alarm ekleyebilirsin.",
                    color = Th.textDim, fontSize = 14.sp)
            }
        } else {
            LazyColumn(
                Modifier.padding(horizontal = 16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                items(alerts, key = { it.id }) { a ->
                    Card {
                        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                            AssetBadge(a.code, a.code.endsWith("GOLD") || a.code.startsWith("GRAM"))
                            Column(
                                Modifier
                                    .weight(1f)
                                    .padding(start = 12.dp)
                            ) {
                                Text(a.code, color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
                                Text(
                                    (if (a.direction == "above") "Üstüne çıkınca • " else "Altına inince • ") + Fmt.tl(a.threshold),
                                    color = Th.textDim, fontSize = 12.sp,
                                )
                            }
                            IconButton(onClick = {
                                scope.launch {
                                    if (runCatching { auth.newApiClient().deleteAlert(a.id) }.isSuccess) {
                                        alerts = alerts.filter { it.id != a.id }
                                    }
                                }
                            }) { Icon(Icons.Filled.Delete, "sil", tint = Th.down) }
                        }
                    }
                }
            }
        }
    }
}
