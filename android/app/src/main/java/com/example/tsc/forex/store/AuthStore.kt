package com.example.tsc.forex.store

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.example.tsc.forex.data.ApiClient
import com.example.tsc.forex.data.ApiException
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.data.TokenResponse
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

/** Holds the auth session and exposes login/register/refresh/logout. */
class AuthStore(private val prefs: Prefs) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private val refreshMutex = Mutex()

    var accessToken by mutableStateOf(prefs.accessToken)
        private set
    var email by mutableStateOf(prefs.email ?: "")
    var isBusy by mutableStateOf(false)
        private set
    var errorMessage by mutableStateOf<String?>(null)

    val isLoggedIn: Boolean get() = !accessToken.isNullOrEmpty()

    /** Called whenever the access token changes so the WS can reconnect. */
    var onAccessTokenChanged: ((String?) -> Unit)? = null

    /** Build an API client wired to this session (auto-refresh on 401). */
    fun newApiClient() = ApiClient(
        tokenProvider = { accessToken },
        onRefresh = { refresh() },
    )

    fun login(email: String, password: String, onSuccess: () -> Unit) =
        run(onSuccess) { newApiClient().login(email.trim(), password) }

    fun register(email: String, password: String, onSuccess: () -> Unit) =
        run(onSuccess) { newApiClient().register(email.trim(), password) }

    private fun run(onSuccess: () -> Unit, block: suspend () -> TokenResponse) {
        isBusy = true; errorMessage = null
        scope.launch {
            try {
                val tok = block()
                persist(tok)
                onSuccess()
            } catch (e: ApiException) {
                errorMessage = when (e.code) {
                    401 -> "E-posta veya şifre hatalı."
                    409 -> "Bu e-posta zaten kayıtlı."
                    else -> "İşlem başarısız (${e.code})."
                }
            } catch (e: Exception) {
                errorMessage = "Sunucuya ulaşılamadı."
            } finally {
                isBusy = false
            }
        }
    }

    /** Single-flight token refresh. Returns the new access token or null. */
    suspend fun refresh(): String? = refreshMutex.withLock {
        val rt = prefs.refreshToken ?: return null
        try {
            val tok = ApiClient().refresh(rt)
            persist(tok)
            tok.accessToken
        } catch (e: Exception) {
            logout()
            null
        }
    }

    fun logout() {
        prefs.clearSession()
        accessToken = null
        onAccessTokenChanged?.invoke(null)
    }

    private fun persist(tok: TokenResponse) {
        prefs.accessToken = tok.accessToken
        prefs.refreshToken = tok.refreshToken
        prefs.email = email
        accessToken = tok.accessToken
        onAccessTokenChanged?.invoke(tok.accessToken)
    }
}
