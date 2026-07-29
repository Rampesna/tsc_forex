package com.example.tsc.forex.data

import android.util.Log
import com.example.tsc.forex.AppConfig
import com.example.tsc.forex.BuildConfig
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject
import java.util.Locale
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicLong

/**
 * Live price WebSocket with automatic reconnect and exponential backoff.
 * Parses server frames and forwards typed events to the listener.
 */
class PriceSocket(
    private val onPrice: (InstrumentPrice) -> Unit,
    private val onNotification: (AppNotification) -> Unit,
    private val onState: (Boolean) -> Unit,
) {
    private val client = OkHttpClient.Builder()
        .pingInterval(20, TimeUnit.SECONDS)
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .build()

    private var ws: WebSocket? = null
    private val closed = AtomicBoolean(false)
    private val generation = AtomicLong(0)
    private var token: String? = null
    private var backoffMs = 1_000L

    fun connect(token: String?) {
        this.token = token
        closed.set(false)
        open()
    }

    fun updateToken(token: String?) {
        this.token = token
        // Reconnect with the new token so per-user alert routing works.
        ws?.cancel()
        if (!closed.get()) open()
    }

    fun reconnect() {
        ws?.cancel()
        if (!closed.get()) open()
    }

    fun disconnect() {
        closed.set(true)
        generation.incrementAndGet()
        ws?.close(1000, "bye")
        ws = null
        onState(false)
    }

    private fun open() {
        val connectionGeneration = generation.incrementAndGet()
        ws?.cancel()
        val locale = Locale.getDefault().toLanguageTag().take(16)
        val req = try {
            val url = websocketHttpUrl(AppConfig.wsUrl).newBuilder()
                .addQueryParameter("platform", "android")
                .addQueryParameter("locale", locale)
                .apply { token?.takeIf(String::isNotEmpty)?.let { addQueryParameter("token", it) } }
                .build()
            Request.Builder()
                .url(url)
                .header("X-TSC-Client-Platform", "android")
                .header("X-TSC-Client-Locale", locale)
                .header("X-TSC-App-Version", BuildConfig.VERSION_NAME)
                .build()
        } catch (error: IllegalArgumentException) {
            // REST bootstrap remains usable even if a debug host override is
            // malformed. Never let transport configuration terminate startup.
            Log.e("TSC.PriceSocket", "Invalid WebSocket endpoint", error)
            onState(false)
            return
        }
        ws = client.newWebSocket(req, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                if (connectionGeneration != generation.get()) return
                backoffMs = 1_000L
                onState(true)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                if (connectionGeneration != generation.get()) return
                handle(text)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                if (connectionGeneration != generation.get()) return
                onState(false)
                scheduleReconnect(connectionGeneration)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                if (connectionGeneration != generation.get()) return
                onState(false)
                if (!closed.get()) scheduleReconnect(connectionGeneration)
            }
        })
    }

    private fun scheduleReconnect(connectionGeneration: Long) {
        if (closed.get() || connectionGeneration != generation.get()) return
        val delay = backoffMs
        backoffMs = (backoffMs * 2).coerceAtMost(15_000L)
        Thread {
            Thread.sleep(delay)
            if (!closed.get() && connectionGeneration == generation.get()) open()
        }.apply { isDaemon = true }.start()
    }

    private fun handle(text: String) {
        val msg = runCatching { JSONObject(text) }.getOrNull() ?: return
        val type = msg.optString("type")
        val payload = when {
            msg.has("data") && msg.get("data") is JSONObject -> msg.getJSONObject("data")
            msg.has("payload") && msg.get("payload") is JSONObject -> msg.getJSONObject("payload")
            else -> msg
        }
        when (type) {
            "instrument_update", "price_update" ->
                if (payload.has("code")) onPrice(InstrumentPrice.from(payload))
            "notification", "alert" -> onNotification(AppNotification.from(payload))
        }
    }
}

/**
 * OkHttp's HttpUrl intentionally accepts only HTTP(S), while its WebSocket
 * client upgrades those requests internally. Normalize WS(S) before adding
 * query parameters so application startup can never fail on a valid socket URL.
 */
internal fun websocketHttpUrl(rawUrl: String): HttpUrl {
    val normalized = when {
        rawUrl.startsWith("wss://", ignoreCase = true) ->
            "https://${rawUrl.substring(6)}"
        rawUrl.startsWith("ws://", ignoreCase = true) ->
            "http://${rawUrl.substring(5)}"
        else -> rawUrl
    }
    return normalized.toHttpUrl()
}
