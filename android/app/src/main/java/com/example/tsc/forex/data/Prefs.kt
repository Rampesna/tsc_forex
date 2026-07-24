package com.example.tsc.forex.data

import android.content.Context
import android.content.SharedPreferences

/** Lightweight persistent key-value store for tokens, favorites and portfolio. */
class Prefs(context: Context) {
    private val sp: SharedPreferences =
        context.getSharedPreferences("tsc_forex", Context.MODE_PRIVATE)

    var accessToken: String?
        get() = sp.getString(KEY_ACCESS, null)
        set(v) = sp.edit().putString(KEY_ACCESS, v).apply()

    var refreshToken: String?
        get() = sp.getString(KEY_REFRESH, null)
        set(v) = sp.edit().putString(KEY_REFRESH, v).apply()

    var email: String?
        get() = sp.getString(KEY_EMAIL, null)
        set(v) = sp.edit().putString(KEY_EMAIL, v).apply()

    var displayName: String?
        get() = sp.getString(KEY_NAME, null)
        set(v) = sp.edit().putString(KEY_NAME, v).apply()

    var avatarUrl: String?
        get() = sp.getString(KEY_AVATAR, null)
        set(v) = sp.edit().putString(KEY_AVATAR, v).apply()

    var hostOverride: String?
        get() = sp.getString(KEY_HOST, null)
        set(v) = sp.edit().putString(KEY_HOST, v).apply()

    var favorites: Set<String>
        get() = sp.getStringSet(KEY_FAVS, emptySet())?.toSet() ?: emptySet()
        // Copy required: SharedPreferences may reuse the same Set instance.
        set(v) = sp.edit().putStringSet(KEY_FAVS, HashSet(v)).apply()

    /** Ordered favorites (drag-to-reorder), stored as a delimited list. */
    var favoriteOrder: List<String>
        get() = sp.getString(KEY_FAV_ORDER, "")?.split("|")?.filter { it.isNotEmpty() } ?: emptyList()
        set(v) = sp.edit().putString(KEY_FAV_ORDER, v.joinToString("|")).apply()

    /** Portfolio holdings serialized as "code:amount:cost" rows joined by ';'. */
    var holdings: String
        get() = sp.getString(KEY_HOLDINGS, "") ?: ""
        set(v) = sp.edit().putString(KEY_HOLDINGS, v).apply()

    var portfolioCache: String
        get() = sp.getString(KEY_PORTFOLIO_CACHE, "") ?: ""
        set(v) = sp.edit().putString(KEY_PORTFOLIO_CACHE, v).apply()

    var portfolioOutbox: String
        get() = sp.getString(KEY_PORTFOLIO_OUTBOX, "") ?: ""
        set(v) = sp.edit().putString(KEY_PORTFOLIO_OUTBOX, v).apply()

    var portfolioLegacyMigrated: Boolean
        get() = sp.getBoolean(KEY_PORTFOLIO_MIGRATED, false)
        set(v) = sp.edit().putBoolean(KEY_PORTFOLIO_MIGRATED, v).apply()

    fun portfolioCache(scope: String): String =
        sp.getString("$KEY_PORTFOLIO_CACHE.$scope", "") ?: ""

    fun setPortfolioCache(scope: String, value: String) {
        sp.edit().putString("$KEY_PORTFOLIO_CACHE.$scope", value).apply()
    }

    fun portfolioOutbox(scope: String): String =
        sp.getString("$KEY_PORTFOLIO_OUTBOX.$scope", "") ?: ""

    fun setPortfolioOutbox(scope: String, value: String) {
        sp.edit().putString("$KEY_PORTFOLIO_OUTBOX.$scope", value).apply()
    }

    fun portfolioLegacyMigrated(scope: String): Boolean =
        sp.getBoolean("$KEY_PORTFOLIO_MIGRATED.$scope", false)

    fun setPortfolioLegacyMigrated(scope: String, value: Boolean) {
        sp.edit().putBoolean("$KEY_PORTFOLIO_MIGRATED.$scope", value).apply()
    }

    fun clearLegacyPortfolioStorage() {
        sp.edit()
            .remove(KEY_PORTFOLIO_CACHE)
            .remove(KEY_PORTFOLIO_OUTBOX)
            .remove(KEY_PORTFOLIO_MIGRATED)
            .apply()
    }

    var deviceId: String?
        get() = sp.getString(KEY_DEVICE_ID, null)
        set(v) = sp.edit().putString(KEY_DEVICE_ID, v).apply()

    fun aiResearchConsent(userId: String): Boolean =
        userId.isNotBlank() && sp.getBoolean("$KEY_AI_CONSENT.$userId", false)

    fun setAiResearchConsent(userId: String, accepted: Boolean) {
        if (userId.isBlank()) return
        sp.edit().putBoolean("$KEY_AI_CONSENT.$userId", accepted).apply()
    }

    fun pendingRequestId(userId: String, operation: String, resource: String): String {
        val key = "$KEY_PENDING.$userId.$operation.$resource"
        val existing = sp.getString(key, null)
        if (!existing.isNullOrBlank()) return existing
        val created = java.util.UUID.randomUUID().toString()
        sp.edit().putString(key, created).apply()
        return created
    }

    fun clearPendingRequestId(userId: String, operation: String, resource: String) {
        sp.edit().remove("$KEY_PENDING.$userId.$operation.$resource").apply()
    }

    fun clearSession() {
        sp.edit().remove(KEY_ACCESS).remove(KEY_REFRESH).remove(KEY_NAME).remove(KEY_AVATAR).apply()
    }

    companion object {
        private const val KEY_ACCESS = "access_token"
        private const val KEY_REFRESH = "refresh_token"
        private const val KEY_EMAIL = "email"
        private const val KEY_NAME = "display_name"
        private const val KEY_AVATAR = "avatar_url"
        private const val KEY_HOST = "host_override"
        private const val KEY_FAVS = "favorites"
        private const val KEY_FAV_ORDER = "favorite_order"
        private const val KEY_HOLDINGS = "holdings"
        private const val KEY_PORTFOLIO_CACHE = "portfolio_cloud_cache_v2"
        private const val KEY_PORTFOLIO_OUTBOX = "portfolio_outbox_v1"
        private const val KEY_PORTFOLIO_MIGRATED = "portfolio_legacy_migrated_v2"
        private const val KEY_DEVICE_ID = "device_id_v1"
        private const val KEY_AI_CONSENT = "ai_research_consent_v1"
        private const val KEY_PENDING = "pending_op_v1"
    }
}
