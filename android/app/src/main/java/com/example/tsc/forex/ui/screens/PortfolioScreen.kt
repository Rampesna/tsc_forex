package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
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
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.PortfolioPerformancePoint
import com.example.tsc.forex.data.PortfolioPosition
import com.example.tsc.forex.data.PortfolioPositionDetail
import com.example.tsc.forex.data.PortfolioTransaction
import com.example.tsc.forex.store.PortfolioStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.AssetBadge
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.UUID

@Composable
fun PortfolioScreen(
    portfolio: PortfolioStore,
    store: PriceStore,
    onPosition: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    var showAdd by remember { mutableStateOf(false) }
    var range by remember { mutableStateOf("1y") }
    val overview = portfolio.overview

    LaunchedEffect(Unit) { portfolio.sync(store.prices, range) }

    Column(Modifier.fillMaxSize().background(Th.bg)) {
        ScreenHeader(
            title = "Portföy",
            subtitle = "Bulut senkronu ve TRY bazlı canlı analiz",
            trailing = {
                IconButton(onClick = { showAdd = true }) {
                    Icon(Icons.Filled.Add, "İşlem ekle", tint = Th.accent)
                }
            },
        )
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
            item {
                portfolio.errorMessage?.let {
                    Card(Modifier.padding(horizontal = 16.dp)) {
                        Text(it, color = Th.down, fontSize = 12.sp)
                    }
                }
            }
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
            item { SummaryCard(overview) }
            item {
                PerformanceCard(
                    points = portfolio.performance,
                    range = range,
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
            Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                SummaryMetric("Bugün", overview?.dayChangeTry ?: 0.0, overview?.dayChangePct)
                SummaryMetric("Gerçekleşmemiş", overview?.unrealizedPnlTry ?: 0.0)
                SummaryMetric("Gerçekleşen", overview?.realizedPnlTry ?: 0.0)
            }
        }
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

@Composable
private fun PerformanceCard(
    points: List<PortfolioPerformancePoint>,
    range: String,
    onRange: (String) -> Unit,
) {
    val firstActive = points.indexOfFirst { it.valueTry != 0.0 || it.costBasisTry != 0.0 }
    val plotted = if (firstActive > 0) points.drop(firstActive) else points
    Card(Modifier.padding(horizontal = 16.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Performans", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                Spacer(Modifier.weight(1f))
                listOf("1m", "3m", "1y", "5y").forEach { item ->
                    Text(
                        item.uppercase(),
                        color = if (range == item) Th.accent else Th.textDim,
                        fontSize = 10.sp,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier
                            .clickable { onRange(item) }
                            .padding(horizontal = 6.dp, vertical = 5.dp),
                    )
                }
            }
            if (plotted.size < 2) {
                Box(Modifier.fillMaxWidth().height(140.dp), contentAlignment = Alignment.Center) {
                    Text("Henüz yeterli veri yok", color = Th.textFaint, fontSize = 12.sp)
                }
            } else {
                Canvas(Modifier.fillMaxWidth().height(140.dp)) {
                    val values = plotted.map { it.valueTry }
                    val low = values.minOrNull() ?: 0.0
                    val high = values.maxOrNull() ?: 1.0
                    val span = maxOf(high - low, high * .01, 1.0)
                    val path = Path()
                    values.forEachIndexed { index, value ->
                        val x = index.toFloat() / (values.size - 1).coerceAtLeast(1) * size.width
                        val y = size.height - ((value - low) / span).toFloat() * (size.height - 12f) - 6f
                        if (index == 0) path.moveTo(x, y) else path.lineTo(x, y)
                    }
                    drawPath(path, color = Th.accent, style = Stroke(width = 2.5f, cap = StrokeCap.Round))
                }
            }
        }
    }
}

@Composable
private fun AllocationCard(positions: List<PortfolioPosition>) {
    Card(Modifier.padding(horizontal = 16.dp)) {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("Dağılım", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
            positions.take(8).forEach { position ->
                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Row {
                        Text(position.code, color = Th.text, fontSize = 11.sp)
                        Spacer(Modifier.weight(1f))
                        Text("%${"%.1f".format(position.allocationPct)}", color = Th.textDim, fontSize = 11.sp)
                    }
                    LinearProgressIndicator(
                        progress = { (position.allocationPct / 100).toFloat().coerceIn(0f, 1f) },
                        color = Th.accent,
                        trackColor = Th.surfaceAlt,
                        modifier = Modifier.fillMaxWidth().height(5.dp),
                    )
                }
            }
        }
    }
}

@Composable
private fun PositionRow(position: PortfolioPosition, onClick: () -> Unit, modifier: Modifier = Modifier) {
    Card(modifier.clickable(onClick = onClick)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            AssetBadge(position.code, position.kind == "gold")
            Column(Modifier.weight(1f).padding(start = 12.dp), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                Text(position.displayName, color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
                Text(
                    "${Fmt.number(position.quantity)} adet • Ort. ${Fmt.money(position.averageCostNative, position.currency)}",
                    color = Th.textDim,
                    fontSize = 10.sp,
                )
            }
            Column(horizontalAlignment = Alignment.End) {
                Text(Fmt.tl(position.valueTry), color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.Bold)
                Text(
                    "${Fmt.signed(position.unrealizedPnlTry)} ₺ (${Fmt.signedPct(position.unrealizedPnlPct)})",
                    color = Th.trend(position.unrealizedPnlTry),
                    fontSize = 10.sp,
                )
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
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri", tint = Th.text) }
            Text(code, color = Th.text, fontSize = 19.sp, fontWeight = FontWeight.Bold)
            Spacer(Modifier.weight(1f))
            IconButton(onClick = { showAdd = true }) { Icon(Icons.Filled.Add, "İşlem ekle", tint = Th.accent) }
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

@Composable
private fun PortfolioTransactionDialog(
    store: PriceStore,
    initialCode: String?,
    transaction: PortfolioTransaction?,
    onDismiss: () -> Unit,
    onSave: (JSONObject) -> Unit,
) {
    val types = listOf("buy", "sell", "dividend", "fee", "deposit", "withdraw")
    var type by remember(transaction?.id) { mutableStateOf(transaction?.type ?: "buy") }
    var typeMenu by remember { mutableStateOf(false) }
    var code by remember(transaction?.id) { mutableStateOf(transaction?.code?.ifEmpty { null } ?: initialCode.orEmpty()) }
    var assetKind by remember(transaction?.id) { mutableStateOf(store.prices[code]?.kind.orEmpty()) }
    var showKindPicker by remember { mutableStateOf(false) }
    var showAssetPicker by remember { mutableStateOf(false) }
    var quantity by remember(transaction?.id) { mutableStateOf(transaction?.quantity?.toString() ?: "") }
    var unitPrice by remember(transaction?.id) {
        mutableStateOf(transaction?.unitPriceNative?.toString() ?: (store.prices[code]?.price?.toString() ?: ""))
    }
    var amount by remember(transaction?.id) { mutableStateOf(transaction?.amountNative?.toString() ?: "") }
    var fee by remember(transaction?.id) { mutableStateOf(transaction?.feeNative?.toString() ?: "0") }
    var note by remember(transaction?.id) { mutableStateOf(transaction?.note ?: "") }
    var occurredDate by remember(transaction?.id) {
        mutableStateOf(transaction?.occurredAt?.take(10) ?: today())
    }
    val trade = type == "buy" || type == "sell"
    val usesAsset = type != "deposit" && type != "withdraw"
    val validDate = validDate(occurredDate)
    val validAmount = if (trade) {
        decimal(quantity) > 0 && decimal(unitPrice) > 0 && decimal(fee) >= 0
    } else {
        decimal(amount) > 0
    }
    LaunchedEffect(code, store.prices.size) {
        if (assetKind.isBlank() && code.isNotBlank()) assetKind = store.prices[code]?.kind.orEmpty()
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = Th.surface,
        title = { Text(if (transaction == null) "Yeni İşlem" else "İşlemi Düzenle", color = Th.text) },
        text = {
            LazyColumn(verticalArrangement = Arrangement.spacedBy(9.dp)) {
                item {
                    Box {
                        OutlinedTextField(
                            value = typeTitle(type), onValueChange = {},
                            readOnly = true, label = { Text("İşlem türü") },
                            modifier = Modifier.fillMaxWidth().clickable { typeMenu = true },
                        )
                        DropdownMenu(expanded = typeMenu, onDismissRequest = { typeMenu = false }) {
                            types.forEach { item ->
                                DropdownMenuItem(
                                    text = { Text(typeTitle(item)) },
                                    onClick = { type = item; typeMenu = false },
                                )
                            }
                        }
                    }
                }
                if (usesAsset) {
                    item {
                        OutlinedTextField(
                            value = instrumentKindTitle(assetKind),
                            onValueChange = {},
                            readOnly = true,
                            label = { Text("Varlık türü") },
                            placeholder = { Text("Tür seç") },
                            modifier = Modifier.fillMaxWidth().clickable { showKindPicker = true },
                        )
                    }
                    item {
                        OutlinedTextField(
                            value = store.prices[code]?.displayName.orEmpty(),
                            onValueChange = {},
                            readOnly = true,
                            enabled = assetKind.isNotBlank(),
                            label = { Text("Varlık") },
                            placeholder = { Text(if (assetKind.isBlank()) "Önce varlık türünü seç" else "Varlık seç") },
                            modifier = Modifier.fillMaxWidth().clickable(enabled = assetKind.isNotBlank()) {
                                showAssetPicker = true
                            },
                        )
                    }
                }
                if (trade) {
                    item { DecimalField("Miktar", quantity) { quantity = it } }
                    item { DecimalField("Birim fiyatı", unitPrice) { unitPrice = it } }
                    item { DecimalField("Komisyon", fee) { fee = it } }
                } else {
                    item { DecimalField("Tutar", amount) { amount = it } }
                }
                item {
                    OutlinedTextField(
                        value = occurredDate,
                        onValueChange = { occurredDate = it.take(10) },
                        label = { Text("İşlem tarihi (YYYY-AA-GG)") },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                item {
                    OutlinedTextField(
                        value = note, onValueChange = { note = it.take(500) },
                        label = { Text("Not") }, modifier = Modifier.fillMaxWidth(),
                    )
                }
            }
        },
        confirmButton = {
            TextButton(
                enabled = validDate && validAmount && (!usesAsset || code.isNotBlank()),
                onClick = {
                    val price = store.prices[code]
                    onSave(PortfolioStore.transactionBody(
                        clientId = transaction?.clientId ?: UUID.randomUUID().toString(),
                        code = if (usesAsset) code else "",
                        type = type,
                        quantity = if (trade) decimal(quantity) else 0.0,
                        unitPrice = if (trade) decimal(unitPrice) else 0.0,
                        amount = if (trade) 0.0 else decimal(amount),
                        fee = if (trade) decimal(fee) else 0.0,
                        currency = if (usesAsset) price?.currency ?: "TRY" else "TRY",
                        usdtry = 0.0,
                        occurredAt = "${occurredDate}T12:00:00Z",
                        note = note,
                    ))
                },
            ) { Text("Kaydet", color = Th.accent) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Vazgeç", color = Th.textDim) } },
    )
    if (showKindPicker) {
        InstrumentKindPickerDialog(
            instruments = store.all(),
            selectedKind = assetKind,
            onDismiss = { showKindPicker = false },
        ) { selectedKind ->
            assetKind = selectedKind
            code = ""
            unitPrice = ""
        }
    }
    if (showAssetPicker && assetKind.isNotBlank()) {
        InstrumentPickerDialog(
            instruments = store.all(),
            kind = assetKind,
            selectedCode = code,
            onDismiss = { showAssetPicker = false },
        ) { instrument ->
            code = instrument.code
            unitPrice = instrument.price.toString()
        }
    }
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
