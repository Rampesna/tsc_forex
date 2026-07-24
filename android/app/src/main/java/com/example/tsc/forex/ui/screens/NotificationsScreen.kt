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
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.AppNotification
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

@Composable
fun NotificationsScreen(
    auth: AuthStore,
    store: PriceStore,
    onBack: (() -> Unit)? = null,
) {
    val scope = rememberCoroutineScope()
    var history by remember { mutableStateOf<List<AppNotification>>(emptyList()) }

    LaunchedEffect(Unit) {
        history = runCatching { auth.newApiClient().notifications() }.getOrDefault(emptyList())
    }

    val live = store.liveNotifications

    Column(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
    ) {
        ScreenHeader("Bildirimler", trailing = {
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (live.isNotEmpty() || history.isNotEmpty()) {
                    TextButton(onClick = {
                        store.clearLiveNotifications()
                        scope.launch {
                            if (runCatching { auth.newApiClient().clearNotifications() }.isSuccess) {
                                history = emptyList()
                            }
                        }
                    }) { Text("Temizle", color = Th.accent, fontSize = 13.sp) }
                }
                if (onBack != null) {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri", tint = Th.text)
                    }
                }
            }
        })

        if (live.isEmpty() && history.isEmpty()) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text("Henüz bildirim yok.", color = Th.textDim, fontSize = 14.sp)
            }
            return@Column
        }

        LazyColumn(
            Modifier.padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            if (live.isNotEmpty()) {
                itemsIndexed(live, key = { i, _ -> "live-$i" }) { index, n ->
                    NotificationCard(n, live = true) { store.removeLiveNotification(index) }
                }
            }
            itemsIndexed(history, key = { _, n -> n.id }) { _, n ->
                NotificationCard(n, live = false) {
                    scope.launch {
                        if (runCatching { auth.newApiClient().deleteNotification(n.id) }.isSuccess) {
                            history = history.filter { it.id != n.id }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun NotificationCard(n: AppNotification, live: Boolean, onDelete: () -> Unit) {
    Card {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    n.title.ifEmpty { "Fiyat alarmı" },
                    color = if (live) Th.accent else Th.text,
                    fontSize = 14.sp, fontWeight = FontWeight.SemiBold,
                )
                if (n.body.isNotEmpty()) Text(n.body, color = Th.textDim, fontSize = 12.sp)
            }
            IconButton(onClick = onDelete) { Icon(Icons.Filled.Close, "sil", tint = Th.textFaint) }
        }
    }
}
