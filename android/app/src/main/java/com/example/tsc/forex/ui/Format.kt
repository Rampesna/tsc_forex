package com.example.tsc.forex.ui

import java.util.Locale
import kotlin.math.abs

/** Turkish-locale number/price formatting helpers. */
object Fmt {
    private val tr: Locale = Locale.forLanguageTag("tr-TR")

    private fun decimals(v: Double): Int = when {
        abs(v) >= 1000 -> 2
        abs(v) >= 1 -> 4
        abs(v) >= 0.01 -> 4
        else -> 6
    }

    fun number(v: Double): String =
        String.format(tr, "%,.${decimals(v)}f", v)

    /** Price with the Turkish Lira symbol, e.g. "₺46,9947". */
    fun tl(v: Double): String = "₺" + number(v)

    fun pct(v: Double): String {
        val sign = if (v > 0) "+" else ""
        return String.format(tr, "%s%.2f%%", sign, v)
    }

    fun signed(v: Double): String {
        val sign = if (v > 0) "+" else ""
        return sign + number(v)
    }
}
