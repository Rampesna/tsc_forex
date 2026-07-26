package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.ResearchArchiveItem
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.BillingStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.ToastOnChange
import com.example.tsc.forex.ui.theme.Th

@Composable
fun ResearchArchiveScreen(
    auth: AuthStore,
    billing: BillingStore,
    onBack: () -> Unit,
    onOpenPaywall: () -> Unit,
) {
    var items by remember { mutableStateOf<List<ResearchArchiveItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selected by remember { mutableStateOf<ResearchArchiveItem?>(null) }

    LaunchedEffect(Unit) {
        loading = true
        error = null
        runCatching { auth.newApiClient().researchHistory() }
            .onSuccess { items = it }
            .onFailure { error = it.message ?: "Arşiv yüklenemedi." }
        loading = false
    }
    ToastOnChange(error) { error = null }

    Column(Modifier.fillMaxSize()) {
        ScreenHeader(
            title = "Araştırma Arşivi",
            subtitle = "Paketinizdeki saklama süresi boyunca",
            leading = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri", tint = Th.text)
                }
            },
        )
        when {
            loading && items.isEmpty() -> {
                Text(
                    "Araştırmalar yükleniyor…",
                    color = Th.textDim,
                    modifier = Modifier.padding(20.dp),
                )
            }
            else -> {
                LazyColumn(
                    Modifier.fillMaxSize().padding(horizontal = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    if (items.isEmpty()) {
                        item {
                            Card {
                                Column(
                                    Modifier.fillMaxWidth().padding(vertical = 18.dp),
                                    horizontalAlignment = Alignment.CenterHorizontally,
                                    verticalArrangement = Arrangement.spacedBy(10.dp),
                                ) {
                                    Text("Arşiviniz henüz boş", color = Th.text, fontSize = 17.sp, fontWeight = FontWeight.Bold)
                                    Text(
                                        "Paketiniz etkinken oluşturduğunuz kaynaklı piyasa araştırmaları burada görünür.",
                                        color = Th.textDim,
                                        fontSize = 12.sp,
                                    )
                                    if (!billing.hasActiveSubscription) {
                                        Button(
                                            onClick = onOpenPaywall,
                                            colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
                                            shape = RoundedCornerShape(12.dp),
                                        ) { Text("Planları İncele", fontWeight = FontWeight.Bold) }
                                    }
                                }
                            }
                        }
                    } else {
                        items(items, key = { it.id }) { item ->
                            Card {
                                Column(
                                    Modifier
                                        .fillMaxWidth()
                                        .clickable { selected = item }
                                        .padding(4.dp),
                                    verticalArrangement = Arrangement.spacedBy(4.dp),
                                ) {
                                    Row {
                                        Text(item.code, color = Th.accent, fontSize = 10.sp, fontWeight = FontWeight.Bold)
                                        Spacer(Modifier.weight(1f))
                                        Text(item.generatedAt.take(16).replace('T', ' '), color = Th.textFaint, fontSize = 9.sp)
                                    }
                                    Text(item.payload.title, color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.Bold, maxLines = 2)
                                    Text(item.payload.summary, color = Th.textDim, fontSize = 11.sp, maxLines = 2)
                                    Text(
                                        "${item.expiresAt.take(10)} tarihine kadar arşivde",
                                        color = Th.textFaint,
                                        fontSize = 9.sp,
                                    )
                                }
                            }
                        }
                    }
                    item { Spacer(Modifier.padding(40.dp)) }
                }
            }
        }
    }

    selected?.let { item ->
        ResearchArchiveDetailSheet(item = item, onDismiss = { selected = null })
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ResearchArchiveDetailSheet(item: ResearchArchiveItem, onDismiss: () -> Unit) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val insight = item.payload
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState, containerColor = Th.bg) {
        Column(
            Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 20.dp)
                .padding(bottom = 36.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row {
                Text(item.code, color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                Spacer(Modifier.weight(1f))
                Text(insight.stanceLabel, color = Th.accent, fontSize = 10.sp, fontWeight = FontWeight.Bold)
            }
            Text(insight.title, color = Th.text, fontSize = 20.sp, fontWeight = FontWeight.Bold)
            Text(insight.summary, color = Th.textDim, fontSize = 13.sp)
            insight.factors.forEach { Text("• $it", color = Th.textDim, fontSize = 12.sp) }
            insight.whatIWouldDo?.let { playbook ->
                Text("BEN OLSAM NE YAPARDIM", color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                Text(playbook.intro, color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                Text("Kısa vade: ${playbook.shortTerm}", color = Th.textDim, fontSize = 11.sp)
                Text("Orta vade: ${playbook.mediumTerm}", color = Th.textDim, fontSize = 11.sp)
                Text("Uzun vade: ${playbook.longTerm}", color = Th.textDim, fontSize = 11.sp)
            }
            if (insight.riskNote.isNotBlank()) {
                Text("⚠ ${insight.riskNote}", color = Th.textDim, fontSize = 11.sp)
            }
            Text(insight.disclaimer, color = Th.textFaint, fontSize = 9.sp)
        }
    }
}
