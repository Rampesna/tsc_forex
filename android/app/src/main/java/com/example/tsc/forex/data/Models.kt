package com.example.tsc.forex.data

import org.json.JSONArray
import org.json.JSONObject

/** A live, user-facing instrument price (currency or gold) quoted in TRY. */
data class InstrumentPrice(
    val code: String,
    val displayName: String,
    val kind: String,          // "currency" | "gold"
    val price: Double,
    val bid: Double,
    val ask: Double,
    val change: Double,
    val changePct: Double,
    val provider: String,
    val timestamp: String,
) {
    val isCurrency get() = kind == "currency"
    val isGold get() = kind == "gold"

    companion object {
        fun from(o: JSONObject) = InstrumentPrice(
            code = o.optString("code"),
            displayName = o.optString("display_name"),
            kind = o.optString("kind"),
            price = o.optDouble("price", 0.0),
            bid = o.optDouble("bid", 0.0),
            ask = o.optDouble("ask", 0.0),
            change = o.optDouble("change", 0.0),
            changePct = o.optDouble("change_pct", 0.0),
            provider = o.optString("provider"),
            timestamp = o.optString("timestamp"),
        )
    }
}

/** OHLC candle for the detail chart. */
data class Candle(
    val time: String,
    val open: Double,
    val high: Double,
    val low: Double,
    val close: Double,
) {
    companion object {
        fun from(o: JSONObject) = Candle(
            time = o.optString("time").ifEmpty { o.optString("timestamp") },
            open = o.optDouble("open", 0.0),
            high = o.optDouble("high", 0.0),
            low = o.optDouble("low", 0.0),
            close = o.optDouble("close", 0.0),
        )
    }
}

/** 24-hour rolling stats for an instrument. */
data class InstrumentStats(
    val code: String,
    val open: Double,
    val high: Double,
    val low: Double,
    val last: Double,
    val count: Long,
) {
    companion object {
        fun from(o: JSONObject) = InstrumentStats(
            code = o.optString("code"),
            open = o.optDouble("open", 0.0),
            high = o.optDouble("high", 0.0),
            low = o.optDouble("low", 0.0),
            last = o.optDouble("last", 0.0),
            count = o.optLong("count", 0),
        )
    }
}

data class TokenResponse(
    val accessToken: String,
    val refreshToken: String,
) {
    companion object {
        fun from(o: JSONObject) = TokenResponse(
            accessToken = o.optString("access_token"),
            refreshToken = o.optString("refresh_token"),
        )
    }
}

data class PriceAlert(
    val id: String,
    val code: String,
    val threshold: Double,
    val direction: String,   // "above" | "below"
    val active: Boolean,
) {
    companion object {
        fun from(o: JSONObject) = PriceAlert(
            id = o.optString("id"),
            code = o.optString("code"),
            threshold = o.optDouble("threshold", 0.0),
            direction = o.optString("direction"),
            active = o.optBoolean("active", true),
        )
    }
}

data class AppNotification(
    val id: String,
    val title: String,
    val body: String,
    val createdAt: String,
) {
    companion object {
        fun from(o: JSONObject) = AppNotification(
            id = o.optString("id"),
            title = o.optString("title"),
            body = o.optString("body").ifEmpty { o.optString("message") },
            createdAt = o.optString("created_at"),
        )
    }
}

/** Parse a JSON array (or a wrapper object holding one) into a typed list. */
inline fun <T> parseArray(raw: String, key: String? = null, crossinline map: (JSONObject) -> T): List<T> {
    val arr: JSONArray = when {
        raw.trimStart().startsWith("[") -> JSONArray(raw)
        else -> {
            val obj = JSONObject(raw)
            when {
                key != null && obj.has(key) -> obj.optJSONArray(key) ?: JSONArray()
                obj.has("data") -> obj.optJSONArray("data") ?: JSONArray()
                else -> JSONArray()
            }
        }
    }
    return buildList { for (i in 0 until arr.length()) add(map(arr.getJSONObject(i))) }
}
