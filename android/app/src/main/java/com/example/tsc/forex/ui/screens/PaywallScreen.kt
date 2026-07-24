package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.auth.findActivity
import com.example.tsc.forex.data.SubscriptionPlan
import com.example.tsc.forex.store.BillingStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

private const val PLAY_MANAGE_URL = "https://play.google.com/store/account/subscriptions"

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PaywallSheet(
    billing: BillingStore,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    val uri = LocalUriHandler.current
    val context = LocalContext.current

    LaunchedEffect(Unit) { billing.refreshNow() }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Th.bg,
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 20.dp)
                .padding(bottom = 32.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Text("AI Araştırma Planları", color = Th.text, fontSize = 22.sp, fontWeight = FontWeight.Bold)
            Text(
                "Daha fazla kaynaklı piyasa araştırması ve planınıza göre artan aylık kullanım hakkı.",
                color = Th.textDim,
                fontSize = 13.sp,
            )

            Card {
                Text(
                    "Ödeme Google Play üzerinden alınır. Aynı hesapla iOS’ta App Store aboneliğiniz varsa haklarınız burada da geçerlidir.",
                    color = Th.textDim,
                    fontSize = 11.sp,
                )
            }

            billing.message?.let { msg ->
                Card {
                    Row(Modifier.fillMaxWidth()) {
                        Text(msg, color = Th.textDim, fontSize = 11.sp, modifier = Modifier.weight(1f))
                        TextButton(onClick = { billing.message = null }) {
                            Text("Kapat", color = Th.accent, fontSize = 11.sp)
                        }
                    }
                }
            }

            when {
                billing.isLoading && billing.catalogPlans.isEmpty() -> {
                    Card {
                        Text("Planlar yükleniyor…", color = Th.textDim, fontSize = 12.sp)
                    }
                }
                billing.catalogError != null && billing.catalogPlans.isEmpty() -> {
                    Card {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(billing.catalogError!!, color = Th.down, fontSize = 12.sp)
                            TextButton(onClick = { scope.launch { billing.refreshNow() } }) {
                                Text("Tekrar Dene", color = Th.accent, fontWeight = FontWeight.Bold)
                            }
                        }
                    }
                }
                else -> {
                    billing.catalogPlans
                        .filter { it.code != "free" }
                        .forEach { plan ->
                            PlanCard(
                                plan = plan,
                                selected = billing.isCurrentPlan(plan.productId),
                                playPrice = billing.formattedPlayPrice(plan.productId),
                                busy = billing.purchasing || billing.isLoading,
                                onPurchase = {
                                    val activity = context.findActivity()
                                    if (activity == null) {
                                        billing.message = "Satın alma ekranı açılamadı."
                                    } else {
                                        billing.purchase(activity, plan.productId)
                                    }
                                },
                            )
                        }
                }
            }

            val manage = billing.serverStatus?.manageUrl?.ifBlank { null } ?: PLAY_MANAGE_URL
            Button(
                onClick = { runCatching { uri.openUri(manage) } },
                colors = ButtonDefaults.buttonColors(containerColor = Th.surfaceAlt, contentColor = Th.accent),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Aboneliği Yönet", fontWeight = FontWeight.Bold) }

            TextButton(
                onClick = { billing.restore() },
                enabled = !billing.isLoading && !billing.purchasing,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Satın Almaları Geri Yükle", color = Th.accent, fontWeight = FontWeight.Bold) }

            TextButton(
                onClick = { scope.launch { billing.refreshNow(); billing.message = "Abonelik durumu yenilendi." } },
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Durumu Yenile", color = Th.accent, fontWeight = FontWeight.Bold) }

            TextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) {
                Text("Kapat", color = Th.textDim)
            }
        }
    }
}

@Composable
private fun PlanCard(
    plan: SubscriptionPlan,
    selected: Boolean,
    playPrice: String?,
    busy: Boolean,
    onPurchase: () -> Unit,
) {
    val fallback = NumberFormat.getCurrencyInstance(Locale("tr", "TR")).format(plan.priceHintTry)
    val priceLabel = playPrice ?: "$fallback / ay"
    val limits = plan.limits
    Card {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(Modifier.fillMaxWidth()) {
                Column(Modifier.weight(1f)) {
                    Text(plan.name, color = Th.text, fontSize = 16.sp, fontWeight = FontWeight.Bold)
                    if (selected) Text("Etkin plan", color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                }
                Text(priceLabel, color = Th.accent, fontSize = 13.sp, fontWeight = FontWeight.Bold)
            }
            Column(
                Modifier
                    .fillMaxWidth()
                    .padding(10.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                QuotaRow("AI araştırması", "${limits.aiResearch}/ ay")
                QuotaRow("Aktif alarm", "${limits.activeAlerts}")
                QuotaRow("İzleme listesi", "${limits.watchlist}")
                QuotaRow("Portföy", "${limits.portfolios}")
                QuotaRow("Portföy sağlık analizi", "${limits.portfolioHealth}/ ay")
                QuotaRow("Dışa aktarma", "${limits.exports}/ ay")
                QuotaRow("Araştırma arşivi", "${limits.researchArchiveDays} gün")
            }
            Text(
                "1 aylık abonelik · otomatik yenilenir · Google Play",
                color = Th.textFaint,
                fontSize = 10.sp,
            )
            if (!selected) {
                Button(
                    onClick = onPurchase,
                    enabled = !busy,
                    colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(
                        if (busy) "İşleniyor…" else "Satın Al",
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
    }
}

@Composable
private fun QuotaRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth()) {
        Text(label, color = Th.textDim, fontSize = 10.sp, modifier = Modifier.weight(1f))
        Text(value, color = Th.text, fontSize = 10.sp, fontWeight = FontWeight.Bold)
    }
}
