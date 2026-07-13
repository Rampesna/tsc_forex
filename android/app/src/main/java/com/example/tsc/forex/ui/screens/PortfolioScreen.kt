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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.store.Holding
import com.example.tsc.forex.store.PortfolioStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.AssetBadge
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.theme.Th

@Composable
fun PortfolioScreen(portfolio: PortfolioStore, store: PriceStore) {
    var showAdd by remember { mutableStateOf(false) }
    val prices = store.prices

    val totalValue = portfolio.totalValue(prices)
    val totalCost = portfolio.totalCost()
    val pl = portfolio.totalPL(prices)
    val plPct = if (totalCost > 0) pl / totalCost * 100 else 0.0

    Column(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
    ) {
        ScreenHeader("Portföy", trailing = {
            IconButton(onClick = { showAdd = true }) { Icon(Icons.Filled.Add, "ekle", tint = Th.accent) }
        })

        Card(modifier = Modifier.padding(horizontal = 16.dp)) {
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text("Toplam Değer", color = Th.textDim, fontSize = 12.sp)
                Text(Fmt.tl(totalValue), color = Th.text, fontSize = 28.sp, fontWeight = FontWeight.Bold)
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Text("Maliyet: ${Fmt.tl(totalCost)}", color = Th.textDim, fontSize = 12.sp)
                    Text(
                        "K/Z: ${Fmt.signed(pl)} (${Fmt.pct(plPct)})",
                        color = Th.trend(pl), fontSize = 12.sp, fontWeight = FontWeight.SemiBold,
                    )
                }
            }
        }

        if (portfolio.holdings.isEmpty()) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text("Portföyün boş.\n+ ile varlık ekle.", color = Th.textDim, fontSize = 14.sp)
            }
        } else {
            LazyColumn(
                Modifier.padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                itemsIndexed(portfolio.holdings, key = { i, _ -> "$i" }) { index, h ->
                    HoldingRow(h, store, onRemove = { portfolio.remove(index) })
                }
            }
        }
    }

    if (showAdd) {
        AddHoldingDialog(
            codes = store.all().map { it.code to it.displayName },
            onDismiss = { showAdd = false },
            onAdd = { code, amount, cost -> portfolio.add(code, amount, cost); showAdd = false },
        )
    }
}

@Composable
private fun HoldingRow(h: Holding, store: PriceStore, onRemove: () -> Unit) {
    val price = store.prices[h.code]?.price ?: 0.0
    val value = price * h.amount
    val pl = value - h.cost * h.amount
    Card {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            AssetBadge(h.code, store.prices[h.code]?.isGold == true)
            Column(
                Modifier
                    .weight(1f)
                    .padding(start = 12.dp)
            ) {
                Text(store.prices[h.code]?.displayName ?: h.code, color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
                Text("${Fmt.number(h.amount)} adet • Maliyet ${Fmt.tl(h.cost)}", color = Th.textDim, fontSize = 11.sp)
            }
            Column(horizontalAlignment = Alignment.End) {
                Text(Fmt.tl(value), color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.Bold)
                Text(Fmt.signed(pl), color = Th.trend(pl), fontSize = 11.sp)
            }
            IconButton(onClick = onRemove) { Icon(Icons.Filled.Delete, "sil", tint = Th.textFaint) }
        }
    }
}

@Composable
private fun AddHoldingDialog(
    codes: List<Pair<String, String>>,
    onDismiss: () -> Unit,
    onAdd: (String, Double, Double) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    var code by remember { mutableStateOf(codes.firstOrNull()?.first ?: "") }
    var amount by remember { mutableStateOf("") }
    var cost by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = Th.surface,
        title = { Text("Varlık ekle", color = Th.text) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Box {
                    Box(
                        Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(10.dp))
                            .background(Th.surfaceAlt)
                            .clickable { expanded = true }
                            .padding(14.dp)
                    ) { Text(code.ifEmpty { "Varlık seç" }, color = Th.text) }
                    DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                        codes.forEach { (c, name) ->
                            DropdownMenuItem(text = { Text("$c • $name") }, onClick = { code = c; expanded = false })
                        }
                    }
                }
                OutlinedTextField(
                    value = amount, onValueChange = { amount = it },
                    label = { Text("Miktar") }, singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                )
                OutlinedTextField(
                    value = cost, onValueChange = { cost = it },
                    label = { Text("Birim maliyet (₺)") }, singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                )
            }
        },
        confirmButton = {
            TextButton(onClick = {
                val a = amount.replace(",", ".").toDoubleOrNull()
                val c = cost.replace(",", ".").toDoubleOrNull()
                if (code.isNotEmpty() && a != null && a > 0 && c != null) onAdd(code, a, c)
            }) { Text("Ekle", color = Th.accent) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Vazgeç", color = Th.textDim) } },
    )
}
