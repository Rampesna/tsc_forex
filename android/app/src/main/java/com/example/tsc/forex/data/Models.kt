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
    val isCrypto get() = kind == "crypto"
    val isStock get() = kind == "stock"

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

data class Profile(
    val id: String,
    val email: String,
    val displayName: String,
    val avatarUrl: String,
    val provider: String,
) {
    companion object {
        fun from(o: JSONObject) = Profile(
            id = o.optString("id"),
            email = o.optString("email"),
            displayName = o.optString("display_name"),
            avatarUrl = o.optString("avatar_url"),
            provider = o.optString("provider"),
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

/**
 * Rich, per-kind market data from GET /instruments/{code}/detail. All fields are
 * optional; only the ones relevant to the instrument kind are populated.
 */
data class InstrumentDetail(
    val code: String,
    val kind: String,
    val priceTry: Double?,
    val priceUsd: Double?,
    // Crypto
    val high24h: Double?, val low24h: Double?, val open24h: Double?,
    val changePct24h: Double?, val quoteVolumeUsd: Double?, val trades: Int?,
    val marketCapUsd: Double?, val fdvUsd: Double?, val dominancePct: Double?, val rank: Int?,
    val circulatingSupply: Double?, val totalSupply: Double?, val maxSupply: Double?,
    val athUsd: Double?, val atlUsd: Double?, val athDate: String?, val atlDate: String?,
    // Stock
    val dayHigh: Double?, val dayLow: Double?, val week52High: Double?, val week52Low: Double?,
    val volume: Double?, val exchange: String?, val longName: String?,
    // Gold
    val onsUsd: Double?, val onsTry: Double?, val usdTry: Double?, val gramHasTry: Double?,
    val gramWeight: Double?, val fineness: Double?, val purityLabel: String?,
    // Currency
    val inverse: Double?, val usdCross: Double?, val iso: String?,
) {
    companion object {
        private fun JSONObject.d(k: String): Double? = if (has(k) && !isNull(k)) optDouble(k) else null
        private fun JSONObject.i(k: String): Int? = if (has(k) && !isNull(k)) optInt(k) else null
        private fun JSONObject.s(k: String): String? = optString(k).ifEmpty { null }

        fun from(o: JSONObject) = InstrumentDetail(
            code = o.optString("code"), kind = o.optString("kind"),
            priceTry = o.d("price_try"), priceUsd = o.d("price_usd"),
            high24h = o.d("high24h"), low24h = o.d("low24h"), open24h = o.d("open24h"),
            changePct24h = o.d("change_pct24h"), quoteVolumeUsd = o.d("quote_volume_usd"), trades = o.i("trades"),
            marketCapUsd = o.d("market_cap_usd"), fdvUsd = o.d("fdv_usd"),
            dominancePct = o.d("dominance_pct"), rank = o.i("rank"),
            circulatingSupply = o.d("circulating_supply"), totalSupply = o.d("total_supply"), maxSupply = o.d("max_supply"),
            athUsd = o.d("ath_usd"), atlUsd = o.d("atl_usd"), athDate = o.s("ath_date"), atlDate = o.s("atl_date"),
            dayHigh = o.d("day_high"), dayLow = o.d("day_low"),
            week52High = o.d("week52_high"), week52Low = o.d("week52_low"),
            volume = o.d("volume"), exchange = o.s("exchange"), longName = o.s("long_name"),
            onsUsd = o.d("ons_usd"), onsTry = o.d("ons_try"), usdTry = o.d("usd_try"), gramHasTry = o.d("gram_has_try"),
            gramWeight = o.d("gram_weight"), fineness = o.d("fineness"), purityLabel = o.s("purity_label"),
            inverse = o.d("inverse"), usdCross = o.d("usd_cross"), iso = o.s("iso"),
        )
    }
}

// ---- Gamification (streak / XP / badges / leaderboard) ----

data class GamBadge(
    val code: String,
    val name: String,
    val description: String,
    val icon: String,
    val earned: Boolean,
    val earnedAt: String?,
) {
    companion object {
        fun from(o: JSONObject) = GamBadge(
            code = o.optString("code"),
            name = o.optString("name"),
            description = o.optString("description"),
            icon = o.optString("icon"),
            earned = o.optBoolean("earned", false),
            earnedAt = o.optString("earned_at").ifEmpty { null },
        )
    }
}

data class GamQuest(
    val code: String,
    val title: String,
    val description: String,
    val xp: Int,
    val completed: Boolean,
) {
    companion object {
        fun from(o: JSONObject) = GamQuest(
            code = o.optString("code"),
            title = o.optString("title"),
            description = o.optString("description"),
            xp = o.optInt("xp", 0),
            completed = o.optBoolean("completed", false),
        )
    }
}

data class GamProfile(
    val xp: Int,
    val level: Int,
    val xpIntoLevel: Int,
    val xpForNextLevel: Int,
    val progress: Double,
    val currentStreak: Int,
    val longestStreak: Int,
    val checkedInToday: Boolean,
    val totalCheckIns: Int,
    val weeklyXp: Int,
    val rank: Int,
    val badges: List<GamBadge>,
    val quests: List<GamQuest>,
) {
    companion object {
        fun from(o: JSONObject) = GamProfile(
            xp = o.optInt("xp", 0),
            level = o.optInt("level", 1),
            xpIntoLevel = o.optInt("xp_into_level", 0),
            xpForNextLevel = o.optInt("xp_for_next_level", 100),
            progress = o.optDouble("progress", 0.0),
            currentStreak = o.optInt("current_streak", 0),
            longestStreak = o.optInt("longest_streak", 0),
            checkedInToday = o.optBoolean("checked_in_today", false),
            totalCheckIns = o.optInt("total_check_ins", 0),
            weeklyXp = o.optInt("weekly_xp", 0),
            rank = o.optInt("rank", 0),
            badges = (o.optJSONArray("badges") ?: JSONArray()).let { arr ->
                buildList { for (i in 0 until arr.length()) add(GamBadge.from(arr.getJSONObject(i))) }
            },
            quests = (o.optJSONArray("quests") ?: JSONArray()).let { arr ->
                buildList { for (i in 0 until arr.length()) add(GamQuest.from(arr.getJSONObject(i))) }
            },
        )
    }
}

data class GamCheckInResult(
    val profile: GamProfile,
    val xpGained: Int,
    val newBadges: List<GamBadge>,
    val alreadyCheckedIn: Boolean,
) {
    companion object {
        fun from(o: JSONObject) = GamCheckInResult(
            profile = GamProfile.from(o.optJSONObject("profile") ?: JSONObject()),
            xpGained = o.optInt("xp_gained", 0),
            newBadges = (o.optJSONArray("new_badges") ?: JSONArray()).let { arr ->
                buildList { for (i in 0 until arr.length()) add(GamBadge.from(arr.getJSONObject(i))) }
            },
            alreadyCheckedIn = o.optBoolean("already_checked_in", false),
        )
    }
}

data class LeaderEntry(
    val rank: Int,
    val userId: String,
    val displayName: String,
    val avatarUrl: String,
    val xp: Int,
    val level: Int,
) {
    companion object {
        fun from(o: JSONObject) = LeaderEntry(
            rank = o.optInt("rank", 0),
            userId = o.optString("user_id"),
            displayName = o.optString("display_name"),
            avatarUrl = o.optString("avatar_url"),
            xp = o.optInt("xp", 0),
            level = o.optInt("level", 1),
        )
    }
}

data class Leaderboard(
    val entries: List<LeaderEntry>,
    val me: LeaderEntry?,
) {
    companion object {
        fun from(o: JSONObject) = Leaderboard(
            entries = (o.optJSONArray("entries") ?: JSONArray()).let { arr ->
                buildList { for (i in 0 until arr.length()) add(LeaderEntry.from(arr.getJSONObject(i))) }
            },
            me = o.optJSONObject("me")?.let { LeaderEntry.from(it) },
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
