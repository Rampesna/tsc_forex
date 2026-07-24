package com.example.tsc.forex.store

import android.util.Base64
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.example.tsc.forex.data.ApiException
import com.example.tsc.forex.data.InstrumentPrice
import com.example.tsc.forex.data.PortfolioOverview
import com.example.tsc.forex.data.PortfolioPerformancePoint
import com.example.tsc.forex.data.PortfolioPosition
import com.example.tsc.forex.data.PortfolioPositionDetail
import com.example.tsc.forex.data.Prefs
import org.json.JSONArray
import org.json.JSONObject
import java.util.UUID

data class Holding(val code: String, val amount: Double, val cost: Double)

class PortfolioStore(
    private val prefs: Prefs,
    private val auth: AuthStore,
) {
    var overview by mutableStateOf<PortfolioOverview?>(null)
        private set
    var portfolios by mutableStateOf<List<PortfolioOverview>>(emptyList())
        private set
    var performance by mutableStateOf<List<PortfolioPerformancePoint>>(emptyList())
        private set
    var loading by mutableStateOf(false)
        private set
    var errorMessage by mutableStateOf<String?>(null)
    var legacyPreview by mutableStateOf<List<Holding>>(emptyList())
        private set
    private var userScope: String? = null
    private var preferredPortfolioId: String? = null

    val activePositions: List<PortfolioPosition>
        get() = overview?.positions?.filter { it.quantity > 0 } ?: emptyList()

    suspend fun sync(prices: Map<String, InstrumentPrice>, range: String = "1y") {
        if (!auth.isLoggedIn) return
        activateScope(jwtUserId(auth.accessToken))
        loading = overview == null
        errorMessage = null
        try {
            val api = auth.newApiClient()
            var available = api.portfolios()
            if (available.isEmpty()) {
                val created = api.createPortfolio("Ana Portföy")
                available = listOf(api.portfolio(created.id))
            }
            portfolios = available
            val preferred = preferredPortfolioId
                ?: overview?.portfolio?.id
            var next = available.firstOrNull { it.portfolio.id == preferred } ?: available.first()
            if (userScope == null) activateScope(next.portfolio.userId)
            migrateLegacyStorage(next)
            val legacy = legacyHoldings()
            if (legacy.isNotEmpty() && !legacyMigrated()) {
                if (next.positions.any { it.quantity > 0 }) {
                    legacyPreview = legacy
                } else {
                    importLegacy(next.portfolio.id, legacy, prices)
                    next = api.portfolio(next.portfolio.id)
                }
            }
            selectLocally(next)
            persistCache(next)
            if (flushOutbox()) {
                next = api.portfolio(next.portfolio.id)
                selectLocally(next)
                persistCache(next)
            }
            performance = runCatching {
                api.portfolioPerformance(next.portfolio.id, range)
            }.getOrDefault(emptyList())
        } catch (e: Exception) {
            errorMessage = e.message ?: "Portföy yüklenemedi."
        } finally {
            loading = false
        }
    }

    suspend fun selectPortfolio(id: String, prices: Map<String, InstrumentPrice>, range: String = "1y") {
        errorMessage = null
        preferredPortfolioId = id
        try {
            val selected = portfolios.firstOrNull { it.portfolio.id == id }
                ?: auth.newApiClient().portfolio(id)
            selectLocally(selected)
            performance = runCatching {
                auth.newApiClient().portfolioPerformance(id, range)
            }.getOrDefault(emptyList())
            // Refresh full detail for selected portfolio
            val fresh = auth.newApiClient().portfolio(id)
            selectLocally(fresh)
            portfolios = portfolios.map { if (it.portfolio.id == fresh.portfolio.id) fresh else it }
                .let { list -> if (list.any { it.portfolio.id == fresh.portfolio.id }) list else list + fresh }
        } catch (e: Exception) {
            errorMessage = e.message ?: "Portföy seçilemedi."
        }
    }

    suspend fun createPortfolio(name: String, prices: Map<String, InstrumentPrice>, range: String = "1y"): Boolean {
        val cleaned = name.trim()
        if (cleaned.isEmpty()) {
            errorMessage = "Geçerli bir portföy adı girin."
            return false
        }
        return try {
            val created = auth.newApiClient().createPortfolio(cleaned)
            val selected = auth.newApiClient().portfolio(created.id)
            portfolios = portfolios.filter { it.portfolio.id != selected.portfolio.id } + selected
            selectLocally(selected)
            performance = runCatching {
                auth.newApiClient().portfolioPerformance(selected.portfolio.id, range)
            }.getOrDefault(emptyList())
            true
        } catch (e: Exception) {
            errorMessage = e.message ?: "Portföy oluşturulamadı."
            false
        }
    }

    suspend fun renameCurrentPortfolio(name: String): Boolean {
        val id = overview?.portfolio?.id ?: return false
        val cleaned = name.trim()
        if (cleaned.isEmpty()) {
            errorMessage = "Geçerli bir portföy adı girin."
            return false
        }
        return try {
            auth.newApiClient().updatePortfolio(id, cleaned)
            val refreshed = auth.newApiClient().portfolio(id)
            selectLocally(refreshed)
            portfolios = portfolios.map { if (it.portfolio.id == id) refreshed else it }
            true
        } catch (e: Exception) {
            errorMessage = e.message ?: "Yeniden adlandırılamadı."
            false
        }
    }

    suspend fun deleteCurrentPortfolio(prices: Map<String, InstrumentPrice>, range: String = "1y"): Boolean {
        val currentId = overview?.portfolio?.id ?: return false
        if (portfolios.size <= 1) {
            errorMessage = "En az bir portföy bulunmalıdır."
            return false
        }
        return try {
            auth.newApiClient().deletePortfolio(currentId)
            portfolios = portfolios.filter { it.portfolio.id != currentId }
            val next = portfolios.first()
            selectLocally(next)
            performance = runCatching {
                auth.newApiClient().portfolioPerformance(next.portfolio.id, range)
            }.getOrDefault(emptyList())
            true
        } catch (e: Exception) {
            errorMessage = e.message ?: "Portföy silinemedi."
            false
        }
    }

    private fun selectLocally(value: PortfolioOverview) {
        overview = value
        preferredPortfolioId = value.portfolio.id
    }

    suspend fun refreshPerformance(range: String) {
        val id = overview?.portfolio?.id ?: return
        performance = runCatching {
            auth.newApiClient().portfolioPerformance(id, range)
        }.getOrDefault(emptyList())
    }

    suspend fun positionDetail(code: String): PortfolioPositionDetail? {
        val id = overview?.portfolio?.id ?: return null
        return runCatching { auth.newApiClient().portfolioPosition(id, code) }
            .onFailure { errorMessage = it.message }
            .getOrNull()
    }

    suspend fun saveTransaction(
        body: JSONObject,
        prices: Map<String, InstrumentPrice>,
        transactionId: String? = null,
    ): Boolean {
        val id = overview?.portfolio?.id ?: return false
        errorMessage = null
        return try {
            if (transactionId == null) {
                auth.newApiClient().createPortfolioTransaction(id, body)
            } else {
                auth.newApiClient().updatePortfolioTransaction(id, transactionId, body)
            }
            sync(prices)
            true
        } catch (e: Exception) {
            if (transactionId == null && e !is ApiException) {
                enqueue(id, body)
                errorMessage = "İşlem çevrimdışı kuyruğa alındı."
                true
            } else {
                errorMessage = e.message ?: "İşlem kaydedilemedi."
                false
            }
        }
    }

    suspend fun deleteTransaction(
        transactionId: String,
        prices: Map<String, InstrumentPrice>,
    ) {
        val id = overview?.portfolio?.id ?: return
        runCatching {
            auth.newApiClient().deletePortfolioTransaction(id, transactionId)
            sync(prices)
        }.onFailure { errorMessage = it.message }
    }

    suspend fun confirmLegacyImport(prices: Map<String, InstrumentPrice>) {
        val id = overview?.portfolio?.id ?: return
        runCatching {
            importLegacy(id, legacyPreview, prices)
            legacyPreview = emptyList()
            sync(prices)
        }.onFailure { errorMessage = it.message }
    }

    fun skipLegacyImport() {
        setLegacyMigrated()
        legacyPreview = emptyList()
    }

    private suspend fun importLegacy(
        portfolioId: String,
        holdings: List<Holding>,
        prices: Map<String, InstrumentPrice>,
    ) {
        val usdtry = (prices["USDTRY"]?.price ?: 1.0).coerceAtLeast(1.0)
        holdings.forEach { holding ->
            val live = prices[holding.code]
            val isUSD = live?.isUSD == true
            val nativeCost = when {
                holding.cost > 0 && isUSD -> holding.cost / usdtry
                holding.cost > 0 -> holding.cost
                else -> live?.price ?: 0.0
            }
            if (holding.amount <= 0 || nativeCost <= 0) return@forEach
            val legacyId = UUID.nameUUIDFromBytes(
                "${holding.code}|${holding.amount}|${holding.cost}".toByteArray(),
            )
            auth.newApiClient().createPortfolioTransaction(
                portfolioId,
                transactionBody(
                    clientId = "android-legacy-v2-$legacyId",
                    code = holding.code,
                    type = "buy",
                    quantity = holding.amount,
                    unitPrice = nativeCost,
                    currency = if (isUSD) "USD" else "TRY",
                    usdtry = if (isUSD) usdtry else 1.0,
                    note = "Eski cihaz portföyünden aktarıldı",
                ),
            )
        }
        setLegacyMigrated()
        prefs.holdings = ""
    }

    private fun legacyHoldings(): List<Holding> =
        prefs.holdings.split(";").filter { it.isNotEmpty() }.mapNotNull { row ->
            val parts = row.split(":")
            if (parts.size == 3) {
                Holding(parts[0], parts[1].toDoubleOrNull() ?: 0.0, parts[2].toDoubleOrNull() ?: 0.0)
            } else null
        }

    private fun enqueue(portfolioId: String, body: JSONObject) {
        val scope = userScope ?: return
        val rows = runCatching { JSONArray(prefs.portfolioOutbox(scope)) }.getOrElse { JSONArray() }
        rows.put(JSONObject().put("portfolio_id", portfolioId).put("body", body))
        prefs.setPortfolioOutbox(scope, rows.toString())
    }

    private suspend fun flushOutbox(): Boolean {
        val scope = userScope ?: return false
        val rows = runCatching { JSONArray(prefs.portfolioOutbox(scope)) }.getOrElse { JSONArray() }
        if (rows.length() == 0) return false
        val remaining = JSONArray()
        var sent = false
        for (index in 0 until rows.length()) {
            val item = rows.getJSONObject(index)
            runCatching {
                auth.newApiClient().createPortfolioTransaction(
                    item.optString("portfolio_id"),
                    item.getJSONObject("body"),
                )
            }.onSuccess { sent = true }.onFailure { remaining.put(item) }
        }
        prefs.setPortfolioOutbox(scope, remaining.toString())
        return sent
    }

    private fun loadCache(scope: String): PortfolioOverview? =
        runCatching {
            val value = prefs.portfolioCache(scope)
            if (value.isBlank()) null else PortfolioOverview.from(JSONObject(value))
        }.getOrNull()

    private fun persistCache(value: PortfolioOverview) {
        userScope?.let { prefs.setPortfolioCache(it, overviewJSON(value).toString()) }
    }

    private fun overviewJSON(value: PortfolioOverview): JSONObject = JSONObject()
        .put("portfolio", JSONObject()
            .put("id", value.portfolio.id)
            .put("user_id", value.portfolio.userId)
            .put("name", value.portfolio.name)
            .put("base_currency", value.portfolio.baseCurrency)
            .put("net_deposits_try", value.portfolio.netDepositsTry)
            .put("version", value.portfolio.version))
        .put("positions", JSONArray(value.positions.map { position ->
            JSONObject()
                .put("id", position.id).put("code", position.code)
                .put("display_name", position.displayName).put("kind", position.kind)
                .put("currency", position.currency).put("quantity", position.quantity)
                .put("average_cost_native", position.averageCostNative)
                .put("average_cost_try", position.averageCostTry)
                .put("current_price_native", position.currentPriceNative)
                .put("current_price_try", position.currentPriceTry)
                .put("value_native", position.valueNative).put("value_try", position.valueTry)
                .put("cost_basis_try", position.costBasisTry)
                .put("unrealized_pnl_try", position.unrealizedPnlTry)
                .put("unrealized_pnl_pct", position.unrealizedPnlPct)
                .put("realized_pnl_try", position.realizedPnlTry)
                .put("day_change_try", position.dayChangeTry)
                .put("day_change_pct", position.dayChangePct)
                .put("allocation_pct", position.allocationPct)
        }))
        .put("total_value_try", value.totalValueTry)
        .put("total_cost_try", value.totalCostTry)
        .put("unrealized_pnl_try", value.unrealizedPnlTry)
        .put("realized_pnl_try", value.realizedPnlTry)
        .put("day_change_try", value.dayChangeTry)
        .put("day_change_pct", value.dayChangePct)
        .put("net_deposits_try", value.netDepositsTry)
        .put("active_positions", value.activePositions)

    private fun activateScope(scope: String?) {
        if (userScope == scope) return
        userScope = scope
        overview = scope?.let(::loadCache)
        performance = emptyList()
        legacyPreview = emptyList()
    }

    private fun legacyMigrated(): Boolean =
        userScope?.let { prefs.portfolioLegacyMigrated(it) } ?: false

    private fun setLegacyMigrated() {
        userScope?.let { prefs.setPortfolioLegacyMigrated(it, true) }
    }

    private fun migrateLegacyStorage(current: PortfolioOverview) {
        val scope = userScope ?: return
        val old = runCatching {
            if (prefs.portfolioCache.isBlank()) null
            else PortfolioOverview.from(JSONObject(prefs.portfolioCache))
        }.getOrNull() ?: return
        if (old.portfolio.id != current.portfolio.id) return
        if (prefs.portfolioOutbox(scope).isBlank() && prefs.portfolioOutbox.isNotBlank()) {
            prefs.setPortfolioOutbox(scope, prefs.portfolioOutbox)
        }
        if (prefs.portfolioLegacyMigrated) {
            prefs.setPortfolioLegacyMigrated(scope, true)
        }
        prefs.clearLegacyPortfolioStorage()
    }

    private fun jwtUserId(token: String?): String? {
        val payload = token?.split(".")?.getOrNull(1) ?: return null
        return runCatching {
            val decoded = Base64.decode(payload, Base64.URL_SAFE or Base64.NO_WRAP or Base64.NO_PADDING)
            JSONObject(String(decoded, Charsets.UTF_8)).optString("uid").ifBlank { null }
        }.getOrNull()
    }

    companion object {
        fun transactionBody(
            clientId: String = UUID.randomUUID().toString(),
            code: String = "",
            type: String,
            quantity: Double = 0.0,
            unitPrice: Double = 0.0,
            amount: Double = 0.0,
            fee: Double = 0.0,
            currency: String = "TRY",
            usdtry: Double = 0.0,
            occurredAt: String = isoNow(),
            note: String = "",
        ): JSONObject = JSONObject()
            .put("client_id", clientId)
            .put("code", code)
            .put("type", type)
            .put("quantity", plainDecimal(quantity))
            .put("unit_price_native", plainDecimal(unitPrice))
            .put("amount_native", plainDecimal(amount))
            .put("fee_native", plainDecimal(fee))
            .put("currency", currency)
            .put("usdtry", plainDecimal(usdtry))
            .put("occurred_at", occurredAt)
            .put("note", note)

        /**
         * Builds RFC3339 `occurred_at` that is never in the future.
         * iOS sends `Date()` for today; Android used to hardcode noon UTC which
         * is still "future" in the morning (TR) and the API rejects it.
         */
        fun occurredAtIso(dateYmd: String): String {
            val zone = java.time.ZoneId.systemDefault()
            val date = runCatching { java.time.LocalDate.parse(dateYmd) }.getOrNull()
                ?: return isoNow()
            val today = java.time.LocalDate.now(zone)
            val instant = if (!date.isBefore(today)) {
                java.time.Instant.now()
            } else {
                date.atTime(12, 0).atZone(zone).toInstant()
            }
            return java.time.format.DateTimeFormatter.ISO_INSTANT.format(instant)
        }

        private fun plainDecimal(v: Double): String =
            java.math.BigDecimal.valueOf(v).stripTrailingZeros().toPlainString()

        private fun isoNow(): String =
            java.time.format.DateTimeFormatter.ISO_INSTANT.format(java.time.Instant.now())
    }
}
