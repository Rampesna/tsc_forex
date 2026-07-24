package com.example.tsc.forex.store

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.example.tsc.forex.auth.GoogleSignIn
import com.example.tsc.forex.auth.GoogleSignInCancelled
import com.example.tsc.forex.auth.GoogleSignInFailed
import com.example.tsc.forex.auth.PasskeyCancelled
import com.example.tsc.forex.auth.PasskeyFailed
import com.example.tsc.forex.auth.PasskeySignIn
import com.example.tsc.forex.auth.PasskeyUnavailable
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
    var displayName by mutableStateOf(prefs.displayName ?: "")
        private set
    var avatarUrl by mutableStateOf(prefs.avatarUrl ?: "")
        private set
    var userId by mutableStateOf<String?>(null)
        private set
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

    /** Credential Manager Google sign-in → backend session. */
    fun loginWithGoogle(context: Context, onSuccess: () -> Unit) {
        isBusy = true; errorMessage = null
        scope.launch {
            try {
                val idToken = GoogleSignIn.idToken(context)
                val tok = ApiClient().googleLogin(idToken)
                persist(tok)
                loadProfile()
                onSuccess()
            } catch (_: GoogleSignInCancelled) {
                errorMessage = "Google girişi iptal edildi."
            } catch (e: GoogleSignInFailed) {
                errorMessage = e.message
            } catch (e: ApiException) {
                errorMessage = when (e.code) {
                    401, 403 -> "Google hesabı doğrulanamadı (${e.code})."
                    else -> "Google girişi başarısız (${e.code})."
                }
            } catch (e: Exception) {
                errorMessage = e.message?.takeIf { it.isNotBlank() } ?: "Google girişi tamamlanamadı."
            } finally {
                isBusy = false
                errorMessage?.let { msg ->
                    try {
                        android.widget.Toast.makeText(context.applicationContext, msg, android.widget.Toast.LENGTH_LONG).show()
                    } catch (_: Exception) {
                        // ignore
                    }
                }
            }
        }
    }

    /** Discoverable passkey (WebAuthn) login via Credential Manager. */
    fun loginWithPasskey(context: Context, onSuccess: () -> Unit) {
        isBusy = true; errorMessage = null
        scope.launch {
            try {
                val tok = PasskeySignIn.login(context)
                persist(tok)
                loadProfile()
                onSuccess()
            } catch (_: PasskeyCancelled) {
                // User dismissed the picker.
            } catch (e: PasskeyUnavailable) {
                errorMessage = e.message
            } catch (e: PasskeyFailed) {
                errorMessage = e.message
            } catch (e: Exception) {
                errorMessage = e.message?.takeIf { it.isNotBlank() } ?: "Geçiş anahtarı girişi başarısız."
            } finally {
                isBusy = false
            }
        }
    }

    /** Fetches profile (name, avatar) and caches it. */
    suspend fun loadProfile() {
        try {
            val p = newApiClient().profile()
            if (p.id.isNotEmpty()) userId = p.id
            if (p.email.isNotEmpty()) { email = p.email; prefs.email = p.email }
            displayName = p.displayName; prefs.displayName = p.displayName
            avatarUrl = p.avatarUrl; prefs.avatarUrl = p.avatarUrl
        } catch (_: Exception) { /* keep cached */ }
    }

    /** Ensures [userId] is available for consent / idempotency keys. */
    suspend fun ensureUserId(): String? {
        if (!userId.isNullOrBlank()) return userId
        loadProfile()
        return userId
    }

    fun updateDisplayName(name: String) {
        scope.launch {
            try {
                val p = newApiClient().updateProfile(name)
                displayName = p.displayName; prefs.displayName = p.displayName
            } catch (e: Exception) {
                errorMessage = "Profil güncellenemedi."
            }
        }
    }

    fun uploadAvatar(bytes: ByteArray, onDone: () -> Unit = {}) {
        scope.launch {
            try {
                val url = newApiClient().uploadAvatar(bytes)
                avatarUrl = url; prefs.avatarUrl = url
                onDone()
            } catch (e: Exception) {
                errorMessage = "Fotoğraf yüklenemedi."
            }
        }
    }

    private fun run(onSuccess: () -> Unit, block: suspend () -> TokenResponse) {
        isBusy = true; errorMessage = null
        scope.launch {
            try {
                val tok = block()
                persist(tok)
                loadProfile()
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
        displayName = ""
        avatarUrl = ""
        userId = null
        onAccessTokenChanged?.invoke(null)
    }

    /**
     * Permanently deletes the account and all data on the server, then signs out.
     * Required for Google Play in-app account-deletion compliance.
     */
    fun deleteAccount(onDone: () -> Unit = {}) {
        isBusy = true; errorMessage = null
        scope.launch {
            try {
                newApiClient().deleteAccount()
            } catch (e: Exception) {
                // Even if the server call fails we still sign the user out locally.
                errorMessage = "Hesap silinemedi, tekrar deneyin."
            } finally {
                logout()
                isBusy = false
                onDone()
            }
        }
    }

    private fun persist(tok: TokenResponse) {
        prefs.accessToken = tok.accessToken
        prefs.refreshToken = tok.refreshToken
        prefs.email = email
        accessToken = tok.accessToken
        onAccessTokenChanged?.invoke(tok.accessToken)
    }
}
