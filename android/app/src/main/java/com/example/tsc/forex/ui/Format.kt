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

    /** Compact large numbers with Turkish suffixes: 1.2 Mn, 3.4 Mr, 5.6 Tn. */
    fun compact(v: Double): String {
        val a = abs(v)
        return when {
            a >= 1_000_000_000_000 -> String.format(tr, "%.2f Tn", v / 1_000_000_000_000)
            a >= 1_000_000_000 -> String.format(tr, "%.2f Mr", v / 1_000_000_000)
            a >= 1_000_000 -> String.format(tr, "%.2f Mn", v / 1_000_000)
            a >= 1_000 -> String.format(tr, "%.2f B", v / 1_000)
            else -> number(v)
        }
    }

    /** USD value, compact when large. */
    fun usd(v: Double): String = "$" + if (abs(v) >= 1_000_000) compact(v) else number(v)

    /** Compact TRY value. */
    fun tlCompact(v: Double): String = "₺" + compact(v)

    /** Unsigned percentage, e.g. "3.42%". */
    fun pctUnsigned(v: Double): String = String.format(tr, "%.2f%%", v)

    /** Trims an ISO date/time to just the date (YYYY-MM-DD). */
    fun dateShort(iso: String): String = iso.take(10)
}
