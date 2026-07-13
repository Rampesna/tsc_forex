package com.example.tsc.forex.store

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.example.tsc.forex.data.InstrumentPrice
import com.example.tsc.forex.data.Prefs

/** A single portfolio position. */
data class Holding(val code: String, val amount: Double, val cost: Double)

/** Local portfolio: holdings with live valuation and P/L. */
class PortfolioStore(private val prefs: Prefs) {
    var holdings by mutableStateOf(load())
        private set

    fun add(code: String, amount: Double, cost: Double) {
        holdings = holdings + Holding(code, amount, cost)
        persist()
    }

    fun remove(index: Int) {
        if (index in holdings.indices) {
            holdings = holdings.toMutableList().also { it.removeAt(index) }
            persist()
        }
    }

    /** Current market value of a holding using the live price. */
    fun value(h: Holding, prices: Map<String, InstrumentPrice>): Double =
        (prices[h.code]?.price ?: 0.0) * h.amount

    fun totalValue(prices: Map<String, InstrumentPrice>): Double =
        holdings.sumOf { value(it, prices) }

    fun totalCost(): Double = holdings.sumOf { it.cost * it.amount }

    fun totalPL(prices: Map<String, InstrumentPrice>): Double =
        totalValue(prices) - totalCost()

    private fun load(): List<Holding> =
        prefs.holdings.split(";").filter { it.isNotEmpty() }.mapNotNull { row ->
            val parts = row.split(":")
            if (parts.size == 3) Holding(parts[0], parts[1].toDoubleOrNull() ?: 0.0, parts[2].toDoubleOrNull() ?: 0.0)
            else null
        }

    private fun persist() {
        prefs.holdings = holdings.joinToString(";") { "${it.code}:${it.amount}:${it.cost}" }
    }
}
