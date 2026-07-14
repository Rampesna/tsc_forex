package com.example.tsc.forex.auth

import android.content.Context
import androidx.credentials.CredentialManager
import androidx.credentials.GetCredentialRequest
import androidx.credentials.GetPublicKeyCredentialOption
import androidx.credentials.PublicKeyCredential
import androidx.credentials.exceptions.GetCredentialCancellationException
import androidx.credentials.exceptions.NoCredentialException
import com.example.tsc.forex.data.ApiClient
import com.example.tsc.forex.data.TokenResponse
import org.json.JSONObject

class PasskeyCancelled : Exception("Geçiş anahtarı girişi iptal edildi")
class PasskeyUnavailable : Exception("Bu cihazda kayıtlı geçiş anahtarı bulunamadı")

/**
 * Usernameless (discoverable) passkey login via Credential Manager (WebAuthn).
 *
 * Flow: POST /auth/passkey/login/begin → get assertion options + opaque session,
 * hand the options to Credential Manager, then POST the authenticator response to
 * /auth/passkey/login/finish which verifies it and returns our session tokens.
 *
 * NOTE: Requires Digital Asset Links (assetlinks.json) served at the RP domain
 * (tsc-api.talhacan.com) listing this app's package + release SHA-256, plus at
 * least one passkey previously registered for the account.
 */
object PasskeySignIn {
    suspend fun login(context: Context): TokenResponse {
        val api = ApiClient()
        val begin = api.passkeyLoginBegin()
        val session = begin.optString("session")
        // go-webauthn returns {"publicKey": {...}}. Credential Manager wants the
        // inner PublicKeyCredentialRequestOptionsJSON.
        val options = begin.optJSONObject("options") ?: begin
        val requestJson = (options.optJSONObject("publicKey") ?: options).toString()

        val option = GetPublicKeyCredentialOption(requestJson)
        val request = GetCredentialRequest.Builder().addCredentialOption(option).build()
        try {
            val result = CredentialManager.create(context).getCredential(context, request)
            val cred = result.credential as? PublicKeyCredential
                ?: error("Beklenmeyen kimlik türü")
            val responseJson = cred.authenticationResponseJson
            return api.passkeyLoginFinish(session, JSONObject(responseJson))
        } catch (e: GetCredentialCancellationException) {
            throw PasskeyCancelled()
        } catch (e: NoCredentialException) {
            throw PasskeyUnavailable()
        }
    }
}
