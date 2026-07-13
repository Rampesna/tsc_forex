package com.example.tsc.forex.data

import com.example.tsc.forex.AppConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.TimeUnit

class ApiException(val code: Int, message: String) : Exception(message)

/**
 * Hand-rolled REST client over OkHttp. Transparently refreshes the access token
 * once on a 401 and retries the original request.
 */
class ApiClient(
    private val tokenProvider: () -> String? = { null },
    private val onRefresh: (suspend () -> String?)? = null,
) {
    private val client = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(20, TimeUnit.SECONDS)
        .build()

    private val json = "application/json; charset=utf-8".toMediaType()

    // ---- Auth ----

    suspend fun login(email: String, password: String): TokenResponse =
        TokenResponse.from(postJson("auth/login", JSONObject().put("email", email).put("password", password), auth = false))

    suspend fun register(email: String, password: String): TokenResponse =
        TokenResponse.from(postJson("auth/register", JSONObject().put("email", email).put("password", password), auth = false))

    suspend fun googleLogin(idToken: String): TokenResponse =
        TokenResponse.from(postJson("auth/google", JSONObject().put("id_token", idToken), auth = false))

    suspend fun refresh(refreshToken: String): TokenResponse =
        TokenResponse.from(postJson("auth/refresh", JSONObject().put("refresh_token", refreshToken), auth = false))

    suspend fun logout(refreshToken: String) {
        runCatching { postJson("auth/logout", JSONObject().put("refresh_token", refreshToken), auth = false) }
    }

    // ---- Market data ----

    suspend fun prices(): List<InstrumentPrice> =
        parseArray(getRaw("instruments/prices"), "prices") { InstrumentPrice.from(it) }

    suspend fun candles(code: String, interval: String): List<Candle> =
        parseArray(getRaw("instruments/$code/history?interval=$interval"), "candles") { Candle.from(it) }

    suspend fun stats(code: String): InstrumentStats =
        InstrumentStats.from(JSONObject(getRaw("instruments/$code/stats")).let { unwrap(it) })

    suspend fun sparklines(points: Int = 30): Map<String, List<Double>> {
        val raw = getRaw("instruments/sparklines?points=$points")
        val obj = JSONObject(raw).let { if (it.has("data")) it.getJSONObject("data") else it }
        val out = HashMap<String, List<Double>>()
        for (key in obj.keys()) {
            val arr = obj.optJSONArray(key) ?: continue
            out[key] = buildList { for (i in 0 until arr.length()) add(arr.optDouble(i, 0.0)) }
        }
        return out
    }

    // ---- Watchlist ----

    suspend fun addWatch(code: String) { postJson("me/watchlist", JSONObject().put("code", code)) }
    suspend fun removeWatch(code: String) { delete("me/watchlist/$code") }

    // ---- Alerts ----

    suspend fun alerts(): List<PriceAlert> =
        parseArray(getRaw("me/alerts"), "alerts") { PriceAlert.from(it) }

    suspend fun createAlert(code: String, threshold: Double, direction: String): PriceAlert =
        PriceAlert.from(unwrap(postJson("me/alerts",
            JSONObject().put("code", code).put("threshold", threshold).put("direction", direction))))

    suspend fun deleteAlert(id: String) { delete("me/alerts/$id") }

    // ---- Notifications ----

    suspend fun notifications(): List<AppNotification> =
        parseArray(getRaw("me/notifications"), "notifications") { AppNotification.from(it) }

    suspend fun deleteNotification(id: String) { delete("me/notifications/$id") }
    suspend fun clearNotifications() { delete("me/notifications") }

    // ---- Push ----

    suspend fun registerDevice(token: String, platform: String = "android") {
        postJson("me/devices", JSONObject().put("token", token).put("platform", platform))
    }

    // ---- Core HTTP ----

    private fun url(path: String) = "${AppConfig.apiBaseUrl}/$path"

    private suspend fun getRaw(path: String): String = execute(
        Request.Builder().url(url(path)).get()
    )

    private suspend fun postJson(path: String, body: JSONObject, auth: Boolean = true): JSONObject {
        val raw = execute(
            Request.Builder().url(url(path)).post(body.toString().toRequestBody(json)),
            auth = auth,
        )
        return if (raw.isBlank()) JSONObject() else JSONObject(raw)
    }

    private suspend fun delete(path: String) {
        execute(Request.Builder().url(url(path)).delete())
    }

    private suspend fun execute(builder: Request.Builder, auth: Boolean = true, isRetry: Boolean = false): String =
        withContext(Dispatchers.IO) {
            val b = builder.header("Accept", "application/json")
            if (auth) tokenProvider()?.let { b.header("Authorization", "Bearer $it") }
            client.newCall(b.build()).execute().use { resp ->
                val text = resp.body?.string().orEmpty()
                if (resp.code == 401 && auth && !isRetry && onRefresh != null) {
                    val newToken = onRefresh.invoke()
                    if (!newToken.isNullOrEmpty()) {
                        // Rebuild the request with the fresh token and retry once.
                        val retry = builder.header("Accept", "application/json")
                            .header("Authorization", "Bearer $newToken")
                        return@withContext client.newCall(retry.build()).execute().use { r2 ->
                            val t2 = r2.body?.string().orEmpty()
                            if (!r2.isSuccessful) throw ApiException(r2.code, t2.ifBlank { r2.message })
                            t2
                        }
                    }
                }
                if (!resp.isSuccessful) throw ApiException(resp.code, text.ifBlank { resp.message })
                text
            }
        }

    private fun unwrap(o: JSONObject): JSONObject =
        if (o.has("data") && o.get("data") is JSONObject) o.getJSONObject("data") else o

    @Suppress("unused")
    private fun JSONArray.objects() = buildList { for (i in 0 until length()) add(getJSONObject(i)) }
}
