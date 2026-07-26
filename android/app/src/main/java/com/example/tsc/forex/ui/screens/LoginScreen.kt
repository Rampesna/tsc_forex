package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import com.example.tsc.forex.ui.LocalizedText as Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.zIndex
import com.example.tsc.forex.R
import com.example.tsc.forex.auth.findActivity
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.ui.appToast
import com.example.tsc.forex.ui.theme.Th

@Composable
fun LoginScreen(
    auth: AuthStore,
    onLoggedIn: () -> Unit,
    asModal: Boolean = false,
    onContinueAsGuest: (() -> Unit)? = null,
) {
    // Use an in-Activity full-screen overlay (not Compose Dialog). Credential
    // Manager attaches to the Activity window; a separate Dialog window often
    // cancels the Google / passkey result after the account picker closes.
    if (asModal) {
        Box(
            Modifier
                .fillMaxSize()
                .zIndex(100f)
                .background(Th.bg.copy(alpha = 0.96f)),
        ) {
            LoginBody(auth, onLoggedIn, onContinueAsGuest)
        }
    } else {
        LoginBody(auth, onLoggedIn, onContinueAsGuest)
    }
}

@Composable
private fun LoginBody(
    auth: AuthStore,
    onLoggedIn: () -> Unit,
    onContinueAsGuest: (() -> Unit)?,
) {
    var isRegister by remember { mutableStateOf(false) }
    var email by remember { mutableStateOf(auth.email) }
    var password by remember { mutableStateOf("") }
    val context = LocalContext.current
    // Credential Manager requires a real Activity; Dialog LocalContext alone is not enough.
    val activity = context.findActivity()

    fun startGoogle() {
        val host = activity
        if (host == null) {
            auth.errorMessage = "Google girişi için Activity bulunamadı."
            context.appToast(auth.errorMessage!!)
            return
        }
        auth.loginWithGoogle(host, onLoggedIn)
    }

    fun startPasskey() {
        val host = activity
        if (host == null) {
            auth.errorMessage = "Geçiş anahtarı için Activity bulunamadı."
            context.appToast(auth.errorMessage!!)
            return
        }
        auth.loginWithPasskey(host, onLoggedIn)
    }

    Box(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
            .statusBarsPadding()
            .navigationBarsPadding()
            .imePadding(),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 28.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(14.dp, Alignment.CenterVertically),
        ) {
            Column(
                Modifier.widthIn(max = 480.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                Image(
                    painter = painterResource(R.drawable.brand_logo),
                    contentDescription = "TSC Forex",
                    contentScale = ContentScale.Fit,
                    modifier = Modifier
                        .size(84.dp)
                        .clip(RoundedCornerShape(20.dp)),
                )

                Text("TSC Forex", color = Th.text, fontSize = 26.sp, fontWeight = FontWeight.Bold)
                Text(
                    "Canlı Altın & Döviz Piyasası",
                    color = Th.textDim, fontSize = 13.sp,
                )

                OutlinedTextField(
                value = email,
                onValueChange = { email = it; auth.email = it },
                label = { Text("E-posta") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                colors = fieldColors(),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text("Şifre") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                colors = fieldColors(),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth(),
                )

            // Keep errors sticky on the login form. Do NOT use ToastOnChange here —
            // it cleared errorMessage immediately so the UI looked "dead".
                auth.errorMessage?.let { msg ->
                    Text(
                        msg,
                        color = Th.down,
                        fontSize = 12.sp,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }

                Button(
                onClick = {
                    if (isRegister) auth.register(email, password, onLoggedIn)
                    else auth.login(email, password, onLoggedIn)
                },
                enabled = !auth.isBusy && email.isNotBlank() && password.length >= 8,
                colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth().heightIn(min = 52.dp).padding(top = 4.dp),
            ) {
                if (auth.isBusy) {
                    CircularProgressIndicator(color = Th.bg, strokeWidth = 2.dp, modifier = Modifier.size(20.dp))
                } else {
                    Text(if (isRegister) "Kayıt ol" else "Giriş yap", fontWeight = FontWeight.Bold)
                }
                }

                Button(
                onClick = ::startGoogle,
                enabled = !auth.isBusy,
                colors = ButtonDefaults.buttonColors(containerColor = Color.White, contentColor = Color.Black),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth().heightIn(min = 52.dp),
            ) {
                Text("G", color = Color(0xFF4285F4), fontWeight = FontWeight.Black, fontSize = 17.sp)
                Text("  Google ile devam et", fontWeight = FontWeight.SemiBold)
                }

                OutlinedButton(
                onClick = ::startPasskey,
                enabled = !auth.isBusy,
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth().heightIn(min = 52.dp),
            ) {
                Text("Geçiş anahtarı ile giriş", color = Th.text, fontWeight = FontWeight.SemiBold)
                }

                TextButton(onClick = { isRegister = !isRegister; auth.errorMessage = null }) {
                    Text(
                        if (isRegister) "Zaten hesabım var • Giriş yap" else "Hesabın yok mu? • Kayıt ol",
                        color = Th.accent, fontSize = 13.sp,
                    )
                }

                if (onContinueAsGuest != null) {
                    TextButton(onClick = onContinueAsGuest) {
                        Text("Üye olmadan piyasaları incele", color = Th.textDim, fontSize = 13.sp, fontWeight = FontWeight.SemiBold)
                    }
                }
            }
        }
    }
}

@Composable
private fun fieldColors() = OutlinedTextFieldDefaults.colors(
    focusedBorderColor = Th.accent,
    unfocusedBorderColor = Th.stroke,
    focusedTextColor = Th.text,
    unfocusedTextColor = Th.text,
    focusedLabelColor = Th.accent,
    unfocusedLabelColor = Th.textDim,
    cursorColor = Th.accent,
    focusedContainerColor = Th.surface,
    unfocusedContainerColor = Th.surface,
)
