package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
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
import com.example.tsc.forex.R
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.ui.theme.Th

@Composable
fun LoginScreen(auth: AuthStore, onLoggedIn: () -> Unit) {
    var isRegister by remember { mutableStateOf(false) }
    var email by remember { mutableStateOf(auth.email) }
    var password by remember { mutableStateOf("") }
    val context = LocalContext.current

    Box(
        Modifier
            .fillMaxSize()
            .background(Th.bg)
            .padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            Modifier.fillMaxWidth(),
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
                if (isRegister) "Yeni hesap oluştur" else "Hesabına giriş yap",
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

            auth.errorMessage?.let {
                Text(it, color = Th.down, fontSize = 12.sp, modifier = Modifier.fillMaxWidth())
            }

            Button(
                onClick = {
                    if (isRegister) auth.register(email, password, onLoggedIn)
                    else auth.login(email, password, onLoggedIn)
                },
                enabled = !auth.isBusy && email.isNotBlank() && password.length >= 8,
                colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 4.dp),
            ) {
                if (auth.isBusy) {
                    CircularProgressIndicator(color = Th.bg, strokeWidth = 2.dp, modifier = Modifier.size(20.dp))
                } else {
                    Text(if (isRegister) "Kayıt ol" else "Giriş yap", fontWeight = FontWeight.Bold)
                }
            }

            Button(
                onClick = { auth.loginWithGoogle(context, onLoggedIn) },
                enabled = !auth.isBusy,
                colors = ButtonDefaults.buttonColors(containerColor = Color.White, contentColor = Color.Black),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("G", color = Color(0xFF4285F4), fontWeight = FontWeight.Black, fontSize = 17.sp)
                Text("  Google ile devam et", fontWeight = FontWeight.SemiBold)
            }

            TextButton(onClick = { isRegister = !isRegister; auth.errorMessage = null }) {
                Text(
                    if (isRegister) "Zaten hesabım var • Giriş yap" else "Hesabın yok mu? • Kayıt ol",
                    color = Th.accent, fontSize = 13.sp,
                )
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
