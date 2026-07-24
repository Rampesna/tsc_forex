package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.outlined.AccountBalance
import androidx.compose.material.icons.outlined.CurrencyBitcoin
import androidx.compose.material.icons.outlined.CurrencyExchange
import androidx.compose.material.icons.outlined.Diamond
import androidx.compose.material.icons.outlined.RadioButtonChecked
import androidx.compose.material.icons.outlined.RadioButtonUnchecked
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.InstrumentPrice
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.theme.Th

internal data class InstrumentKindOption(
    val id: String,
    val title: String,
    val subtitle: String,
    val icon: ImageVector,
)

internal val instrumentKindOptions = listOf(
    InstrumentKindOption("gold", "Altın", "Gram, çeyrek, tam ve diğer altınlar", Icons.Outlined.Diamond),
    InstrumentKindOption("currency", "Döviz", "Dolar, euro ve diğer para birimleri", Icons.Outlined.CurrencyExchange),
    InstrumentKindOption("crypto", "Kripto Para", "Bitcoin, Ethereum ve diğer kriptolar", Icons.Outlined.CurrencyBitcoin),
    InstrumentKindOption("stock", "Hisse Senedi", "BIST ve global şirket hisseleri", Icons.Outlined.AccountBalance),
)

internal fun instrumentKindTitle(kind: String) =
    instrumentKindOptions.firstOrNull { it.id == kind }?.title.orEmpty()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun InstrumentKindPickerDialog(
    instruments: List<InstrumentPrice>,
    selectedKind: String,
    onDismiss: () -> Unit,
    onSelect: (String) -> Unit,
) {
    var search by remember { mutableStateOf("") }
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val options = instrumentKindOptions.filter {
        search.isBlank() || it.title.contains(search, true) || it.subtitle.contains(search, true)
    }
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Th.bg,
        dragHandle = null,
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .fillMaxHeight(0.92f)
                .padding(horizontal = 16.dp),
        ) {
            PickerTopBar(title = "Varlık Türü", onCancel = onDismiss)
            Spacer(Modifier.height(8.dp))
            Column(
                Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(22.dp))
                    .background(Th.surface)
                    .padding(vertical = 4.dp),
            ) {
                options.forEachIndexed { index, option ->
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .clickable {
                                onSelect(option.id)
                                onDismiss()
                            }
                            .padding(horizontal = 16.dp, vertical = 14.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        BoxIcon(option.icon)
                        Column(Modifier.weight(1f)) {
                            Text(option.title, color = Th.text, fontSize = 16.sp, fontWeight = FontWeight.SemiBold)
                            Text(option.subtitle, color = Th.textDim, fontSize = 12.sp)
                        }
                        Text(
                            "${instruments.count { it.kind == option.id }}",
                            color = Th.textDim,
                            fontSize = 14.sp,
                        )
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = null,
                            tint = Th.textFaint,
                            modifier = Modifier.size(20.dp),
                        )
                    }
                    if (index < options.lastIndex) {
                        HorizontalDivider(color = Th.stroke, thickness = 0.5.dp, modifier = Modifier.padding(start = 56.dp))
                    }
                }
            }
            Spacer(Modifier.height(12.dp))
            BottomSearchField(
                value = search,
                onValueChange = { search = it },
                placeholder = "Varlık türü ara",
            )
            Spacer(Modifier.height(20.dp))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun InstrumentPickerDialog(
    instruments: List<InstrumentPrice>,
    kind: String,
    selectedCode: String,
    onDismiss: () -> Unit,
    onSelect: (InstrumentPrice) -> Unit,
) {
    var search by remember { mutableStateOf("") }
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val title = instrumentKindTitle(kind)
    val matches = instruments
        .filter { it.kind == kind }
        .filter { search.isBlank() || it.displayName.contains(search, true) || it.code.contains(search, true) }
        .sortedBy { it.displayName }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Th.bg,
        dragHandle = null,
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .fillMaxHeight(0.92f)
                .padding(horizontal = 16.dp),
        ) {
            PickerTopBar(title = title, onCancel = onDismiss)
            Spacer(Modifier.height(8.dp))
            LazyColumn(
                Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(22.dp))
                    .background(Th.surface),
            ) {
                items(matches, key = { it.code }) { instrument ->
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .clickable {
                                onSelect(instrument)
                                onDismiss()
                            }
                            .padding(horizontal = 16.dp, vertical = 13.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text(instrument.displayName, color = Th.text, fontSize = 16.sp, fontWeight = FontWeight.SemiBold)
                            Text(Fmt.money(instrument.price, instrument.currency), color = Th.textDim, fontSize = 13.sp)
                        }
                        Icon(
                            imageVector = if (selectedCode == instrument.code) {
                                Icons.Outlined.RadioButtonChecked
                            } else {
                                Icons.Outlined.RadioButtonUnchecked
                            },
                            contentDescription = null,
                            tint = if (selectedCode == instrument.code) Th.accent else Th.textFaint,
                            modifier = Modifier.size(22.dp),
                        )
                    }
                    HorizontalDivider(color = Th.stroke, thickness = 0.5.dp, modifier = Modifier.padding(start = 16.dp))
                }
            }
            Spacer(Modifier.height(12.dp))
            BottomSearchField(
                value = search,
                onValueChange = { search = it },
                placeholder = "$title ara",
            )
            Spacer(Modifier.height(20.dp))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
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
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val matches = instruments
        .filter { it.kind == kind }
        .filter { search.isBlank() || it.displayName.contains(search, true) || it.code.contains(search, true) }
        .sortedBy { it.displayName }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Th.bg,
        dragHandle = null,
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .fillMaxHeight(0.92f)
                .padding(horizontal = 16.dp),
        ) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                TextButton(onClick = onDismiss) {
                    Text(
                        "Vazgeç",
                        color = Th.accent,
                        modifier = Modifier
                            .clip(RoundedCornerShape(20.dp))
                            .background(Th.surface)
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                    )
                }
                Spacer(Modifier.weight(1f))
                Text("İzlenecek (${selected.size}/$maxSelection)", color = Th.text, fontWeight = FontWeight.Bold)
                Spacer(Modifier.weight(1f))
                TextButton(
                    enabled = selected.isNotEmpty(),
                    onClick = {
                        onDone(instruments.filter { selected.contains(it.code) }.map { it.code }.take(maxSelection))
                        onDismiss()
                    },
                ) { Text("Bitti", color = Th.accent, fontWeight = FontWeight.Bold) }
            }
            Spacer(Modifier.height(8.dp))
            Row(
                Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
                horizontalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                instrumentKindOptions.forEach { option ->
                    Text(
                        option.title,
                        color = if (kind == option.id) Th.bg else Th.textDim,
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier
                            .clip(RoundedCornerShape(20.dp))
                            .background(if (kind == option.id) Th.accent else Th.surface)
                            .clickable { kind = option.id; search = "" }
                            .padding(horizontal = 12.dp, vertical = 8.dp),
                    )
                }
            }
            Spacer(Modifier.height(10.dp))
            LazyColumn(
                Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(22.dp))
                    .background(Th.surface),
            ) {
                items(matches, key = { it.code }) { instrument ->
                    val isOn = selected.contains(instrument.code)
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .clickable {
                                selected = when {
                                    isOn -> selected - instrument.code
                                    selected.size < maxSelection -> selected + instrument.code
                                    else -> selected
                                }
                            }
                            .padding(horizontal = 16.dp, vertical = 13.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text(instrument.displayName, color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
                            Text(Fmt.money(instrument.price, instrument.currency), color = Th.textDim, fontSize = 12.sp)
                        }
                        Text(if (isOn) "●" else "○", color = if (isOn) Th.accent else Th.textFaint, fontSize = 18.sp)
                    }
                    HorizontalDivider(color = Th.stroke, thickness = 0.5.dp, modifier = Modifier.padding(start = 16.dp))
                }
            }
            Spacer(Modifier.height(12.dp))
            BottomSearchField(
                value = search,
                onValueChange = { search = it },
                placeholder = "${instrumentKindTitle(kind)} ara",
            )
            Spacer(Modifier.height(20.dp))
        }
    }
}

@Composable
private fun PickerTopBar(title: String, onCancel: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(top = 8.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TextButton(onClick = onCancel) {
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
        Spacer(Modifier.weight(1f))
        Text(title, color = Th.text, fontSize = 17.sp, fontWeight = FontWeight.Bold)
        Spacer(Modifier.weight(1f))
        Spacer(Modifier.padding(horizontal = 40.dp))
    }
}

@Composable
private fun BottomSearchField(value: String, onValueChange: (String) -> Unit, placeholder: String) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        singleLine = true,
        placeholder = { Text(placeholder, color = Th.textFaint) },
        leadingIcon = {
            Icon(Icons.Outlined.Search, null, tint = Th.textFaint, modifier = Modifier.size(20.dp))
        },
        shape = RoundedCornerShape(28.dp),
        colors = OutlinedTextFieldDefaults.colors(
            focusedBorderColor = Th.stroke,
            unfocusedBorderColor = Th.stroke,
            focusedContainerColor = Th.surface,
            unfocusedContainerColor = Th.surface,
            focusedTextColor = Th.text,
            unfocusedTextColor = Th.text,
            cursorColor = Th.accent,
        ),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun BoxIcon(icon: ImageVector) {
    Box(
        Modifier
            .size(40.dp)
            .clip(CircleShape)
            .background(Th.accent.copy(alpha = 0.14f)),
        contentAlignment = Alignment.Center,
    ) {
        Icon(icon, contentDescription = null, tint = Th.accent, modifier = Modifier.size(20.dp))
    }
}
