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
        get() = sp.getStringSet(KEY_FAVS, emptySet()) ?: emptySet()
        set(v) = sp.edit().putStringSet(KEY_FAVS, v).apply()

    /** Ordered favorites (drag-to-reorder), stored as a delimited list. */
    var favoriteOrder: List<String>
        get() = sp.getString(KEY_FAV_ORDER, "")?.split("|")?.filter { it.isNotEmpty() } ?: emptyList()
        set(v) = sp.edit().putString(KEY_FAV_ORDER, v.joinToString("|")).apply()

    /** Portfolio holdings serialized as "code:amount:cost" rows joined by ';'. */
    var holdings: String
        get() = sp.getString(KEY_HOLDINGS, "") ?: ""
        set(v) = sp.edit().putString(KEY_HOLDINGS, v).apply()

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
    }
}
