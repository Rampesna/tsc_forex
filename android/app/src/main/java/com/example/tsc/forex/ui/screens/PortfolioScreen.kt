package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.OpenInNew
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.WorkspacePremium
import androidx.compose.material.icons.outlined.CreateNewFolder
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material.icons.outlined.FavoriteBorder
import androidx.compose.material.icons.outlined.IosShare
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.MoreHoriz
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.UnfoldMore
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import com.example.tsc.forex.ui.LocalizedText as Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.FileProvider
import com.example.tsc.forex.data.ApiException
import com.example.tsc.forex.data.Candle
import com.example.tsc.forex.data.PortfolioHealth
import com.example.tsc.forex.data.PortfolioPerformancePoint
import com.example.tsc.forex.data.PortfolioPosition
import com.example.tsc.forex.data.PortfolioPositionDetail
import com.example.tsc.forex.data.PortfolioTransaction
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.BillingStore
import com.example.tsc.forex.store.PortfolioStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.AssetBadge
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ChartMode
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.PriceChart
import com.example.tsc.forex.ui.PrimaryActionButton
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.SegmentedControl
import com.example.tsc.forex.ui.ToastOnChange
import com.example.tsc.forex.ui.theme.Th
import androidx.compose.ui.graphics.Color
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.io.File
import java.text.SimpleDateFormat
import java.time.Instant
import java.time.ZoneOffset
import java.util.Date
import java.util.Locale
import java.util.UUID
import android.content.Intent
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.rememberModalBottomSheetState
import kotlin.math.max
import kotlin.math.min

@Composable
fun PortfolioScreen(
    portfolio: PortfolioStore,
    store: PriceStore,
    auth: AuthStore,
    billing: BillingStore,
    prefs: Prefs,
    onPosition: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var showAdd by remember { mutableStateOf(false) }
    var showPaywall by remember { mutableStateOf(false) }
    var showCreatePortfolio by remember { mutableStateOf(false) }
    var showRenamePortfolio by remember { mutableStateOf(false) }
    var confirmDeletePortfolio by remember { mutableStateOf(false) }
    var portfolioNameInput by remember { mutableStateOf("") }
    var showPortfolioMenu by remember { mutableStateOf(false) }
    var showPortfolioPicker by remember { mutableStateOf(false) }
    var range by remember { mutableStateOf("1y") }
    var chartMetric by remember { mutableStateOf("value") }
    var portfolioHealth by remember { mutableStateOf<PortfolioHealth?>(null) }
    var premiumBusy by remember { mutableStateOf<String?>(null) }
    val overview = portfolio.overview

    LaunchedEffect(Unit) {
        portfolio.sync(store.prices, range)
        billing.refreshNow()
    }

    fun quotaText(feature: String): String {
        val q = billing.serverStatus?.usage?.get(feature) ?: return "Paket özelliği"
        return "Bu ay ${q.remaining}/${q.limit} kaldı"
    }

    fun handlePremiumError(e: Throwable) {
        when {
            e is ApiException && e.code == 402 -> showPaywall = true
            e is ApiException && e.code == 429 ->
                portfolio.errorMessage = "Çok sık istek gönderildi. Kısa bir süre bekleyip tekrar deneyin."
            e is ApiException && e.code == 409 && e.serverCode == "operation_in_progress" ->
                portfolio.errorMessage =
                    "İşlem halen tamamlanıyor. Biraz sonra tekrar deneyin; kullanım hakkınız yeniden düşülmeyecek."
            else -> portfolio.errorMessage = e.message ?: "İşlem başarısız."
        }
    }

    suspend fun runHealth() {
        val id = overview?.portfolio?.id ?: return
        val userId = auth.ensureUserId() ?: run {
            portfolio.errorMessage = "Hesap bilgileri doğrulanamadı. Profilinizi yenileyip tekrar deneyin."
            return
        }
        val op = "portfolio-health"
        val requestId = prefs.pendingRequestId(userId, op, id)
        premiumBusy = "health"
        try {
            portfolioHealth = auth.newApiClient().portfolioHealth(id, requestId)
            prefs.clearPendingRequestId(userId, op, id)
            billing.refreshNow()
        } catch (e: Exception) {
            if (e !is ApiException || !e.shouldRetainRequestId) {
                prefs.clearPendingRequestId(userId, op, id)
            }
            handlePremiumError(e)
        } finally {
            premiumBusy = null
        }
    }

    suspend fun runExport() {
        val id = overview?.portfolio?.id ?: return
        val userId = auth.ensureUserId() ?: run {
            portfolio.errorMessage = "Hesap bilgileri doğrulanamadı. Profilinizi yenileyip tekrar deneyin."
            return
        }
        val op = "portfolio-export"
        val requestId = prefs.pendingRequestId(userId, op, id)
        premiumBusy = "export"
        try {
            val bytes = auth.newApiClient().exportPortfolio(id, requestId)
            val date = SimpleDateFormat("yyyy-MM-dd", Locale.US).format(Date())
            val file = File(context.cacheDir, "portfoy-${id.take(8)}-$date.csv")
            file.writeBytes(bytes)
            prefs.clearPendingRequestId(userId, op, id)
            billing.refreshNow()
            val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
            val share = Intent(Intent.ACTION_SEND).apply {
                type = "text/csv"
                putExtra(Intent.EXTRA_STREAM, uri)
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
            context.startActivity(Intent.createChooser(share, "Portföy CSV"))
        } catch (e: Exception) {
            if (e !is ApiException || !e.shouldRetainRequestId) {
                prefs.clearPendingRequestId(userId, op, id)
            }
            handlePremiumError(e)
        } finally {
            premiumBusy = null
        }
    }

    Column(Modifier.fillMaxSize().background(Th.bg)) {
        ScreenHeader(
            title = "Portföy",
            trailing = {
                Box(
                    Modifier
                        .size(32.dp)
                        .clip(RoundedCornerShape(16.dp))
                        .background(Th.accent)
                        .clickable { showAdd = true },
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Filled.Add, "İşlem ekle", tint = Th.bg, modifier = Modifier.size(20.dp))
                }
            },
        )
        ToastOnChange(portfolio.errorMessage) { portfolio.errorMessage = null }

        if (portfolio.loading && overview == null) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text("Portföy senkronize ediliyor…", color = Th.textDim)
            }
            return@Column
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (portfolio.legacyPreview.isNotEmpty()) {
                item {
                    Card(Modifier.padding(horizontal = 16.dp)) {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text("Eski portföy bulundu", color = Th.text, fontWeight = FontWeight.Bold)
                            Text(
                                "${portfolio.legacyPreview.size} yerel pozisyon için içe aktarma onayı gerekiyor.",
                                color = Th.textDim,
                                fontSize = 12.sp,
                            )
                            portfolio.legacyPreview.take(8).forEach { holding ->
                                Row(Modifier.fillMaxWidth()) {
                                    Text(holding.code, color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                                    Spacer(Modifier.weight(1f))
                                    Text("${Fmt.number(holding.amount)} adet", color = Th.textDim, fontSize = 11.sp)
                                }
                            }
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Button(
                                    onClick = { scope.launch { portfolio.confirmLegacyImport(store.prices) } },
                                    colors = ButtonDefaults.buttonColors(containerColor = Th.accent),
                                ) { Text("İçe aktar") }
                                TextButton(onClick = portfolio::skipLegacyImport) { Text("Atla", color = Th.textDim) }
                            }
                        }
                    }
                }
            }
            item {
                Card(Modifier.padding(horizontal = 16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text("AKTİF PORTFÖY", color = Th.textFaint, fontSize = 9.sp, fontWeight = FontWeight.Bold)
                            Row(
                                Modifier.clickable { showPortfolioPicker = true },
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(6.dp),
                            ) {
                                Text(
                                    overview?.portfolio?.name ?: "Portföy seç",
                                    color = Th.text,
                                    fontSize = 16.sp,
                                    fontWeight = FontWeight.Bold,
                                )
                                Icon(
                                    Icons.Outlined.UnfoldMore,
                                    contentDescription = null,
                                    tint = Th.accent,
                                    modifier = Modifier.size(18.dp),
                                )
                            }
                        }
                        IconButton(onClick = {
                            portfolioNameInput = ""
                            showCreatePortfolio = true
                        }) {
                            Icon(
                                Icons.Outlined.CreateNewFolder,
                                contentDescription = "Yeni portföy",
                                tint = Th.accent,
                                modifier = Modifier.size(22.dp),
                            )
                        }
                        Box {
                            IconButton(onClick = { showPortfolioMenu = true }) {
                                Icon(
                                    Icons.Outlined.MoreHoriz,
                                    contentDescription = "Portföy menüsü",
                                    tint = Th.textDim,
                                    modifier = Modifier.size(22.dp),
                                )
                            }
                            DropdownMenu(expanded = showPortfolioMenu, onDismissRequest = { showPortfolioMenu = false }) {
                                DropdownMenuItem(
                                    text = { Text("Yeniden Adlandır") },
                                    leadingIcon = {
                                        Icon(Icons.Outlined.Edit, null, tint = Th.textDim, modifier = Modifier.size(18.dp))
                                    },
                                    onClick = {
                                        showPortfolioMenu = false
                                        portfolioNameInput = overview?.portfolio?.name.orEmpty()
                                        showRenamePortfolio = true
                                    },
                                )
                                DropdownMenuItem(
                                    text = { Text("Portföyü Sil", color = Th.down) },
                                    leadingIcon = {
                                        Icon(Icons.Outlined.Delete, null, tint = Th.down, modifier = Modifier.size(18.dp))
                                    },
                                    enabled = portfolio.portfolios.size > 1,
                                    onClick = {
                                        showPortfolioMenu = false
                                        confirmDeletePortfolio = true
                                    },
                                )
                            }
                        }
                    }
                }
            }
            item { SummaryCard(overview) }
            item {
                Card(Modifier.padding(horizontal = 16.dp)) {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text("Portföy Araçları", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                                Text(
                                    "Dağılımı değerlendir veya işlem defterini paylaşılabilir CSV olarak al.",
                                    color = Th.textDim,
                                    fontSize = 11.sp,
                                )
                            }
                            Icon(
                                Icons.Filled.WorkspacePremium,
                                contentDescription = null,
                                tint = Th.accent,
                                modifier = Modifier.size(20.dp),
                            )
                        }
                        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                            Button(
                                onClick = { scope.launch { runHealth() } },
                                enabled = premiumBusy == null,
                                colors = ButtonDefaults.buttonColors(containerColor = Th.surfaceAlt, contentColor = Th.text),
                                modifier = Modifier.weight(1f),
                                shape = RoundedCornerShape(11.dp),
                            ) {
                                Column(horizontalAlignment = Alignment.Start) {
                                    Icon(
                                        Icons.Outlined.FavoriteBorder,
                                        contentDescription = null,
                                        tint = Th.accent,
                                        modifier = Modifier.size(18.dp).padding(bottom = 4.dp),
                                    )
                                    Text(
                                        if (premiumBusy == "health") "Analiz…" else "Sağlık Analizi",
                                        fontWeight = FontWeight.Bold,
                                        fontSize = 12.sp,
                                    )
                                    Text(quotaText("portfolio_health"), color = Th.textFaint, fontSize = 9.sp)
                                }
                            }
                            Button(
                                onClick = { scope.launch { runExport() } },
                                enabled = premiumBusy == null,
                                colors = ButtonDefaults.buttonColors(containerColor = Th.surfaceAlt, contentColor = Th.text),
                                modifier = Modifier.weight(1f),
                                shape = RoundedCornerShape(11.dp),
                            ) {
                                Column(horizontalAlignment = Alignment.Start) {
                                    Icon(
                                        Icons.Outlined.IosShare,
                                        contentDescription = null,
                                        tint = Th.accent,
                                        modifier = Modifier.size(18.dp).padding(bottom = 4.dp),
                                    )
                                    Text(
                                        if (premiumBusy == "export") "Aktarılıyor…" else "CSV Dışa Aktar",
                                        fontWeight = FontWeight.Bold,
                                        fontSize = 12.sp,
                                    )
                                    Text(quotaText("exports"), color = Th.textFaint, fontSize = 9.sp)
                                }
                            }
                        }
                    }
                }
            }
            portfolioHealth?.let { health ->
                item {
                    Card(Modifier.padding(horizontal = 16.dp)) {
                        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                Column(Modifier.weight(1f)) {
                                    Text("Portföy Sağlığı", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                                    Text(health.generatedAt.take(16).replace('T', ' '), color = Th.textFaint, fontSize = 9.sp)
                                }
                                Column(horizontalAlignment = Alignment.End) {
                                    val color = when {
                                        health.score >= 85 -> Th.up
                                        health.score >= 60 -> Th.accent
                                        else -> Th.down
                                    }
                                    Text("${health.score}", color = color, fontSize = 24.sp, fontWeight = FontWeight.Bold)
                                    Text(health.grade, color = color, fontSize = 9.sp, fontWeight = FontWeight.Bold)
                                }
                            }
                            Text(health.summary, color = Th.textDim, fontSize = 12.sp)
                            health.findings.forEach { finding ->
                                val sev = when (finding.severity) {
                                    "high" -> Th.down
                                    "medium" -> Th.accent
                                    else -> Th.up
                                }
                                Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                    Text("● ${finding.title}", color = sev, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                                    Text(finding.detail, color = Th.textDim, fontSize = 10.sp)
                                }
                            }
                            Text(
                                "Bu değerlendirme eğitim ve risk farkındalığı içindir; yatırım tavsiyesi değildir.",
                                color = Th.textFaint,
                                fontSize = 9.sp,
                            )
                        }
                    }
                }
            }
            item {
                PerformanceCard(
                    points = portfolio.performance,
                    range = range,
                    metric = chartMetric,
                    onMetric = { chartMetric = it },
                    onRange = {
                        range = it
                        scope.launch { portfolio.refreshPerformance(it) }
                    },
                )
            }
            if (portfolio.activePositions.isEmpty()) {
                item {
                    Card(Modifier.padding(horizontal = 16.dp)) {
                        Column(
                            Modifier.fillMaxWidth().padding(vertical = 20.dp),
                            horizontalAlignment = Alignment.CenterHorizontally,
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            Text("Portföyün boş", color = Th.text, fontSize = 18.sp, fontWeight = FontWeight.Bold)
                            Text("İlk alış işlemini ekleyerek başlayın.", color = Th.textDim, fontSize = 12.sp)
                            Button(
                                onClick = { showAdd = true },
                                colors = ButtonDefaults.buttonColors(containerColor = Th.accent),
                            ) { Text("İşlem ekle") }
                        }
                    }
                }
            } else {
                item { AllocationCard(portfolio.activePositions) }
                item {
                    Text(
                        "Pozisyonlar",
                        color = Th.text,
                        fontSize = 15.sp,
                        fontWeight = FontWeight.Bold,
                        modifier = Modifier.padding(horizontal = 16.dp),
                    )
                }
                items(portfolio.activePositions, key = { it.id }) { position ->
                    PositionRow(
                        position,
                        onClick = { onPosition(position.code) },
                        modifier = Modifier.padding(horizontal = 16.dp),
                    )
                }
            }
            item { Spacer(Modifier.height(90.dp)) }
        }
    }

    if (showAdd) {
        PortfolioTransactionDialog(
            store = store,
            initialCode = null,
            transaction = null,
            onDismiss = { showAdd = false },
            onSave = { body ->
                scope.launch {
                    if (portfolio.saveTransaction(body, store.prices)) showAdd = false
                }
            },
        )
    }
    if (showPaywall) {
        PaywallSheet(billing = billing, onDismiss = { showPaywall = false })
    }
    if (showPortfolioPicker) {
        AlertDialog(
            onDismissRequest = { showPortfolioPicker = false },
            containerColor = Th.surface,
            title = { Text("Portföy Seç", color = Th.text) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    portfolio.portfolios.forEach { item ->
                        val selected = item.portfolio.id == overview?.portfolio?.id
                        Text(
                            item.portfolio.name,
                            color = if (selected) Th.accent else Th.text,
                            fontWeight = if (selected) FontWeight.Bold else FontWeight.SemiBold,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clip(RoundedCornerShape(10.dp))
                                .background(if (selected) Th.accent.copy(alpha = 0.12f) else Th.surfaceAlt)
                                .clickable {
                                    showPortfolioPicker = false
                                    portfolioHealth = null
                                    scope.launch { portfolio.selectPortfolio(item.portfolio.id, store.prices, range) }
                                }
                                .padding(12.dp),
                        )
                    }
                }
            },
            confirmButton = {},
            dismissButton = { TextButton(onClick = { showPortfolioPicker = false }) { Text("Vazgeç", color = Th.textDim) } },
        )
    }
    if (showCreatePortfolio || showRenamePortfolio) {
        AlertDialog(
            onDismissRequest = { showCreatePortfolio = false; showRenamePortfolio = false },
            containerColor = Th.surface,
            title = {
                Text(
                    if (showCreatePortfolio) "Yeni Portföy" else "Portföyü Yeniden Adlandır",
                    color = Th.text,
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    if (showCreatePortfolio) {
                        Text(
                            "Planınızdaki portföy hakkı kapsamında ayrı bir takip alanı oluşturun.",
                            color = Th.textDim,
                            fontSize = 12.sp,
                        )
                    }
                    OutlinedTextField(
                        value = portfolioNameInput,
                        onValueChange = { portfolioNameInput = it },
                        label = { Text("Portföy adı") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        val ok = if (showCreatePortfolio) {
                            portfolio.createPortfolio(portfolioNameInput, store.prices, range)
                        } else {
                            portfolio.renameCurrentPortfolio(portfolioNameInput)
                        }
                        if (ok) {
                            showCreatePortfolio = false
                            showRenamePortfolio = false
                            portfolioNameInput = ""
                        }
                    }
                }) { Text(if (showCreatePortfolio) "Oluştur" else "Kaydet", color = Th.accent, fontWeight = FontWeight.Bold) }
            },
            dismissButton = {
                TextButton(onClick = { showCreatePortfolio = false; showRenamePortfolio = false }) {
                    Text("Vazgeç", color = Th.textDim)
                }
            },
        )
    }
    if (confirmDeletePortfolio) {
        AlertDialog(
            onDismissRequest = { confirmDeletePortfolio = false },
            containerColor = Th.surface,
            title = { Text("Portföy silinsin mi?", color = Th.text) },
            text = {
                Text(
                    "Bu portföydeki tüm işlem ve performans geçmişi kalıcı olarak silinir.",
                    color = Th.textDim,
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmDeletePortfolio = false
                    scope.launch {
                        portfolioHealth = null
                        portfolio.deleteCurrentPortfolio(store.prices, range)
                    }
                }) { Text("Kalıcı Olarak Sil", color = Th.down, fontWeight = FontWeight.Bold) }
            },
            dismissButton = {
                TextButton(onClick = { confirmDeletePortfolio = false }) { Text("Vazgeç", color = Th.textDim) }
            },
        )
    }
}

@Composable
private fun SummaryCard(overview: com.example.tsc.forex.data.PortfolioOverview?) {
    Card(Modifier.padding(horizontal = 16.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("Toplam Değer", color = Th.textDim, fontSize = 12.sp)
            Text(
                Fmt.tl(overview?.totalValueTry ?: 0.0),
                color = Th.text,
                fontSize = 29.sp,
                fontWeight = FontWeight.Bold,
            )
            BoxWithConstraints(Modifier.fillMaxWidth()) {
                if (maxWidth < 380.dp) {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        SummaryMetricRow("Bugün", overview?.dayChangeTry ?: 0.0, overview?.dayChangePct)
                        SummaryMetricRow("Gerçekleşmemiş K/Z", overview?.unrealizedPnlTry ?: 0.0)
                        SummaryMetricRow("Gerçekleşen K/Z", overview?.realizedPnlTry ?: 0.0)
                    }
                } else {
                    Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                        SummaryMetric("Bugün", overview?.dayChangeTry ?: 0.0, overview?.dayChangePct)
                        SummaryMetric("Gerçekleşmemiş", overview?.unrealizedPnlTry ?: 0.0)
                        SummaryMetric("Gerçekleşen", overview?.realizedPnlTry ?: 0.0)
                    }
                }
            }
        }
    }
}

@Composable
private fun SummaryMetricRow(label: String, value: Double, pct: Double? = null) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(label, color = Th.textDim, fontSize = 11.sp, modifier = Modifier.weight(1f))
        Text(
            buildString {
                append(Fmt.signed(value))
                append(" ₺")
                if (pct != null) append("  (${Fmt.signedPct(pct)})")
            },
            color = Th.trend(value),
            fontSize = 12.sp,
            fontWeight = FontWeight.SemiBold,
            textAlign = TextAlign.End,
        )
    }
}

@Composable
private fun RowScope.SummaryMetric(label: String, value: Double, pct: Double? = null) {
    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
        Text(label, color = Th.textDim, fontSize = 10.sp, maxLines = 1)
        Text("${Fmt.signed(value)} ₺", color = Th.trend(value), fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
        if (pct != null) Text(Fmt.signedPct(pct), color = Th.trend(value), fontSize = 9.sp)
    }
}

private val PerformanceValueColor = Th.accent
private val PerformanceCostColor = Color(0xFFF3C43D)
private val PerformancePnlColor = Color(0xFFE559B7)

@Composable
private fun PerformanceCard(
    points: List<PortfolioPerformancePoint>,
    range: String,
    metric: String,
    onMetric: (String) -> Unit,
    onRange: (String) -> Unit,
) {
    val firstActive = points.indexOfFirst { it.valueTry != 0.0 || it.costBasisTry != 0.0 }
    val plotted = if (firstActive > 0) points.drop(firstActive) else points
    var selectedEpochMs by remember { mutableStateOf<Long?>(null) }
    var chartReset by remember { mutableIntStateOf(0) }

    LaunchedEffect(metric, range, plotted.size) {
        selectedEpochMs = null
        chartReset += 1
    }

    val candles = remember(plotted, metric) {
        performanceCandles(plotted) {
            when (metric) {
                "pnl" -> it.totalPnlTry
                "cost" -> it.costBasisTry
                else -> it.valueTry
            }
        }
    }
    val seriesTitle = when (metric) {
        "pnl" -> "Toplam K/Z"
        "cost" -> "Maliyet"
        else -> "Değer"
    }
    val seriesColor = when (metric) {
        "pnl" -> PerformancePnlColor
        "cost" -> PerformanceCostColor
        else -> PerformanceValueColor
    }
    val trendUp = remember(candles) {
        val first = candles.firstOrNull()?.close ?: return@remember true
        val last = candles.lastOrNull()?.close ?: return@remember true
        last >= first
    }
    val selectedPoint = remember(selectedEpochMs, plotted) {
        val ms = selectedEpochMs ?: return@remember null
        val day = Instant.ofEpochMilli(ms).atZone(ZoneOffset.UTC).toLocalDate().toString()
        plotted.firstOrNull { it.date.take(10) == day }
    }
    val snapshot = selectedPoint ?: plotted.lastOrNull()

    Card(Modifier.padding(horizontal = 16.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("Portföy Performansı", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
            Text("Varlıkların gerçek günlük fiyat geçmişine göre", color = Th.textDim, fontSize = 11.sp)
            val metricOptions = listOf("Değer", "Toplam K/Z", "Maliyet")
            val metricIDs = listOf("value", "pnl", "cost")
            SegmentedControl(
                options = metricOptions,
                selectedIndex = metricIDs.indexOf(metric).coerceAtLeast(0),
                onSelected = {
                    onMetric(metricIDs[it])
                    selectedEpochMs = null
                    chartReset += 1
                },
            )
            if (snapshot != null) {
                Column(
                    Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(12.dp))
                        .background(Th.surfaceAlt)
                        .padding(12.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    val label = if (selectedPoint == null) "SON DURUM" else "SEÇİLİ GÜN"
                    Text(
                        "$label · ${snapshot.date.take(10)}",
                        color = Th.textFaint,
                        fontSize = 10.sp,
                        fontWeight = FontWeight.Bold,
                    )
                    Row(Modifier.fillMaxWidth()) {
                        Text("Değer", color = PerformanceValueColor, fontSize = 11.sp, modifier = Modifier.weight(1f))
                        Text(Fmt.tl(snapshot.valueTry), color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                    }
                    Row(Modifier.fillMaxWidth()) {
                        Text("Maliyet", color = PerformanceCostColor, fontSize = 11.sp, modifier = Modifier.weight(1f))
                        Text(Fmt.tl(snapshot.costBasisTry), color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                    }
                    Row(Modifier.fillMaxWidth()) {
                        Text("Toplam K/Z", color = PerformancePnlColor, fontSize = 11.sp, modifier = Modifier.weight(1f))
                        Text(
                            Fmt.tl(snapshot.totalPnlTry),
                            color = Th.trend(snapshot.totalPnlTry),
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                }
            }
            BoxWithConstraints(Modifier.fillMaxWidth()) {
                PriceChart(
                    candles = candles,
                    mode = ChartMode.LINE,
                    trendUp = trendUp,
                    currency = "TRY",
                    resetToken = chartReset,
                    heightDp = if (maxWidth < 360.dp) 190 else 220,
                    seriesTitle = seriesTitle,
                    seriesColor = seriesColor,
                    showsMovingAverages = false,
                    showsVolume = false,
                    showsOHLCSelection = false,
                    stickySelection = true,
                    emptyTitle = "Henüz yeterli performans verisi yok",
                    emptySubtitle = "İşlem ekledikçe geçmiş portföy değeri burada oluşur.",
                    interactionHint = "Sürükle · Yakınlaştır · Dokun: gün detayı",
                    onSelectionChange = { selectedEpochMs = it },
                )
            }
            val rangeIDs = listOf("1m", "3m", "1y", "5y")
            SegmentedControl(
                options = listOf("1 AY", "3 AY", "1 YIL", "5 YIL"),
                selectedIndex = rangeIDs.indexOf(range).coerceAtLeast(0),
                onSelected = {
                    onRange(rangeIDs[it])
                    selectedEpochMs = null
                    chartReset += 1
                },
            )
        }
    }
}

private fun performanceCandles(
    points: List<PortfolioPerformancePoint>,
    valueOf: (PortfolioPerformancePoint) -> Double,
): List<Candle> = points.mapIndexedNotNull { index, point ->
    val day = point.date.take(10)
    if (!day.matches(Regex("""\d{4}-\d{2}-\d{2}"""))) return@mapIndexedNotNull null
    val close = valueOf(point)
    val open = if (index > 0) valueOf(points[index - 1]) else close
    Candle(
        time = "${day}T00:00:00Z",
        open = open,
        high = max(open, close),
        low = min(open, close),
        close = close,
        count = 0,
    )
}

@Composable
private fun AllocationCard(positions: List<PortfolioPosition>) {
    val colors = listOf(
        Th.accent,
        Color(0xFF6C8CFF),
        Color(0xFFA873E8),
        Color(0xFFF3B84B),
        Color(0xFF43C59E),
        Color(0xFFF06A78),
        Color(0xFF57A7D8),
        Color(0xFF9BA7B8),
    )
    val ordered = positions.filter { it.allocationPct > 0 }.sortedByDescending { it.allocationPct }
    val shown = ordered.take(7)
    val slices = shown.mapIndexed { index, position ->
        AllocationSlice(position.id, position.code, position.allocationPct, colors[index])
    }.toMutableList()
    val remainder = ordered.drop(shown.size).sumOf { it.allocationPct }
    if (remainder > 0.01) {
        slices += AllocationSlice("other", "Diğer", remainder, colors.last())
    }

    Card(Modifier.padding(horizontal = 16.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("Dağılım", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
            BoxWithConstraints(Modifier.fillMaxWidth()) {
                if (maxWidth >= 350.dp) {
                    Row(
                        Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(18.dp),
                    ) {
                        AllocationDonut(slices, positions.size)
                        AllocationLegend(slices, Modifier.weight(1f))
                    }
                } else {
                    Column(
                        Modifier.fillMaxWidth(),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(16.dp),
                    ) {
                        AllocationDonut(slices, positions.size)
                        AllocationLegend(slices, Modifier.fillMaxWidth())
                    }
                }
            }
        }
    }
}

private data class AllocationSlice(
    val id: String,
    val label: String,
    val percentage: Double,
    val color: Color,
)

@Composable
private fun AllocationDonut(slices: List<AllocationSlice>, assetCount: Int) {
    val total = slices.sumOf { it.percentage }.coerceAtLeast(0.0001)
    Box(Modifier.size(146.dp), contentAlignment = Alignment.Center) {
        Canvas(Modifier.fillMaxSize()) {
            val stroke = 22.dp.toPx()
            drawCircle(color = Th.surfaceAlt, style = Stroke(width = stroke))
            var start = -90f
            slices.forEach { slice ->
                val sweep = ((slice.percentage / total) * 360.0).toFloat()
                drawArc(
                    color = slice.color,
                    startAngle = start,
                    sweepAngle = (sweep - 0.8f).coerceAtLeast(0.2f),
                    useCenter = false,
                    style = Stroke(width = stroke),
                )
                start += sweep
            }
        }
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text("$assetCount", color = Th.text, fontSize = 22.sp, fontWeight = FontWeight.Bold)
            Text("varlık", color = Th.textDim, fontSize = 10.sp, fontWeight = FontWeight.SemiBold)
        }
    }
}

@Composable
private fun AllocationLegend(slices: List<AllocationSlice>, modifier: Modifier = Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(9.dp)) {
        slices.forEach { slice ->
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.size(9.dp).clip(androidx.compose.foundation.shape.CircleShape).background(slice.color))
                Text(
                    slice.label,
                    color = Th.text,
                    fontSize = 11.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    modifier = Modifier.padding(horizontal = 8.dp).weight(1f),
                )
                Text(Fmt.pctUnsigned(slice.percentage), color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.SemiBold)
            }
        }
    }
}

@Composable
private fun PositionRow(position: PortfolioPosition, onClick: () -> Unit, modifier: Modifier = Modifier) {
    Card(modifier.clickable(onClick = onClick)) {
        BoxWithConstraints(Modifier.fillMaxWidth()) {
            val compact = maxWidth < 360.dp
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                AssetBadge(position.code, position.kind == "gold")
                Column(
                    Modifier
                        .weight(1f)
                        .padding(start = 12.dp, end = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(3.dp),
                ) {
                    Text(
                        position.displayName,
                        color = Th.text,
                        fontSize = 14.sp,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                    )
                    Text(
                        if (compact) {
                            "${Fmt.number(position.quantity)} adet"
                        } else {
                            "${Fmt.number(position.quantity)} adet • Ort. ${Fmt.money(position.averageCostNative, position.currency)}"
                        },
                        color = Th.textDim,
                        fontSize = 10.sp,
                        maxLines = 1,
                        overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                    )
                }
                Column(horizontalAlignment = Alignment.End) {
                    Text(Fmt.tl(position.valueTry), color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.Bold)
                    Text(
                        Fmt.signedPct(position.unrealizedPnlPct),
                        color = Th.trend(position.unrealizedPnlTry),
                        fontSize = 10.sp,
                    )
                }
            }
        }
    }
}

@Composable
fun PortfolioPositionScreen(
    code: String,
    portfolio: PortfolioStore,
    store: PriceStore,
    onBack: () -> Unit,
    onAsset: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    var detail by remember { mutableStateOf<PortfolioPositionDetail?>(null) }
    var edit by remember { mutableStateOf<PortfolioTransaction?>(null) }
    var showAdd by remember { mutableStateOf(false) }
    LaunchedEffect(code, portfolio.overview?.portfolio?.version) {
        detail = portfolio.positionDetail(code)
    }
    val position = detail?.position ?: portfolio.activePositions.firstOrNull { it.code == code }

    Column(Modifier.fillMaxSize().background(Th.bg)) {
        Box(
            Modifier
                .fillMaxWidth()
                .statusBarsPadding()
                .padding(horizontal = 8.dp, vertical = 8.dp),
        ) {
            IconButton(onClick = onBack, modifier = Modifier.align(Alignment.CenterStart)) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri", tint = Th.text)
            }
            Text(
                code,
                color = Th.text,
                style = MaterialTheme.typography.titleLarge,
                modifier = Modifier.align(Alignment.Center),
            )
            IconButton(onClick = { showAdd = true }, modifier = Modifier.align(Alignment.CenterEnd)) {
                Icon(Icons.Filled.Add, "İşlem ekle", tint = Th.accent)
            }
        }
        if (position == null) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text("Pozisyon yükleniyor…", color = Th.textDim)
            }
        } else {
            LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                item {
                    Card(Modifier.padding(horizontal = 16.dp)) {
                        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                            Text(position.displayName, color = Th.text, fontSize = 19.sp, fontWeight = FontWeight.Bold)
                            Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                                PositionMetric("Miktar", Fmt.number(position.quantity))
                                PositionMetric("Portföy Payı", "%${"%.1f".format(position.allocationPct)}")
                            }
                            Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                                PositionMetric("Ortalama Maliyet", Fmt.money(position.averageCostNative, position.currency))
                                PositionMetric("Native Değer", Fmt.money(position.valueNative, position.currency))
                            }
                            Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                                PositionMetric("TRY Değeri", Fmt.tl(position.valueTry))
                                PositionMetric(
                                    "Günlük K/Z",
                                    "${Fmt.signed(position.dayChangeTry)} ₺ (${Fmt.signedPct(position.dayChangePct)})",
                                    position.dayChangeTry,
                                )
                            }
                            Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                                PositionMetric(
                                    "Toplam K/Z",
                                    "${Fmt.signed(position.unrealizedPnlTry)} ₺ (${Fmt.signedPct(position.unrealizedPnlPct)})",
                                    position.unrealizedPnlTry,
                                )
                                Spacer(Modifier.weight(1f))
                            }
                            Button(
                                onClick = { onAsset(code) },
                                colors = ButtonDefaults.buttonColors(containerColor = Th.surfaceAlt),
                            ) {
                                Icon(Icons.AutoMirrored.Filled.OpenInNew, null, Modifier.size(16.dp))
                                Text(" Varlık detayı ve grafik")
                            }
                        }
                    }
                }
                item {
                    Text(
                        "İşlem Geçmişi",
                        color = Th.text,
                        fontSize = 15.sp,
                        fontWeight = FontWeight.Bold,
                        modifier = Modifier.padding(horizontal = 16.dp),
                    )
                }
                items(detail?.transactions ?: emptyList(), key = { it.id }) { transaction ->
                    Card(Modifier.padding(horizontal = 16.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text(typeTitle(transaction.type), color = Th.text, fontWeight = FontWeight.SemiBold)
                                Text(transaction.occurredAt.take(10), color = Th.textDim, fontSize = 10.sp)
                            }
                            Text(
                                if (transaction.quantity > 0) "${Fmt.number(transaction.quantity)} adet"
                                else Fmt.money(transaction.amountNative, transaction.currency),
                                color = Th.text,
                                fontSize = 12.sp,
                            )
                            IconButton(onClick = { edit = transaction }) {
                                Icon(Icons.Filled.Edit, "Düzenle", tint = Th.textDim, modifier = Modifier.size(18.dp))
                            }
                            IconButton(onClick = {
                                scope.launch {
                                    portfolio.deleteTransaction(transaction.id, store.prices)
                                    detail = portfolio.positionDetail(code)
                                }
                            }) {
                                Icon(Icons.Filled.Delete, "Sil", tint = Th.down, modifier = Modifier.size(18.dp))
                            }
                        }
                    }
                }
                item { Spacer(Modifier.height(40.dp)) }
            }
        }
    }

    if (showAdd || edit != null) {
        PortfolioTransactionDialog(
            store = store,
            initialCode = code,
            transaction = edit,
            onDismiss = { showAdd = false; edit = null },
            onSave = { body ->
                scope.launch {
                    val ok = portfolio.saveTransaction(body, store.prices, edit?.id)
                    if (ok) {
                        showAdd = false
                        edit = null
                        detail = portfolio.positionDetail(code)
                    }
                }
            },
        )
    }
}

@Composable
private fun RowScope.PositionMetric(label: String, value: String, trend: Double? = null) {
    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
        Text(label, color = Th.textDim, fontSize = 10.sp)
        Text(value, color = trend?.let(Th::trend) ?: Th.text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun PortfolioTransactionDialog(
    store: PriceStore,
    initialCode: String?,
    transaction: PortfolioTransaction?,
    onDismiss: () -> Unit,
    onSave: (JSONObject) -> Unit,
) {
    var type by remember(transaction?.id) { mutableStateOf(transaction?.type ?: "buy") }
    var code by remember(transaction?.id) { mutableStateOf(transaction?.code?.ifEmpty { null } ?: initialCode.orEmpty()) }
    var assetKind by remember(transaction?.id) { mutableStateOf(store.prices[code]?.kind.orEmpty()) }
    var showKindPicker by remember { mutableStateOf(false) }
    var showAssetPicker by remember { mutableStateOf(false) }
    var quantity by remember(transaction?.id) { mutableStateOf(transaction?.quantity?.takeIf { it > 0 }?.toString() ?: "0") }
    var unitPrice by remember(transaction?.id) {
        mutableStateOf(transaction?.unitPriceNative?.takeIf { it > 0 }?.toString() ?: (store.prices[code]?.price?.toString() ?: "0"))
    }
    var amount by remember(transaction?.id) { mutableStateOf(transaction?.amountNative?.toString() ?: "0") }
    var fee by remember(transaction?.id) { mutableStateOf(transaction?.feeNative?.toString() ?: "0") }
    var note by remember(transaction?.id) { mutableStateOf(transaction?.note ?: "") }
    var occurredDate by remember(transaction?.id) {
        mutableStateOf(transaction?.occurredAt?.take(10) ?: today())
    }
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val trade = type == "buy" || type == "sell"
    val usesAsset = type != "deposit" && type != "withdraw"
    val validDate = validDate(occurredDate)
    val validAmount = if (trade) {
        decimal(quantity) > 0 && decimal(unitPrice) > 0 && decimal(fee) >= 0
    } else {
        decimal(amount) > 0
    }
    val canSave = validDate && validAmount && (!usesAsset || (assetKind.isNotBlank() && code.isNotBlank()))
    LaunchedEffect(code, store.prices.size) {
        if (assetKind.isBlank() && code.isNotBlank()) assetKind = store.prices[code]?.kind.orEmpty()
    }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Th.bg,
        dragHandle = null,
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp).padding(bottom = 28.dp)) {
            Box(
                Modifier
                    .fillMaxWidth()
                    .padding(vertical = 8.dp),
                contentAlignment = Alignment.Center,
            ) {
                TextButton(
                    onClick = onDismiss,
                    modifier = Modifier.align(Alignment.CenterStart),
                ) {
                    Text(
                        "Vazgeç",
                        color = Th.accent,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier
                            .clip(RoundedCornerShape(20.dp))
                            .background(Th.surface)
                            .padding(horizontal = 14.dp, vertical = 7.dp),
                    )
                }
                Text(
                    if (transaction == null) "Yeni İşlem" else "İşlemi Düzenle",
                    color = Th.text,
                    fontSize = 17.sp,
                    fontWeight = FontWeight.Bold,
                    textAlign = TextAlign.Center,
                    maxLines = 1,
                    modifier = Modifier.padding(horizontal = 104.dp),
                )
            }

            Column(
                Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(18.dp))
                    .background(Th.surface),
            ) {
                Column(
                    Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 12.dp, vertical = 12.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text("İşlem türü", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.SemiBold)
                    SegmentedControl(
                        options = listOf("Alış", "Satış", "Diğer"),
                        selectedIndex = when (type) {
                            "buy" -> 0
                            "sell" -> 1
                            else -> 2
                        },
                        onSelected = {
                            type = when (it) {
                                0 -> "buy"
                                1 -> "sell"
                                else -> if (type in listOf("dividend", "fee", "deposit", "withdraw")) type else "dividend"
                            }
                        },
                    )
                    if (type !in listOf("buy", "sell")) {
                        val otherTypes = listOf("dividend", "fee", "deposit", "withdraw")
                        SegmentedControl(
                            options = listOf("Temettü", "Ücret", "Yatır", "Çek"),
                            selectedIndex = otherTypes.indexOf(type).coerceAtLeast(0),
                            onSelected = { type = otherTypes[it] },
                        )
                    }
                }
                HorizontalRule()
                if (usesAsset) {
                    FormSelectRow(
                        "Varlık türü",
                        instrumentKindTitle(assetKind).ifBlank { "Tür seç" },
                        accent = assetKind.isNotBlank(),
                    ) { showKindPicker = true }
                    FormSelectRow(
                        "Varlık",
                        store.prices[code]?.displayName ?: "Varlık seç",
                        accent = code.isNotBlank(),
                        enabled = assetKind.isNotBlank(),
                        search = true,
                    ) { showAssetPicker = true }
                }
                if (trade) {
                    FormFieldRow("Miktar", quantity) { quantity = it }
                    FormFieldRow("Birim fiyatı", unitPrice) { unitPrice = it }
                    FormFieldRow("Komisyon", fee) { fee = it }
                } else {
                    FormFieldRow("Tutar", amount) { amount = it }
                }
                FormFieldRow("İşlem tarihi", occurredDate) { occurredDate = it.take(10) }
                OutlinedTextField(
                    value = note,
                    onValueChange = { note = it.take(500) },
                    placeholder = { Text("Not (opsiyonel)", color = Th.textFaint) },
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 8.dp),
                    colors = androidx.compose.material3.OutlinedTextFieldDefaults.colors(
                        focusedBorderColor = Th.stroke,
                        unfocusedBorderColor = Th.stroke,
                        focusedTextColor = Th.text,
                        unfocusedTextColor = Th.text,
                        cursorColor = Th.accent,
                    ),
                )
            }
            Spacer(Modifier.height(14.dp))
            PrimaryActionButton(
                text = if (transaction == null) "İşlemi ekle" else "Değişiklikleri kaydet",
                enabled = canSave,
                onClick = {
                    val price = store.prices[code]
                    onSave(
                        PortfolioStore.transactionBody(
                            clientId = transaction?.clientId ?: UUID.randomUUID().toString(),
                            code = if (usesAsset) code else "",
                            type = type,
                            quantity = if (trade) decimal(quantity) else 0.0,
                            unitPrice = if (trade) decimal(unitPrice) else 0.0,
                            amount = if (trade) 0.0 else decimal(amount),
                            fee = if (trade) decimal(fee) else 0.0,
                            currency = if (usesAsset) price?.currency ?: "TRY" else "TRY",
                            usdtry = 0.0,
                            occurredAt = PortfolioStore.occurredAtIso(occurredDate),
                            note = note,
                        ),
                    )
                },
            )
        }
    }

    if (showKindPicker) {
        InstrumentKindPickerDialog(
            instruments = store.all(),
            selectedKind = assetKind,
            onDismiss = { showKindPicker = false },
            onSelect = { selectedKind ->
                assetKind = selectedKind
                code = ""
                unitPrice = "0"
            },
        )
    }
    if (showAssetPicker && assetKind.isNotBlank()) {
        InstrumentPickerDialog(
            instruments = store.all(),
            kind = assetKind,
            selectedCode = code,
            onDismiss = { showAssetPicker = false },
            onSelect = { instrument ->
                code = instrument.code
                unitPrice = instrument.price.toString()
            },
        )
    }
}

@Composable
private fun FormSelectRow(
    label: String,
    value: String,
    accent: Boolean = false,
    enabled: Boolean = true,
    search: Boolean = false,
    onClick: () -> Unit,
) {
    Column(
        Modifier
            .fillMaxWidth()
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Text(label, color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.SemiBold)
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                value,
                color = if (accent) Th.accent else Th.text,
                fontSize = 15.sp,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Icon(
                imageVector = if (search) Icons.Outlined.Search else Icons.Outlined.UnfoldMore,
                contentDescription = null,
                tint = if (accent) Th.accent else Th.textFaint,
                modifier = Modifier.size(18.dp),
            )
        }
    }
    HorizontalRule()
}

@Composable
private fun FormFieldRow(label: String, value: String, onChange: (String) -> Unit) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 9.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Text(label, color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.SemiBold)
        OutlinedTextField(
            value = value,
            onValueChange = onChange,
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            modifier = Modifier.fillMaxWidth(),
            colors = androidx.compose.material3.OutlinedTextFieldDefaults.colors(
                focusedBorderColor = Th.stroke,
                unfocusedBorderColor = Th.stroke,
                focusedTextColor = Th.text,
                unfocusedTextColor = Th.text,
                cursorColor = Th.accent,
            ),
        )
    }
    HorizontalRule()
}

@Composable
private fun HorizontalRule() {
    Box(Modifier.fillMaxWidth().padding(start = 16.dp).height(0.5.dp).background(Th.stroke))
}

@Composable
private fun DecimalField(label: String, value: String, onChange: (String) -> Unit) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(label) },
        singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
        modifier = Modifier.fillMaxWidth(),
    )
}

private fun decimal(value: String) = value.replace(",", ".").toDoubleOrNull() ?: 0.0

private fun today(): String =
    SimpleDateFormat("yyyy-MM-dd", Locale.US).format(Date())

private fun validDate(value: String): Boolean {
    if (!Regex("""\d{4}-\d{2}-\d{2}""").matches(value)) return false
    val parsed = runCatching {
        SimpleDateFormat("yyyy-MM-dd", Locale.US).apply { isLenient = false }.parse(value)
    }.getOrNull()
    return parsed != null && !parsed.after(Date())
}

private fun typeTitle(type: String) = when (type) {
    "buy" -> "Alış"
    "sell" -> "Satış"
    "dividend" -> "Temettü"
    "fee" -> "Ücret"
    "deposit" -> "Para yatırma"
    "withdraw" -> "Para çekme"
    else -> type
}
