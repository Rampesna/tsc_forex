package com.example.tsc.forex.auth

import android.content.Context
import androidx.credentials.CredentialManager
import androidx.credentials.GetCredentialRequest
import androidx.credentials.exceptions.GetCredentialCancellationException
import com.example.tsc.forex.AppConfig
import com.google.android.libraries.identity.googleid.GetSignInWithGoogleOption
import com.google.android.libraries.identity.googleid.GoogleIdTokenCredential

class GoogleSignInCancelled : Exception("Google girişi iptal edildi")

/**
 * Runs the Credential Manager "Sign in with Google" flow and returns a Google
 * ID token whose audience is our web client ID. The backend (/auth/google)
 * verifies it and issues our own session.
 */
object GoogleSignIn {
    suspend fun idToken(context: Context): String {
        val option = GetSignInWithGoogleOption.Builder(AppConfig.GOOGLE_WEB_CLIENT_ID).build()
        val request = GetCredentialRequest.Builder().addCredentialOption(option).build()
        try {
            val result = CredentialManager.create(context).getCredential(context, request)
            val cred = result.credential
            if (cred is androidx.credentials.CustomCredential &&
                cred.type == GoogleIdTokenCredential.TYPE_GOOGLE_ID_TOKEN_CREDENTIAL
            ) {
                return GoogleIdTokenCredential.createFrom(cred.data).idToken
            }
            error("Beklenmeyen kimlik türü")
        } catch (e: GetCredentialCancellationException) {
            throw GoogleSignInCancelled()
        }
    }
}
