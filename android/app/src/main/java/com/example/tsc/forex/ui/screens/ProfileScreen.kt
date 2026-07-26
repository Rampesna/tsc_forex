package com.example.tsc.forex.ui.screens

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.theme.Th
import java.io.ByteArrayOutputStream

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileSheet(auth: AuthStore, onDismiss: () -> Unit) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val context = LocalContext.current
    var name by remember { mutableStateOf(auth.displayName) }
    var savedName by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        auth.loadProfile()
        name = auth.displayName
    }

    val picker = rememberLauncherForActivityResult(
        ActivityResultContracts.PickVisualMedia(),
    ) { uri ->
        if (uri != null) {
            val bytes = downscaleFromUri(context, uri)
            if (bytes != null) auth.uploadAvatar(bytes)
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState, containerColor = Th.bg) {
        Column(
            Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 20.dp)
                .padding(bottom = 36.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            RowHeader(title = "Profil", subtitle = "Hesap bilgileri", onClose = onDismiss)

            Card {
                Column(
                    Modifier.fillMaxWidth().padding(vertical = 6.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Box(
                        Modifier.size(96.dp).clip(CircleShape).background(Th.accent.copy(alpha = 0.18f)),
                        contentAlignment = Alignment.Center,
                    ) {
                        if (auth.avatarUrl.isNotEmpty()) {
                            AsyncImage(
                                model = auth.avatarUrl,
                                contentDescription = null,
                                contentScale = ContentScale.Crop,
                                modifier = Modifier.fillMaxSize(),
                            )
                        } else {
                            Text(
                                (auth.displayName.ifEmpty { auth.email }).take(1).uppercase(),
                                color = Th.accent,
                                fontWeight = FontWeight.Bold,
                                fontSize = 34.sp,
                            )
                        }
                    }
                    TextButton(
                        onClick = {
                            picker.launch(
                                PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly),
                            )
                        },
                    ) { Text("Fotoğrafı değiştir", color = Th.accent, fontWeight = FontWeight.SemiBold) }
                    Text(
                        auth.displayName.ifEmpty { auth.email.ifEmpty { "Kullanıcı" } },
                        color = Th.text,
                        fontSize = 18.sp,
                        fontWeight = FontWeight.Bold,
                    )
                    if (auth.email.isNotEmpty()) {
                        Text(auth.email, color = Th.textDim, fontSize = 13.sp)
                    }
                }
            }

            Card {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("GÖRÜNEN AD", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    OutlinedTextField(
                        value = name,
                        onValueChange = { name = it; savedName = false },
                        label = { Text("Görünen ad") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Button(
                        onClick = {
                            auth.updateDisplayName(name.trim())
                            savedName = true
                        },
                        enabled = name.isNotBlank(),
                        shape = RoundedCornerShape(10.dp),
                        colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(if (savedName) "Kaydedildi ✓" else "Kaydet", fontWeight = FontWeight.Bold) }
                }
            }
        }
    }
}

@Composable
private fun RowHeader(title: String, subtitle: String, onClose: () -> Unit) {
    Box(Modifier.fillMaxWidth()) {
        IconButton(onClick = onClose, modifier = Modifier.align(Alignment.CenterStart)) {
            Icon(Icons.AutoMirrored.Filled.ArrowBack, "Kapat", tint = Th.text)
        }
        Column(
            Modifier.align(Alignment.Center),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(title, color = Th.text, fontSize = 22.sp, fontWeight = FontWeight.Bold)
            Text(subtitle, color = Th.textDim, fontSize = 12.sp)
        }
    }
}

private fun downscaleFromUri(context: android.content.Context, uri: android.net.Uri): ByteArray? {
    return try {
        val input = context.contentResolver.openInputStream(uri) ?: return null
        val original = input.use { BitmapFactory.decodeStream(it) } ?: return null
        val maxSide = 512
        val scale = minOf(1f, maxSide.toFloat() / maxOf(original.width, original.height))
        val scaled = if (scale < 1f) {
            Bitmap.createScaledBitmap(
                original,
                (original.width * scale).toInt(),
                (original.height * scale).toInt(),
                true,
            )
        } else original
        ByteArrayOutputStream().use { out ->
            scaled.compress(Bitmap.CompressFormat.JPEG, 85, out)
            out.toByteArray()
        }
    } catch (_: Exception) {
        null
    }
}
