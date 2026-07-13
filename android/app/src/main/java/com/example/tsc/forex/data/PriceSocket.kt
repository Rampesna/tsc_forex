package com.example.tsc.forex.data

import com.example.tsc.forex.AppConfig
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean

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
        ws?.close(1000, "bye")
        ws = null
        onState(false)
    }

    private fun open() {
        val urlStr = token?.takeIf { it.isNotEmpty() }?.let { "${AppConfig.wsUrl}?token=$it" } ?: AppConfig.wsUrl
        val req = Request.Builder().url(urlStr).build()
        ws = client.newWebSocket(req, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                backoffMs = 1_000L
                onState(true)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handle(text)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                onState(false)
                scheduleReconnect()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                onState(false)
                if (!closed.get()) scheduleReconnect()
            }
        })
    }

    private fun scheduleReconnect() {
        if (closed.get()) return
        val delay = backoffMs
        backoffMs = (backoffMs * 2).coerceAtMost(15_000L)
        Thread {
            Thread.sleep(delay)
            if (!closed.get()) open()
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
