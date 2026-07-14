package com.example.tsc.forex.store

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshots.SnapshotStateList
import com.example.tsc.forex.data.AppNotification
import com.example.tsc.forex.data.InstrumentPrice
import com.example.tsc.forex.data.PriceSocket
import com.example.tsc.forex.data.Prefs
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/** Live market state: REST bootstrap + WebSocket stream + favorites. */
class PriceStore(private val prefs: Prefs, private val auth: AuthStore) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    val prices = mutableStateMapOf<String, InstrumentPrice>()
    val sparklines = mutableStateMapOf<String, List<Double>>()
    val liveNotifications: SnapshotStateList<AppNotification> = mutableStateListOf()

    var connected by mutableStateOf(false)
        private set
    var favorites by mutableStateOf(prefs.favorites)
        private set
    var favoriteOrder by mutableStateOf(prefs.favoriteOrder.ifEmpty { prefs.favorites.toList() })
        private set

    private var socket: PriceSocket? = null
    private var started = false

    fun start() {
        if (started) return
        started = true
        auth.onAccessTokenChanged = { token -> socket?.updateToken(token) }
        bootstrap()
        socket = PriceSocket(
            onPrice = { ip -> applyPrice(ip) },
            onNotification = { n -> liveNotifications.add(0, n) },
            onState = { c -> scope.launch { connected = c } },
        ).also { it.connect(auth.accessToken) }
    }

    fun reconnect() { socket?.reconnect() }

    private fun bootstrap() {
        scope.launch {
            val api = auth.newApiClient()
            runCatching { api.prices() }.getOrNull()?.forEach { applyPrice(it) }
            runCatching { api.sparklines(30) }.getOrNull()?.let { sparklines.putAll(it) }
        }
    }

    fun refresh() = bootstrap()

    private fun applyPrice(ip: InstrumentPrice) {
        prices[ip.code] = ip
        // Maintain a short rolling sparkline buffer per instrument.
        val buf = (sparklines[ip.code] ?: emptyList()).toMutableList()
        buf.add(ip.price)
        if (buf.size > 40) buf.removeAt(0)
        sparklines[ip.code] = buf
    }

    // ---- Derived collections ----

    fun all(): List<InstrumentPrice> = prices.values.sortedBy { it.code }
    fun gold(): List<InstrumentPrice> = all().filter { it.isGold }
    fun currencies(): List<InstrumentPrice> = all().filter { it.isCurrency }
    fun crypto(): List<InstrumentPrice> = all().filter { it.kind == "crypto" }
    fun stocks(): List<InstrumentPrice> = all().filter { it.kind == "stock" }

    fun orderedFavorites(): List<InstrumentPrice> =
        favoriteOrder.mapNotNull { prices[it] }.filter { favorites.contains(it.code) }

    /** Top daily gainers (positive change only), highest first. */
    fun topGainers(n: Int = 10): List<InstrumentPrice> =
        prices.values.filter { it.changePct > 0 }
            .sortedByDescending { it.changePct }
            .take(n)

    // ---- Favorites ----

    fun isFavorite(code: String) = favorites.contains(code)

    fun toggleFavorite(code: String) {
        val newFavs = favorites.toMutableSet()
        val newOrder = favoriteOrder.toMutableList()
        if (newFavs.contains(code)) {
            newFavs.remove(code); newOrder.remove(code)
            scope.launch { runCatching { auth.newApiClient().removeWatch(code) } }
        } else {
            newFavs.add(code); if (!newOrder.contains(code)) newOrder.add(code)
            scope.launch { runCatching { auth.newApiClient().addWatch(code) } }
        }
        favorites = newFavs
        favoriteOrder = newOrder
        prefs.favorites = newFavs
        prefs.favoriteOrder = newOrder
    }

    fun moveFavorite(from: Int, to: Int) {
        val list = orderedFavorites().map { it.code }.toMutableList()
        if (from !in list.indices || to !in list.indices) return
        val item = list.removeAt(from)
        list.add(to, item)
        val missing = favoriteOrder.filter { !list.contains(it) && favorites.contains(it) }
        favoriteOrder = list + missing
        prefs.favoriteOrder = favoriteOrder
    }

    // ---- Notifications ----

    fun clearLiveNotifications() = liveNotifications.clear()
    fun removeLiveNotification(index: Int) {
        if (index in liveNotifications.indices) liveNotifications.removeAt(index)
    }
}
