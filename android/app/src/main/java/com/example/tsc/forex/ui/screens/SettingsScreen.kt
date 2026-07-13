package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.AppConfig
import com.example.tsc.forex.BuildConfig
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.theme.Th

@Composable
fun SettingsScreen(auth: AuthStore, store: PriceStore, prefs: Prefs) {
    Column(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
    ) {
        ScreenHeader("Ayarlar")

        Column(
            Modifier.padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Card {
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text("HESAP", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    Text(auth.email.ifEmpty { "—" }, color = Th.text, fontSize = 15.sp)
                    Text("TSC Forex v${BuildConfig.VERSION_NAME}", color = Th.textFaint, fontSize = 11.sp)
                }
            }

            if (AppConfig.allowsServerOverride) {
                DebugServerCard(store, prefs)
            }

            Button(
                onClick = { auth.logout() },
                colors = ButtonDefaults.buttonColors(containerColor = Th.down.copy(alpha = 0.15f), contentColor = Th.down),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Çıkış yap", fontWeight = FontWeight.Bold) }
        }
    }
}

@Composable
private fun DebugServerCard(store: PriceStore, prefs: Prefs) {
    var host by remember { mutableStateOf(prefs.hostOverride ?: "") }
    Card {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("GELİŞTİRİCİ · SUNUCU", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            OutlinedTextField(
                value = host,
                onValueChange = { host = it },
                label = { Text("Sunucu (host:port)") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = {
                    prefs.hostOverride = host.ifBlank { null }
                    AppConfig.hostOverride = host.ifBlank { null }
                    store.reconnect()
                    store.refresh()
                }, shape = RoundedCornerShape(10.dp)) { Text("Kaydet & Bağlan") }
            }
        }
    }
}
