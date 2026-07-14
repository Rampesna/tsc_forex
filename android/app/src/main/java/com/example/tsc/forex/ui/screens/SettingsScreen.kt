package com.example.tsc.forex.ui.screens

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.example.tsc.forex.AppConfig
import com.example.tsc.forex.BuildConfig
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.theme.Th
import java.io.ByteArrayOutputStream

@Composable
fun SettingsScreen(auth: AuthStore, store: PriceStore, prefs: Prefs) {
    val uriHandler = LocalUriHandler.current
    var confirmDelete by remember { mutableStateOf(false) }

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
            ProfileCard(auth)

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("YASAL", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    LegalRow("Gizlilik Politikası") { uriHandler.openUri("https://tsc.talhacan.com/privacy") }
                    LegalRow("Kullanım Şartları") { uriHandler.openUri("https://tsc.talhacan.com/terms") }
                    LegalRow("Destek & Yardım") { uriHandler.openUri("https://tsc.talhacan.com/support") }
                }
            }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text("UYGULAMA", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    Text("TSC Forex v${BuildConfig.VERSION_NAME}", color = Th.textFaint, fontSize = 11.sp)
                }
            }

            if (AppConfig.allowsServerOverride) {
                DebugServerCard(store, prefs)
            }

            Button(
                onClick = { auth.logout() },
                colors = ButtonDefaults.buttonColors(containerColor = Th.surfaceAlt, contentColor = Th.text),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Çıkış yap", fontWeight = FontWeight.Bold) }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("TEHLİKELİ BÖLGE", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    Text("Hesabını silersen tüm verilerin (favoriler, alarmlar, portföy, bildirimler, ödüller) kalıcı olarak kaldırılır. Bu işlem geri alınamaz.",
                        color = Th.textDim, fontSize = 12.sp)
                    Button(
                        onClick = { confirmDelete = true },
                        enabled = !auth.isBusy,
                        colors = ButtonDefaults.buttonColors(containerColor = Th.down.copy(alpha = 0.15f), contentColor = Th.down),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Hesabı Sil", fontWeight = FontWeight.Bold) }
                }
            }
        }
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            containerColor = Th.surface,
            title = { Text("Hesabını sil?", color = Th.text) },
            text = { Text("Tüm verilerin kalıcı olarak silinecek. Bu işlem geri alınamaz.", color = Th.textDim) },
            confirmButton = {
                TextButton(onClick = { confirmDelete = false; auth.deleteAccount() }) {
                    Text("Hesabı Sil", color = Th.down, fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) { Text("Vazgeç", color = Th.textDim) }
            },
        )
    }
}

@Composable
private fun LegalRow(title: String, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, color = Th.text, fontSize = 14.sp, modifier = Modifier.weight(1f))
        Text("↗", color = Th.textFaint, fontSize = 14.sp)
    }
}

@Composable
private fun ProfileCard(auth: AuthStore) {
    val context = LocalContext.current
    var name by remember { mutableStateOf(auth.displayName) }
    var savedName by remember { mutableStateOf(false) }

    val picker = rememberLauncherForActivityResult(
        ActivityResultContracts.PickVisualMedia()
    ) { uri ->
        if (uri != null) {
            val bytes = downscaleFromUri(context, uri)
            if (bytes != null) auth.uploadAvatar(bytes)
        }
    }

    Card {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("PROFİL", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            Row(verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Box(
                    Modifier.size(56.dp).clip(CircleShape).background(Th.accent.copy(alpha = 0.18f)),
                    contentAlignment = Alignment.Center,
                ) {
                    if (auth.avatarUrl.isNotEmpty()) {
                        AsyncImage(
                            model = auth.avatarUrl, contentDescription = null,
                            contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize(),
                        )
                    } else {
                        Text((auth.displayName.ifEmpty { auth.email }).take(1).uppercase(),
                            color = Th.accent, fontWeight = FontWeight.Bold, fontSize = 22.sp)
                    }
                }
                Column {
                    Text(auth.displayName.ifEmpty { auth.email.ifEmpty { "Kullanıcı" } },
                        color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
                    if (auth.email.isNotEmpty())
                        Text(auth.email, color = Th.textDim, fontSize = 12.sp)
                    TextButton(
                        onClick = {
                            picker.launch(PickVisualMediaRequest(
                                ActivityResultContracts.PickVisualMedia.ImageOnly))
                        },
                        contentPadding = androidx.compose.foundation.layout.PaddingValues(0.dp),
                    ) { Text("Fotoğrafı değiştir", color = Th.accent, fontSize = 12.sp) }
                }
            }

            OutlinedTextField(
                value = name,
                onValueChange = { name = it; savedName = false },
                label = { Text("Görünen ad") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Button(
                onClick = { auth.updateDisplayName(name.trim()); savedName = true },
                enabled = name.isNotBlank(),
                shape = RoundedCornerShape(10.dp),
                colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
            ) { Text(if (savedName) "Kaydedildi ✓" else "Kaydet", fontWeight = FontWeight.Bold) }
        }
    }
}

/** Reads an image Uri, downscales the longest side to ~512px, returns JPEG bytes. */
private fun downscaleFromUri(context: android.content.Context, uri: android.net.Uri): ByteArray? {
    return try {
        val input = context.contentResolver.openInputStream(uri) ?: return null
        val original = input.use { BitmapFactory.decodeStream(it) } ?: return null
        val maxSide = 512
        val scale = minOf(1f, maxSide.toFloat() / maxOf(original.width, original.height))
        val scaled = if (scale < 1f) {
            Bitmap.createScaledBitmap(original,
                (original.width * scale).toInt(), (original.height * scale).toInt(), true)
        } else original
        ByteArrayOutputStream().use { out ->
            scaled.compress(Bitmap.CompressFormat.JPEG, 85, out)
            out.toByteArray()
        }
    } catch (e: Exception) {
        null
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
