package com.example.tsc.forex.data

import com.example.tsc.forex.AppConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone
import java.util.UUID
import java.util.concurrent.TimeUnit

class ApiException(
    val code: Int,
    message: String,
    val serverCode: String? = null,
) : Exception(message) {
    /** Keep idempotency key for in-flight / rate-limited operations. */
    val shouldRetainRequestId: Boolean
        get() = (code == 409 && serverCode == "operation_in_progress") || code == 429

    companion object {
        fun from(code: Int, body: String): ApiException {
            val obj = runCatching { JSONObject(body) }.getOrNull()
            val msg = obj?.optString("error")?.ifBlank { null }
                ?: obj?.optString("message")?.ifBlank { null }
                ?: body.ifBlank { "HTTP $code" }
            val serverCode = obj?.optString("code")?.ifBlank { null }
            return ApiException(code, msg, serverCode)
        }
    }
}

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
        .readTimeout(70, TimeUnit.SECONDS)
        .build()

    private val json = "application/json; charset=utf-8".toMediaType()

    // ---- Auth ----

    suspend fun login(email: String, password: String): TokenResponse =
        TokenResponse.from(postJson("auth/login", JSONObject().put("email", email).put("password", password), auth = false))

    suspend fun register(email: String, password: String): TokenResponse {
        val body = JSONObject().put("email", email).put("password", password)
        postJson("auth/register", body, auth = false)
        return login(email, password)
    }

    suspend fun googleLogin(idToken: String): TokenResponse =
        TokenResponse.from(postJson("auth/google", JSONObject().put("id_token", idToken), auth = false))

    suspend fun appleLogin(idToken: String, fullName: String): TokenResponse =
        TokenResponse.from(postJson("auth/apple",
            JSONObject().put("id_token", idToken).put("full_name", fullName), auth = false))

    suspend fun refresh(refreshToken: String): TokenResponse =
        TokenResponse.from(postJson("auth/refresh", JSONObject().put("refresh_token", refreshToken), auth = false))

    /** Begins usernameless passkey login; returns {options, session}. */
    suspend fun passkeyLoginBegin(): JSONObject =
        postJson("auth/passkey/login/begin", JSONObject(), auth = false)

    /** Finishes passkey login with the signed session + authenticator response. */
    suspend fun passkeyLoginFinish(session: String, credential: JSONObject): TokenResponse =
        TokenResponse.from(postJson(
            "auth/passkey/login/finish",
            JSONObject().put("session", session).put("credential", credential),
            auth = false,
        ))

    suspend fun logout(refreshToken: String) {
        runCatching { postJson("auth/logout", JSONObject().put("refresh_token", refreshToken), auth = false) }
    }

    // ---- Market data ----

    suspend fun prices(): List<InstrumentPrice> =
        parseArray(getRaw("instruments/prices"), "prices") { InstrumentPrice.from(it) }

    suspend fun candles(
        code: String,
        interval: String,
        fromMs: Long = 0L,
        toMs: Long = 0L,
        limit: Int = 0,
        latestSession: Boolean = false,
    ): List<Candle> {
        var path = "instruments/$code/history?interval=$interval"
        fun enc(ms: Long) = java.net.URLEncoder.encode(java.time.Instant.ofEpochMilli(ms).toString(), "UTF-8")
        if (fromMs > 0L) path += "&from=" + enc(fromMs)
        if (toMs > 0L) path += "&to=" + enc(toMs)
        if (limit > 0) path += "&limit=$limit"
        if (latestSession) path += "&session=latest"
        return parseArray(getRaw(path), "candles") { Candle.from(it) }
    }

    suspend fun stats(code: String): InstrumentStats =
        InstrumentStats.from(JSONObject(getRaw("instruments/$code/stats")).let { unwrap(it) })

    suspend fun detail(code: String): InstrumentDetail =
        InstrumentDetail.from(unwrap(JSONObject(getRaw("instruments/$code/detail"))))

    suspend fun marketInsight(
        code: String,
        deviceId: String,
        requestId: String,
        onJobId: (String) -> Unit = {},
    ): MarketInsight {
        var jobId = ""
        for (i in 0 until 120) {
            try {
                val body = JSONObject()
                    .put("code", code)
                    .put("device_id", deviceId)
                    .put("request_id", requestId)
                if (jobId.isNotEmpty()) body.put("job_id", jobId)
                val response = unwrap(postJson("me/ai/market-insight", body))
                val pendingJob = response.optString("job_id")
                if (pendingJob.isEmpty()) return MarketInsight.from(response)
                jobId = pendingJob
                onJobId(jobId)
                delay(3_000)
            } catch (e: ApiException) {
                if (e.code == 409 && e.serverCode == "operation_in_progress") {
                    delay(1_500)
                } else {
                    throw e
                }
            }
        }
        throw ApiException(504, "Derin araştırma beklenenden uzun sürdü; biraz sonra yeniden deneyin.")
    }

    suspend fun billingPlans(): BillingPlansResponse =
        BillingPlansResponse.from(unwrap(JSONObject(getRaw("billing/plans"))))

    suspend fun billingStatus(deviceId: String): BillingStatus {
        val enc = java.net.URLEncoder.encode(deviceId, "UTF-8")
        return BillingStatus.from(unwrap(JSONObject(getRaw("me/billing/status?device_id=$enc"))))
    }

    /** Verifies a Google Play purchase token with the backend (Android only). */
    suspend fun syncGooglePurchase(
        packageName: String,
        productId: String,
        purchaseToken: String,
        deviceId: String,
    ): BillingStatus {
        val body = JSONObject()
            .put("package_name", packageName)
            .put("product_id", productId)
            .put("purchase_token", purchaseToken)
            .put("device_id", deviceId)
        return BillingStatus.from(unwrap(postJson("me/billing/google/sync", body)))
    }

    suspend fun researchHistory(limit: Int = 50): List<ResearchArchiveItem> {
        val bounded = limit.coerceIn(1, 100)
        return parseArray(getRaw("me/ai/research-history?limit=$bounded"), "items") {
            ResearchArchiveItem.from(it)
        }
    }

    suspend fun portfolioHealth(id: String, requestId: String): PortfolioHealth =
        PortfolioHealth.from(unwrap(postJson(
            "me/portfolios/$id/health-check",
            JSONObject().put("request_id", requestId),
        )))

    suspend fun exportPortfolio(id: String, requestId: String): ByteArray {
        val body = JSONObject().put("request_id", requestId).toString().toRequestBody(json)
        return executeBytes(Request.Builder().url(url("me/portfolios/$id/export")).post(body))
    }

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

    /** Server-side favorites for the authenticated user (`GET /me/watchlist`). */
    suspend fun watchlist(): List<String> {
        val o = JSONObject(getRaw("me/watchlist"))
        val arr = o.optJSONArray("codes") ?: return emptyList()
        return buildList {
            for (i in 0 until arr.length()) {
                val code = arr.optString(i).trim()
                if (code.isNotEmpty()) add(code)
            }
        }
    }

    suspend fun addWatch(code: String) { postJson("me/watchlist", JSONObject().put("code", code)) }
    suspend fun removeWatch(code: String) { delete("me/watchlist/$code") }

    // ---- Alerts ----

    suspend fun alerts(): List<PriceAlert> =
        parseArray(getRaw("me/alerts"), "alerts") { PriceAlert.from(it) }

    suspend fun createAlert(code: String, threshold: Double, direction: String): PriceAlert =
        PriceAlert.from(unwrap(postJson("me/alerts",
            JSONObject().put("code", code).put("threshold", threshold).put("direction", direction))))

    suspend fun updateAlert(id: String, direction: String, threshold: Double): PriceAlert =
        PriceAlert.from(unwrap(postJson(
            "me/alerts/$id",
            JSONObject().put("direction", direction).put("threshold", threshold),
            method = "PATCH",
        )))

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

    // ---- Gamification ----

    suspend fun gamification(): GamProfile =
        GamProfile.from(unwrap(JSONObject(getRaw("me/gamification"))))

    suspend fun gamificationCheckIn(): GamCheckInResult =
        GamCheckInResult.from(unwrap(postJson("me/gamification/checkin", JSONObject())))

    suspend fun leaderboard(limit: Int = 50): Leaderboard =
        Leaderboard.from(unwrap(JSONObject(getRaw("me/gamification/leaderboard?limit=$limit"))))

    suspend fun gamificationExperience(): GamificationExperience =
        GamificationExperience.from(unwrap(JSONObject(getRaw("me/gamification/experience"))))

    suspend fun saveDailyPlan(
        risk: String,
        codes: List<String>,
        learningGoal: String,
        reflection: String,
    ): GamCheckInResult {
        val body = JSONObject()
            .put("risk_appetite", risk)
            .put("watch_codes", JSONArray(codes))
            .put("learning_goal", learningGoal)
            .put("reflection", reflection)
        return GamCheckInResult.from(unwrap(postJson("me/gamification/daily-plan", body)))
    }

    suspend fun submitQuiz(lesson: String, question: String, answer: Int): QuizAttemptResult =
        QuizAttemptResult.from(unwrap(postJson("me/gamification/quiz-attempts", JSONObject()
            .put("lesson_code", lesson)
            .put("question_code", question)
            .put("answer", answer)
            .put("idempotency_key", UUID.randomUUID().toString()))))

    suspend fun completeDiscipline(code: String, note: String): Pair<Boolean, Int> {
        val response = unwrap(postJson("me/gamification/discipline/$code", JSONObject().put("note", note)))
        return response.optBoolean("awarded") to response.optInt("xp_gained")
    }

    suspend fun createPaperTrade(
        code: String,
        side: String,
        quantity: Double,
        entryPrice: Double,
        thesis: String,
        riskBudgetPct: Double,
        invalidation: String,
    ): PaperTrade {
        val formatter = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }
        val response = unwrap(postJson("me/gamification/paper-trades", JSONObject()
            .put("client_id", UUID.randomUUID().toString())
            .put("code", code)
            .put("side", side)
            .put("quantity", quantity)
            .put("entry_price", entryPrice)
            .put("thesis", thesis)
            .put("risk_budget_pct", riskBudgetPct)
            .put("invalidation", invalidation)
            .put("occurred_at", formatter.format(Date()))))
        return PaperTrade.from(response.optJSONObject("trade") ?: response)
    }

    suspend fun reviewPaperTrade(id: String, followedPlan: Boolean, retrospective: String): Pair<PaperTrade, Int> {
        val response = unwrap(postJson(
            "me/gamification/paper-trades/$id/review",
            JSONObject().put("followed_plan", followedPlan).put("retrospective", retrospective),
            method = "PATCH",
        ))
        return PaperTrade.from(response.optJSONObject("trade") ?: JSONObject()) to response.optInt("xp_gained")
    }

    suspend fun saveGamificationPrivacy(privacy: GamificationPrivacy): GamificationPrivacy =
        GamificationPrivacy.from(unwrap(postJson(
            "me/gamification/privacy", privacy.json(), method = "PATCH",
        )))

    suspend fun createStudyGroup(name: String): StudyGroupOverview =
        StudyGroupOverview.from(unwrap(postJson(
            "me/gamification/study-groups", JSONObject().put("name", name),
        )))

    suspend fun joinStudyGroup(inviteCode: String) {
        postJson("me/gamification/study-groups/join", JSONObject().put("invite_code", inviteCode))
    }

    // ---- Portfolio ledger ----

    suspend fun portfolios(): List<PortfolioOverview> =
        parseArray(getRaw("me/portfolios"), "portfolios") { PortfolioOverview.from(it) }

    suspend fun createPortfolio(name: String): PortfolioMeta =
        PortfolioMeta.from(unwrap(postJson("me/portfolios", JSONObject().put("name", name))))

    suspend fun updatePortfolio(id: String, name: String): PortfolioMeta =
        PortfolioMeta.from(unwrap(postJson(
            "me/portfolios/$id",
            JSONObject().put("name", name),
            method = "PATCH",
        )))

    suspend fun deletePortfolio(id: String) {
        delete("me/portfolios/$id")
    }

    suspend fun portfolio(id: String): PortfolioOverview =
        PortfolioOverview.from(unwrap(JSONObject(getRaw("me/portfolios/$id"))))

    suspend fun portfolioPosition(id: String, code: String): PortfolioPositionDetail =
        PortfolioPositionDetail.from(unwrap(JSONObject(getRaw("me/portfolios/$id/positions/$code"))))

    suspend fun createPortfolioTransaction(id: String, body: JSONObject): PortfolioTransaction {
        val response = postJson("me/portfolios/$id/transactions", body)
        return PortfolioTransaction.from(response.optJSONObject("transaction") ?: unwrap(response))
    }

    suspend fun updatePortfolioTransaction(
        portfolioId: String,
        transactionId: String,
        body: JSONObject,
    ): PortfolioTransaction =
        PortfolioTransaction.from(unwrap(postJson(
            "me/portfolios/$portfolioId/transactions/$transactionId",
            body,
            method = "PATCH",
        )))

    suspend fun deletePortfolioTransaction(portfolioId: String, transactionId: String) {
        delete("me/portfolios/$portfolioId/transactions/$transactionId")
    }

    suspend fun portfolioPerformance(id: String, range: String = "1y"): List<PortfolioPerformancePoint> =
        parseArray(
            getRaw("me/portfolios/$id/performance?range=$range&points=365"),
            "points",
        ) { PortfolioPerformancePoint.from(it) }

    // ---- Account ----

    /** Permanently deletes the user and all their data (store compliance). */
    suspend fun deleteAccount() { delete("me/account") }

    // ---- Profile ----

    suspend fun profile(): Profile = Profile.from(unwrap(JSONObject(getRaw("me/profile"))))

    suspend fun updateProfile(displayName: String): Profile =
        Profile.from(unwrap(postJson("me/profile", JSONObject().put("display_name", displayName), auth = true, method = "PATCH")))

    /** Uploads avatar bytes as multipart/form-data; returns the public URL. */
    suspend fun uploadAvatar(bytes: ByteArray): String = withContext(Dispatchers.IO) {
        val part = MultipartBody.Builder().setType(MultipartBody.FORM)
            .addFormDataPart("avatar", "avatar.jpg",
                bytes.toRequestBody("image/jpeg".toMediaType(), 0, bytes.size))
            .build()
        val b = Request.Builder().url(url("me/avatar")).post(part).header("Accept", "application/json")
        tokenProvider()?.let { b.header("Authorization", "Bearer $it") }
        client.newCall(b.build()).execute().use { resp ->
            val text = resp.body?.string().orEmpty()
            if (resp.code == 401 && onRefresh != null) {
                val nt = onRefresh.invoke()
                if (!nt.isNullOrEmpty()) {
                    val retry = Request.Builder().url(url("me/avatar")).post(part)
                        .header("Accept", "application/json").header("Authorization", "Bearer $nt")
                    return@withContext client.newCall(retry.build()).execute().use { r2 ->
                        val t2 = r2.body?.string().orEmpty()
                        if (!r2.isSuccessful) throw ApiException.from(r2.code, t2.ifBlank { r2.message })
                        JSONObject(t2).optString("avatar_url")
                    }
                }
            }
            if (!resp.isSuccessful) throw ApiException.from(resp.code, text.ifBlank { resp.message })
            JSONObject(text).optString("avatar_url")
        }
    }

    // ---- Core HTTP ----

    private fun url(path: String) = "${AppConfig.apiBaseUrl}/$path"

    private suspend fun getRaw(path: String): String = execute(
        Request.Builder().url(url(path)).get()
    )

    private suspend fun postJson(path: String, body: JSONObject, auth: Boolean = true, method: String = "POST"): JSONObject {
        val reqBody = body.toString().toRequestBody(json)
        val builder = Request.Builder().url(url(path))
        when (method) {
            "PATCH" -> builder.patch(reqBody)
            "PUT" -> builder.put(reqBody)
            else -> builder.post(reqBody)
        }
        val raw = execute(builder, auth = auth)
        return if (raw.isBlank()) JSONObject() else JSONObject(raw)
    }

    private suspend fun delete(path: String) {
        execute(Request.Builder().url(url(path)).delete())
    }

    private suspend fun executeBytes(builder: Request.Builder, auth: Boolean = true, isRetry: Boolean = false): ByteArray =
        withContext(Dispatchers.IO) {
            val b = builder.header("Accept", "*/*")
            if (auth) tokenProvider()?.let { b.header("Authorization", "Bearer $it") }
            client.newCall(b.build()).execute().use { resp ->
                val bytes = resp.body?.bytes() ?: ByteArray(0)
                if (resp.code == 401 && auth && !isRetry && onRefresh != null) {
                    val newToken = onRefresh.invoke()
                    if (!newToken.isNullOrEmpty()) {
                        val retry = builder.header("Accept", "*/*")
                            .header("Authorization", "Bearer $newToken")
                        return@withContext client.newCall(retry.build()).execute().use { r2 ->
                            val b2 = r2.body?.bytes() ?: ByteArray(0)
                            if (!r2.isSuccessful) {
                                throw ApiException.from(r2.code, b2.toString(Charsets.UTF_8).ifBlank { r2.message })
                            }
                            b2
                        }
                    }
                }
                if (!resp.isSuccessful) {
                    throw ApiException.from(resp.code, bytes.toString(Charsets.UTF_8).ifBlank { resp.message })
                }
                bytes
            }
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
                            if (!r2.isSuccessful) throw ApiException.from(r2.code, t2.ifBlank { r2.message })
                            t2
                        }
                    }
                }
                if (!resp.isSuccessful) throw ApiException.from(resp.code, text.ifBlank { resp.message })
                text
            }
        }

    private fun unwrap(o: JSONObject): JSONObject =
        if (o.has("data") && o.get("data") is JSONObject) o.getJSONObject("data") else o

    @Suppress("unused")
    private fun JSONArray.objects() = buildList { for (i in 0 until length()) add(getJSONObject(i)) }
}
