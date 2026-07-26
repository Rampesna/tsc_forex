package com.example.tsc.forex.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import com.example.tsc.forex.ui.LocalizedText as Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.InstrumentPrice
import com.example.tsc.forex.ui.theme.Th

/**
 * Per-instrument circular icon. Each asset class gets its own visual identity:
 * crypto uses brand colors + ticker glyphs, currencies show flag emoji, gold
 * shows a coin glyph, stocks show a monogram on a stable per-symbol color.
 */
@Composable
fun AssetIcon(ip: InstrumentPrice, size: Int = 36) {
    when {
        ip.isCrypto -> CryptoIcon(ip.code, size)
        ip.isCurrency -> CurrencyIcon(ip.code, size)
        ip.isGold -> GoldIcon(size)
        ip.isStock -> StockIcon(ip.code, size)
        else -> MonogramIcon(ip.code.take(2), Th.surfaceAlt, Th.text, size)
    }
}

// ---- Crypto ----

private data class Brand(val color: Color, val glyph: String)

private val CRYPTO_BRANDS: Map<String, Brand> = mapOf(
    "BTC" to Brand(Color(0xFFF7931A), "₿"),
    "ETH" to Brand(Color(0xFF627EEA), "Ξ"),
    "BNB" to Brand(Color(0xFFF3BA2F), "BNB"),
    "SOL" to Brand(Color(0xFF14F195), "SOL"),
    "XRP" to Brand(Color(0xFF23292F), "XRP"),
    "ADA" to Brand(Color(0xFF0033AD), "ADA"),
    "DOGE" to Brand(Color(0xFFC2A633), "Ð"),
    "TRX" to Brand(Color(0xFFEF0027), "TRX"),
    "AVAX" to Brand(Color(0xFFE84142), "AVAX"),
    "DOT" to Brand(Color(0xFFE6007A), "DOT"),
    "LINK" to Brand(Color(0xFF2A5ADA), "LINK"),
    "MATIC" to Brand(Color(0xFF8247E5), "MATIC"),
    "LTC" to Brand(Color(0xFF345D9D), "Ł"),
    "ATOM" to Brand(Color(0xFF2E3148), "ATOM"),
    "XLM" to Brand(Color(0xFF08B5E5), "XLM"),
    "SHIB" to Brand(Color(0xFFFFA409), "SHIB"),
    "BCH" to Brand(Color(0xFF0AC18E), "BCH"),
)

@Composable
private fun CryptoIcon(code: String, size: Int) {
    val sym = code.removeSuffix("USD").removeSuffix("-USD").uppercase()
    val brand = CRYPTO_BRANDS[sym] ?: Brand(Th.accent, sym.take(3))
    val fg = if (brand.color.luminanceHigh()) Color.Black else Color.White
    Box(
        Modifier.size(size.dp).clip(CircleShape).background(brand.color),
        contentAlignment = Alignment.Center,
    ) {
        Text(brand.glyph, color = fg, fontSize = glyphSize(brand.glyph, size).sp, fontWeight = FontWeight.Black, maxLines = 1)
    }
}

// ---- Currency (flag emoji by ISO) ----

private val CURRENCY_FLAGS: Map<String, String> = mapOf(
    "USD" to "🇺🇸", "EUR" to "🇪🇺", "GBP" to "🇬🇧", "JPY" to "🇯🇵", "CHF" to "🇨🇭",
    "CAD" to "🇨🇦", "AUD" to "🇦🇺", "CNY" to "🇨🇳", "RUB" to "🇷🇺", "SAR" to "🇸🇦",
    "AED" to "🇦🇪", "KWD" to "🇰🇼", "INR" to "🇮🇳", "KRW" to "🇰🇷", "BRL" to "🇧🇷",
    "MXN" to "🇲🇽", "ZAR" to "🇿🇦", "AZN" to "🇦🇿", "KZT" to "🇰🇿", "NOK" to "🇳🇴",
    "SEK" to "🇸🇪", "DKK" to "🇩🇰", "PLN" to "🇵🇱", "HUF" to "🇭🇺", "CZK" to "🇨🇿",
    "ILS" to "🇮🇱", "QAR" to "🇶🇦", "SGD" to "🇸🇬", "HKD" to "🇭🇰", "NZD" to "🇳🇿",
    "THB" to "🇹🇭", "IDR" to "🇮🇩", "MYR" to "🇲🇾", "PHP" to "🇵🇭", "TWD" to "🇹🇼",
    "UAH" to "🇺🇦", "RON" to "🇷🇴", "BGN" to "🇧🇬", "GEL" to "🇬🇪", "TRY" to "🇹🇷",
)

@Composable
private fun CurrencyIcon(code: String, size: Int) {
    val iso = code.uppercase().let { c ->
        CURRENCY_FLAGS.keys.firstOrNull { c.contains(it) } ?: c.take(3)
    }
    val flag = CURRENCY_FLAGS[iso]
    if (flag != null) {
        Box(
            Modifier.size(size.dp).clip(CircleShape).background(Th.surfaceAlt),
            contentAlignment = Alignment.Center,
        ) { Text(flag, fontSize = (size * 0.55).sp) }
    } else {
        MonogramIcon(iso.take(3), Th.surfaceAlt, Th.text, size)
    }
}

// ---- Gold ----

@Composable
private fun GoldIcon(size: Int) {
    Box(
        Modifier.size(size.dp).clip(CircleShape).background(Color(0xFFD4AF37).copy(alpha = 0.20f)),
        contentAlignment = Alignment.Center,
    ) {
        Text("Au", color = Color(0xFFD4AF37), fontSize = (size * 0.36).sp, fontWeight = FontWeight.Black)
    }
}

// ---- Stock (stable color per symbol) ----

private val STOCK_COLORS = listOf(
    Color(0xFF2962FF), Color(0xFF00897B), Color(0xFF6D4C41), Color(0xFFAD1457),
    Color(0xFF5E35B1), Color(0xFF00838F), Color(0xFFEF6C00), Color(0xFF388E3C),
)

@Composable
private fun StockIcon(code: String, size: Int) {
    val color = STOCK_COLORS[(code.hashCode() and 0x7fffffff) % STOCK_COLORS.size]
    MonogramIcon(code.take(2).uppercase(), color, Color.White, size)
}

// ---- Shared ----

@Composable
private fun MonogramIcon(label: String, bg: Color, fg: Color, size: Int) {
    Box(
        Modifier.size(size.dp).clip(CircleShape).background(bg),
        contentAlignment = Alignment.Center,
    ) {
        Text(label, color = fg, fontSize = (size * 0.34).sp, fontWeight = FontWeight.Bold, maxLines = 1)
    }
}

private fun glyphSize(glyph: String, size: Int): Double = when (glyph.length) {
    1 -> size * 0.5
    2, 3 -> size * 0.3
    else -> size * 0.24
}

/** Rough perceived-luminance check to pick a legible foreground. */
private fun Color.luminanceHigh(): Boolean =
    (0.299 * red + 0.587 * green + 0.114 * blue) > 0.6
