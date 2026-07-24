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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
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
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.example.tsc.forex.AppConfig
import com.example.tsc.forex.BuildConfig
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.BillingStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.ToastOnChange
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

@Composable
fun SettingsScreen(
    auth: AuthStore,
    store: PriceStore,
    billing: BillingStore,
    prefs: Prefs,
    onOpenArchive: () -> Unit,
) {
    val uriHandler = LocalUriHandler.current
    val scope = rememberCoroutineScope()
    var confirmDelete by remember { mutableStateOf(false) }
    var showPaywall by remember { mutableStateOf(false) }
    var showProfile by remember { mutableStateOf(false) }
    var acceptedAI by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        auth.loadProfile()
        billing.refreshNow()
    }
    LaunchedEffect(auth.userId) {
        acceptedAI = prefs.aiResearchConsent(auth.userId.orEmpty())
    }
    ToastOnChange(billing.message) { billing.message = null }
    ToastOnChange(billing.catalogError) { billing.catalogError = null }
    ToastOnChange(auth.errorMessage) { auth.errorMessage = null }

    Column(Modifier.fillMaxSize().background(Th.bg)) {
        ScreenHeader("Ayarlar", "Hesap & uygulama")
        Column(
            Modifier
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = 100.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Card {
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Text("HESAP", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(10.dp))
                            .clickable { showProfile = true }
                            .padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        Box(
                            Modifier.size(44.dp).clip(CircleShape).background(Th.accent.copy(alpha = 0.18f)),
                            contentAlignment = Alignment.Center,
                        ) {
                            if (auth.avatarUrl.isNotEmpty()) {
                                AsyncImage(
                                    model = auth.avatarUrl,
                                    contentDescription = null,
                                    contentScale = ContentScale.Crop,
                                    modifier = Modifier.fillMaxSize(),
                                )
                            } else {
                                Text(
                                    (auth.displayName.ifEmpty { auth.email }).take(1).uppercase(),
                                    color = Th.accent,
                                    fontWeight = FontWeight.Bold,
                                    fontSize = 18.sp,
                                )
                            }
                        }
                        Column(Modifier.weight(1f)) {
                            Text(
                                auth.displayName.ifEmpty { auth.email.ifEmpty { "Kullanıcı" } },
                                color = Th.text,
                                fontSize = 15.sp,
                                fontWeight = FontWeight.SemiBold,
                            )
                            Text("Profili düzenle", color = Th.textDim, fontSize = 12.sp)
                        }
                        Text("›", color = Th.textFaint, fontSize = 18.sp)
                    }
                    Button(
                        onClick = { billing.clearSession(); auth.logout() },
                        colors = ButtonDefaults.buttonColors(
                            containerColor = Th.down.copy(alpha = 0.16f),
                            contentColor = Th.down,
                        ),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Çıkış Yap", fontWeight = FontWeight.Bold) }
                }
            }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("ABONELİK", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                if (billing.hasActiveSubscription) "Abonelik etkin" else "Ücretsiz plan",
                                color = Th.text,
                                fontSize = 15.sp,
                                fontWeight = FontWeight.SemiBold,
                            )
                            Text(
                                billing.serverStatus?.plan?.name ?: "AI araştırma kullanımını planına göre yönet",
                                color = Th.textDim,
                                fontSize = 11.sp,
                            )
                        }
                        billing.aiRemaining?.let {
                            Text("$it hak", color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        }
                    }
                    Button(
                        onClick = { billing.message = null; showPaywall = true },
                        colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(
                            if (billing.hasActiveSubscription) "Planları Gör" else "Planları İncele",
                            fontWeight = FontWeight.Bold,
                        )
                    }
                    TextButton(onClick = {
                        val url = billing.serverStatus?.manageUrl?.ifBlank { null }
                            ?: "https://play.google.com/store/account/subscriptions"
                        runCatching { uriHandler.openUri(url) }
                    }) { Text("Aboneliği Yönet", color = Th.accent, fontWeight = FontWeight.SemiBold) }
                    TextButton(
                        onClick = { billing.restore() },
                        enabled = !billing.isLoading && !billing.purchasing,
                    ) { Text("Satın Almaları Geri Yükle", color = Th.accent, fontWeight = FontWeight.SemiBold) }
                    TextButton(onClick = {
                        scope.launch { billing.refreshNow(); billing.message = "Durum yenilendi." }
                    }) { Text("Durumu Yenile", color = Th.accent, fontWeight = FontWeight.SemiBold) }
                    TextButton(onClick = onOpenArchive) {
                        Text("Araştırma Arşivi  ›", color = Th.accent, fontWeight = FontWeight.SemiBold)
                    }
                }
            }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("AI VE VERİ KULLANIMI", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    Text(
                        "AI araştırmasında piyasa verileri, açık web kaynakları ve tek yönlü türetilen takma adlı güvenlik kimliği analiz ve kötüye kullanım önleme için OpenAI'a gönderilir; kişisel profil ve portföy tutarları gönderilmez.",
                        color = Th.textDim,
                        fontSize = 12.sp,
                    )
                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text("AI araştırma onayı", color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.SemiBold)
                            Text(
                                if (acceptedAI) "Onay verildi" else "Onay verilmedi",
                                color = Th.textDim,
                                fontSize = 11.sp,
                            )
                        }
                        if (acceptedAI) {
                            TextButton(onClick = {
                                prefs.setAiResearchConsent(auth.userId.orEmpty(), false)
                                acceptedAI = false
                            }) { Text("Onayı Geri Çek", color = Th.down, fontWeight = FontWeight.Bold) }
                        }
                    }
                }
            }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("YASAL", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    LegalRow("Gizlilik Politikası") { uriHandler.openUri("https://tsc.talhacan.com/privacy") }
                    LegalRow("Kullanım Şartları") { uriHandler.openUri("https://tsc.talhacan.com/terms") }
                    LegalRow("Destek & Yardım") { uriHandler.openUri("https://tsc.talhacan.com/support") }
                }
            }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("UYGULAMA", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    InfoRow("Sürüm", BuildConfig.VERSION_NAME)
                    InfoRow("İzlenen varlık", "${store.prices.size}")
                }
            }

            if (AppConfig.allowsServerOverride) {
                Card {
                    Text(
                        "Geliştirici sunucu ayarı yalnızca debug derlemesinde görünür.",
                        color = Th.textFaint,
                        fontSize = 11.sp,
                    )
                }
            }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("TEHLİKELİ BÖLGE", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    Text(
                        "Hesabını silersen izleme listen, alarmların, portföyün, bildirimlerin ve ödül verilerin kalıcı olarak kaldırılır. Aktif abonelik otomatik iptal edilmez.",
                        color = Th.textDim,
                        fontSize = 12.sp,
                    )
                    Button(
                        onClick = { confirmDelete = true },
                        enabled = !auth.isBusy,
                        colors = ButtonDefaults.buttonColors(
                            containerColor = Th.down.copy(alpha = 0.16f),
                            contentColor = Th.down,
                        ),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Hesabı Sil", fontWeight = FontWeight.Bold) }
                }
            }

            Text(
                "TSC Forex · Canlı Altın & Döviz",
                color = Th.textFaint,
                fontSize = 11.sp,
                modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
            )
        }
    }

    if (showPaywall) {
        PaywallSheet(billing = billing, onDismiss = { showPaywall = false })
    }
    if (showProfile) {
        ProfileSheet(auth = auth, onDismiss = { showProfile = false })
    }
    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            containerColor = Th.surface,
            title = { Text("Hesabını sil?", color = Th.text) },
            text = {
                Text(
                    "Tüm verilerin kalıcı olarak silinecek. Bu işlem geri alınamaz.",
                    color = Th.textDim,
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    billing.clearSession()
                    auth.deleteAccount()
                }) { Text("Hesabı Sil", color = Th.down, fontWeight = FontWeight.Bold) }
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
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, color = Th.text, fontSize = 14.sp, modifier = Modifier.weight(1f))
        Text("↗", color = Th.textFaint, fontSize = 12.sp)
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth()) {
        Text(label, color = Th.textDim, fontSize = 13.sp, modifier = Modifier.weight(1f))
        Text(value, color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.SemiBold)
    }
}
