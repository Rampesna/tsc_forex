package com.example.tsc.forex.auth

import android.content.Context
import androidx.credentials.CredentialManager
import androidx.credentials.GetCredentialRequest
import androidx.credentials.GetPublicKeyCredentialOption
import androidx.credentials.PublicKeyCredential
import androidx.credentials.exceptions.GetCredentialCancellationException
import androidx.credentials.exceptions.GetCredentialException
import androidx.credentials.exceptions.NoCredentialException
import com.example.tsc.forex.data.ApiClient
import com.example.tsc.forex.data.ApiException
import com.example.tsc.forex.data.TokenResponse
import org.json.JSONObject

class PasskeyCancelled : Exception("Geçiş anahtarı girişi iptal edildi")
class PasskeyUnavailable(message: String = "Bu cihazda kayıtlı geçiş anahtarı bulunamadı") : Exception(message)
class PasskeyFailed(message: String) : Exception(message)

/**
 * Usernameless (discoverable) passkey login via Credential Manager (WebAuthn).
 *
 * Requires Digital Asset Links at https://<rpId>/.well-known/assetlinks.json
 * for package com.talhacan.tscforex + the app signing SHA-256.
 */
object PasskeySignIn {
    suspend fun login(context: Context): TokenResponse {
        val activity = context.requireActivity()
        val api = ApiClient()
        val begin = try {
            api.passkeyLoginBegin()
        } catch (e: ApiException) {
            throw PasskeyFailed("Sunucu geçiş anahtarı oturumu başlatamadı (${e.code}).")
        } catch (e: Exception) {
            throw PasskeyFailed(e.message ?: "Sunucuya ulaşılamadı.")
        }

        val session = begin.optString("session")
        if (session.isBlank()) throw PasskeyFailed("Geçiş anahtarı oturumu alınamadı.")

        // go-webauthn returns {"publicKey": {...}}. Credential Manager wants the
        // inner PublicKeyCredentialRequestOptionsJSON.
        val options = begin.optJSONObject("options") ?: begin
        val requestJson = (options.optJSONObject("publicKey") ?: options).toString()

        val option = GetPublicKeyCredentialOption(requestJson)
        val request = GetCredentialRequest.Builder().addCredentialOption(option).build()
        try {
            val result = CredentialManager.create(activity).getCredential(activity, request)
            val cred = result.credential as? PublicKeyCredential
                ?: throw PasskeyFailed("Beklenmeyen kimlik türü")
            val responseJson = cred.authenticationResponseJson
            return try {
                api.passkeyLoginFinish(session, JSONObject(responseJson))
            } catch (e: ApiException) {
                throw PasskeyFailed("Geçiş anahtarı doğrulanamadı (${e.code}).")
            }
        } catch (e: GetCredentialCancellationException) {
            throw PasskeyCancelled()
        } catch (e: NoCredentialException) {
            throw PasskeyUnavailable(
                "Bu cihazda bu uygulama için kayıtlı geçiş anahtarı yok. " +
                    "Önce iOS’ta veya web’de passkey ekleyin; Android Digital Asset Links eşleşmeli.",
            )
        } catch (e: GetCredentialException) {
            val detail = e.errorMessage?.toString()?.ifBlank { null } ?: e.type.ifBlank { e.message }
            throw PasskeyFailed(
                detail?.let { "Geçiş anahtarı girişi başarısız: $it" }
                    ?: "Geçiş anahtarı girişi başarısız (Digital Asset Links / RP domain).",
            )
        }
    }
}
