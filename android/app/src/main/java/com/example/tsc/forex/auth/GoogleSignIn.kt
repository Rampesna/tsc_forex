package com.example.tsc.forex.auth

import android.content.Context
import android.util.Log
import androidx.credentials.CredentialManager
import androidx.credentials.CustomCredential
import androidx.credentials.GetCredentialRequest
import androidx.credentials.exceptions.GetCredentialCancellationException
import androidx.credentials.exceptions.GetCredentialException
import androidx.credentials.exceptions.NoCredentialException
import com.example.tsc.forex.AppConfig
import com.google.android.libraries.identity.googleid.GetGoogleIdOption
import com.google.android.libraries.identity.googleid.GetSignInWithGoogleOption
import com.google.android.libraries.identity.googleid.GoogleIdTokenCredential

class GoogleSignInCancelled : Exception("Google girişi iptal edildi")

class GoogleSignInFailed(message: String) : Exception(message)

/**
 * Runs the Credential Manager "Sign in with Google" flow and returns a Google
 * ID token whose audience is our web client ID. The backend (/auth/google)
 * verifies it and issues our own session.
 *
 * Requires a Google Cloud **Android** OAuth client with package
 * `com.talhacan.tscforex` + this build's SHA-1. Local builds and Google Play
 * builds are signed with different certificates, so each SHA-1 must have its
 * own Android OAuth client. A missing client often surfaces as
 * [GetCredentialCancellationException] right after the account is selected.
 */
object GoogleSignIn {
    private const val TAG = "GoogleSignIn"

    suspend fun idToken(context: Context): String {
        val activity = context.requireActivity()
        val manager = CredentialManager.create(activity)
        Log.i(TAG, "starting Google sign-in on ${activity.javaClass.simpleName}")

        // Prefer the explicit Sign-In button flow (matches the UI CTA). Fall back
        // to GetGoogleIdOption only when no credential UI is available.
        try {
            return requestToken(
                manager,
                activity,
                GetCredentialRequest.Builder()
                    .addCredentialOption(
                        GetSignInWithGoogleOption.Builder(AppConfig.GOOGLE_WEB_CLIENT_ID).build(),
                    )
                    .build(),
                label = "GetSignInWithGoogleOption",
            )
        } catch (e: NoCredentialException) {
            Log.w(TAG, "Sign-In button flow: no credential, trying GetGoogleIdOption")
        }

        try {
            return requestToken(
                manager,
                activity,
                GetCredentialRequest.Builder()
                    .addCredentialOption(
                        GetGoogleIdOption.Builder()
                            .setFilterByAuthorizedAccounts(false)
                            .setServerClientId(AppConfig.GOOGLE_WEB_CLIENT_ID)
                            .setAutoSelectEnabled(false)
                            .build(),
                    )
                    .build(),
                label = "GetGoogleIdOption",
            )
        } catch (e: NoCredentialException) {
            throw GoogleSignInFailed(
                "Cihazda kullanılabilir Google hesabı bulunamadı.",
            )
        }
    }

    private suspend fun requestToken(
        manager: CredentialManager,
        activity: android.app.Activity,
        request: GetCredentialRequest,
        label: String,
    ): String {
        try {
            val result = manager.getCredential(activity, request)
            val cred = result.credential
            Log.i(TAG, "$label ok type=${cred.type}")
            if (cred is CustomCredential &&
                cred.type == GoogleIdTokenCredential.TYPE_GOOGLE_ID_TOKEN_CREDENTIAL
            ) {
                val token = GoogleIdTokenCredential.createFrom(cred.data).idToken
                if (token.isBlank()) throw GoogleSignInFailed("Google ID token boş döndü.")
                return token
            }
            throw GoogleSignInFailed("Beklenmeyen kimlik türü (${cred.type})")
        } catch (e: GetCredentialCancellationException) {
            val detail = e.errorMessage?.toString()?.ifBlank { null } ?: e.message
            Log.e(TAG, "$label CANCELLED type=${e.type} detail=$detail", e)
            // Misconfigured Android OAuth client commonly looks like a user cancel.
            val looksLikeConfig =
                detail?.contains("reauth", ignoreCase = true) == true ||
                    detail?.contains("developer console", ignoreCase = true) == true
            if (looksLikeConfig) {
                throw GoogleSignInFailed(
                    "Google girişi şu anda tamamlanamıyor. Lütfen birkaç dakika sonra tekrar deneyin.",
                )
            }
            throw GoogleSignInCancelled()
        } catch (e: NoCredentialException) {
            Log.w(TAG, "$label no credential: ${e.type} ${e.errorMessage}", e)
            throw e
        } catch (e: GetCredentialException) {
            Log.e(TAG, "$label failed: type=${e.type} msg=${e.errorMessage}", e)
            val detail = e.errorMessage?.toString()?.ifBlank { null }
                ?: e.type.takeIf { it.isNotBlank() }
                ?: e.message
            throw GoogleSignInFailed(
                when {
                    detail?.contains("10:", ignoreCase = true) == true ||
                        detail?.contains("Developer console", ignoreCase = true) == true ->
                        "Google girişi şu anda kullanılamıyor. Lütfen daha sonra tekrar deneyin."
                    else -> "Google girişi tamamlanamadı. Lütfen tekrar deneyin."
                },
            )
        }
    }
}
