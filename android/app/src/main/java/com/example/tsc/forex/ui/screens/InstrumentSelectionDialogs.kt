package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.InstrumentPrice
import com.example.tsc.forex.ui.theme.Th

internal data class InstrumentKindOption(
    val id: String,
    val title: String,
    val subtitle: String,
)

internal val instrumentKindOptions = listOf(
    InstrumentKindOption("gold", "Altın", "Gram, çeyrek, tam ve diğer altınlar"),
    InstrumentKindOption("currency", "Döviz", "Dolar, euro ve diğer para birimleri"),
    InstrumentKindOption("crypto", "Kripto Para", "Bitcoin, Ethereum ve diğer kriptolar"),
    InstrumentKindOption("stock", "Hisse Senedi", "BIST ve global şirket hisseleri"),
)

internal fun instrumentKindTitle(kind: String) =
    instrumentKindOptions.firstOrNull { it.id == kind }?.title.orEmpty()

@Composable
internal fun InstrumentKindPickerDialog(
    instruments: List<InstrumentPrice>,
    selectedKind: String,
    onDismiss: () -> Unit,
    onSelect: (String) -> Unit,
) {
    var search by remember { mutableStateOf("") }
    val options = instrumentKindOptions.filter {
        search.isBlank() || it.title.contains(search, true) || it.subtitle.contains(search, true)
    }
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = Th.surface,
        title = { Text("Varlık Türü", color = Th.text) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = search,
                    onValueChange = { search = it },
                    label = { Text("Varlık türü ara") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                options.forEach { option ->
                    SelectionRow(
                        title = option.title,
                        subtitle = "${option.subtitle} · ${instruments.count { it.kind == option.id }} varlık",
                        selected = selectedKind == option.id,
                    ) {
                        onSelect(option.id)
                        onDismiss()
                    }
                }
            }
        },
        confirmButton = {},
        dismissButton = { TextButton(onClick = onDismiss) { Text("Vazgeç") } },
    )
}

@Composable
internal fun InstrumentPickerDialog(
    instruments: List<InstrumentPrice>,
    kind: String,
    selectedCode: String,
    onDismiss: () -> Unit,
    onSelect: (InstrumentPrice) -> Unit,
) {
    var search by remember { mutableStateOf("") }
    val matches = instruments
        .filter { it.kind == kind }
        .filter { search.isBlank() || it.displayName.contains(search, true) || it.code.contains(search, true) }
        .sortedBy { it.displayName }
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = Th.surface,
        title = { Text(instrumentKindTitle(kind), color = Th.text) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = search,
                    onValueChange = { search = it },
                    label = { Text("Varlık ara") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                androidx.compose.foundation.lazy.LazyColumn(
                    modifier = Modifier.fillMaxWidth().heightIn(max = 390.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    items(matches.size) { index ->
                        val instrument = matches[index]
                        SelectionRow(
                            title = instrument.displayName,
                            subtitle = "${if (instrument.currency == "USD") "$" else "₺"}${instrument.price}",
                            selected = selectedCode == instrument.code,
                        ) {
                            onSelect(instrument)
                            onDismiss()
                        }
                    }
                }
            }
        },
        confirmButton = {},
        dismissButton = { TextButton(onClick = onDismiss) { Text("Vazgeç") } },
    )
}

@Composable
internal fun InstrumentMultiPickerDialog(
    instruments: List<InstrumentPrice>,
    initialCodes: List<String>,
    maxSelection: Int,
    onDismiss: () -> Unit,
    onDone: (List<String>) -> Unit,
) {
    var selected by remember { mutableStateOf(initialCodes.toSet()) }
    var kind by remember { mutableStateOf("gold") }
    var search by remember { mutableStateOf("") }
    val matches = instruments
        .filter { it.kind == kind }
        .filter { search.isBlank() || it.displayName.contains(search, true) || it.code.contains(search, true) }
        .sortedBy { it.displayName }

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = Th.surface,
        title = { Text("İzlenecek Varlıklar (${selected.size}/$maxSelection)", color = Th.text) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(9.dp)) {
                if (selected.isNotEmpty()) {
                    Row(
                        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        instruments.filter { selected.contains(it.code) }.forEach { instrument ->
                            Text(
                                "${instrument.displayName} ×",
                                color = Th.accent,
                                fontSize = 11.sp,
                                fontWeight = FontWeight.SemiBold,
                                modifier = Modifier
                                    .background(Th.surfaceAlt, RoundedCornerShape(20.dp))
                                    .clickable { selected = selected - instrument.code }
                                    .padding(horizontal = 10.dp, vertical = 7.dp),
                            )
                        }
                    }
                }
                Row(
                    Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(5.dp),
                ) {
                    instrumentKindOptions.forEach { option ->
                        Text(
                            option.title,
                            color = if (kind == option.id) Th.accent else Th.textDim,
                            fontSize = 11.sp,
                            fontWeight = FontWeight.SemiBold,
                            modifier = Modifier
                                .background(Th.surfaceAlt, RoundedCornerShape(20.dp))
                                .clickable { kind = option.id; search = "" }
                                .padding(horizontal = 10.dp, vertical = 7.dp),
                        )
                    }
                }
                OutlinedTextField(
                    value = search,
                    onValueChange = { search = it },
                    label = { Text("${instrumentKindTitle(kind)} ara") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                androidx.compose.foundation.lazy.LazyColumn(
                    modifier = Modifier.fillMaxWidth().heightIn(max = 330.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    items(matches.size) { index ->
                        val instrument = matches[index]
                        SelectionRow(
                            title = instrument.displayName,
                            subtitle = if (selected.contains(instrument.code)) "Seçildi" else "Seç",
                            selected = selected.contains(instrument.code),
                        ) {
                            selected = when {
                                selected.contains(instrument.code) -> selected - instrument.code
                                selected.size < maxSelection -> selected + instrument.code
                                else -> selected
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(
                enabled = selected.isNotEmpty(),
                onClick = {
                    onDone(instruments.filter { selected.contains(it.code) }.map { it.code }.take(maxSelection))
                    onDismiss()
                },
            ) { Text("Bitti", color = Th.accent) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Vazgeç") } },
    )
}

@Composable
private fun SelectionRow(
    title: String,
    subtitle: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .background(Th.surfaceAlt, RoundedCornerShape(10.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column {
            Text(title, color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
            Text(subtitle, color = Th.textDim, fontSize = 10.sp)
        }
        Spacer(Modifier.weight(1f))
        Text(if (selected) "✓" else "›", color = if (selected) Th.accent else Th.textFaint)
    }
}
