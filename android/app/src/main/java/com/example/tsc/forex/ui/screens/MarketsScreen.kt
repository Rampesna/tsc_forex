package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.KeyboardArrowUp
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.outlined.StarBorder
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.InstrumentPrice
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.AssetIcon
import com.example.tsc.forex.ui.ChangePill
import com.example.tsc.forex.ui.Fmt
import com.example.tsc.forex.ui.Sparkline
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.theme.Th

private enum class Cat(val label: String) {
    FAV("Favoriler"), ALL("Tümü"), GOLD("Altın"), CUR("Döviz"), CRYPTO("Kripto"), STOCK("Hisse")
}

@Composable
fun MarketsScreen(
    store: PriceStore,
    auth: AuthStore,
    onOpen: (String) -> Unit,
    onNotifications: () -> Unit,
    onRequestAuth: () -> Unit,
) {
    var query by rememberSaveable { mutableStateOf("") }
    var cat by rememberSaveable { mutableStateOf(Cat.ALL) }
    var editFavs by remember { mutableStateOf(false) }
    val notifCount = store.liveNotifications.size

    val base: List<InstrumentPrice> = when (cat) {
        Cat.FAV -> store.orderedFavorites()
        Cat.ALL -> store.all()
        Cat.GOLD -> store.gold()
        Cat.CUR -> store.currencies()
        Cat.CRYPTO -> store.crypto()
        Cat.STOCK -> store.stocks()
    }
    val list = if (query.isBlank()) base else base.filter {
        it.displayName.contains(query, true) || it.code.contains(query, true)
    }

    Column(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
    ) {
        ScreenHeader(
            title = "Piyasalar",
            subtitle = "Altın · Döviz · Kripto · Hisse",
            trailing = {
                IconButton(onClick = onNotifications) {
                BadgedBox(badge = {
                    if (notifCount > 0) {
                        Badge(containerColor = Th.down) {
                            Text(if (notifCount > 9) "9+" else "$notifCount", fontSize = 9.sp)
                        }
                    }
                }) {
                    Icon(Icons.Filled.Notifications, "Bildirimler", tint = Th.textDim)
                }
                }
            },
        )

        OutlinedTextField(
            value = query,
            onValueChange = { query = it },
            placeholder = { Text("Ara: Gram Altın, Dolar, Euro…", color = Th.textFaint) },
            leadingIcon = {
                Icon(Icons.Outlined.Search, contentDescription = null, tint = Th.textDim)
            },
            singleLine = true,
            keyboardOptions = KeyboardOptions.Default,
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = Th.accent,
                unfocusedBorderColor = Th.stroke,
                focusedTextColor = Th.text,
                unfocusedTextColor = Th.text,
                cursorColor = Th.accent,
                focusedContainerColor = Th.surface,
                unfocusedContainerColor = Th.surface,
            ),
            shape = RoundedCornerShape(12.dp),
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp),
        )

        // Category segmented control (scrollable — more classes than fit a row)
        LazyRow(
            Modifier
                .fillMaxWidth()
                .padding(top = 12.dp, bottom = 10.dp),
            contentPadding = PaddingValues(start = 16.dp, end = 24.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(Cat.entries.toList(), key = { it.name }) { c ->
                val selected = c == cat
                FilterChip(
                    selected = selected,
                    onClick = {
                            if (c == Cat.FAV && !auth.isLoggedIn) {
                                onRequestAuth()
                                return@FilterChip
                            }
                            cat = c
                            editFavs = false
                    },
                    label = {
                    Text(
                        c.label,
                        color = if (selected) Color.White else Th.textDim,
                        fontSize = 13.sp,
                        fontWeight = if (selected) FontWeight.Bold else FontWeight.SemiBold,
                    )
                    },
                    shape = RoundedCornerShape(12.dp),
                    colors = FilterChipDefaults.filterChipColors(
                        containerColor = Th.surface,
                        selectedContainerColor = Th.accent,
                    ),
                    border = FilterChipDefaults.filterChipBorder(
                        enabled = true,
                        selected = selected,
                        borderColor = Th.stroke,
                        selectedBorderColor = Th.accent,
                    ),
                )
            }
        }

        if (cat == Cat.FAV && list.isNotEmpty()) {
            Row(
                Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 2.dp),
                horizontalArrangement = Arrangement.End,
            ) {
                TextButton(onClick = { editFavs = !editFavs }) {
                    Text(
                        if (editFavs) "Sıralamayı bitir" else "Sıralamayı düzenle",
                        color = Th.accent,
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }
        }

        if (list.isEmpty()) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text(
                    if (cat == Cat.FAV) "Henüz favori eklemedin." else "Sonuç bulunamadı.",
                    color = Th.textDim, fontSize = 14.sp,
                )
            }
        } else {
            LazyColumn(contentPadding = PaddingValues(bottom = 24.dp)) {
                if (cat == Cat.FAV && editFavs) {
                    itemsIndexed(list, key = { _, it -> it.code }) { index, ip ->
                        ReorderRow(ip, store.sparklines[ip.code].orEmpty(),
                            canUp = index > 0, canDown = index < list.size - 1,
                            onUp = { store.moveFavorite(index, index - 1) },
                            onDown = { store.moveFavorite(index, index + 1) })
                    }
                } else {
                    items(list, key = { it.code }) { ip ->
                        MarketRow(ip, store.sparklines[ip.code].orEmpty(),
                            isFav = store.isFavorite(ip.code),
                            onFav = {
                                if (!auth.isLoggedIn) onRequestAuth()
                                else store.toggleFavorite(ip.code)
                            },
                            onClick = { onOpen(ip.code) })
                    }
                }
            }
        }
    }
}

@Composable
private fun MarketRow(
    ip: InstrumentPrice,
    spark: List<Double>,
    isFav: Boolean,
    onFav: () -> Unit,
    onClick: () -> Unit,
) {
    BoxWithConstraints(
        Modifier
            .fillMaxWidth()
            .heightIn(min = 72.dp)
            .clickable { onClick() }
            .padding(horizontal = 12.dp, vertical = 8.dp),
    ) {
        val compact = maxWidth < 380.dp
        Row(
            Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            AssetIcon(ip)
            Column(
                Modifier
                    .weight(1f)
                    .padding(start = 10.dp, end = 6.dp),
            ) {
                Text(
                    ip.displayName,
                    color = Th.text,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                )
                Text(
                    ip.code,
                    color = Th.textFaint,
                    fontSize = 11.sp,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                )
            }
            if (!compact) {
                Sparkline(
                    spark,
                    up = ip.changePct >= 0,
                    modifier = Modifier
                        .width(52.dp)
                        .height(28.dp)
                        .padding(end = 8.dp),
                )
            }
            Column(
                horizontalAlignment = Alignment.End,
                modifier = Modifier.widthIn(min = if (compact) 82.dp else 94.dp, max = 112.dp),
            ) {
                Text(
                    Fmt.money(ip.price, ip.currency),
                    color = Th.text,
                    fontSize = if (compact) 12.sp else 14.sp,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                )
                if (ip.isUSD && !compact) {
                    Text("≈ ${Fmt.tl(ip.tryValue)}", color = Th.textFaint, fontSize = 10.sp, maxLines = 1)
                }
                ChangePill(ip.changePct)
            }
            IconButton(onClick = onFav, modifier = Modifier.size(44.dp)) {
                Icon(
                    if (isFav) Icons.Filled.Star else Icons.Outlined.StarBorder,
                    contentDescription = if (isFav) "Favorilerden çıkar" else "Favoriye ekle",
                    tint = if (isFav) Th.accent else Th.textFaint,
                    modifier = Modifier.size(20.dp),
                )
            }
        }
    }
}

@Composable
private fun ReorderRow(
    ip: InstrumentPrice,
    spark: List<Double>,
    canUp: Boolean,
    canDown: Boolean,
    onUp: () -> Unit,
    onDown: () -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .heightIn(min = 72.dp)
            .padding(horizontal = 12.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AssetIcon(ip)
        Column(
            Modifier
                .weight(1f)
                .padding(start = 12.dp),
        ) {
            Text(ip.displayName, color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, maxLines = 1)
            Text(Fmt.money(ip.price, ip.currency), color = Th.textDim, fontSize = 12.sp, maxLines = 1)
        }
        IconButton(onClick = onUp, enabled = canUp) {
            Icon(Icons.Filled.KeyboardArrowUp, "yukarı", tint = if (canUp) Th.text else Th.textFaint)
        }
        IconButton(onClick = onDown, enabled = canDown) {
            Icon(Icons.Filled.KeyboardArrowDown, "aşağı", tint = if (canDown) Th.text else Th.textFaint)
        }
    }
}
